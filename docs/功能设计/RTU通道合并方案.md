# RTU 与通道合并方案

> 状态: 设计阶段 | 日期: 2026-08-09（修订）

---

## 现状分析

### 两个独立配置

| 对象 | 管理器 | XML 文件 | 关键字段 |
|---|---|---|---|
| **RTU** | `DbConfigXMLManager` | `db_config.xml` | ID, Name, Type, Devices |
| **ChannelGroup** | `FeConfigXMLManager` | `fe_config.xml` | ID, Name, Model.Parameters(**STATION_BASE_ID**) |
| **Channel** | `FeConfigXMLManager` | `fe_config.xml` | ID, Name, Plugins, **PossibleOwner.ID** |

### 绑定关系（三方 1:1:1）

```
RTU (ChannelGroupID, ChannelID)        ← 新增字段记录绑定
  ↕ STATION_BASE_ID（Model 参数）
ChannelGroup（唯一 Channel）
  ↓ 1:1
Channel (PossibleOwner.ID = RTU.ID)
```

### 当前流程（两步，完全独立）

```
1. POST /api/v1/log/dbcfg/rtu                      → 创建 RTU（得到 rtuID）
2. POST /api/v1/log/fecfg/cgroup/:cgid/channel     → 创建 ChannelGroup + Channel（手动绑定）
```

**问题**：两个操作完全独立，前端需要知道绑定关系，容易遗漏或配置错误。

---

## 目标流程（一步 + 三方绑定）

```
POST /api/v1/log/dbcfg/rtu
    → 创建 RTU（db_config.xml）
    → 自动创建 ChannelGroup + Channel（fe_config.xml）
        ChannelGroup.Model.Parameters.STATION_BASE_ID = rtuID
        Channel.PossibleOwner.ID = rtuID（int）
    → RTU 记录 ChannelGroupID + ChannelID，保存回 db_config.xml
```

查询 RTU 时，能直接看到绑定的通道组号、通道号和通道完整信息。

### 有通道的 RTU 类型

| 类型      | 常量                   | 有通道 | 说明                            |
| ------- | -------------------- | --- | ----------------------------- |
| 采集 RTU  | `RtuTypeAcqu` (1)    | ✅   | 自动创建通道组+通道                    |
| 转发 RTU  | `RtuTypeTrans` (2)   | ✅   | 自动创建通道组+通道                    |
| AGC RTU | `RtuTypeAgc` (3)     | ✅   | 自动创建通道组+通道                    |
| 监视 RTU  | `RtuTypeMonit` (4)   | ❌   | -1 | -1 |
| 公式 RTU  | `RtuTypeFormula` (5) | ❌   | -1 | -1 |

前端通过 `channelgroupid >= 0` 判断 RTU 是否有通道，-1 表示无通道。

### 默认命名规则

| 对象 | RTU 类型 | 命名格式 | 示例 |
|---|---|---|---|
| 通道组 | 采集 RTU（RtuTypeAcqu） | `{rtu名}的通道组` | `储能RTU1的通道组` |
| 通道组 | 转发 RTU（RtuTypeTrans） | `{rtu名}的转发通道` | `转发RTU1的转发通道` |
| 通道组 | AGC RTU（RtuTypeAgc） | `{rtu名}的通道组` | `AGC的通道组` |
| 通道 | 所有类型 | `{rtu名}的通道` | `储能RTU1的通道` |

---

## RTU 结构扩展

```go
type RTU struct {
    BaseInfo
    XMLName        xml.Name  `xml:"RTU" json:"-"`
    Type           int       `xml:"Type,attr" json:"type"`
    RtuAddr        string    `xml:"RtuAddr,attr" json:"rtuaddr"`
    MemOfCabinet   string    `xml:"MemOfCabinet,attr" json:"memofcabinet"`
    InService      string    `xml:"InService,attr" json:"inservice"`
    IsConnected    string    `xml:"IsConnected,attr" json:"isconnected"`
    ChannelGroupID int       `xml:"ChannelGroupID,attr" json:"channelgroupid"` // 新增：绑定的通道组 ID
    ChannelID      int       `xml:"ChannelID,attr" json:"channelid"`          // 新增：绑定的通道 ID
    Devices        []*Device `xml:"Device" json:"device"`

    // 内存字段（不写入 XML，启动时填充）
    Channel        *Channel  `xml:"-" json:"channel,omitempty"`
}
```

