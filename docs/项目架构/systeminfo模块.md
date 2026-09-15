# systeminfo 模块

> 路径: `systeminfo/` | 核心职责: 系统级信息获取与配置

---

## 文件结构

| 文件 | 职责 |
|---|---|
| `systeminfo.go` | 系统信息获取（CPU/内存/磁盘/进程） |
| `debianipconfig.go` | Debian 网络配置（nmcli 操作） |

---

## 网络配置 (`debianipconfig.go`)

通过调用系统 `nmcli` 命令管理网络：

```go
type NetworkDeviceInfo struct {
    DeviceName  string `json:"device_name"`
    ConnName    string `json:"conn_name"`
    AddressMode string `json:"address_mode"` // static/dhcp
    IPAddress   string `json:"ip_address"`
    SubnetMask  string `json:"subnet_mask"`
    Gateway     string `json:"gateway"`
    DNS         string `json:"dns"`
    BackupDNS   string `json:"backup_dns"`
}
```

### 主要接口

- `GetAllNetworkDevices()` — 查询所有网络设备
- `ConfigNetworkDevice(...)` — 配置网络（静态IP/DHCP）

---

## 系统信息

通过 ZMQ 与系统监控进程通信获取：

| 信息 | 说明 |
|---|---|
| CPU 使用率 | 实时推送 |
| 内存使用 | 实时推送 |
| 磁盘使用 | 实时推送 |
| 进程状态 | 进程列表及运行状态 |

ZMQ 端点: `tcp://127.0.0.1:9971` (REQ) / `tcp://127.0.0.1:9972` (SUB)

---

## 相关笔记

- [[项目架构总览]]
- [[zmq模块]]
- [[handlers模块]]
