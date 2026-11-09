# API 路由地图

> 所有路由定义在 `routes/routes.go`，前缀 `/api/v1`，认证组 `/api/v1/log`

---

## 认证相关（无需登录）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/user/login` | 用户登录（成功后响应 `license` 字段合并许可证状态） |
| POST | `/api/v1/user/refresh_token` | 刷新 Token |

---

## 用户管理 `/api/v1/log/user`

| 方法 | 路径 | Handler | 说明 | 权限 |
|---|---|---|---|---|
| POST | `/updatepwd` | `UpdatePwd` | 修改密码 | 所有用户 |
| POST | `/resetpwd` | `ResetPwd` | 重置密码 | admin |
| POST | `/kickout` | `KickOutUser` | 踢出用户 | admin |
| GET | `/onlines` | `OnlinesUser` | 在线用户列表 | admin |
| POST | `/logout` | `LogoutUser` | 登出 | 所有用户 |
| GET | `/info` | `UserInfo` | 当前用户信息 | 所有用户 |
| GET | `/all` | `AllUsers` | 所有用户列表 | admin |

---

## 注册码许可 `/api/v1/log/license`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| GET | `/machine_serial` | `GetMachineSerial` | 查询本机机器码（向厂商申请注册码用，持久化后跨请求一致） |
| POST | `/register` | `RegisterLicense` | 提交注册码（校验并保存） |
| GET | `/status` | `GetLicenseStatus` | 查询许可证状态（有效 / 无效（含过期）/ 未注册） |

---

## WebSocket `/api/v1/log/ws`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| GET | `/push` | `WebSocketHandler` | WebSocket 连接，接收实时推送 |

---

## 系统管理 `/api/v1/log/system`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| GET | `/project/sn` | `GetProjectSn` | 设备序列号 |
| POST | `/project/sn` | `SetProjectSn` | 设置序列号（调试） |
| GET | `/modelnum` | `GetModelNum` | 设备型号 |
| POST | `/modelnum` | `SetModelNum` | 设置设备型号 |
| GET | `/hostname` | `GetHostname` | 主机名 |
| POST | `/hostname` | `SetHostname` | 设置主机名 |
| GET | `/network/config` | `GetNetworkConfig` | 网络配置 |
| POST | `/network/config` | `SetNetworkConfig` | 设置网络配置 |
| POST | `/reboot` | `SystemReboot` | 重启系统 |
| GET | `/ws/sysinfo` | `WsGetSystemInfo` | 系统信息（WebSocket） |
| GET | `/ws/process` | `WsGetProcessInfo` | 进程信息（WebSocket） |
| POST | `/process/:procid` | `OperProcess` | 操作进程 |
| GET | `/process/:procid/logdump` | `WsGetProcessLog` | 进程日志（WebSocket） |
| GET | `/i18n` | `GetI18N` | 国际化翻译 |
| GET | `/dataunit` | `GetDataUnit` | 点位单位 |
| GET | `/zone` | `GetZoneInfo` | 时区信息 |
| GET | `/timezone` | `GetSystemTime` | 系统时间 |
| POST | `/timezone` | `SetSystemTimeZone` | 设置时区 |
| POST | `/time` | `SetSystemTime` | 设置时间 |
| GET | `/baseinfo` | `GetSysBaseInfo` | 基础信息 |
| POST | `/baseinfo` | `SetSysBaseInfo` | 设置基础信息 |
| GET | `/netconfig` | `GetSysNetConfig` | 网络设备配置 |
| POST | `/netconfig` | `SetSysNetConfig` | 配置网络设备 |
| GET | `/log_list/process/:processname` | `GetLogList` | 日志文件列表 |
| POST | `/download/log_file` | `DownloadLogFile` | 下载日志文件 |

---

## 升级管理 `/api/v1/log/operations`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| POST | `/update/:filename` | `RunUpdate` | 执行升级 |
| GET | `/backup` | `Backup` | 备份 |
| GET | `/download/:filename` | `DownloadFile` | 下载文件 |

---

