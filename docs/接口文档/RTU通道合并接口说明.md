# RTU 通道合并接口说明

> 日期: 2026-08-09 | 状态: 已实现
> 修订: 2026-10-09 协议多层化（`plugins` 按 ID 区间分层，`type` 改为层号字符串）

---

## 概述

将 RTU 与通道（ChannelGroup + Channel）的配置从两步独立操作合并为一步自动绑定。创建 RTU 时自动创建通道组和通道，删除 RTU 时同步删除。

### 核心变化

| 操作 | 改造前 | 改造后 |
|---|---|---|
| 新建 RTU | 前端分两步调用：创建 RTU → 手动创建通道组+通道并绑定 | 一步调用，自动创建并绑定 |
| 删除 RTU | 前端需分别删除 RTU 和通道 | 一步调用，同步删除 |
| 查询 RTU | 返回的 RTU 信息无通道字段 | 返回 `channelgroupid`、`channelid`、`channel`（完整通道信息） |

### 有通道的 RTU 类型

| 类型 | Type 值 | 有通道 | 说明 |
|---|---|---|---|
| 采集 RTU | 1 | ✅ | 自动创建通道组+通道 |
| 转发 RTU | 2 | ✅ | 自动创建通道组+通道 |
| AGC RTU | 3 | ✅ | 自动创建通道组+通道 |
| 监视 RTU | 4 | ❌ | `channelgroupid=-1, channelid=-1` |
| 公式 RTU | 5 | ❌ | `channelgroupid=-1, channelid=-1` |

前端通过 `channelgroupid >= 0` 判断 RTU 是否有通道。

### 协议多层化（2026-10-09 变更）

通道的 `plugins` 由「APP / LINK 两层」改为「**按插件 ID 区间划分的多层**」：

- **层号**由插件 `id` 推导：`id/1000 + 1`（`0~999`→第 1 层，`1000~1999`→第 2 层，……，无上限）。
- 每个通道**每层至多一个协议**，且必须**从第 1 层起连续**（选了第 N 层则 1..N-1 层均已选）；允许不选任何层。
- `plugin.type` 由 `"APP"`/`"LINK"` 改为**层号字符串** `"1"`/`"2"`/…（类型仍是字符串，仅取值变化）。
- 协议的**选项来源**为接口 8 `GET /fecfg/plugins`（按层分组返回）。
- 写入 `fe_config.xml` 时按层升序排列，`<PLUGIN ... TYPE="1">`。

---

## 接口列表

### 1. 新建 RTU（改造）

**POST** `/api/v1/log/dbcfg/rtu`

#### 请求体

```json
{
  "name": "储能RTU1",
  "type": 1,
  "rtuaddr": "",
  "memofcabinet": "",
  "inservice": "1",
  "isconnected": "0"
}
```

| 字段           | 类型     | 必填  | 说明                                   |
| ------------ | ------ | --- | ------------------------------------ |
| name         | string | ✅   | RTU 名称                               |
| type         | int    | ✅   | RTU 类型：1=采集, 2=转发, 3=AGC, 4=监视, 5=公式 |
| rtuaddr      | string | ❌   | RTU 地址（自动生成，无需传入）                    |
| memofcabinet | string | ❌   | 柜体描述                                 |
| inservice    | string | ❌   | 是否投运                                 |
| isconnected  | string | ❌   | 是否连接                                 |

#### 响应（有通道的 RTU 类型：1/2/3）