- 无通道的 RTU：`ChannelGroupID=-1`, `ChannelID=-1`, `Channel=nil`
- 有通道的 RTU：`ChannelGroupID>=0`, `ChannelID>=0`, `Channel` 指向完整通道信息

---

## autoSave 模式（内部方法控制是否立即落盘）

### 设计思路

Go 没有默认参数，采用**内部方法 + autoSave 布尔参数**模式：

- 内部方法 `addRTU(rtu, autoSave)` 控制是否立即 `save()`
- 公开方法 `AddRTU(rtu)` 始终 `autoSave=true`（保持 API 兼容）
- 编排方法中传 `autoSave=false`，最后统一 `save()` 一次

### dbconfig.go 改造

```go
// ==================== 内部方法：控制 autoSave ====================

// addRTU 内部无锁版本，autoSave=true 时立即写文件
func (m *DbConfigXMLManager) addRTU(rtu *RTU, autoSave bool) error {
    rtus, err := m.getSingleTable("rtu")
    if err != nil {
        return err
    }
    rtu.ID, err = m.generateRTUID(rtu.Type)
    if err != nil {
        return err
    }
    rtu.RtuAddr = rtu.ID

    // 层次表
    m.db.Hierarchy.RTUs = append(m.db.Hierarchy.RTUs, rtu)
    // 单表添加拷贝
    copyRtu := *rtu
    rtus.RTUs = append(rtus.RTUs, &copyRtu)

    if autoSave {
        return m.save()
    }
    return nil
}

// deleteRTU 内部无锁版本，autoSave=true 时立即写文件
func (m *DbConfigXMLManager) deleteRTU(rtuID string, autoSave bool) error {
    rtu, err := m.getRTUFromHierarchy(rtuID)
    if err != nil {
        return err
    }
    if _, err = m.getRTUFromSingle(rtuID); err != nil {
        return err
    }
    rtus, err := m.getSingleTable("rtu")
    if err != nil {
        return err
    }

    delID := make(map[string]struct{})
    delID[rtuID] = struct{}{}

    // 先删除 RTU 下所有的 device
    for i := range rtu.Devices {
        m.deleteDeviceFromRTU(rtuID, rtu.Devices[i].ID)
    }

    deleteSliceItemByIDs(&m.db.Hierarchy.RTUs, delID)
    deleteSliceItemByIDs(&rtus.RTUs, delID)

    rtuIDInt, err := strconv.Atoi(rtuID)
    if err != nil {
        log.Errorf("RTU ID[%s]转换失败: %v", rtuID, err)
    } else {
        delete(m.rtuID, rtuIDInt)
    }

    if autoSave {
        return m.save()
    }
    return nil
}

// setRTUChannelInternal 设置 RTU 绑定的通道组号和通道号（不 save）
func (m *DbConfigXMLManager) setRTUChannelInternal(rtuID string, channelGroupID, channelID int) error {
    rtu, err := m.getRTUFromHierarchy(rtuID)
    if err != nil {
        return err
    }
    rtu.ChannelGroupID = channelGroupID
    rtu.ChannelID = channelID

    // 同步更新单表
    sRtu, err := m.getRTUFromSingle(rtuID)
    if err != nil {
        return err
    }
    sRtu.ChannelGroupID = channelGroupID
    sRtu.ChannelID = channelID

    return nil
}

// ==================== 公开方法：保持 API 兼容 ====================

// AddRTU 对外接口，始终 save（原有调用者不受影响）
func (m *DbConfigXMLManager) AddRTU(rtu *RTU) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.addRTU(rtu, true)
}

// DeleteRTU 对外接口，始终 save
func (m *DbConfigXMLManager) DeleteRTU(rtuID string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.deleteRTU(rtuID, true)
}

// ==================== 编排方法：原子操作 ====================

// AddRTUWithChannel 创建 RTU 并自动创建通道组+通道，统一加锁，一次 save
func (m *DbConfigXMLManager) AddRTUWithChannel(rtu *RTU, channelGroupID, channelID int) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 1. 创建 RTU（不 save）
    if err := m.addRTU(rtu, false); err != nil {
        return err
    }

    // 2. 设置绑定关系（不 save）
    if err := m.setRTUChannelInternal(rtu.ID, channelGroupID, channelID); err != nil {
        return err
    }

    // 3. 统一 save 一次
    return m.save()
}

// DeleteRTUWithChannel 删除 RTU 并同步删除通道组+通道，统一加锁，一次 save
func (m *DbConfigXMLManager) DeleteRTUWithChannel(rtuID string, deleteCGFunc func(rtuID string) error) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 1. 删除通道组+通道（独立 save，由 feConfig 管理）
    if err := deleteCGFunc(rtuID); err != nil {
        log.Warnf("DeleteRTUWithChannel: delete channel group failed: %v", err)
        // 通道组删除失败不阻塞 RTU 删除（可能是无通道 RTU）
    }

    // 2. 删除 RTU（不 save，最后统一 save）
    if err := m.deleteRTU(rtuID, false); err != nil {
        return err
    }

    // 3. 统一 save
    return m.save()
}
```

