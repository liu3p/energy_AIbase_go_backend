# handlers 模块

> 路径: `handlers/` | 核心职责: 所有 HTTP 接口的业务逻辑处理

---

## 文件结构

| 文件 | 行数 | 职责 |
|---|---|---|
| `users.go` | — | 用户管理（登录/登出/改密/踢人/在线列表） |
| `license.go` | — | 注册码许可（机器码查询/注册码提交/状态校验） |
| `system.go` | 1500+ | 系统全局（序列号、主机名、国际化、网络配置、时区、进程管理、日志） |
| `faultrecording.go` | 1300+ | 录波文件管理（解析、查询、下载、导出） |
| `agcconfig.go` | — | AGC 策略与设备模型 CRUD |
| `dbconfig.go` | — | RTU/设备/测点配置 CRUD |
| `feconfig.go` | — | 前置通道组/通道配置 |
| `datamonitor.go` | — | 数据监控（通道状态、测点数据、遥控遥调） |
| `sort_helper.go` | — | 通用排序工具（Parameters 字母序 + 特殊参数优先） |
| `operation.go` | — | 升级/备份/下载 |
| `ems.go` | — | EMS 网关查询（遥测/遥信/属性） |
| `gateway.go` | — | 网关信息管理 |
| `wshandler.go` | — | WebSocket 连接管理 |
| `wssysteminfo.go` | — | 系统信息 WebSocket 推送 |
| `wslogpush.go` | — | 日志推送 WebSocket |
| `rediscall.go` | — | Redis 调用相关 |

---

## 核心全局变量

```go
var ClientMgr *emsclient.Manager     // EMS 客户端管理器
var OnlineUsers sync.Map             // 在线用户表 (uname → *OnlineUser)
```

### OnlineUser 结构

```go
type OnlineUser struct {
    Uname            string
    Utype            string            // admin/config/ops/guest
    State            int               // 1=已登录, 2=WebSocket 在线
    Pwd              string
    AccessToken      string
    AccessTokenExp   int64
    RefreshToken     string
    RefreshTokenExp  int64
    PushConn         *websocket.Conn   // WebSocket 连接
    LastUpdateCache  int64
}
```

---

## 国际化数据

```go
var I18nData      map[string]TransEntry    // i18n 翻译 (zh/en)
var DataUnit       map[string]DataUnitEntry // 点位单位与取值
var I18nDataModbus map[string]TransEntry    // Modbus 国际化
var DataUnitModbus map[string]DataUnitEntry // Modbus 单位
var LoginRecordData map[string]interface{}  // 登录记录
var AllModelInfo   map[string]interface{}   // 总览模型信息
```

---

## 关键处理流程

### 录波文件 (`faultrecording.go`)

1. 查找录波文件 → 按日期组织路径 `/home/eco_ems/eco_ems/dat/wave/{date}/{file}`
2. 解析二进制文件 → GBK 路径转换（`utf8Str2gbkStr`）
3. 提取 YC（遥测）统计数据 + YX（遥信）变位记录
4. 支持前端展示波形数据

### Parameters 排序 (`sort_helper.go`)

对 `channel.plugins.parameters` 数组排序：`STATION_ID ASDU_ADDR` 置顶，其余按字母序。通过拷贝+排序实现，不修改 fe_config 原始数据。

涉及接口：`GetRTUs`、`GetRTUsExceptPoints`、`GetAllChannels`、`GetChannelPlugins`、`GetRtuChannels`。

### 协议多层化 (`feconfig.go` + `sort_helper.go`)

通道 `plugins` 由「APP/LINK 两层」改为「按插件 `id` 区间分层」（层号 = `id/1000+1`）：每层至多一个、必须从第 1 层起连续；`plugin.type` 由 `"APP"/"LINK"` 变为层号字符串 `"1"/"2"`。`GetChannelPlugins` 返回按层分组的数组；`UpdateRTU` / `AddCGChannel` / `UpdateCGChannel` 写入前做层连续性校验（不合法返回 400）。详见 [RTU通道合并接口说明](../接口文档/RTU通道合并接口说明.md)。

### 遥测数据格式化 (`datamonitor.go`)

WS 接口 `GetPointsData` 推送 analog 数据时，`rawvalue` 和 `currvalue` 保留三位小数。通过 `formatAnalogValues` 拷贝+格式化实现，不修改 C++ 层原始数据。

### station_model 参数值格式化 (`agcconfig.go`)

`GetAgcData` 接口返回 `station_model` 时，递归遍历 JSON 树，对 `para[].value` 和 `dyn_para[].value` 中数字形式的值保留至多三位小数。

- `para[].value`（string 类型）：格式化后仍为 string，非数字原样返回
- `dyn_para[].value`（float64 类型）：格式化后转为 string，保留三位小数，非数字原样返回
- 通过 `formatStationModelParaValues` 递归拷贝+格式化实现，不修改 8500 服务原始数据
- `strategy_model` 不涉及此处理

### homepage show_value 格式化 (`homepage.go` + `homepage/format.go`)

**config 接口**：`GetHomepageConfig` 返回首页配置时，对 `type == 1`（遥测 Analog）且 `show_value` 为数字形式的值保留至多三位小数；非数字或 `type != 1` 时不变。通过 `formatHomepageConfigShowValue` 拷贝+格式化实现，不修改 ConfigManager 内存中的原始配置。

**WebSocket 推送**：`/ws/push` 的 `homepage_data` topic 推送实时数据时，`value` 字段为数字形式时保留至多三位小数，仍以字符串返回。通过 `transformToHomepageData` 中调用 `roundToAtMost3Decimal` 格式化，不修改 C++ 层原始数据。

- `roundToAtMost3Decimal` 在 `homepage/format.go` 内定义（`homepage` 包无法引用 `handlers` 包）
- `handlers/agcconfig.go` 内有同名函数，用于 config 接口的 show_value 和 station_model 格式化

### 注册码许可 (`license.go`)

文件写于 `$ARM_SF/etc/license/`（`license.dat` 注册码、`machine_serial` 机器码），写前 `MkdirAll` 兜底建目录。设计细节见 [[功能设计/注册码license功能实现方案]]。

| Handler | 说明 |
|---|---|
| `GetMachineSerial` | 查询机器码：优先读持久化的 `machine_serial`；不存在则调用 Cgo `GenerateMachineSerial()` 生成并落盘，保证前端每次查询一致 |
| `RegisterLicense` | 提交注册码：`ValidateLicense` 校验通过后经 `SaveLicenseToFile` 写入 `license.dat`（签名错误/机码不匹配/已过期均 400） |
| `GetLicenseStatus` / `GetLicenseStatusData` | 状态：读文件原文实时校验，三态——有效 / 无效（含已过期）/ 未注册（文件不存在/为空） |

- **登录合并**：`LoginUser` 登录成功后调用 `GetLicenseStatusData()`，把许可证状态并入响应 `license` 字段（语义同 `/license/status`）；登录失败不触发读取。

### 进程管理 (`wssysteminfo.go`)

- 通过 ZMQ 获取系统信息（CPU/内存/磁盘）
- 通过 WebSocket 实时推送进程状态
- 支持进程启停操作

---

## 相关笔记

- [[项目架构总览]]
- [[API路由地图]]
- [[emsclient模块]]
- [[用户认证体系]]
- [[WebSocket推送机制]]
- [[功能设计/注册码license功能实现方案]]