## AGC 策略 `/api/v1/log/agc`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| GET | `/allstrategies` | `AllStrategies` | 所有策略列表 |
| GET | `/strategies` | `GetStrategies` | 获取策略 |
| POST | `/strategy` | `AddStrategy` | 新增策略 |
| POST | `/strategies` | `AddStrategies` | 批量新增策略 |
| DELETE | `/strategy/:strategyname` | `RemoveStrategy` | 删除策略 |
| GET | `/strategy/:strategyname/params` | `GetStrategyParams` | 获取策略参数 |
| POST | `/strategy/:strategyname/params` | `SetStrategyParams` | 设置策略参数 |
| POST | `/strategy/:strategyname/run` | `RunStrategy` | 运行策略 |
| GET | `/strategy/:strategyname/runconflict` | `CheckRunConflict` | 检查运行冲突 |
| DELETE | `/strategy/:strategyname/run` | `StopStrategy` | 停止策略 |

---

## AGC 设备模型 `/api/v1/log/agc`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| GET | `/devicetype/all_devicetypes` | `GetAvailableAgcDeviceTypes` | 所有设备类型 |
| GET | `/devicetype/:device_type` | `GetDeviceChildren` | 子设备类型 |
| POST | `/device/:device_name` | `AddAgcDevice` | 新增设备 |
| GET | `/all_device_names` | `GetDeviceNameTree` | 所有设备名称树 |
| GET | `/all_devices` | `GetAgcDevice` | 所有设备 |
| GET | `/device/:device_name` | `GetAgcDevice` | 指定设备信息 |
| PUT | `/device/:device_name` | `PutAgcDevice` | 修改设备 |
| DELETE | `/device/:device_name` | `DeleteAgcDevice` | 删除设备 |
| POST | `/rtu_points` | `GetAgcRtuPoints` | 获取 AGC RTU 测点 |
| GET | `/station_model` | `GetAgcData(1)` | 厂站模型（para/dyn_para.value 数字保留三位小数） |
| GET | `/strategy_model` | `GetAgcData(2)` | 策略模型 |
| GET | `/agc_data` | `GetAgcData(3)` | 厂站+策略模型（station_model 同上格式化） |

---

## 数据库配置 `/api/v1/log/dbcfg`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| GET | `/rtus` | `GetRTUs` | 所有 RTU（`channel.plugins.parameters` 按字母序，`STATION_ID ASDU_ADDR` 置顶） |
| GET | `/rtus/except_points` | `GetRTUsExceptPoints` | RTU 列表（不含测点，parameters 排序同上） |
| GET | `/channels` | `GetAllChannels` | 所有通道（扁平列表，含 RTU 归属信息，parameters 排序同上） |
| POST | `/rtu` | `AddRTU` | 添加 RTU |
| PUT | `/rtu/:rid` | `UpdateRTU` | 修改 RTU（含通道多层协议 `channel.plugins`） |
| DELETE | `/rtu/:rid` | `DeleteRTU` | 删除 RTU |
| GET | `/rtu/:rid/devices` | `GetRtuDevices` | RTU 下所有设备 |
| GET | `/rtu/:rid/device/:did` | `GetRtuDevice` | 指定设备 |
| POST | `/rtu/:rid/device` | `AddRtuDevice` | 添加设备 |
| PUT | `/rtu/:rid/device/:did` | `UpdateRtuDevice` | 修改设备 |
| DELETE | `/rtu/:rid/device/:did` | `DeleteRtuDevice` | 删除设备 |
| GET | `/rtu/:rid/device/:did/points` | `GetPointsFromDevice` | 设备下所有测点 |
| GET | `/rtu/:rid/device/:did/type/:type/points` | `GetPointsFromDeviceByType` | 按类型获取测点 |
| POST | `/trans_points` | `GetTransPointsByType` | 获取转发测点 |
| GET | `/rtu/:rid/device/:did/excel` | `GetRtuDeviceExcel` | 导出 Excel |
| POST | `/rtu/:rid/device/:did/import_excel` | `ImportPointsFromExcel` | 导入 Excel |
| POST | `/rtu/:rid/device/:did/points` | `AddRtuDevicePoints` | 添加测点 |
| DELETE | `/rtu/:rid/device/:did/points` | `DeleteRtuDevicePoints` | 删除测点 |
| DELETE | `/rtu/:rid/device/:did/points_by_id` | `DeleteRtuDevicePointByID` | 按 ID 删除测点 |
| POST | `/rtu/:rid/device/:did/update_points` | `UpdateDevicePointsByID` | 按 ID 更新测点 |
| GET | `/rtu/:rid/device/:did/point/:pid/formula` | `GetPointFormula` | 获取公式 |
| POST | `/rtu/:rid/device/:did/point/:pid/formula` | `UpdatePointFormula` | 更新公式 |
| POST | `/formula/check` | `CheckFormula` | 校验公式 |
| GET | `/points/:ptype` | `GetPointsByType` | 按类型获取所有测点 |