---

## 启动时加载（内存中建立关联）

```go
// DbConfigXMLManager 初始化后调用
func (m *DbConfigXMLManager) LoadRTUChannelRelation() error {
    fecfg, err := GetFeConfigXMLManager()
    if err != nil {
        return err
    }

    for _, rtu := range m.db.Hierarchy.RTUs {
        if rtu.ChannelID < 0 {
            continue
        }
        // 通过 ChannelGroupID 从 fe_config 查到 ChannelGroup，再找唯一 Channel
        cg, err := fecfg.GetChannelGroupByID(rtu.ChannelGroupID)
        if err != nil {
            log.Warnf("RTU %s bound channel group %d not found", rtu.ID, rtu.ChannelGroupID)
            continue
        }
        if len(cg.Channels) == 0 {
            log.Warnf("RTU %s channel group %d has no channel", rtu.ID, rtu.ChannelGroupID)
            continue
        }
        rtu.Channel = &cg.Channels[0] // 1:1，取唯一通道
    }
    return nil
}
```

### 启动流程

```
main.go
    ↓
GetDbConfigManager()     → 加载 db_config.xml（含 ChannelGroupID、ChannelID）
GetFeConfigXMLManager()  → 加载 fe_config.xml
    ↓
dbcfg.LoadRTUChannelRelation()
    → 遍历 RTU，用 ChannelGroupID 从 fe_config 查到 ChannelGroup → Channels[0]
    → 填充 rtu.Channel（内存指针）
    ↓
后续查询 RTU 时，直接返回 rtu.Channel，无需跨模块查询
```

---

## FeConfigXMLManager 新增方法

```go
// CreateChannelGroupWithChannel 自动创建通道组+通道（1:1），返回 (channelGroupID, channelID, error)
// 内部逻辑：
//   1. 生成 ChannelGroupID（递增）
//   2. 创建 ChannelGroup（含 Model，STATION_BASE_ID = rtuID）
//   3. 在 ChannelGroup 下创建唯一 Channel（PossibleOwner.ID = rtuID）
//   4. save
func (m *FeConfigXMLManager) CreateChannelGroupWithChannel(cgParam ChannelGroupParam, chParam ChannelParam) (int, int, error)

// DeleteChannelGroupByStationID 按 STATION_BASE_ID 删除 ChannelGroup + Channel（删除 RTU 时调用）
// 内部逻辑：
//   1. 遍历 ChannelGroups，找到 Model.Parameters 中 STATION_BASE_ID == rtuID 的
//   2. 删除该 ChannelGroup（含其下所有 Channel）
//   3. save
func (m *FeConfigXMLManager) DeleteChannelGroupByStationID(rtuID string) error

// GetChannelGroupByStationID 按 STATION_BASE_ID 查询 ChannelGroup
func (m *FeConfigXMLManager) GetChannelGroupByStationID(rtuID string) (*ChannelGroup, error)
```

> 注：现有 `GetChannelsByRtuID` 方法已实现按 `STATION_BASE_ID` 查找 ChannelGroup 的逻辑，可复用。

---

## handler 层编排

### 创建 RTU（改造后）

