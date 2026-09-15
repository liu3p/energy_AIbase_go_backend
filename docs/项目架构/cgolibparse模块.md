# cgolibparse 模块

> 路径: `cgolibparse/` | 核心职责: Cgo 桥接层，调用 C++ 库

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `cgodbmsg.go` | **实时数据查询核心**（DBMsgManager 单例，OID 查询，遥控遥调） |
| `cgoformula.go` | 公式解析接口（eq_validate / eq_eval） |
| `cgoagcapi.go` | AGC 计算接口（Cgo 调用 C++） |
| `cgofeparse.go` | FE（前置）解析桥接 |
| `libparsesrc/cgo_pluginparse.cpp` | C++ 实现：FES 通道消息处理 |

---

## DBMsgManager — 实时数据查询（`cgodbmsg.go`）

### 架构

```
DBMsgManager（单例）
    │
    ├─ 启动时：DBCGOInit() 初始化 C 库
    │
    ├─ 后台协程 startGetPointsValue()：每 2 秒轮询
    │   ├─ 遍历 allPointsID（已注册的 OID，uint64 编码）
    │   ├─ 按 tableNo 分组（遥测/遥信/...）
    │   └─ 调用 C.GetFieldsValues() 批量查询 → 缓存到 pointsValueMap
    │
    └─ 对外接口：RegisterPointsID(idNameMap, registerIDs)
        ├─ getValue()    → 从缓存读取 []PointValue
        └─ unRegister() → 取消订阅（引用计数归零时真正删除）
```

### 使用方式

```go
mgr, _ := cgolibparse.GetDBMsgManagerInstance()

// 注册
getValue, unRegister, err := mgr.RegisterPointsID(idNameMap, []string{"1-1-102-1"})

// 读取
values := getValue()

// 取消
unRegister()
```

### PointValue 结构

```go
type PointValue struct {
    OID          uint64 // uint64 编码
    PointID      string // 原始 OID 字符串
    Name         string // 显示名
    CurrentValue string // 当前值（主显示值）
    RawValue     string // 原始值
    CtlValue     string // 遥调遥控预置值
    Dead         string // 死区值
    HisPeriod    string // 历史存盘周期
    Quality      string // 质量码
    SendTime     string // 发送时间
}
```

### Cgo 调用链

```go
C.DBCGOInit()                    // 初始化 C 库
C.GetFieldsValues(               // 批量查询字段值
    clientID,                    // 唯一客户端 ID
    oids,                        // OID 数组（uint64）
    oidCount,
    fields,                      // 查询字段名数组
    fieldCount,
    &jsonResult,                 // 返回 JSON
)
// 返回：[{oid: uint64, values: [raw, curr, ctl, dead, hisperiod, quality, sendtime]}]
```

### 引用计数机制

`RegisterPointsID` 内部维护 `pointsIDCount` 引用计数：
- 多个调用者注册同一 OID → 计数 +1
- `unRegister()` → 计数 -1
- 计数归零 → 真正从轮询列表移除

---

## 其他 Cgo 接口

### 遥控遥调（`cgodbmsg.go`）

| 函数 | 说明 |
|---|---|
| `SendYkCmd(pointID, cmd, action)` | 遥控命令（分合闸） |
| `SendYtCmd(pointID, cmd, value, isFast)` | 遥调命令（设定值） |
| `SendManualSetCmd(pointID, flag, value, type)` | 手动置值 |

### 公式解析（`cgoformula.go`）

| 函数 | 说明 |
|---|---|
| `EqValidate(expr)` | 验证公式合法性 |
| `EqValidateWithVars(expr, varNames)` | 带变量名验证 |
| `EqEval(expr, varNames, varValues)` | 计算公式值 |

### FES 通道消息（`cgo_pluginparse.cpp`）

| 消息类型 | 常量 | 说明 |
|---|---|---|
| `MT_DISPLAY_CODE` | 11003 | 显示编码（通道日志） |
| `MT_SUBSCRIBE_CHANNEL_STATUS` | 11021 | 通道状态订阅 |
| `MT_RELOAD_RESULT` | 11011 | 重载结果 |

---

## 构建说明

```bash
CGO_LDFLAGS="-L${ARM_SF}/lib/scada -L${ARM_SF}/lib" go build -o arm-web-backend
```

- 依赖 `${ARM_SF}/lib/scada` 和 `${ARM_SF}/lib` 下的 C++ 库（`libdb_client_cgo`、`libscada_client`、`libequation`）
- 目标平台为 ARM 嵌入式 Linux

---

## 对 C++ 工程师的意义

这是项目中与 C++ 直接相关的部分，你对 C++ 代码逻辑（Poco JSON、内存流、mail slot 通信机制、C 接口设计）的理解在这里最有价值。

---

## 相关笔记

- [[项目架构总览]]
- [[handlers模块]]
- [[首页模块设计]]
