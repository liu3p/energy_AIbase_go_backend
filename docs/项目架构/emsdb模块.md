# emsdb 模块

> 路径: `emsdb/` | 核心职责: 数据库操作（InfluxDB 时序数据 + 用户数据 + 网关数据）

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `influxdbhandler.go` | InfluxDB 时序数据库操作（历史数据查询/写入/导出） |
| `userdbhandler.go` | 用户数据库操作 |
| `gatewayhandler.go` | 网关数据处理 |
| `systemhandler.go` | 系统数据处理 |
| `oldgatewayhandler.go` | 旧版网关数据（兼容） |

### influxdbhandler.go 关键函数

| 函数 | 用途 |
|---|---|
| `QueryHistoryData` | 按类型批量查询历史数据，时间插值网格锚定 `start_time`（`aggregateWindow(every, offset, fn:last, createEmpty:false)`，首点由 `first()` 补入），遥信取整、遥测/遥脉保留 3 位小数（`/his/query`、`/his/home` 用） |
| `QueryRawHistoryData` | 查询单测点原始数据，**不聚合、不插值、不取整**（`/his/export` 用） |
| `WriteRawHistoryCSV` | 把原始数据写成 `时间,测点ID,值` 长表 CSV，带 UTF-8 BOM |
| `ExportHistoryDataToCSV` | 导出为宽表 CSV 并落盘到服务端（`/his` 用） |
| `GetQueryAPI` | 懒初始化并返回 InfluxDB `api.QueryAPI` |

---

## InfluxDB 集成

使用 `influxdata/influxdb-client-go/v2` 客户端连接 InfluxDB，用于：
- 历史遥测数据存储
- 历史遥信数据存储
- 数据导出（`ExportInfluxDBData` 接口）

---

## 相关笔记

- [[项目架构总览]]
- [[handlers模块]]
