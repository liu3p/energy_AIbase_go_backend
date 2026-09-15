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