---

## 前置通道配置 `/api/v1/log/fecfg`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| GET | `/cgroups` | `GetChannelGroups` | 通道组列表 |
| POST | `/cgroup` | `AddChannelGroup` | 新增通道组 |
| PUT | `/cgroup/:cgid` | `UpdateChannelGroup` | 修改通道组 |
| DELETE | `/cgroup/:cgid` | `DeleteChannelGroup` | 删除通道组 |
| GET | `/cgroup/:cgid/channels` | `GetCGChannels` | 通道组下通道 |
| POST | `/cgroup/:cgid/channel` | `AddCGChannel` | 新增通道 |
| PUT | `/cgroup/:cgid/channel/:cid` | `UpdateCGChannel` | 修改通道 |
| DELETE | `/cgroup/:cgid/channel/:cid` | `DeleteCGChannel` | 删除通道 |
| GET | `/plugins` | `GetChannelPlugins` | 协议插件列表（按层分组，层号=`id/1000+1`，只返回非空层；parameters 排序同上） |
| POST | `/notifyreload` | `NotifyFeConfigReload` | 通知 FE 重载配置 |
| POST | `/channels/restart` | `RestartChannels` | 重启指定通道（kill -9 fe_channel 进程） |

---

## 数据监控 `/api/v1/log/datamonitor`

| 方法   | 路径                                                 | Handler                   | 说明       |
| ---- | -------------------------------------------------- | ------------------------- | -------- |
| GET  | `/rtu/:rid/channels`                               | `GetRtuChannels`          | 查询通道（parameters 排序同上）    |
| GET  | `/cgroup/:cgid/channel/:cid/status`                | `GetRtuChannelStatus`     | 通道状态（WS） |
| GET  | `/cgroup/:cgid/channel/:cid/plugin/:pluginid/logs` | `GetRtuChannelLogs`       | 通道日志（WS） |
| GET  | `/rtu/:rid/device/:did/points/:ptype`              | `GetPointsData`           | 测点数据（WS，analog 类型 rawvalue/currvalue 保留三位小数） |
| POST | `/rtu/:rid/device/:did/point/:pid/control`         | `SetPointControl`         | 遥控       |
| POST | `/rtu/:rid/device/:did/point/:pid/regulate`        | `SetPointRegulate`        | 遥调       |
| GET  | `/channels/status`                                 | `GetAllRtuChannelsStatus` | 所有通道状态   |

---

## 首页配置 `/api/v1/log/homepage`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| GET | `/config` | `GetHomepageConfig` | 获取首页配置（type==1 的 show_value 保留三位小数） |
| POST | `/config/topology` | `UpdateHomepageTopology` | 更新拓扑图配置 |
| POST | `/config/realtime` | `UpdateHomepageRealtime` | 更新实时数据配置 |
| POST | `/config/basic_info` | `UpdateHomepageBasicInfo` | 更新基本信息配置 |
| POST | `/config/power_level` | `UpdateHomepagePowerLevel` | 更新功率等级配置 |

---

## InfluxDB `/api/v1/log/influxdb`

| 方法 | 路径 | Handler | 说明 |
|---|---|---|---|
| POST | `/his` | `ExportInfluxDBData` | 导出历史数据到服务端 CSV 文件（仅落盘，不返回文件内容） |
| POST | `/his/query` | `QueryInfluxDBData` | 查询历史数据（支持时间插值） |
| POST | `/his/home` | `QueryHomeHistory` | 首页历史数据查询（自动从配置读取 OID，返回 topology + power_level 两个维度） |
| GET | `/his/export` | `ExportRawHistory` | 导出单测点原始历史数据为 CSV（不做插值/取整，文件流直接返回） |

---

## 非认证公开接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/swagger/*any` | Swagger 文档 |
| GET | `/api/v1/system/modelnum` | 设备型号（无需登录） |
| GET | `/api/v1/system/i18n` | 国际化翻译（无需登录） |
| GET | `/api/v1/system/i18nmb` | Modbus 国际化（无需登录） |
| GET | `/api/v1/system/dataunit` | 点位单位（无需登录） |
| GET | `/api/v1/system/dataunitmb` | Modbus 单位（无需登录） |
| GET | `/api/v1/system/zone` | 时区信息（无需登录） |

---

## 相关笔记

- [[项目架构总览]]
- [[用户认证体系]]
- [[handlers模块]]
- [[功能设计/注册码license功能实现方案]]
