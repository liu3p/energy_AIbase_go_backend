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
| `homepage_data` | 首页实时数据 | 后端→前端，推送最新值（数字 value 保留三位小数） |

### homepage_data Content 结构

```go
// homepage/manager.go:13
type HomepagePointData struct {
    OID      string `json:"oid"`       // 测点 OID
    Value    string `json:"value"`     // 当前值（数字保留三位小数，非数字原样）
    Quality  string `json:"quality"`   // 质量码
    SendTime string `json:"sendtime"`  // 发送时间
    Dead     string `json:"dead"`      // 死区值
}
```

推送频率：每 2 秒一次。前端发送 `{"topic":"homepage_subscribe"}` 触发订阅，`{"topic":"homepage_unsubscribe"}` 停止。

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

---

## 数据监控 WebSocket（独立连接）

数据监控的 WebSocket 接口与上述推送机制不同：每个接口独立建立 WS 连接，不复用 `/ws/push`。

### 接口列表

| 路径 | Handler | 推送频率 | 说明 |
|---|---|---|---|
| `/datamonitor/rtu/:rid/device/:did/points/:ptype` | `GetPointsData` | 1秒 | 测点实时数据 |
| `/datamonitor/cgroup/:cgid/channel/:cid/status` | `GetRtuChannelStatus` | 1秒 | 通道状态 |
| `/datamonitor/cgroup/:cgid/channel/:cid/plugin/:pluginid/logs` | `GetRtuChannelLogs` | 实时 | 通道报文 |

### GetPointsData 数据流

```
前端 GET /datamonitor/rtu/{rid}/device/{did}/points/{ptype} (带 JWT)
    ↓
Upgrade HTTP → WebSocket
    ↓
getPointsAndDBMsg(rid, did, ptype) → 获取测点 ID 列表 + DBMsgManager
    ↓
conn.WriteJSON(CMD{Code:0, Total:N})  → 推送测点总数
    ↓
startWSReader → 接收前端控制指令（分页/搜索）
    ↓
dbMsg.RegisterPointsID(ids) → 注册订阅，返回 getValue 闭包
    ↓
ticker 每秒 → getValue() → conn.WriteJSON(pointValues)
```

### 前端控制指令（CMD）

```json
{"code": 1, "page": 0, "pagesize": 50}     // 分页查询
{"code": 2, "name": "电压"}                  // 模糊搜索
{"code": 3, "page": 0, "pagesize": 50}      // 按上次搜索结果分页
```

| code | 说明 |
|---|---|
| 1 | 直接分页（page * pagesize） |
| 2 | 按名称模糊搜索，返回匹配的测点 ID |
| 3 | 在上次搜索结果上分页 |

### analog 数据格式化

当 `ptype == "analog"` 时，推送的 `PointValue` 中 `rawvalue` 和 `currvalue` 保留三位小数（如 `"123.457"`、`"0.100"`）。格式化在推送前动态执行，不修改 C++ 层原始数据。非数值字符串原样返回。

---

## 相关笔记

- [[emsclient模块]]
- [[handlers模块]]
- [[用户认证体系]]
