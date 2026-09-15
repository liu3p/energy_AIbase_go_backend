# WebSocket 推送机制

> 涉及模块: `handlers/wshandler.go` `emsclient/emsrpclientmgr.go` `emsdatahub/`

---

## 连接流程

```
前端 GET /api/v1/log/ws/push (带 JWT)
         ↓
WebSocketHandler (upgrade HTTP → WS)
         ↓
更新 OnlineUser.State = 2, 存储 PushConn
Manager.PushConn(uname, conn)
         ↓
启动 ping/pong 心跳协程
         ↓
循环 ReadMessage (保持连接)
```

---

## 推送架构

```
EMS 核心进程
    ↓ (TCP RPC Push 消息)
emsclient.Manager
    ↓ pushRespChan (缓冲 1024)
Manager.RunPushMsg() 协程
    ↓ 分发到各 WebSocket 连接
handlers.PushWs(uname, msg)
    ↓
前端浏览器
```

---

## 心跳机制

```go
// wshandler.go
go func() {
    noPongTimes := 0
    for {
        select {
        case <-getPong:
            noPongTimes = 0
        case <-time.After(10 * time.Second):
            // 发送 ping
            ClientMgr.PushMsg(&PushResp{Topic: "ping", Content: uname})
            if noPongTimes > 2 {
                // 超过 3 次无 pong，关闭连接
                conn.Close()
                return
            }
            noPongTimes++
        }
    }
}()
```

- 每 10 秒发送一次 `ping`
- 超过 **3 次**未收到 pong → 强制关闭 WebSocket
- 前端需实现 pong 回复逻辑

---

## 推送消息格式

```go
type PushResp struct {
    Topic   string      `json:"topic"`   // 消息主题
    Content interface{} `json:"content"` // 消息内容
}
```

### Topic 类型

| Topic | 说明 | Content 格式 |
|---|---|---|
| `ping` | 心跳 | uname 字符串 |
| `close` | 关闭连接 | uname 字符串 |
| `status` | 设备状态 | `{deviceSn: 1/2}` (1=离线, 2=在线) |
| `tm` | 遥测数据 | `{deviceSn: [{k, v, at, mt}]}` |
| `ts` | 遥信数据 | `{deviceSn: [{k, v, at, mt}]}` |
| `prop` | 属性数据 | `{deviceSn: [{k, v, at, mt}]}` |
| `homepage_subscribe` | 首页订阅 | 前端→后端，触发实时数据推送 |
| `homepage_unsubscribe` | 首页取消订阅 | 前端→后端，停止推送 |
| `homepage_data` | 首页实时数据 | 后端→前端，推送最新值 |

---

## 断开处理

```go
defer func() {
    ClientMgr.PopConn(uname)
    onlines.State = 1
    onlines.PushConn = nil
}()
```

- 从 `Manager.clientConns` 移除
- 更新 `OnlineUser` 状态为已登录但未连接

---

## 相关笔记

- [[emsclient模块]]
- [[handlers模块]]
- [[用户认证体系]]