```go
func AddRTU(c *gin.Context) {
    rtu := emsxmldb.RTU{}
    c.ShouldBindJSON(&rtu)

    if rtu.Name == "" {
        c.JSON(400, ...)
        return
    }

    dbcfg, _ := emsxmldb.GetDbConfigManager()

    // 非通道类型 RTU：直接创建，不创建通道
    if !rtuHasChannel(rtu.Type) {
        err := dbcfg.AddRTU(&rtu)
        if err != nil {
            c.JSON(500, ...)
            return
        }
        c.JSON(200, gin.H{"code": 200, "data": gin.H{"rtuid": rtu.ID}})
        return
    }

    // 有通道的 RTU：先创建通道组+通道，再创建 RTU 并绑定
    fecfg, _ := emsxmldb.GetFeConfigXMLManager()

    cgParam := emsxmldb.ChannelGroupParam{
        Name:           channelGroupName(rtu),
        StationBaseID:  "0", // 临时值，CreateChannelGroupWithChannel 内部会用 rtuID 覆盖
    }
    chParam := emsxmldb.ChannelParam{
        Name: rtu.Name + "的通道",
    }

    cgID, chID, err := fecfg.CreateChannelGroupWithChannel(cgParam, chParam)
    if err != nil {
        c.JSON(500, ...)
        return
    }

    // 创建 RTU 并绑定通道，统一加锁，一次 save
    err = dbcfg.AddRTUWithChannel(&rtu, cgID, chID)
    if err != nil {
        // 回滚：删除刚创建的通道组
        fecfg.DeleteChannelGroup(cgID)
        c.JSON(500, ...)
        return
    }

    c.JSON(200, gin.H{
        "code": 200,
        "data": gin.H{
            "rtuid":           rtu.ID,
            "channelgroupid":  cgID,
            "channelid":       chID,
        },
    })
}

// rtuHasChannel 判断 RTU 类型是否有通道
func rtuHasChannel(rtuType int) bool {
    return rtuType == emsxmldb.RtuTypeAcqu ||
        rtuType == emsxmldb.RtuTypeTrans ||
        rtuType == emsxmldb.RtuTypeAgc
}

// channelGroupName 根据 RTU 类型生成通道组名称
func channelGroupName(rtu emsxmldb.RTU) string {
    if rtu.Type == emsxmldb.RtuTypeTrans {
        return rtu.Name + "的转发通道"
    }
    return rtu.Name + "的通道组"
}
```

### 删除 RTU（改造后）

```go
func DeleteRTU(c *gin.Context) {
    rtuID := c.Param("rid")

    dbcfg, _ := emsxmldb.GetDbConfigManager()
    fecfg, _ := emsxmldb.GetFeConfigXMLManager()

    // 删除 RTU + 通道组，统一加锁
    err := dbcfg.DeleteRTUWithChannel(rtuID, func(id string) error {
        return fecfg.DeleteChannelGroupByStationID(id)
    })
    if err != nil {
        c.JSON(500, ...)
        return
    }

    c.JSON(200, ...)
}
```

---

## 数据流总结

### 创建

```
handler.AddRTU
  ├─ rtuHasChannel(type)?
  │   ├─ 否 → dbcfg.AddRTU(rtu)                    // 单独 save
  │   └─ 是 ↓
  ├─ fecfg.CreateChannelGroupWithChannel(cg, ch)    // feConfig save
  │   → 返回 (cgID, chID)
  └─ dbcfg.AddRTUWithChannel(rtu, cgID, chID)       // dbConfig 原子 save
      ├─ addRTU(rtu, false)
      ├─ setRTUChannelInternal(rtu.ID, cgID, chID)
      └─ save()                                     // 一次写入
      失败 → fecfg.DeleteChannelGroup(cgID)         // 回滚通道组
```

### 删除

```
handler.DeleteRTU
  └─ dbcfg.DeleteRTUWithChannel(rtuID, deleteFunc)
      ├─ deleteFunc(rtuID)                          // feConfig: 删通道组+通道
      │   → fecfg.DeleteChannelGroupByStationID
      ├─ deleteRTU(rtuID, false)                    // 内存删除
      └─ save()                                     // 一次写入
```

### 查询

