---
name: oid-format-detail
description: EMS OID 由 rtuid-devid-tableno-gin 四部分组成，tableno 固定绑定测点类型（101-116 全部映射）
metadata: 
  node_type: memory
  type: project
  originSessionId: e1faacce-77e9-498d-9aef-b04c94a964bb
---

## OID 格式

## 字符串形式

OID 为 XML 中测点的完整 ID，由三个 `-` 分成 4 部分，是测点的全局唯一键：

```
"1-1-102-1"
 │ │ │   └─ gin      编号（点号）
 │ │ └──── tableNo   表号，固定绑定测点类型
 │ └─────── devid     装置ID
 └────────── rtuid     RTU ID
```

## uint64 压缩形式

OID 可编码为 64 位无符号整数，用于 Cgo 层高效传递（`cgolibparse/cgodbmsg.go`）。

### 位域布局

```
|<- 16 bit ->|<- 16 bit ->|<- 8 bit ->|<- 24 bit ->|
     rtuID       deviceID     tableNo       gin
```

| 字段 | 位宽 | 掩码 | 示例值 |
|------|------|------|--------|
| rtuID | 16 bit | `0xFFFF` | 1 |
| deviceID | 16 bit | `0xFFFF` | 1 |
| tableNo | 8 bit | `0xFF` | 102 (0x66) |
| gin | 24 bit | `0xFFFFFF` | 1 |

### 编码公式

```go
(rtuID & 0xFFFF) << 48 | (deviceID & 0xFFFF) << 32 | (tableNo & 0xFF) << 24 | (gin & 0xFFFFFF)
```

示例：`"1-1-102-1"` → `(1<<48) | (1<<32) | (102<<24) | 1` = `0x0001000166000001`

### 解码函数

位于 `cgolibparse/cgodbmsg.go:310`：

```go
func pOIDToUint64(pOIDStr string) (uint64, error) {
    var rtuID, deviceID, tableNo, gin uint64
    n, err := fmt.Sscanf(pOIDStr, "%d-%d-%d-%d", &rtuID, &deviceID, &tableNo, &gin)
    if err != nil || n != 4 {
        return 0, fmt.Errorf("invalid pOID format: %s", pOIDStr)
    }
    rtuID &= 0xFFFF
    deviceID &= 0xFFFF
    tableNo &= 0xFF
    gin &= 0xFFFFFF
    return (rtuID << 48) | (deviceID << 32) | (tableNo << 24) | gin, nil
}
```

### 容量限制

- rtuID 最大 65535
- deviceID 最大 65535
- tableNo 最大 255
- gin 最大 16777215（约 1677 万）

## tableNo 与测点类型绑定关系

数据来源：`emsxmldb/dbconfig.go` initDB 中的 Single.Tables 定义。

| tableNo | 表名 | 测点类型 | XML 层级路径 | 对应结构体 | Unit 字段 |
|---------|------|---------|-------------|-----------|----------|
| 101 | digital | 遥信 YX | Device.YX.Digitals | `*Digital` | 无 |
| 102 | analog | 遥测 YC | Device.YC.Analogs | `*Analog` | `p.Unit` |
| 103 | control | 遥控 YK | Device.YK.Controls | `*Control` | 无 |
| 104 | regulate | 遥调 YT | Device.YT.Regulates | `*Regulate` | 无 |
| 105 | pulse | 遥脉 YM | Device.YM.Pulses | `*Pulse` | 无 |
| 106 | trans_digital | 转发遥信 | Device.YX.Digitals | `*Digital` | 无 |
| 107 | trans_analog | 转发遥测 | Device.YC.Analogs | `*Analog` | `p.Unit` |
| 108 | trans_control | 转发遥控 | Device.YK.Controls | `*Control` | 无 |
| 109 | trans_regulate | 转发遥调 | Device.YT.Regulates | `*Regulate` | 无 |
| 110 | trans_pulse | 转发遥脉 | Device.YM.Pulses | `*Pulse` | 无 |
| 111 | formula | 公式 | — | — | — |
| 112 | formula_factor | 公式因子 | — | — | — |
| 115 | attribute | 属性 SX | Device.SX.Attributes | `*Attribute` | 无 |
| 116 | trans_attribute | 转发属性 | Device.SX.Attributes | `*Attribute` | 无 |
| 201 | rtu | RTU 层级 | DB.RTUs | `*RTU` | — |
| 202 | device | 装置层级 | RTU.Devices | `*Device` | — |

普通类型和对应转发类型共用同一个 Device 层级字段（如 102/107 都是 `Device.YC.Analogs`），区分转发/普通依据的是 **rtuid 范围**：rtuid 500-699 为转发 RTU，其下所有测点均为转发表。详见 [[emsxmldb模块#转发 RTU（rtuid 500-699）]]。

## OID 排序规则

`GetPointsData` WebSocket 接口（`handlers/datamonitor.go`）返回的测点数据按 OID 最后一段数字（**gin**）的数值升序排列。

排序逻辑位于 `getPointsAndDBMsg` 函数，使用 `lastOIDSegment` 提取 gin 值后调用 `sort.Slice`：

```go
sort.Slice(ids, func(i, j int) bool {
    return lastOIDSegment(ids[i]) < lastOIDSegment(ids[j])
})
```

排序效果：`0-0-0-1`, `0-0-0-2`, `0-0-0-3`, `0-0-0-10`, `0-0-0-20`（而非字符串排序的 `1, 10, 2, 20, 3`）。