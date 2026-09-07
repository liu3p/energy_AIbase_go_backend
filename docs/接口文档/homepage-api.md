# EMS 首页模块接口文档

> 版本: v1.0 | 日期: 2026-07-25 | 基础路径: `/api/v1/log/homepage`

---

## 1. 查询首页配置

**`GET /homepage/config`**

返回完整首页配置，每个有 `oid` 的条目会自动补充对应测点的 `name`（测点名称）和 `unit`（单位）。

**响应** `200 OK`

```json
{
  "code": 200,
  "data": {
    "topology": [
      {
        "no": 1,
        "show_name": "储能",
        "used": 1,
        "oid": "1-1-102-1",
        "type": 1,      ￼{
        "show_unit": "",
        "name": "储能PCS有功功率",
        "unit": "kW"
      }
    ],
    "realtime": [
      {
        "no": 1,
        "show_name": "",
        "oid": "1-1-102-1",
        "type": 2,
        "table": { "0": "未发生", "1": "发生" },
        "name": "故障信号",
        "unit": ""
      }
    ],
    "basic_info": [
      {
        "no": 1,
        "show_name": "电站名称",
        "type": 3,
        "show_value": "xxx电站",
        "unit": "",
        "name": "",
        "unit": ""
      }
    ]
  }
}
```

### 字段说明

| 字段           | 类型     | 说明                                                       |
| ------------ | ------ | -------------------------------------------------------- |
| `no`         | int    | 序号，唯一标识该条目，从 1 开始连续递增                                    |
| `show_name`  | string | 展示名称（可为空，为空时前端可使用 `name`）                                |
| `used`       | int    | 是否启用：1=启用，0=禁用（仅 `topology` 有）                           |
| `oid`        | string | 绑定的测点 ID，格式：`rtuID-deviceID-tableNo-gin`，如 `"1-1-102-1"` |
| `type`       | int    | 值类型：1=实时值，2=查表值，3=固定值                                    |
| `show_unit`  | string | 展示单位（可为空，为空时前端可使用 `unit`）                                |
| `table`      | object | 值→文本映射，仅 `type=2` 时存在，如 `{"0":"未发生","1":"发生"}`           |
| `show_value` | string | 固定显示值，仅 `type=3` 时存在                                     |
| `name`       | string | 测点原始名称（后端补充，只读）                                          |
| `unit`       | string | 测点原始单位（后端补充，只读）                                          |

---

## 2. 修改拓扑配置

**`POST /homepage/config/topology`**

整体替换 topology 数组，传入完整的数组内容。

**请求体**：数组

```json
[
  {
    "no": 1,
    "show_name": "储能",
    "used": 1,
    "oid": "1-1-102-1",
    "type": 1,
    "show_unit": ""
  },
  {
    "no": 2,
    "show_name": "电网",
    "used": 1,
    "oid": "1-1-102-2",
    "type": 1,
    "show_unit": ""
  }
]
```

### `no` 字段处理规则

| 情况 | 处理方式 |
|---|---|
| `no` 值未重复 | 保留原值 |
| `no` 值重复 | 后端自动分配下一个可用编号 |

最终保存的配置中 `no` 从 1 开始连续递增，无空洞、无重复。

**响应** `200 OK`

```json
{ "code": 200, "message": "success" }
```

---

## 3. 修改实时数据配置

**`POST /homepage/config/realtime`**

整体替换 realtime 数组，格式与接口 2 相同，`no` 去重规则一致。

**请求体**：数组（同接口 2 的格式）

**响应** `200 OK`

```json
{ "code": 200, "message": "success" }
```

---

## 4. 修改基础信息配置

**`POST /homepage/config/basic_info`**

整体替换 basic_info 数组，格式与接口 2 相同，`no` 去重规则一致。

**请求体**：数组（同接口 2 的格式）

**响应** `200 OK`

```json
{ "code": 200, "message": "success" }
```

---

## 5. 修改电力等级配置

**`POST /homepage/config/power_level`**

整体替换 power_level 数组，格式与接口 2 相同，`no` 去重规则一致。

**请求体**：数组（同接口 2 的格式）