```
handler.GetRTUs
  └─ dbcfg.GetAllRTUs()
      → 返回 []*RTU，含 ChannelGroupID, ChannelID, Channel（启动时已填充）
      前端通过 channelgroupid >= 0 判断是否有通道
```

### 查询全部通道（新增）

```
handler.GetAllChannels
  ├─ dbcfg.GetAllRTUs()
  │   → 构建 map[channelGroupID]*RTU
  └─ fecfg.ListAllChannelGroups()
      → 遍历每个 ChannelGroup 下的 Channels
      → 通过 channelGroupID 关联 RTU 信息
      → 扁平化输出 []ChannelVO
```

响应结构体 `ChannelVO`：

```go
type ChannelVO struct {
    ID               int               `json:"id"`
    Name             string            `json:"name"`
    Plugins          []emsxmldb.Plugin `json:"plugins"`
    RTUID            string            `json:"rtuid"`
    RTUName          string            `json:"rtuname"`
    ChannelGroupID   int               `json:"channelgroupid"`
    ChannelGroupName string            `json:"channelgroupname"`
}
```

修改文件：

| 文件 | 修改内容 |
|---|---|
| `handlers/dbconfig.go` | 新增 `ChannelVO` 结构体 + `GetAllChannels` handler |
| `routes/routes.go` | 在 `dbcfgGroup` 中注册 `GET /channels` 路由 |

---

## 待确认事项

| 问题 | 选项 | 状态 |
|---|---|---|
| 绑定方式 | 三方 1:1:1：RTU 记录 ChannelGroupID+ChannelID，ChannelGroup 通过 STATION_BASE_ID 绑定 RTU，Channel 通过 PossibleOwner.ID 绑定 RTU | ✅ 已确定 |
| 通道组处理 | 一个通道组仅一个通道，自动创建通道组+通道（内部抽象，不改 fe_config.xml 格式） | ✅ 已确定 |
| 新建时 Channel 配置 | 不需要配置，填空，与原新建通道接口一致 | ✅ 已确定 |
| 已有 RTU 处理 | 不管，无存量项目 | ✅ 已确定 |
| 删除策略 | 同步删除 ChannelGroup+Channel，后续接口统一以 RTU 为 key | ✅ 已确定 |
| autoSave 模式 | 内部方法 addRTU(rtu, autoSave) 控制是否立即 save，编排方法统一原子写入 | ✅ 已确定 |
| 有通道 RTU 类型 | 采集(1)、转发(2)、AGC(3) 有通道；监视(4)、公式(5) 无通道 | ✅ 已确定 |
| 无通道标志 | 无通道 RTU 的 ChannelGroupID=-1, ChannelID=-1，前端通过 channelgroupid >= 0 判断 | ✅ 已确定 |
| 全部通道查询 | 新增 GET /dbcfg/channels，扁平化所有通道，附带 RTU 归属信息 | ✅ 已确定 |

## 更新通道数据（统一到 RTU 修改接口）

### 方案

通道配置的修改统一到 `PUT /api/v1/log/dbcfg/rtu/:rid`，请求体在原有 RTU 字段基础上增加通道相关字段：

```json
{
  "name": "储能RTU1",
  "inservice": "1",
  "channelname": "储能RTU1的通道",
  "channlecomment": "注释",
  "plugins": [...]
}
```

- 有通道字段 → 同时更新 `ChannelGroup` 和 `Channel`
- 无通道字段 → 只更新 RTU 信息

不需要独立的 `PUT /rtu/:rid/channel` 接口，前端一个请求搞定。

| 问题 | 严重程度 | 说明 |
|---|---|---|
| `LoadRTUChannelRelation` 未实现 | 重要 | 启动时 `rtu.Channel` 内存字段不会被填充，查询 RTU 时拿不到通道完整信息 |
| `rtuIDToInt` 返回 0 有歧义 | 轻微 | 0 可能与 `Host.ID` 冲突，建议失败时返回 -1 |
| `GenerateRTUID` 泄漏 | 中等 | 创建 RTU 失败时，预生成的 ID 不会释放，建议回滚路径补上 `ReleaseRTUID` |

---

## 更新通道数据（通过 RTU ID）

---

## 相关笔记

- [[项目架构总览]]
- [[emsxmldb模块]]
