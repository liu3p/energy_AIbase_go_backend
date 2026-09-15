# emsclient 模块

> 路径: `emsclient/` | 核心职责: 与 EMS 核心进程建立 TCP RPC 连接，发送命令/接收响应

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `emsrpcclient.go` | 底层 RPC 客户端（TCP 连接、消息收发、序列号管理） |
| `emsrpclientmgr.go` | 客户端管理器（Manager），管理 WebSocket 连接、消息推送 |
| `emsrpcgateway.go` | 网关查询接口（遥测/遥信/属性/设备/遥控） |
| `emsrpcuser.go` | 用户相关 RPC |
| `emsrpcsyscall.go` | 系统调用 RPC |
| `emsrpcredis.go` | Redis 请求暂存 |

---

## RPC 协议

**自定义 TCP 二进制协议**，消息结构：

```
┌──────────┬──────────┬─────────┬──────────┐
│ bodyLen  │  msgSeq  │ msgType │  method  │  ← 13 字节头（BigEndian）
│ (4 byte) │ (4 byte) │ (1 byte)│ (4 byte) │
├──────────┴──────────┴─────────┴──────────┤
│              body (JSON)                  │  ← 变长消息体
└─────────────────────────────────────────┘
```

**消息类型常量** (`emsrpcclient.go`):

| 常量 | 值 | 说明 |
|---|---|---|
| `MessageTypeCall` | 10 | 调用请求 |
| `MessageTypeReply` | 11 | 回复响应 |
| `MessageTypePush` | 12 | 推送（无需回复） |
| `MessageTypeError` | 13 | 错误 |

---

## 核心类型

### EmsRPCClient（底层连接）

```go
type EmsRPCClient struct {
    conn     net.Conn   // TCP 连接
    shutDown bool       // 关闭标志
}
```

- `Dial(address)` — 建立 TCP 连接
- `send(method, seq, body)` — 发送请求（加 13 字节头）
- `recv()` — 接收响应（解析头 + body）
- `Close()` — 关闭连接

### EmsRPCClientBase（带重连的客户端）

```go
type EmsRPCClientBase struct {
    serverAddr    string
    client        *EmsRPCClient
    seq           uint32              // 序列号（原子递增）
    timeout       time.Duration       // 调用超时时间
    calls         map[uint32]*Call    // 活跃调用表
    stopConnect   atomic.Bool
    RunOK         atomic.Bool
}
```

### EmsClient（完整客户端）

```go
type EmsClient struct {
    EmsRPCClientBase
    mgr *Manager
}
```

### Manager（客户端管理器）

```go
type Manager struct {
    clientConns sync.Map              // uname → *websocket.Conn
    RedisReq    sync.Map              // Redis 请求暂存
    pushRespChan chan *PushResp        // 推送消息通道（缓冲 1024）
}
```

---

## RPC 方法常量 (`emsrpcgateway.go`)

| 常量 | 值 | 说明 |
|---|---|---|
| `QueryGatewayMethod` | 100 | 查询网关 |
| `QueryDevMethod` | 200 | 查询设备 |
| `QueryYxMethod` | 201 | 查询遥信 |
| `QueryYcMethod` | 202 | 查询遥测 |
| `QueryKwhMethod` | 203 | 查询电度 |
| `QueryPropMethod` | 204 | 查询属性 |
| `QueryRemoteMethod` | 205 | 远程遥控 |
| `PushMethod` | 1000 | 推送消息 |
| `PushRedisMethod` | 1001 | Redis 推送 |
| `PushRemoteMethod` | 1002 | 远程推送 |
| `PushLogDumpTaskMethod` | 1003 | 日志转储任务推送 |

---

## 推送机制

`Manager` 通过 `pushRespChan`（缓冲 1024）接收推送消息，`RunPushMsg()` 协程循环消费并分发到已连接的 WebSocket。

---

## 相关笔记

- [[项目架构总览]]
- [[WebSocket推送机制]]
- [[handlers模块]]