**响应** `200 OK`

```json
{ "code": 200, "message": "success" }
```

---

## 6. 查询首页历史数据

**`POST /api/v1/log/influxdb/his/home`**

自动从首页配置中读取 Topology 和 PowerLevel 的 OID，按类型分组后批量查询 InfluxDB 历史数据，一次性返回两个维度的历史数据。

**请求体**

```json
{
  "start_time": "2026-01-01T00:00:00Z",
  "end_time": "2026-01-02T00:00:00Z",
  "interval": 300
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `start_time` | string | 是 | 开始时间，RFC3339 格式 |
| `end_time` | string | 是 | 结束时间，RFC3339 格式，必须晚于 `start_time` |
| `interval` | int | 否 | 时间间隔（秒），默认 300 秒，最小 300 秒 |

**响应** `200 OK`

```json
{
  "code": 200,
  "data": {
    "topology": [
      {
        "time": "2026-01-01T00:00:00Z",
        "data": {
          "1-1-102-1": 32.4,
          "1-1-102-2": 750.1
        }
      }
    ],
    "power_level": [
      {
        "time": "2026-01-01T00:00:00Z",
        "data": {
          "1-1-102-3": 10.5
        }
      }
    ]
  },
  "msg": "查询成功"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.topology` | array | 拓扑维度的历史数据（来自配置中 `topology` 数组的 OID） |
| `data.power_level` | array | 电力等级维度的历史数据（来自配置中 `power_level` 数组的 OID） |
| `data.*[].time` | string | 时间点，RFC3339 格式 |
| `data.*[].data` | object | 该时间点各测点的值，key 为测点 ID |

**错误响应**

```json
{ "code": 400, "message": "参数 start_time 和 end_time 必填" }
{ "code": 400, "message": "start_time 必须早于 end_time" }
{ "code": 500, "message": "获取首页配置失败: ..." }
```

> 注：单类型查询失败不阻塞其他类型，失败的类型结果中不含该类型数据。

---

## 错误响应

```json
{ "code": 400, "message": "请求参数错误" }
{ "code": 500, "message": "配置保存失败" }
不为200的都是错误
```

---

## type 值对照表

| type | 含义  | 必填字段          | 说明                       |
| ---- | --- | ------------- | ------------------------ |
| 1    | 实时值 | `oid`         | 根据 实时数据直接展示              |
| 2    | 查表值 | `oid`、`table` | 根据 实时数据，通过 `table` 映射为文本 |
| 3    | 固定值 | `show_value`  | 无需绑点，直接展示固定文本            |
## 6. 首页实时数据推送（WebSocket）

基于已有的 WebSocket 连接（`/api/v1/log/ws/push`，登录时建立），前端发送订阅消息，后端广播实时数据。

### 订阅

前端发送：

```json
{ "topic": "homepage_subscribe" }
```

后端收到后：
1. 读取首页配置，收集所有 `type=1/2` 的 OID
2. 与已注册的 OID 对比，若配置有变化则重新注册
3. 启动全局推送协程，每 2 秒广播一次

### 取消订阅

前端发送：

```json
{ "topic": "homepage_unsubscribe" }
```

连接断开时后端自动取消订阅，无需前端处理。

### 推送数据

后端每 2 秒推送一条消息，广播到所有已连接用户：

```json
{
  "topic": "homepage_data",
  "content": [
    { "oid": "1-1-102-1", "value": "750.1", "quality": "0", "sendtime": "2026-07-25T10:00:00", "dead": "0" },
    { "oid": "1-1-102-2", "value": "1", "quality": "0", "sendtime": "2026-07-25T10:00:00", "dead": "1" }
  ]
}
```

### 推送字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| `oid` | string | 测点 ID，用于匹配前端配置中的 `oid` 字段 |
| `value` | string | 当前值（CurrentValue），type=2 时前端通过 `table` 映射为文本 |
| `quality` | string | 质量码 |
| `sendtime` | string | 数据发送时间（RFC3339 格式） |
| `dead` | string | 死数标识，直接透传 C 库返回值，`"0"`=正常，非 `"0"`=数据超时未更新 |