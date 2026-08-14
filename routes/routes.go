package routes

import (
	"webserver-arm/handlers"
	"webserver-arm/middleware"

	"github.com/gin-gonic/gin"

	_ "webserver-arm/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func LoadRoutes(r *gin.Engine) {

	v1Group := r.Group("/api/v1")
	loginedGroup := v1Group.Group("/log")
	loginedGroup.Use(middleware.Auth())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 用户接口
	userGroup := loginedGroup.Group("/user")
	userGroup.POST("/updatepwd", handlers.UpdatePwd)
	userGroup.POST("/resetpwd", handlers.ResetPwd)   // admin
	userGroup.POST("/kickout", handlers.KickOutUser) // admin
	userGroup.GET("/onlines", handlers.OnlinesUser)  // admin
	v1Group.POST("/user/login", handlers.LoginUser)
	v1Group.POST("/user/refresh_token", handlers.RefreshToken)
	userGroup.POST("/logout", handlers.LogoutUser)
	userGroup.GET("/info", handlers.UserInfo)
	userGroup.GET("/all", handlers.AllUsers) // admin

	// 推送
	loginedGroup.GET("/ws/push", handlers.WebSocketHandler)

	// 系统全局  comment by zhh 2026.4.1 这部分接口已弃用
	systemGroup := loginedGroup.Group("/system")
	systemGroup.GET("/project/sn", handlers.GetProjectSn)  // 设备序列号
	systemGroup.POST("/project/sn", handlers.SetProjectSn) // 仅限调试人员  configure
	v1Group.GET("/system/modelnum", handlers.GetModelNum)  // 设备型号
	systemGroup.POST("/modelnum", handlers.SetModelNum)
	systemGroup.GET("/hostname", handlers.GetHostname) // 主机名
	systemGroup.POST("/hostname", handlers.SetHostname)
	// TODO 软件版本号， docker环境变量？

	// TODO 系统信息，cpu 内存，磁盘，网络，订阅zmq（带推送）

	systemGroup.GET("/network/config", handlers.GetNetworkConfig)
	systemGroup.POST("/network/config", handlers.SetNetworkConfig)
	systemGroup.POST("/reboot", handlers.SystemReboot)

	systemGroup.GET("/ws/sysinfo", handlers.WsGetSystemInfo)
	systemGroup.GET("/ws/process", handlers.WsGetProcessInfo)
	systemGroup.POST("/process/:procid", handlers.OperProcess) //  访客不允许
	systemGroup.GET("/process/:procid/logdump", handlers.WsGetProcessLog)

	// 国际化翻译
	systemGroup.GET("/i18n", handlers.GetI18N)

	v1Group.GET("/system/i18n", handlers.GetI18N)
	v1Group.GET("/system/i18nmb", handlers.GetI18NModbus)
	// 点位的单位
	systemGroup.GET("/dataunit", handlers.GetDataUnit)
	v1Group.GET("/system/dataunit", handlers.GetDataUnit)
	v1Group.GET("/system/dataunitmb", handlers.GetDataUnitModbus)

	// 历史记录查询

	// 时区信息
	systemGroup.GET("/zone", handlers.GetZoneInfo)
	systemGroup.GET("/timezone", handlers.GetSystemTime)
	systemGroup.POST("/timezone", handlers.SetSystemTimeZone)
	systemGroup.POST("/time", handlers.SetSystemTime) //

	v1Group.GET("/system/zone", handlers.GetZoneInfo)

	systemGroup.GET("/baseinfo", handlers.GetSysBaseInfo)
	systemGroup.POST("/baseinfo", handlers.SetSysBaseInfo)

	systemGroup.GET("/netconfig", handlers.GetSysNetConfig)
	systemGroup.POST("/netconfig", handlers.SetSysNetConfig)
	systemGroup.GET("/log_list/process/:processname", handlers.GetLogList)
	systemGroup.POST("/download/log_file", handlers.DownloadLogFile)

	// 升级管理
	operationGroup := loginedGroup.Group("/operations")
	operationGroup.POST("/update/:filename", handlers.RunUpdate)
	operationGroup.GET("/backup", handlers.Backup)
	operationGroup.GET("/download/:filename", handlers.DownloadFile)

	// agc strategy

	agcGroup := loginedGroup.Group("/agc")
	agcGroup.GET("/allstrategies", handlers.AllStrategies)
	agcGroup.GET("/strategies", handlers.GetStrategies)
	agcGroup.POST("/strategy", handlers.AddStrategy)
	agcGroup.POST("/strategies", handlers.AddStrategies)
	agcGroup.DELETE("/strategy/:strategyname", handlers.RemoveStrategy)
	agcGroup.GET("/strategy/:strategyname/params", handlers.GetStrategyParams)
	agcGroup.POST("/strategy/:strategyname/params", handlers.SetStrategyParams)
	agcGroup.POST("/strategy/:strategyname/run", handlers.RunStrategy)
	agcGroup.GET("/strategy/:strategyname/runconflict", handlers.CheckRunConflict)
	agcGroup.DELETE("/strategy/:strategyname/run", handlers.StopStrategy)

	// agc model
	agcGroup.GET("/devicetype/all_devicetypes", handlers.GetAvailableAgcDeviceTypes) // 获取所有设备类型
	agcGroup.GET("/devicetype/:device_type", handlers.GetDeviceChildren)             // 获取子设备类型

	agcGroup.POST("/device/:device_name", handlers.AddAgcDevice)      // 新增设备
	agcGroup.GET("/all_device_names", handlers.GetDeviceNameTree)     // 获取所有设备名称
	agcGroup.GET("/all_devices", handlers.GetAgcDevice)               // 获取所有设备
	agcGroup.GET("/device/:device_name", handlers.GetAgcDevice)       // 获取某个指定设备信息
	agcGroup.PUT("/device/:device_name", handlers.PutAgcDevice)       // 修改设备信息
	agcGroup.DELETE("/device/:device_name", handlers.DeleteAgcDevice) // 删除设备

	agcGroup.POST("/rtu_points", handlers.GetAgcRtuPoints)  // 获取agc rtu可添加的测点信息
	agcGroup.GET("/agc_data", handlers.GetAgcData(3))       // 获取ml模型数据
	agcGroup.GET("/station_model", handlers.GetAgcData(1))  // 获取ml模型数据
	agcGroup.GET("/strategy_model", handlers.GetAgcData(2)) // 获取ml模型数据

	// db config
	dbcfgGroup := loginedGroup.Group("/dbcfg")
	dbcfgGroup.GET("/rtus", handlers.GetRTUs)                                          //  获取所有RTU
	dbcfgGroup.GET("/rtus/except_points", handlers.GetRTUsExceptPoints)                //  获取所有RTU，不包含测点信息
	dbcfgGroup.POST("/rtu", handlers.AddRTU)                                           //  添加RTU
	dbcfgGroup.PUT("/rtu/:rid", handlers.UpdateRTU)                                    //  修改RTU
	dbcfgGroup.DELETE("/rtu/:rid", handlers.DeleteRTU)                                 //  删除RTU
	dbcfgGroup.GET("/rtu/:rid/devices", handlers.GetRtuDevices)                        //  获取RTU下的所有设备
	dbcfgGroup.GET("/rtu/:rid/device/:did", handlers.GetRtuDevice)                     //  获取RTU下的某个指定设备
	dbcfgGroup.POST("/rtu/:rid/device", handlers.AddRtuDevice)                         //  添加RTU下的设备
	dbcfgGroup.POST("/source_rtu/:rid/source_device/:did", handlers.AddTransRtuDevice) //  添加转发RTU下的设备
	dbcfgGroup.PUT("/rtu/:rid/device/:did", handlers.UpdateRtuDevice)                  //  修改RTU下的设备
	dbcfgGroup.DELETE("/rtu/:rid/device/:did", handlers.DeleteRtuDevice)               //  删除RTU下的设备

	dbcfgGroup.GET("/rtu/:rid/device/:did/points", handlers.GetPointsFromDevice)                  //  获取RTU下的设备下的所有点
	dbcfgGroup.GET("/rtu/:rid/device/:did/type/:type/points", handlers.GetPointsFromDeviceByType) //  获取RTU下的设备下所有指定类型的点
	dbcfgGroup.POST("/trans_points", handlers.GetTransPointsByType)                               //  获取RTU下的设备下所有指定类型的点
	dbcfgGroup.GET("/rtu/:rid/device/:did/excel", handlers.GetRtuDeviceExcel)                     //  导出RTU下的设备下的所有点
	dbcfgGroup.POST("/rtu/:rid/device/:did/import_excel", handlers.ImportPointsFromExcel)         //  导入execl文件
	dbcfgGroup.POST("/rtu/:rid/device/:did/points", handlers.AddRtuDevicePoints)                  //  添加RTU下的设备下的点
	dbcfgGroup.DELETE("/rtu/:rid/device/:did/points", handlers.DeleteRtuDevicePoints)             //  删除RTU下的设备下的点
	dbcfgGroup.DELETE("/rtu/:rid/device/:did/points_by_id", handlers.DeleteRtuDevicePointByID)    //  根据id删除RTU下的设备下的点
	dbcfgGroup.POST("/rtu/:rid/device/:did/update_points", handlers.UpdateDevicePointsByID)       //  根据id更新RTU下的设备下的点

	dbcfgGroup.GET("/rtu/:rid/device/:did/point/:pid/formula", handlers.GetPointFormula)
	dbcfgGroup.POST("/rtu/:rid/device/:did/point/:pid/formula", handlers.UpdatePointFormula)
	//dbcfgGroup.DELETE("/rtu/:rid/device/:did/points/:pid/formula", handlers.DeletePointFormula)
	dbcfgGroup.POST("/formula/check", handlers.CheckFormula)
	dbcfgGroup.GET("/points/:ptype", handlers.GetPointsByType)

	// fe config
	fecfgGroup := loginedGroup.Group("/fecfg")
	fecfgGroup.GET("/cgroups", handlers.GetChannelGroups)                     // 查询全部通道组
	fecfgGroup.POST("/cgroup", handlers.AddChannelGroup)                      // 新增通道组
	fecfgGroup.PUT("/cgroup/:cgid", handlers.UpdateChannelGroup)              // 修改通道组
	fecfgGroup.DELETE("/cgroup/:cgid", handlers.DeleteChannelGroup)           // 删除通道组
	fecfgGroup.GET("/cgroup/:cgid/channels", handlers.GetCGChannels)          // 查询通道组下的所有通道
	fecfgGroup.POST("/cgroup/:cgid/channel", handlers.AddCGChannel)           // 新增通道
	fecfgGroup.PUT("/cgroup/:cgid/channel/:cid", handlers.UpdateCGChannel)    // 修改通道
	fecfgGroup.DELETE("/cgroup/:cgid/channel/:cid", handlers.DeleteCGChannel) /// 删除通道
	// 获取通道的协议插件
	fecfgGroup.GET("/plugins", handlers.GetChannelPlugins)
	fecfgGroup.POST("/notifyreload", handlers.NotifyFeConfigReload)

	// data monitor
	datamonitorGroup := loginedGroup.Group("/datamonitor")
	datamonitorGroup.GET("/rtu/:rid/channels", handlers.GetRtuChannels)                                  // query channels
	datamonitorGroup.GET("/cgroup/:cgid/channel/:cid/status", handlers.GetRtuChannelStatus)              // ws push channel status
	datamonitorGroup.GET("/cgroup/:cgid/channel/:cid/plugin/:pluginid/logs", handlers.GetRtuChannelLogs) // ws push channel logs
	datamonitorGroup.GET("/rtu/:rid/device/:did/points/:ptype", handlers.GetPointsData)                  // ws
	datamonitorGroup.POST("/rtu/:rid/device/:did/point/:pid/control", handlers.SetPointControl)
	datamonitorGroup.POST("/rtu/:rid/device/:did/point/:pid/regulate", handlers.SetPointRegulate)
	datamonitorGroup.GET("/channels/status", handlers.GetAllRtuChannelsStatus)

	influxDBGroup := loginedGroup.Group("/influxdb")
	influxDBGroup.POST("/his", handlers.ExportInfluxDBData)
	influxDBGroup.POST("/his/query", handlers.QueryInfluxDBData)
	influxDBGroup.POST("/his/home", handlers.QueryHomeHistory)

	// homepage config
	homepageGroup := loginedGroup.Group("/homepage")
	homepageGroup.GET("/config", handlers.GetHomepageConfig)
	homepageGroup.POST("/config/topology", handlers.UpdateHomepageTopology)
	homepageGroup.POST("/config/realtime", handlers.UpdateHomepageRealtime)
	homepageGroup.POST("/config/basic_info", handlers.UpdateHomepageBasicInfo)
	homepageGroup.POST("/config/power_level", handlers.UpdateHomepagePowerLevel)
}
