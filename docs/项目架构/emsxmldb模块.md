# emsxmldb 模块

> 路径: `emsxmldb/` | 核心职责: XML 配置文件的读写管理

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `dbconfig.go` | RTU/设备/测点 XML 配置（核心，2000+ 行） |
| `userxml.go` | 用户 XML 配置（登录/CRUD） |
| `feconfig.go` | 前置通道 XML 配置 |
| `systemconfig.go` | 系统 XML 配置 |

---

## 数据模型（`dbconfig.go`）

### 核心接口

```go
type IDer interface {
    GetID() string
    SetID(string)
    GetName() string
}

type Pointer interface {
    IDer
    GetGin() string
    SetGin(string)
    GetTrans() string
    GetDataSource() string
    Clone() Pointer
}
```

### 层级结构

```
DB
├── Single（单表）  ← 所有测点的扁平列表
│   ├── analog (遥测, 102) / trans_analog (转发遥测, 107)
│   ├── digital (遥信, 101) / trans_digital (转发遥信, 106)
│   ├── pulse (脉冲, 105) / trans_pulse (转发脉冲, 110)
│   ├── regulate (遥调, 104) / trans_regulate (转发遥调, 109)
│   ├── control (遥控, 103) / trans_control (转发遥控, 108)
│   ├── attribute (属性, 115) / trans_attribute (转发属性, 116)
│   ├── formula (公式, 111)
│   └── formula_factor (公式因子, 112)
└── Hierarchy（层级）
    └── RTU[]
        └── Device[]
            └── Points (YC/YX/YK/YT/YM/SX)
```

### 关键类型

```go
type BaseInfo struct {
    ID   string `xml:"ID,attr" json:"id"`
    Name string `xml:"Name,attr" json:"name"`
}

type BasePoint struct {
    BaseInfo
    Gin        string `xml:"Gin,attr"`       // 点号
    Trans      string `xml:"Trans,attr"`     // 是否转发
    MqttKey    string `xml:"MqttKey,attr"`   // MQTT 键
    DataSource string `xml:"DataSource,attr"`
    Path       string `xml:"Path,attr"`
}
```

### DbConfigXMLManager

```go
type DbConfigXMLManager struct {
    filePath        string            // XML 文件路径
    exportJsonPath  string            // AGC RTU 导出路径
    mu              sync.Mutex        // 并发安全锁
    db              *DB               // 内存中的 XML 数据
    rtuID           map[int]struct{}  // 已存在 RTU ID
    deviceID        int               // 最大设备 ID
    formulaID       int               // 最大公式 ID
    templateFile    *excelize.File    // Excel 模板
    appendixMapping *AppendixMapping  // 附录映射
}
```

---

## 用户管理 (`userxml.go`)

用户配置存储于 XML 文件: `{configBasePath}/webdata/user.xml`

```go
type User struct {
    Id   string `xml:"Id,attr"`
    Name string `xml:"Name,attr"`
    Pwd  string `xml:"Pwd,attr"`
    Type int    `xml:"Type,attr"`   // 1=admin, 2=config, 3=ops, 4=guest
    Role string `xml:"Role"`        // 角色英文名（冗余）
}
```

- `Login(name, pwd)` — 读取 XML 校验密码
- `ReadUsersFromXML()` / `WriteUsersToXML()` — CRUD 操作
- 使用 `xmlFileMutex` 保证并发安全

---

## 并发安全

- `DbConfigXMLManager` 使用 `sync.Mutex` 保护所有读写操作
- `colPathsCache` 使用 `sync.Map` 缓存反射路径

---

## tableNo 与测点类型绑定关系

tableNo 嵌入在测点的完整 OID 中（第 3 段），固定绑定测点物理类型，不区分普通/转发：

| tableNo | 表名 | 测点类型 | Device 层级字段 |
|---------|------|---------|----------------|
| 101 | digital | 遥信 YX | `Device.YX.Digitals` |
| 102 | analog | 遥测 YC | `Device.YC.Analogs` |
| 103 | control | 遥控 YK | `Device.YK.Controls` |
| 104 | regulate | 遥调 YT | `Device.YT.Regulates` |
| 105 | pulse | 遥脉 YM | `Device.YM.Pulses` |
| 106 | trans_digital | 转发遥信 | `Device.YX.Digitals` |
| 107 | trans_analog | 转发遥测 | `Device.YC.Analogs` |
| 108 | trans_control | 转发遥控 | `Device.YK.Controls` |
| 109 | trans_regulate | 转发遥调 | `Device.YT.Regulates` |
| 110 | trans_pulse | 转发遥脉 | `Device.YM.Pulses` |
| 111 | formula | 公式 | `Table.Formulas` |
| 112 | formula_factor | 公式因子 | `Table.FormulaFactors` |
| 115 | attribute | 属性 SX | `Device.SX.Attributes` |
| 116 | trans_attribute | 转发属性 | `Device.SX.Attributes` |

普通类型（101-105, 115）和对应转发类型（106-110, 116）共用同一个 Device 字段。在 Single 表的 `addItemsToSingleTable` 中，两者也共用同一个切片（如 `"analog"` 和 `"trans_analog"` 都 append 到 `Table.Analogs`）。

## 转发 RTU（rtuid 500-699）

rtuid 在 **500-699** 范围内的 RTU 为**转发 RTU**，其下所有 Device 的测点全部为转发表。区分普通点和转发点依据的是 **rtuid 所在范围**，而非 tableNo 本身。

在 Hierarchy 层级结构中，转发点与普通点同样存在于 `RTU → Device → Points` 层级中。查找时先按 rtuid 定位到对应 RTU，再按 tableNo 定位到具体 Device 字段，最后按 gin 匹配即可。

---

## Excel 导入导出

通过 `excelize/v2` 支持：
- `ExportAllPointsToBuffer()` — 导出测点到 Excel buffer
- `ImportPointsFromExcel()` — 从 Excel 导入测点
- 内置模板文件 (`template.xlsx`)

---

## 相关笔记

- [[项目架构总览]]
- [[handlers模块]]
- [[用户认证体系]]