```json
{
  "code": 200,
  "message": "AddRTU success",
  "data": {
    "rtuid": "1",
    "channelgroupid": 0,
    "channelid": 0
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| data.rtuid | string | 新建 RTU 的 ID |
| data.channelgroupid | int | 自动创建的通道组 ID |
| data.channelid | int | 自动创建的通道 ID |

#### 响应（无通道的 RTU 类型：4/5）

```json
{
  "code": 200,
  "message": "AddRTU success"
}
```

#### 自动创建的命名规则

| 对象 | RTU 类型 | 命名格式 | 示例 |
|---|---|---|---|
| 通道组 | 采集 (1) | `{rtu名}的通道组` | `储能RTU1的通道组` |
| 通道组 | 转发 (2) | `{rtu名}的转发通道` | `转发RTU1的转发通道` |
| 通道组 | AGC (3) | `{rtu名}的通道组` | `AGC的通道组` |
| 通道 | 所有类型 | `{rtu名}的通道` | `储能RTU1的通道` |

#### 错误码

| code | 说明 |
|---|---|
| 400 | rtu.name 为空 |
| 500 | RTU ID 生成失败、通道创建失败等 |

---

### 2. 删除 RTU（改造）

**DELETE** `/api/v1/log/dbcfg/rtu/{rid}`

#### 路径参数

| 参数 | 类型 | 说明 |
|---|---|---|
| rid | string | RTU ID |

#### 说明

- 有通道的 RTU：同步删除关联的通道组和通道
- 无通道的 RTU：仅删除 RTU 本身

#### 响应

```json
{
  "code": 200,
  "message": "DeleteRTU success"
}
```

---

### 3. 查询全部 RTU（改造）

**GET** `/api/v1/log/dbcfg/rtus`

#### 查询参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| `include_channel` | bool | 否 | `true` | 是否返回 `channel` 字段。传 `false` 或 `0` 可省略通道详情，仅保留 `channelgroupid` 和 `channelid` |

#### 响应

```json
[
  {
    "id": "1",
    "name": "储能RTU1",
    "type": 1,
    "rtuaddr": "1",
    "memofcabinet": "",
    "inservice": "1",
    "isconnected": "0",
    "channelgroupid": 0,
    "channelid": 0,
    "channel": {
      "id": 0,
      "name": "储能RTU1的通道",
      "comment": "",
      "onduty": 0,
      "servergroup": 1,
      "plugins": [],
      "possibleowner": { "id": 1 }
    },
    "device": [...]
  },
  {
    "id": "800",
    "name": "系统监视",
    "type": 4,
    "channelgroupid": -1,
    "channelid": -1,
    "device": [...]
  }
]
```

| 字段 | 类型 | 说明 |
|---|---|---|
| channelgroupid | int | 绑定的通道组 ID，`-1` 表示无通道 |
| channelid | int | 绑定的通道 ID，`-1` 表示无通道 |
| channel | object \| null | 通道完整信息，无通道时为 `null`（JSON 中省略） |

> **Parameters 排序**：`channel.plugins[].parameters` 数组按字母序排列，`STATION_ID ASDU_ADDR`（单个参数名）始终排在最前。排序在返回时动态执行，不修改 fe_config 原始数据。

---

### 4. 查询全部 RTU（不含测点，改造）

**GET** `/api/v1/log/dbcfg/rtus/except_points`

#### 查询参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| `include_channel` | bool | 否 | `true` | 是否返回 `channel` 字段，行为同接口 3 |

响应结构同接口 3，但 `device` 中不含五遥测点数据，仅保留设备基础信息。`channelgroupid`、`channelid`、`channel` 字段行为一致。

---

### 5. 修改 RTU（含通道配置）

**PUT** `/api/v1/log/dbcfg/rtu/{rid}`

RTU 信息和通道配置在同一个接口修改，前端一次请求更新全部。

#### 路径参数

| 参数 | 类型 | 说明 |
|---|---|---|
| rid | string | RTU ID |

#### 请求体

> 字段名以 Go 结构体 `emsxmldb.RTU` 的 JSON tag 为准：通道为嵌套对象 `channel`，**不是平铺字段**。

```json
{
  "name": "储能RTU1",
  "type": 1,
  "inservice": "1",
  "memofcabinet": "A01",
  "channelgroupid": 0,
  "channelid": 0,
  "channelgroupname": "储能RTU1_CG",
  "for_transfer": "0",
  "channel": {
    "id": 0,
    "name": "储能RTU1的通道",
    "comment": "修改后的注释",
    "possibleowner": { "id": 1 },
    "plugins": [
      { "id": 100,  "name": "tcp_server",     "parameters": [ { "name": "TCP_PORT", "value": "503" } ] },
      { "id": 1100, "name": "modbus_general", "parameters": [ { "name": "SLAVE_ID", "value": "1" } ] }
    ]
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| name | string | ✅ | RTU 名称 |
| type | int | ✅ | RTU 类型（1 采集 / 2 转发 / 3 AGC / 4 监视 / 5 公式） |
| inservice | string | ❌ | 是否在服务 |
| memofcabinet | string | ❌ | 柜体信息 |
| channelgroupname | string | ❌ | 通道组名称（写通道组 `NAME`） |
| for_transfer | string | ❌ | 通道组 MODEL 的 `FOR_TRANSFER` |
| channel | object | ❌ | 通道信息；仅「有通道的 RTU 类型」+ `channel != null` 时处理 |
| channel.name | string | ❌ | 通道名称 |
| channel.comment | string | ❌ | 通道注释 |
| channel.possibleowner.id | int | ❌ | 归属 RTU ID，缺省用 Host.ID |
| channel.plugins | array | ❌ | **多层协议选择**，可为空数组 |
| channel.plugins[].id | int | ✅ | 所选协议插件 ID（层号由 `id/1000+1` 推导） |
| channel.plugins[].parameters | array | ❌ | 该协议的参数键值对 |
| channel.plugins[].name | string | ❌ | 协议名；不传则写出空 `NAME` |
| channel.plugins[].type | string | ❌ | **服务端忽略，一律由 `id` 推导** |

#### 多层协议规则（`channel.plugins`）

- 层号 = `id/1000 + 1`（`0~999`→1，`1000~1999`→2，……）。
- 每层至多选一个；必须从第 1 层起**连续**；允许 `[]`。
- 服务端按**层升序**写入 `fe_config.xml`，`TYPE` 写为层号字符串。

#### 处理流程

```
PUT /rtu/:rid
    ↓
1. 更新 RTU 信息（db_config.xml）→ save
2. 若有 channel → UpdateChannelGroup(name, for_transfer) → save
               → UpdateChannel(plugins) → 校验层连续性 → 按层升序 → save
```

#### 响应

```json
{ "code": 200, "message": "UpdateRTU success" }
```

#### 错误码

| code | 说明 | 示例 message |
|---|---|---|
| 400 | RTU 名称为空 / `rtuaddr` 非法 | `rtu.Name is empty` |
| 400 | 协议跳层（不连续） | `plugin layer invalid: 协议必须从第 1 层起连续选择，缺少第 1 层` |
| 400 | 同层重复 | `plugin layer invalid: 第 1 层只能选择一个协议` |
| 400 | 插件 ID 非法 | `plugin layer invalid: 插件 ID -1 无效` |
| 500 | 更新失败 | — |

> **原子性说明**：处理流程为多步落盘（RTU → 通道组 → 通道）。当通道协议校验失败返回 400 时，`db_config.xml` 与通道组的 `NAME`/`FOR_TRANSFER` 可能**已落盘**，仅通道协议未改。该「半写」属既有问题，本次未改动。

---

### 6. 查询全部通道（新增）

**GET** `/api/v1/log/dbcfg/channels`

扁平化所有通道组下的通道，附带所属 RTU 信息。

#### 响应

```json
[
  {
    "id": 1,
    "name": "储能RTU1_通道1",
    "plugins": [
      {
        "id": 100,
        "name": "tcp_server",
        "type": "1",
        "parameters": [
          { "name": "IP", "value": "192.168.1.100", "valuelist": "" }
        ]
      },
      {
        "id": 1100,
        "name": "modbus_general",
        "type": "2",
        "parameters": [
          { "name": "SLAVE_ID", "value": "1", "valuelist": "" }
        ]
      }
    ],
    "rtuid": "1",
    "rtuname": "储能RTU1",
    "channelgroupid": 0,
    "channelgroupname": "储能RTU1_CG"
  }
]
```

| 字段 | 类型 | 说明 |
|---|---|---|
| id | int | 通道 ID |
| name | string | 通道名称 |
| plugins | array | 通道关联的协议插件列表（≤ 每层一个，按层升序） |
| plugins[].name | string | 协议名，来自 `fe_config.xml` 的 `PLUGIN@NAME`；**写入时未传 name 则为空串**（服务端不回填） |
| plugins[].type | string | 层号字符串 `"1"`/`"2"`/…（由 `plugins[].id/1000+1` 推导） |
| rtuid | string | 所属 RTU 的 ID，未绑定时为空字符串 |
| rtuname | string | 所属 RTU 的名称，未绑定时为空字符串 |
| channelgroupid | int | 所属通道组 ID |
| channelgroupname | string | 所属通道组名称 |

> **Parameters 排序**：`plugins[].parameters` 数组按字母序排列，`STATION_ID ASDU_ADDR`（单个参数名）始终排在最前。排序在返回时动态执行，不修改原始数据。

#### 数据来源

聚合两个数据源：

| 数据源 | 管理器 | 提供的数据 |
|---|---|---|
| db_config.xml | `DbConfigXMLManager` | RTU 列表，构建 `ChannelGroupID → RTU` 映射 |
| fe_config.xml | `FeConfigXMLManager` | 通道组列表，每个通道组下包含通道 |

#### 边界情况

| 场景 | 返回值 |
|---|---|
| 通道组下无通道 | 该通道组不产生任何元素 |
| 通道组未绑定 RTU | `rtuid` 和 `rtuname` 返回空字符串 |
| 系统中无任何通道 | 返回空数组 `[]` |

#### 使用场景

- **前端通道列表展示**：展示所有通道，包含通道名称、所属 RTU、所属通道组
- **WebSocket 连接参数获取**：返回数据可直接用于建立 WS 订阅连接

```javascript
// 建立 WebSocket 连接示例（订阅某一层插件的报文日志）
const channel = channels[0];
// 多层协议时需指定订阅哪一层（例如 type === "1" 的第 1 层插件）
const plugin = channel.plugins.find(p => p.type === "1");
const wsUrl = `/api/v1/log/datamonitor/cgroup/${channel.channelgroupid}/channel/${channel.id}/plugin/${plugin.id}/logs`;
```

---

### 7. 重启指定通道（新增）

**POST** `/api/v1/log/fecfg/channels/restart`

通过 `kill -9` 杀死 fe_channel 进程，c_fe_channel_manage 看门狗自动重启该进程。

#### 请求体

```json
[
  {
    "channel_group_id": 1,
    "channel_ids": [1, 2]
  },
  {
    "channel_group_id": 2,
    "channel_ids": [3]
  }
]
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `channel_group_id` | int | ✅ | 通道组 ID（>= 0） |
| `channel_ids` | int[] | ✅ | 该通道组下要重启的通道 ID 列表（每个元素 >= 0） |

> 不同通道组下的通道 ID 可能重复，因此需要 channel_group_id + channel_id 唯一定位一个通道。

#### 响应（全部成功）

```json
{
  "code": 200,
  "message": "restart channels completed",
  "data": {
    "success": [
      {"channel_group_id": 1, "channel_ids": [1, 2]},
      {"channel_group_id": 2, "channel_ids": [3]}
    ],
    "fail": []
  }
}
```

#### 响应（部分失败）

```json
{
  "code": 200,
  "message": "restart channels completed",
  "data": {
    "success": [
      {"channel_group_id": 1, "channel_ids": [1]}
    ],
    "fail": [
      {"channel_group_id": 1, "channel_ids": [2]},
      {"channel_group_id": 2, "channel_ids": [3]}
    ]
  }
}
```

#### 响应字段说明

请求和响应使用相同的数据结构：

| 字段 | 类型 | 说明 |
|---|---|---|
| `data.success` | array | 重启成功的通道列表 |
| `data.success[].channel_group_id` | int | 通道组 ID |
| `data.success[].channel_ids` | int[] | 该通道组下重启成功的通道 ID 列表 |
| `data.fail` | array | 重启失败的通道列表 |
| `data.fail[].channel_group_id` | int | 通道组 ID |
| `data.fail[].channel_ids` | int[] | 该通道组下重启失败的通道 ID 列表 |

#### 错误码

| HTTP 状态码 | 说明 | 示例 |
|---|---|---|
| 400 | 请求体格式错误或为空 | `"request body is required"` |
| 400 | channel_group_id 为负数 | `"item[0].channel_group_id must be >= 0"` |
| 400 | channel_ids 缺失或为空数组 | `"item[0].channel_ids is required and cannot be empty"` |
| 400 | channel_ids 中包含负数 | `"item[0].channel_ids[1] must be >= 0, got -1"` |
| 500 | feconfig 加载失败 | 内部错误信息 |

#### 重启原理

```
1. 通过 channel_group_id + channel_id 查找通道的 SERVER_GROUP_ID（即 server_id）
2. ps aux | grep 'fe_channel' | grep ' <server_id>$' → 获取进程 PID
3. kill -9 <pid> → 强制杀死 fe_channel 进程
4. c_fe_channel_manage 看门狗（每 3 秒检查）检测到进程死亡
5. 连续 3 次后自动重启 fe_channel 进程，通道恢复
```

从 kill 到通道恢复约 3-10 秒。

---

### 8. 获取协议选项（按层）

**GET** `/api/v1/log/fecfg/plugins`

返回全部可选协议，**按层（`id/1000+1`）分组**，供前端按层选择；只返回非空层。

#### 响应

```json
[
  {
    "type": "1",
    "plugins": [
      { "id": 100, "name": "tcp_server", "type": "1",
        "parameters": [ { "name": "TCP_PORT", "value": "503", "valuelist": "" } ] }
    ]
  },
  {
    "type": "2",
    "plugins": [
      { "id": 1100, "name": "modbus_general", "type": "2",
        "parameters": [ { "name": "SLAVE_ID", "value": "1", "valuelist": "" } ] }
    ]
  }
]
```

| 字段 | 类型 | 说明 |
|---|---|---|
| type | string | 层号字符串 `"1"`/`"2"`/… |
| plugins | array | 该层可选协议 |
| plugins[].id | int | 协议插件 ID |
| plugins[].name | string | 协议名 |
| plugins[].type | string | 层号字符串，与所属层一致 |
| plugins[].parameters | array | 协议参数（含默认值与 `valuelist`） |

> `parameters` 排序规则同接口 6。
> ⚠️ **变更（2026-10-09）**：原响应为 `{ "appplugin": [...], "linkplugin": [...] }`，现改为**按层分组的数组**。

---

## 内部数据流（供开发参考）

### 创建流程

```
handler.AddRTU
  ├─ rtuHasChannel(type)?
  │   ├─ 否 → ChannelGroupID=-1, ChannelID=-1
  │   │       dbcfg.AddRTU(rtu)                          // save
  │   └─ 是 ↓
  ├─ dbcfg.GenerateRTUID(type) → rtuID                   // 加锁，预生成 ID
  ├─ fecfg.CreateChannelGroupWithChannel(cg, ch)          // feConfig save
  │   → 返回 (cgID, chID)
  └─ dbcfg.AddRTUWithChannel(rtu, cgID, chID)            // dbConfig 原子 save
      ├─ addRTUWithPreAssignedID(rtu, false)
      ├─ setRTUChannelInternal(rtu.ID, cgID, chID)
      └─ save()                                          // 一次写入
      失败 → fecfg.DeleteChannelGroup(cgID)               // 回滚通道组
```

### 删除流程

```
handler.DeleteRTU
  └─ dbcfg.DeleteRTUWithChannel(rtuID, deleteFunc)
      ├─ deleteFunc → fecfg.DeleteChannelGroupByStationID  // feConfig save
      ├─ deleteRTU(rtuID, false)                           // 内存删除
      └─ save()                                            // 一次写入
```

### 启动加载

```
main.go
  → GetDbConfigManager()        // 加载 db_config.xml（含 ChannelGroupID, ChannelID）
  → GetFeConfigXMLManager()     // 加载 fe_config.xml
  → dbcfg.LoadRTUChannelRelation()
      → 遍历 RTU，ChannelGroupID >= 0 的，用 ChannelGroupID 查 fe_config
      → 找到 ChannelGroup.Channels[0]，填充 rtu.Channel（内存指针）
```

---

## XML 格式变化

### db_config.xml（RTU 节点新增两个属性）

```xml
<!-- 改造前 -->
<RTU ID="1" Name="储能RTU1" Type="1" RtuAddr="1" ...>

<!-- 改造后（有通道） -->
<RTU ID="1" Name="储能RTU1" Type="1" RtuAddr="1" ChannelGroupID="0" ChannelID="0" ...>

<!-- 改造后（无通道） -->
<RTU ID="800" Name="系统监视" Type="4" RtuAddr="800" ChannelGroupID="-1" ChannelID="-1" ...>
```

### fe_config.xml

通道组和通道的创建复用现有 XML 格式，不引入新节点。协议多层化后，`<PLUGIN>` 节点：

- 按**层升序**排列；
- `TYPE` 属性由 `"APP"`/`"LINK"` 变为**层号字符串** `"1"`/`"2"`/…（由 `ID` 推导）。

```xml
<CHANNEL ID="0" NAME="储能RTU1_CG" ...>
  <PLUGIN ID="100"  NAME="tcp_server"     TYPE="1"> ... </PLUGIN>
  <PLUGIN ID="1100" NAME="modbus_general" TYPE="2"> ... </PLUGIN>
  <POSSIBLE_OWNER ID="1"/>
</CHANNEL>
```

> 存量数据中 `TYPE` 若为非数字（`"APP"`）或与 `ID` 矛盾，服务启动时会以 `ID` 推断的层号覆盖并打印告警，不影响启动。

---

## 兼容性

| 项目 | 说明 |
|---|---|
| 现有 RTU | 不处理存量，项目无存量数据 |
| 原有 AddRTU 接口 | 对外签名不变，内部行为增加通道创建 |
| 原有 DeleteRTU 接口 | 对外签名不变，内部行为增加通道删除 |
| 原有 fecfg 手动接口 | `/api/v1/log/fecfg/cgroup/:cgid/channel[/:cid]` 仍可独立使用；但请求体由 `appplugin`/`linkplugin` 两个对象改为 `plugins` 数组（多层），并同样做层连续性校验 |
| 原有 datamonitor 模块 | `GetRtuChannels` 仍通过 `STATION_BASE_ID` 反查，不受影响 |
| 协议多层化（2026-10-09） | 通道 `plugins` 由两层（APP/LINK）改为按 ID 区间的多层；`plugins[].type` 取值由 `"APP"/"LINK"` 变为层号字符串。依赖 C 侧插件 `class_id` 已完成重编号 |
