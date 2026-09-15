# 测点 OID 排序修改方案

## 1. 需求

`GET /api/v1/log/datamonitor/rtu/:rid/device/:did/points/:ptype` 接口返回的测点数据，需要按照 OID 最后一段数字（gin）的数值顺序排列。

**当前问题**：使用 `sort.Strings(ids)` 字符串排序，导致 `"0-0-0-10"` 排在 `"0-0-0-2"` 前面。

## 2. 数据流分析

```
getPointsAndDBMsg(rid, did, ptype)
  → dbcfg.GetPointsIDFromDeviceByType()  // 返回 map[string]string (id → name)
  → ids = map 的 key 列表
  → sort.Strings(ids)  // ❌ 当前：字符串排序
  → 返回 allIDs

GetPointsData (WebSocket handler)
  → 首次注册：getRegisterPointsID(code=1, page=0) 取 allIDs 前 N 个
  → 分页注册：getRegisterPointsID(code=1) 按 page 取 allIDs 中的片段
  → 模糊查询：getRegisterPointsID(code=2) 按 name 过滤
  → registerPointsFunc(registerIDs)
      → dbMsg.RegisterPointsID(idNameMap, registerIDs)
          → getValue() 按 registerIDs 顺序返回 []PointValue
```

**排序的关键点**：`allIDs` 的顺序决定了分页返回的顺序，只要 `allIDs` 排序正确，后续所有分页和推送都自动有序。

## 3. 修改方案

### 3.1 修改 `getPointsAndDBMsg` 函数

**文件**：`handlers/datamonitor.go:362`

将 `sort.Strings(ids)` 替换为按 OID 最后一段数字排序：

```go
// getPointsAndDBMsg 获取所有测点
func getPointsAndDBMsg(rid, did, ptype string) (map[string]string, []string, *cgolibparse.DBMsgManager, error) {
    // 获取所有测点id和name
    dbcfg, err := emsxmldb.GetDbConfigManager()
    if err != nil {
        return nil, nil, nil, fmt.Errorf("GetPointsData GetDbConfigManager failed:", err)
    }
    idNameMap, err := dbcfg.GetPointsIDFromDeviceByType(rid, did, ptype)
    if err != nil {
        return nil, nil, nil, fmt.Errorf("GetPointsData GetPointsFromDevice failed:", err)
    }
    ids := make([]string, 0, len(idNameMap))
    for id := range idNameMap {
        ids = append(ids, id)
    }

    // 按 OID 最后一段数字（gin）排序
    sort.Slice(ids, func(i, j int) bool {
        return lastOIDSegment(ids[i]) < lastOIDSegment(ids[j])
    })

    // 获取前置数据管理类
    dbMsg, err := cgolibparse.GetDBMsgManagerInstance()
    if err != nil {
        return nil, nil, nil, fmt.Errorf("GetPointsData GetDBMsgManagerInstance failed:", err)
    }
    return idNameMap, ids, dbMsg, nil
}
```

### 3.2 新增辅助函数

**文件**：`../../handlers/datamonitor.go`（在 `getPointsAndDBMsg` 前添加）

```go
// lastOIDSegment 提取 OID 字符串最后一段的数字值
// OID 格式: "rtuID-deviceID-tableNo-gin"，如 "0-0-0-1"
// 返回 gin 的数值，用于排序；解析失败返回 0
func lastOIDSegment(oid string) uint64 {
    idx := strings.LastIndex(oid, "-")
    if idx == -1 || idx == len(oid)-1 {
        return 0
    }
    val, err := strconv.ParseUint(oid[idx+1:], 10, 64)
    if err != nil {
        return 0
    }
    return val
}
```

## 4. 排序效果对比

输入 OID 列表：`["0-0-0-1", "0-0-0-10", "0-0-0-2", "0-0-0-20", "0-0-0-3"]`

| 排序方式 | 结果 |
|---------|------|
| **修改前**（`sort.Strings`） | `0-0-0-1`, `0-0-0-10`, `0-0-0-2`, `0-0-0-20`, `0-0-0-3` |
| **修改后**（按 gin 数值） | `0-0-0-1`, `0-0-0-2`, `0-0-0-3`, `0-0-0-10`, `0-0-0-20` |

## 5. 影响范围

| 位置 | 影响 |
|------|------|
| `getPointsAndDBMsg` | ✅ 唯一修改点 |
| `allIDs`（返回值） | 排序方式改变 |
| `getRegisterPointsID`（分页/查询） | 无代码修改，但分页结果顺序随 `allIDs` 改变 |
| `RegisterPointsID` → `getValue()` | 无代码修改，返回顺序随 `registerIDs` 改变 |
| `../../homepage/manager.go` | 调用 `RegisterPointsID`，同样受益于排序 |

## 6. 修改文件清单

| 文件 | 修改内容 |
|------|---------|
| `../../handlers/datamonitor.go` | 新增 `lastOIDSegment` 函数；`getPointsAndDBMsg` 中替换排序逻辑 |

## 7. 测试要点

1. **基本排序**：OID `0-0-0-1` 到 `0-0-0-5`，验证按数值 1→2→3→4→5 排列
2. **多位数排序**：OID 包含 `10`、`20`，验证不出现 `10` 排在 `2` 前面的情况
3. **分页连续性**：翻页后数据顺序保持连续
4. **模糊查询**：按名称搜索后返回的数据仍然有序
5. **边界情况**：OID 格式异常（无 `-` 分隔符）时返回默认值 0，不 panic
