# agcconfig 模块

> 路径: `agcconfig/` | 核心职责: AGC（自动发电控制）策略与设备模型管理

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `agcstragety.go` | AGC 策略管理（CRUD、运行控制） |
| `agcmodel.go` | AGC 设备模型管理（设备类型、设备实例） |

---

## 核心功能

### 策略管理

- 策略 CRUD（增删改查）
- 策略参数管理
- 策略运行/停止
- 运行冲突检查

### 设备模型

- 设备类型树（父子关系）
- 设备实例 CRUD
- 设备名称树查询
- RTU 测点查询（供 AGC 选择测点）

---

## 使用模式

通过单例模式获取管理实例：

```go
ins, err := agcconfig.GetModelInstance()
```

---

## 依赖

- [[emsxmldb模块]] — XML 配置持久化
- [[cgolibparse模块]] — Cgo 公式解析（策略参数校验）
- [[emsclient模块]] — RPC 查询 EMS 实时数据

---

## 相关笔记

- [[项目架构总览]]
- [[handlers模块]]
- [[API路由地图]]
