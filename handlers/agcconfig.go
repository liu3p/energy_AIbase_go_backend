package handlers

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"webserver-arm/agcconfig"
	"webserver-arm/cgolibparse"
	"webserver-arm/emsxmldb"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
)

func AllStrategies(c *gin.Context) {
	log.Info("AllStrategies in ...")

	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("AllStrategies: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	strategies, err := ins.GetAllStrategies()
	if err != nil {
		log.Errorf("AllStrategies: GetAllStrategies failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	type ResultItem struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	var result []ResultItem
	for _, s := range strategies {
		result = append(result, ResultItem{
			Name: s.Name,
			Desc: s.Desc,
		})
	}
	log.Info("AllStrategies:", result)
	c.JSON(http.StatusOK, result)
}

func GetStrategies(c *gin.Context) {
	log.Info("GetStrategies in ...")
	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("GetStrategies: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp, err := ins.GetStrategies()
	if err != nil {
		log.Errorf("GetStrategies: GetStrategies failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func AddStrategy(c *gin.Context) {
	log.Info("AddStrategy in ...")
	agcStrategy := agcconfig.Strategy{}
	err := c.ShouldBindJSON(&agcStrategy)
	if err != nil {
		log.Errorf("AddStrategy: ShouldBindJSON failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("agcStrategy:", agcStrategy)

	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("AddStrategy: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = ins.AddStrategy(agcStrategy.Name)
	if err != nil {
		log.Errorf("AddStrategy: AddStrategy failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 添加策略后需要更新模型配置里面的动态库列表
	modelIns, err := agcconfig.GetModelInstance()
	if err != nil {
		log.Errorf("AddStrategy: GetModelInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = modelIns.UpdateParaAndDynPara()
	if err != nil {
		log.Errorf("AddStrategy: UpdateParaAndDynPara failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	log.Info("AddStrategy OK")
	c.JSON(http.StatusOK, agcStrategy)
	return
}

func AddStrategies(c *gin.Context) {
	log.Info("AddStrategies in ...")
	var agcStrategy []agcconfig.Strategy
	err := c.ShouldBindJSON(&agcStrategy)
	if err != nil {
		log.Errorf("AddStrategies: ShouldBindJSON failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("AddStrategies: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	for _, s := range agcStrategy {
		err := ins.AddStrategy(s.Name)
		if err != nil {
			log.Errorf("AddStrategies: AddStrategy failed: '%w'", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
			return
		}
	}

	// 添加策略后需要更新模型配置里面的动态库列表
	modelIns, err := agcconfig.GetModelInstance()
	if err != nil {
		log.Errorf("AddStrategies: GetModelInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = modelIns.UpdateParaAndDynPara()
	if err != nil {
		log.Errorf("AddStrategies: UpdateParaAndDynPara failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	log.Info("AddStrategies OK")
	c.JSON(http.StatusOK, agcStrategy)
}

func RemoveStrategy(c *gin.Context) {
	strategyName := c.Param("strategyname")
	log.Info("RemoveStrategy in ... strategyName:", strategyName)
	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("RemoveStrategy: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = ins.RemoveStrategy(strategyName)
	if err != nil {
		log.Errorf("RemoveStrategy: RemoveStrategy failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 删除策略后需要更新模型配置里面的动态库列表
	modelIns, err := agcconfig.GetModelInstance()
	if err != nil {
		log.Errorf("RemoveStrategy: GetModelInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = modelIns.UpdateParaAndDynPara()
	if err != nil {
		log.Errorf("RemoveStrategy: UpdateParaAndDynPara failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	log.Info("RemoveStrategy OK")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "RemoveStrategy OK"})
}

func GetStrategyParams(c *gin.Context) {
	strategyName := c.Param("strategyname")
	log.Info("GetStrategyParams in ... strategyName:", strategyName)

	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("GetStrategyParams: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	strategyParams, err := ins.GetStrategyParams(strategyName)
	if err != nil {
		log.Errorf("GetStrategyParams: GetStrategyParams failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, strategyParams)
}

func SetStrategyParams(c *gin.Context) {
	strategyName := c.Param("strategyname")
	log.Info("SetStrategyParams in ...  strategyname:", strategyName)

	var settings []cgolibparse.SettingItem
	err := c.ShouldBindJSON(&settings)
	if err != nil {
		log.Errorf("SetStrategyParams: ShouldBindJSON failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("settings:", settings)
	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("SetStrategyParams: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = ins.SetStrategyParams(strategyName, settings)
	if err != nil {
		log.Errorf("SetStrategyParams: SetStrategyParams failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "SetStrategyParams success"})
	log.Info("SetStrategyParams success")
	return
}

func RunStrategy(c *gin.Context) {
	strategyName := c.Param("strategyname")
	log.Info("RunStrategy in ... strategyName:", strategyName)
	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("RunStrategy: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = ins.SetStrategyParams(strategyName, []cgolibparse.SettingItem{{Name: "Activate", Value: "1", Type: "DIGITAL"}})
	if err != nil {
		log.Errorf("RunStrategy: SetStrategyParams failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "RunStrategy success"})
	log.Info("RunStrategy success")
}

func CheckRunConflict(c *gin.Context) {
	strategyName := c.Param("strategyname")
	log.Info("CheckRunConflict in ... strategyName:", strategyName)
	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("CheckRunConflict: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	conflicts, err := ins.CheckPriorityConflict(strategyName)
	if err != nil {
		log.Errorf("CheckRunConflict: CheckPriorityConflict failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("CheckRunConflict ", strategyName, " over, conflicts:", conflicts)
	c.JSON(http.StatusOK, conflicts)
}

func StopStrategy(c *gin.Context) {
	strategyName := c.Param("strategyname")
	log.Info("StopStrategy in ... strategyName:", strategyName)
	ins, err := agcconfig.GetStrategyInstance()
	if err != nil {
		log.Errorf("StopStrategy: GetStrategyInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = ins.SetStrategyParams(strategyName, []cgolibparse.SettingItem{{Name: "Activate", Value: "0", Type: "DIGITAL"}})
	if err != nil {
		log.Errorf("StopStrategy: SetStrategyParams failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "StopStrategy success"})
}

// @Summary		添加agc设备
// @Description	添加agc设备，需先登录 modify by zhh 2026.4.8
// @ID			add-agc-device
// @Tags		agc
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param 		device_name 	path 		string 				true 	"待添加设备的父设备名称"
// @param       device          body        agcconfig.Device 	true 	"设备信息"
// @Success		200				{object}	agcconfig.Device	"添加成功，返回子设备信息"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router		/api/v1/log/agc/device/{device_name} [POST]
func AddAgcDevice(c *gin.Context) {
	log.Info("AddAgcDevice in ...")
	var device agcconfig.Device
	// 前端参数反序列化
	err := c.BindJSON(&device)
	if err != nil {
		log.Errorf("AddAgcDevice: BindJSON failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 获取模型管理单例
	ins, err := agcconfig.GetModelInstance()
	if err != nil {
		log.Errorf("AddAgcDevice: GetModelInstance failed '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 调用单例接口添加设备
	err = ins.AddAgcDevice(c.Param("device_name"), &device)
	if err != nil {
		log.Errorf("AddAgcDevice: add device failed '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, device)
}

// @Summary		查询agc设备信息
// @Description	查询agc设备信息，需先登录 modify by zhh 2026.4.8
// @ID			get-agc-device
// @Tags		agc
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param 		device_name 	path 		string 				true 	"待查询设备名称"
// @Success		200				{object}	agcconfig.Device	"查询成功，返回设备信息"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router		/api/v1/log/agc/device/{device_name} [GET]
func GetAgcDevice(c *gin.Context) {
	log.Info("GetAgcDevice in ...")

	// 获取模型管理单例
	ins, err := agcconfig.GetModelInstance()
	if err != nil {
		log.Errorf("GetAgcDevice: GetModelInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 调用单例查询设备信息
	device, err := ins.GetAgcDevice(c.Param("device_name"))
	if err != nil {
		log.Errorf("GetAgcDevice: get device failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, device)
}

// @Summary		修改agc设备信息
// @Description	修改agc信息，需先登录 modify by zhh 2026.4.8
// @ID			update-agc-device
// @Tags		agc
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param 		device_name 	path 		string 				true 	"待修改设备名称"
// @param       device          body        agcconfig.Device 	true 	"设备信息"
// @Success		200				{object}	agcconfig.Device	"修改成功，返回修改后的设备信息"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router		/api/v1/log/agc/device/{device_name} [PUT]
func PutAgcDevice(c *gin.Context) {
	log.Info("PutAgcDevice in ...")

	// 获取前端参数
	stationData, err := c.GetRawData()
	if err != nil {
		log.Errorf("PutAgcDevice: GetRawData failed: '%v'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 获取模型管理单例
	ins, err := agcconfig.GetModelInstance()
	if err != nil {
		log.Errorf("PutAgcDevice: GetModelInstance failed: '%v'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 调用单例更新设备信息
	device, err := ins.UpdateAgcDevice(c.Param("device_name"), string(stationData))
	if err != nil {
		log.Errorf("PutAgcDevice: UpdateAgcDevice failed: '%v'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, device)
}

// @Summary		删除agc设备
// @Description	删除agc设备，需先登录 modify by zhh 2026.4.8
// @ID			delete-agc-device
// @Tags		agc
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param 		device_name 	path 		string 				true 	"待删除设备名称"
// @Success		200				{object}	agcconfig.Device	"删除成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router		/api/v1/log/agc/device/{device_name} [DELETE]
func DeleteAgcDevice(c *gin.Context) {
	stname := c.Param("device_name")
	log.Info("DeleteAgcDevice in ... stname:", stname)

	// 获取模型管理单例
	ins, err := agcconfig.GetModelInstance()
	if err != nil {
		log.Errorf("DeleteAgcDevice: GetModelInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 调用单例删除设备
	err = ins.RemoveAgcDevice(stname)
	if err != nil {
		log.Errorf("DeleteAgcDevice: RemoveAgcDevice failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "DeleteAgcDevice success"})
}

// @Summary		获取所有设备列表
// @Description	获取所有设备列表，需先登录 modify by zhh 2026.4.2
// @ID			get-all-devicetypes
// @Tags		agc
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string		true	"JWT Token"	default(<token>)
// @Success		200				{object}	[]string			"查询成功，返回所有设备的字符串数组"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router		/api/v1/log/agc/devicetype/all_devicetypes [GET]
func GetAvailableAgcDeviceTypes(c *gin.Context) {
	log.Info("GetAvailableAgcDeviceTypes in ...")

	children, err := cgolibparse.GetDeviceChildren("")
	if err != nil {
		log.Errorf("GetAvailableAgcDeviceTypes: GetDeviceChildren failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, children)
}

// @Summary		获取当前设备下可以挂在哪些子设备
// @Description	获取当前设备下可以挂在哪些子设备，需先登录 add by zhh 2026.4.2
// @ID			get-children-devicetypes
// @Tags		agc
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string		true	"JWT Token"	default(<token>)
// @Param 		device_type 	path 		string 		true 	"设备类型"
// @Success		200				{object}	[]string			"查询成功，返回设备子类型字符串数组"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router		/api/v1/log/agc/devicetype/{device_type} [GET]
func GetDeviceChildren(c *gin.Context) {
	log.Info("GetDeviceChildren in ...")

	deviceType := c.Param("device_type")
	children, err := cgolibparse.GetDeviceChildren(deviceType)
	if err != nil {
		log.Errorf("GetDeviceChildren: GetDeviceChildren failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, children)
}

// @Summary		获取【根厂站】的设备名称层级树
// @Description	获取【根厂站】的设备名称层级树，需先登录 add by zhh 2026.4.2
// @ID			get-device-name
// @Tags		agc
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string		true	"JWT Token"	default(<token>)
// @Success		200				{object}	agcconfig.DeviceNameNode	"查询成功，返回设备子类型字符串数组"
// @Failure		401				{object}	models.CommonResp			"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp			"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp			"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router		/api/v1/log/agc/devicetype/{device_type} [GET]
func GetDeviceNameTree(c *gin.Context) {
	log.Info("GetDeviceNameTree in ...")

	// 获取模型管理单例
	ins, err := agcconfig.GetModelInstance()
	if err != nil {
		log.Errorf("GetDeviceNameTree: GetModelInstance failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 调用单例删除设备
	deviceName, err := ins.GetDeviceNameTree()
	if err != nil {
		log.Errorf("GetDeviceNameTree: GetDeviceNameTree failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, deviceName)
}

type GetAgcRtuPointsReq struct {
	SelectType string `json:"selectType"`
	Name       string `json:"name"`
	ParaType   string `json:"paraType"`
	DataType   string `json:"dataType"`
}

// @Summary		获取AGC RTU可添加的测点信息
// @Description	根据设备名称、参数类型、数据类型筛选获取AGC RTU可添加的测点列表，需登录 add by zhh 2026.4.21
// @ID			get-agc-rtu-points
// @Tags		agc
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param 		req 			body		GetAgcRtuPointsReq 			true	"查询参数"
// @Success		200				{object}	[]agcconfig.AgcRtuModelPoints	"查询成功，返回AGC RTU测点列表"
// @Failure		401				{object}	models.CommonResp				"认证失败"
// @Failure		403				{object}	models.CommonResp				"拒绝访问"
// @Failure		500				{object}	models.CommonResp				"服务器内部错误"
// @Router		/api/v1/log/agc/rtu_points [POST]
func GetAgcRtuPoints(c *gin.Context) {
	log.Info("GetAgcRtuPoints in ...")
	req := GetAgcRtuPointsReq{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Errorf("GetAgcRtuPoints: ShouldBindJSON failed: '%w'", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 获取dbcfg单例
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetAgcRtuPoints: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// 获取agc rtu下所有已添加的测点
	allPointsPath, err := dbcfg.GetAgcRtuPointsName(req.DataType)
	if err != nil {
		log.Errorf("GetAgcRtuPoints: GetAgcRtuPointsID failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	if req.SelectType == "model" {
		// 获取模型管理单例
		modelIns, err := agcconfig.GetModelInstance()
		if err != nil {
			log.Errorf("GetAgcRtuPoints: GetModelInstance failed: '%w'", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
			return
		}

		// 调用单例获取rtu点信息
		agcPoints := modelIns.GetAgcRtuModelPoints(req.Name, req.ParaType, req.DataType, allPointsPath)
		c.JSON(http.StatusOK, agcPoints)
	} else if req.SelectType == "strategy" {
		// 获取策略管理单例
		strategyIns, err := agcconfig.GetStrategyInstance()
		if err != nil {
			log.Errorf("GetAgcRtuPoints: GetStrategyInstance failed: '%w'", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
			return
		}
		// 调用单例获取rtu点信息
		agcStrategyPoints := strategyIns.GetAgcRtuModelPoints(req.Name, req.ParaType, req.DataType, allPointsPath)
		c.JSON(http.StatusOK, agcStrategyPoints)
	} else {
		c.JSON(http.StatusInternalServerError, "selectType参数错误")
	}
}

// valueFormatKeys 需要格式化 value 字段的 key 集合
var valueFormatKeys = map[string]bool{
	"para":     true,
	"dyn_para": true,
}

// formatStationModelParaValues 递归遍历 station_model JSON 树，
// 对所有 para[].value 和 dyn_para[].value 为数字形式的值保留至多三位小数。
// 返回格式化后的新结构，不修改原始数据。
func formatStationModelParaValues(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		cp := make(map[string]interface{}, len(v))
		for key, val := range v {
			if valueFormatKeys[key] {
				cp[key] = formatValueArray(val)
			} else {
				cp[key] = formatStationModelParaValues(val)
			}
		}
		return cp
	case []interface{}:
		cp := make([]interface{}, len(v))
		for i, item := range v {
			cp[i] = formatStationModelParaValues(item)
		}
		return cp
	default:
		return v
	}
}

// formatValueArray 处理 para/dyn_para 数组，对每个元素的 value 字段做三位小数格式化
func formatValueArray(arr interface{}) interface{} {
	items, ok := arr.([]interface{})
	if !ok {
		return arr
	}
	cp := make([]interface{}, len(items))
	for i, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			cp[i] = item
			continue
		}
		cm := make(map[string]interface{}, len(m))
		for k, v := range m {
			cm[k] = v
		}
		if val, exists := cm["value"]; exists && val != nil {
			cm["value"] = formatNumericValue(val)
		}
		cp[i] = cm
	}
	return cp
}

// formatNumericValue 对 value 做三位小数格式化，支持 string 和 float64 类型
// string 返回 string，float64 返回 float64，保持原类型不变
func formatNumericValue(val interface{}) interface{} {
	switch v := val.(type) {
	case string:
		return roundToAtMost3Decimal(v)
	case float64:
		return math.Round(v*1000) / 1000
	default:
		return val
	}
}

// roundToAtMost3Decimal 数字字符串保留至多三位小数（去尾零），非数字原样返回
func roundToAtMost3Decimal(s string) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	formatted := strconv.FormatFloat(f, 'f', 3, 64)
	if dot := strings.Index(formatted, "."); dot != -1 {
		formatted = strings.TrimRight(formatted, "0")
		formatted = strings.TrimRight(formatted, ".")
	}
	return formatted
}

func GetAgcData(t int) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info("GetAgcData in ...")

		client := &http.Client{Timeout: 5 * time.Second}

		fetchModel := func(model string) interface{} {
			resp, err := client.Get("http://127.0.0.1:8500/?model=" + model)
			if err != nil {
				log.Errorf("GetAgcData: fetch %s failed: %v", model, err)
				return nil
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Errorf("GetAgcData: read %s body failed: %v", model, err)
				return nil
			}

			var result interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				log.Errorf("GetAgcData: unmarshal %s failed: %v", model, err)
				return nil
			}
			return result
		}

		result := gin.H{}

		switch t {
		case 1:
			result["station_model"] = formatStationModelParaValues(fetchModel("station_model"))
		case 2:
			result["strategy_model"] = fetchModel("strategy_model")
		case 3:
			result["station_model"] = formatStationModelParaValues(fetchModel("station_model"))
			result["strategy_model"] = fetchModel("strategy_model")
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type, expected 1, 2 or 3"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
