# handlers 模块

> 路径: `handlers/` | 核心职责: 所有 HTTP 接口的业务逻辑处理

---

## 文件结构

| 文件 | 行数 | 职责 |
|---|---|---|
| `users.go` | — | 用户管理（登录/登出/改密/踢人/在线列表） |
| `system.go` | 1500+ | 系统全局（序列号、主机名、国际化、网络配置、时区、进程管理、日志） |
| `faultrecording.go` | 1300+ | 录波文件管理（解析、查询、下载、导出） |
| `agcconfig.go` | — | AGC 策略与设备模型 CRUD |
| `dbconfig.go` | — | RTU/设备/测点配置 CRUD |
| `feconfig.go` | — | 前置通道组/通道配置 |
| `datamonitor.go` | — | 数据监控（通道状态、测点数据、遥控遥调） |
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
