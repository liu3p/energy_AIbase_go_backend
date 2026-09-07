# zmq 模块

> 路径: `zmq/` | 核心职责: ZMQ 客户端，与系统监控进程通信

---

## ZmqClient (`zmqclient.go`)

基于 `pebbe/zmq4` 库，支持两种模式：

### DEALER 模式（请求-响应）

```go
func NewZmqClient(serverAddr, clientID string) (*ZmqClient, error)
```

- 用于系统信息查询（REQ/REP）
- 端点: `tcp://127.0.0.1:9971`
- 接收超时: 60 秒

### SUB 模式（订阅-发布）

```go
func NewZmqSub(serverAddr, clientID, topic string) (*ZmqClient, error)
```

- 用于系统信息订阅推送
- 端点: `tcp://127.0.0.1:9972`

---

## 类型帧协议

ZMQ 消息前会附加 8 字节**类型标识帧**（`buildTypeFrame`），每个参数占 4 bit：

```
string → 1
int    → 2
float  → 3
[]byte → 4
空     → 0
```

最多支持 16 个参数，编码到一个 `uint64` 中。

---

## 主要接口

| 方法 | 说明 |
|---|---|
| `Send(frame...)` | 发送消息（带类型帧） |
| `Recv()` | 接收消息 |
| `SimpleGetMsg(frame...)` | 发送+接收（同步） |
| `SendAndLoopReceiveMsg(frame...)` | 发送+循环接收（返回 channel） |

---

## 使用场景

`systeminfo` 模块通过 ZMQ 获取：
- CPU 使用率
- 内存使用率
- 磁盘信息
- 进程列表和状态

---

## 相关笔记

- [[项目架构总览]]
- [[systeminfo模块]]
