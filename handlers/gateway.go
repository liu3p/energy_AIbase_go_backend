package handlers

import (
	"net/http"
	"time"
	"webserver-arm/emsclient"
	"webserver-arm/emsdb"
	"webserver-arm/log"
	"webserver-arm/middleware"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
)

const (
	LW_HIS_TOPIC     = "LW_HIS_TOPIC"
	LW_HIS_TOPIC_RET = "LW_HIS_TOPIC_RET"
)

//nolint:govet
type QueryGatewaysResp struct {
	Topic   string                         `json:"topic"`
	Content map[string]*models.GatewayInfo `json:"content"`
}

// type HisReqMsg struct {
// 	gormsg.Rmsg
// 	CodeName string `json:"codename"`
// }

func needUpdateCache(c *gin.Context) bool {
	uName := c.GetString(middleware.UNameKey)
	if data, ok := OnlineUsers.Load(uName); ok {
		online, match := data.(*OnlineUser)
		if match && (time.Now().Unix()-online.LastUpdateCache) < 3600 { // one hour
			log.Error("no need update cache. now:", time.Now().Unix(), ", last:", online.LastUpdateCache, ", diff:", (time.Now().Unix() - online.LastUpdateCache))
			return false
		}
		online.LastUpdateCache = time.Now().Unix()
		OnlineUsers.Store(uName, online)
	}
	return true
}

// @Summary		查询网关
// @Description	查询全部网关信息
// @ID				query-gateways
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	QueryGatewaysResp	"查询成功 {\"topic\": \"gateways\", \"content\": {\"gateway_sn1\":{\"name\":\"\",\"address\":\"\", \"status\":2}, "gateway_sn2\":{\"name\":\"\",\"address\":\"\", \"status\":2}}}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways [get]
func QueryGateways(c *gin.Context) {
	log.Info("QueryGateways in ...")
	gwInfos, err := emsdb.GetGateways()
	if err != nil {
		log.Error("GetGateways error: %v", err)
		errResp := &models.CommonResp{Code: http.StatusNotFound, Message: "gateway not found"}
		c.JSON(http.StatusNotFound, errResp)
		return
	}

	result := QueryGatewaysResp{
		Topic:   "gateways",
		Content: make(map[string]*models.GatewayInfo),
	}
	for _, gwInfo := range gwInfos {
		result.Content[gwInfo.Sn] = &models.GatewayInfo{Sn: gwInfo.Sn, Name: gwInfo.Name, Status: gwInfo.Status}
		log.Info("QueryGateways sn:", gwInfo.Sn, ", name:", gwInfo.Name)
	}

	c.JSON(http.StatusOK, result)
}

type DevInfoSet map[string]models.DeviceInfo

var allDeviceInfo DevInfoSet
var gwDeviceInfo map[string]DevInfoSet

func getAllDevice() (allDev DevInfoSet, gwDev map[string]DevInfoSet, err error) {

	if allDeviceInfo != nil && gwDeviceInfo != nil {
		return allDeviceInfo, gwDeviceInfo, nil
	}

	if allDeviceInfo == nil {
		allDeviceInfo = make(DevInfoSet)
	}

	if gwDeviceInfo == nil {
		gwDeviceInfo = make(map[string]DevInfoSet)
	}

	gwInfos, err := emsdb.GetGateways() //ClientMgr.GetGatewayInfo("")
	if err != nil {
		return nil, nil, err
	}

	for _, gwInfo := range gwInfos {
		devData, err := emsdb.GetDevices(gwInfo.Sn)
		if err != nil {
			log.Debug("getAllDevice GetDevices err:", err.Error())
			return allDeviceInfo, gwDeviceInfo, err
		}

		for devSn, devinfo := range devData.Content {
			allDeviceInfo[devSn] = devinfo
		}

		gwDeviceInfo[gwInfo.Sn] = devData.Content

	}

	return allDeviceInfo, gwDeviceInfo, nil
}

// @Summary		查询子设备
// @Description	查询指定网关下子设备信息
// @ID				query-devices
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Success		200				{object}	models.QueryDevicesResp	"查询成功 {\"topic\": \"devices\", \"content\": {\"device_sn1\":{\"name\":\"\",\"dev_code\":\"\"}, "device_sn2\":{\"name\":\"\",\"dev_code\":\"\"}}}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/devices [get]
func QueryDevices(c *gin.Context) {
	log.Info("QueryDevices in ...")
	gatewaySn := c.Param("gateway_sn")

	devData, err := emsdb.GetDevices(gatewaySn)
	if err != nil {
		log.Error("GetDevices error:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	c.JSON(http.StatusOK, devData)
}

// @Summary		查询子设备遥测
// @Description	查询指定网关下某子设备遥测
// @ID				query-devices-yc
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Param			device_sn		path		string				true	"子设备sn"	default(11223355esscab1pcs1)
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"tm\",\"content\":{\"11223355esscab1pcs1\":[{\"at\":1699263620341,\"k\":\"Uab\",\"mt\":1699263620341,\"v\":400}]}}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/devices/{device_sn}/telemetry [get]
func QueryDeviceTelemetry(c *gin.Context) {
	gatewaySn := c.Param("gateway_sn")
	deviceSn := c.Param("device_sn")
	updateCache := needUpdateCache(c)
	log.Info("QueryDeviceTelemetry in. gateway_sn:", gatewaySn, ",deviceSn:", deviceSn, ", updateCache:", updateCache)

	teleData, err := emsdb.GetDevicesYc([]string{deviceSn}, updateCache)
	if err != nil {
		log.Error("QueryDeviceTelemetry GetDevicesYc err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	c.JSON(http.StatusOK, teleData)
}

type DeviceList []string

// @Summary		查询多个子设备遥测
// @Description	查询指定网关下多个子设备遥测
// @ID				query-multi-devices-yc
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Param			devices			body		DeviceList			true	"设备sn数组"	default([\"11223355esscab1pcs1\",\"11223355esscab1ac11\"])
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"tm\",\"content\":{\"11223355esscab1pcs1\":[{\"at\":1699263620341,\"k\":\"Uab\",\"mt\":1699263620341,\"v\":400}],\"11223355esscab1ac11\":[{\"at\":1699263620341,\"k\":\"Runstate\",\"mt\":1699263620341,\"v\":0}]}}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/dev_telemetry [post]
func QueryMultiDeviceTelemetry(c *gin.Context) {
	reqInfo := make([]string, 0)
	err := c.ShouldBindJSON(&reqInfo)
	if err != nil {
		log.Error("ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}
	gatewaySn := c.Param("gateway_sn")
	updateCache := needUpdateCache(c)
	log.Info("QueryMultiDeviceTelemetry in. gateway_sn:", gatewaySn, " dev_sns:", reqInfo, ", updateCache:", updateCache)

	teleData, err := emsdb.GetDevicesYc(reqInfo, updateCache)
	if err != nil {
		log.Error("QueryDeviceTelemetry GetDevicesYc err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	c.JSON(http.StatusOK, teleData)
}

// @Summary		查询子设备遥信
// @Description	查询指定网关下某子设备遥信
// @ID				query-devices-yx
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Param			device_sn		path		string				true	"子设备sn"	default(11223355esscab1pcs1)
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"ts\",\"content\":{\"11223355esscab1pcs1\":[{\"at\":1699263620341,\"k\":\"Uab\",\"mt\":1699263620341,\"v\":400}]}}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/devices/{device_sn}/telesignal [get]
func QueryDeviceTelesignal(c *gin.Context) {
	gatewaySn := c.Param("gateway_sn")
	deviceSn := c.Param("device_sn")

	var reqInfo emsclient.QueryYXReq
	reqInfo = append(reqInfo, deviceSn)

	updateCache := needUpdateCache(c)

	log.Info("QueryDeviceTelesignal in. gateway_sn:", gatewaySn, ",reqInfo:", reqInfo, ", updateCache:", updateCache)

	teleData, err := emsdb.GetDevicesYx([]string{deviceSn}, updateCache)
	if err != nil {
		log.Error("QueryDeviceTelemetry GetDevicesYx err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	c.JSON(http.StatusOK, teleData)
}

// @Summary		查询多个子设备遥信
// @Description	查询指定网关下多个子设备遥信
// @ID				query-multi-devices-yx
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Param			devices			body		DeviceList			true	"设备sn数组"	default(["11223355esscab1pcs1","11223355esscab1ac11"])
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"ts\",\"content\":{\"11223355esscab1pcs1\":[{\"at\":1699263620341,\"k\":\"Uab\",\"mt\":1699263620341,\"v\":400}],\"11223355esscab1ac11\":[{\"at\":1699263620341,\"k\":\"Runstate\",\"mt\":1699263620341,\"v\":0}]}}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/dev_telesignal [post]
func QueryMultiDeviceTelesignal(c *gin.Context) {
	log.Info("QueryMultiDeviceTelesignal in ...")

	reqInfo := make([]string, 0)
	err := c.ShouldBindJSON(&reqInfo)
	if err != nil {
		log.Error("ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}
	gatewaySn := c.Param("gateway_sn")
	updateCache := needUpdateCache(c)
	log.Info("QueryMultiDeviceTelemetry in. gateway_sn:", gatewaySn, ", dev_sns:", reqInfo, ", updateCache:", updateCache)

	teleData, err := emsdb.GetDevicesYx(reqInfo, updateCache)
	if err != nil {
		log.Error("QueryDeviceTelemetry GetDevicesYx err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	c.JSON(http.StatusOK, teleData)
}

// @Summary		查询网关属性
// @Description	查询指定网关下属性
// @ID				query-gateway-prop
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"prop\",\"content\":[{\"at\":1699263620341,\"k\":\"Uab\",\"mt\":1699263620341,\"v\":400}]}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务器内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/prop [get]
func QueryGatewayProp(c *gin.Context) {
	// TODO
	c.JSON(http.StatusNotFound, models.CommonResp{Code: http.StatusNotFound, Message: "gateway not found"})
	// gatewaySn := c.Param("gateway_sn")

	// log.Info("QueryGatewayProp in. gateway_sn:", gatewaySn)

	// var reqInfo emsclient.QueryPropReq
	// reqInfo = append(reqInfo, gatewaySn)

	// client := ClientMgr.GetClient(gatewaySn)
	// if client == nil {
	// 	log.Error("GetClient Failed, gateway_sn:", gatewaySn)
	// 	c.JSON(http.StatusNotFound, models.CommonResp{Code: http.StatusNotFound, Message: "gateway not found"})
	// 	return
	// }

	// resp, err := client.QueryProp(&reqInfo)
	// if err != nil {
	// 	log.Debug("QueryGatewayProp QueryProp err: ", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Server Error"})
	// 	return
	// }

	// log.Debug("QueryGatewayProp resp:", resp)
	// c.JSON(http.StatusOK, resp.Data)

	/*
		gatewaySn := c.Param("gateway_sn")

		log.Info("QueryDeviceProp in. gateway_sn:", gatewaySn)

		var reqInfo emsclient.QueryPropReq
		reqInfo = append(reqInfo, gatewaySn)

		// updateCache := needUpdateCache(c) need?

		propData, err := emsdb.GetDevicesProp(reqInfo)
		if err != nil {
			log.Error("QueryGatewayProp GetDevicesProp err:", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			return
		}

		c.JSON(http.StatusOK, propData)*/
}

// @Summary		查询子设备属性
// @Description	查询指定网关下某子设备属性
// @ID				query-devices-prop
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Param			device_sn		path		string				true	"子设备sn"	default(11223355esscab1pcs1)
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"prop\",\"content\":{\"11223355esscab1pcs1\":[{\"at\":1699263620341,\"k\":\"Uab\",\"mt\":1699263620341,\"v\":400}]}}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/devices/{device_sn}/prop [get]
func QueryDeviceProp(c *gin.Context) {
	// gatewaySn := c.Param("gateway_sn")
	// deviceSn := c.Param("device_sn")

	// log.Info("QueryDeviceProp in. gateway_sn:", gatewaySn, ",device_sn:", deviceSn)

	// var reqInfo emsclient.QueryPropReq
	// reqInfo = append(reqInfo, deviceSn)

	// // updateCache := needUpdateCache(c) need?

	// propData, err := emsdatahub.GetDevicesProp(reqInfo)
	// if err != nil {
	// 	log.Error("QueryDeviceProp GetDevicesProp err:", err)
	// 	c.JSON(http.StatusInternalServerEc.JSON(http.forbi)rror, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Service Error"})
	// 	return
	// }

	// c.JSON(http.StatusOK, propData)
	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
		Message: "Internal Service Error"})
}

type DeviceRemoteData struct {
	MqttID string  `json:"mqttid"`
	Value  float32 `json:"value"`
}

// @Summary		查询子设备遥控点
// @Description	查询指定网关下某子设备的遥控点
// @ID				query-devices-remote
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Param			device_sn		path		string				true	"子设备sn"	default(esscab1pcs1)
// @Success		200				{object}	DeviceRemoteData		"查询成功 [{\"mqttid\":\"OnOff\",\"value\":0}]"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/devices/{device_sn}/remote [get]
func QueryDeviceRemote(c *gin.Context) {
	gatewaySn := c.Param("gateway_sn")
	deviceSn := c.Param("device_sn")

	log.Info("QueryDeviceRemote in. gateway_sn:", gatewaySn, ",device_sn:", deviceSn)

	userType := c.GetString(middleware.UTypeKey)
	if userType != UserTypeOper && userType != UserTypeConfig {
		log.Error("QueryDeviceRemote  just return empty. userType is :", userType)
		result := make([]DeviceRemoteData, 0)
		c.JSON(http.StatusOK, result)
		return
	}

	values, err := emsdb.GetDevicesYk(deviceSn, false)
	if err != nil {
		log.Error("QueryDeviceAdjust GetDevicesYt err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}
	result := make([]DeviceRemoteData, 0)
	for k, v := range values {
		entry := DeviceRemoteData{
			MqttID: k,
			Value:  float32(v),
		}
		result = append(result, entry)
	}

	log.Info("QueryDeviceRemote result:", result)
	c.JSON(http.StatusOK, result)
}

// @Summary		查询子设备遥设点
// @Description	查询指定网关下某子设备的遥设点
// @ID				query-devices-adjust
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn"
// @Param			device_sn		path		string				true	"子设备sn"	default(esscab1pcs1)
// @Success		200				{object}	DeviceRemoteData		"查询成功 [{\"mqttid\":\"OnOff\",\"value\":0}]"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/devices/{device_sn}/adjust [get]
func QueryDeviceAdjust(c *gin.Context) {
	gatewaySn := c.Param("gateway_sn")
	deviceSn := c.Param("device_sn")

	log.Info("QueryDeviceAdjust in. gateway_sn:", gatewaySn, ",device_sn:", deviceSn)

	userType := c.GetString(middleware.UTypeKey)
	if userType != UserTypeOper && userType != UserTypeConfig {
		log.Error("QueryDeviceAdjust just return empty. userType is :", userType)
		result := make([]DeviceRemoteData, 0)
		c.JSON(http.StatusOK, result)
		return
	}

	values, err := emsdb.GetDevicesYt(deviceSn, false)
	if err != nil {
		log.Error("QueryDeviceAdjust GetDevicesYt err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	result := make([]DeviceRemoteData, 0)
	for k, v := range values {
		entry := DeviceRemoteData{
			MqttID: k,
			Value:  v,
		}
		result = append(result, entry)
	}

	log.Info("QueryDeviceAdjust result:", result)
	c.JSON(http.StatusOK, result)
}

// Seq    string `json:"seq"`
// Methon string `json:"method"`
// Data   string `json:"data"`
//
//nolint:govet
type RemoteReq struct {
	ProjSn   string      `json:"sn"`
	Checkpwd string      `json:"checkpwd"`
	Content  interface{} `json:"content"`
}

type RemoteReqState struct {
	State int `json:"state"`
}

// @Summary		子设备遥控/调
// @Description	查询指定网关下某子设备属性
// @ID				devices-remote
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			gateway_sn		path		string						true	"网关sn"		Example("esscab1")
// @Param			device_sn		path		string						true	"子设备sn"		example("esscab1ac1")
// @Param			method			path		string						true	"方法名"		example("OnOff")
// @Param			RemoteReq		body		RemoteReq					true	"遥控/调。content的内容是{state: 0}格式的json结构，【不是字符串】，数值填写在state中。 checkpwd的内容是用户密码做校验用"
// @Success		200				{object}	emsclient.PushRemoteResp	"下发成功 {\"topic\": \"\", \"content\": \"6\"}"
// @Failure		400				{object}	models.CommonResp			"下发失败 5-应答失败,7-发令超时,8-撤消,9-应答超时,10-异常,11-遥控/调点不存在 {\"code\": 5, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp			"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp			"没找到gateway {\"code\": 404, \"msg\": \"\"}"
// @Failure		408				{object}	models.CommonResp			"遥控下发响应超时 {\"code\": 408, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp			"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/devices/{device_sn}/remote/{method} [post]
func DeviceRemote(c *gin.Context) {
	log.Info("DeviceRemote in ...")
	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
		Message: "Internal Service Error"})

	// gatewaySn := c.Param("gateway_sn")
	// deviceSn := c.Param("device_sn")
	// method := c.Param("method") // mqttid
	// remoteReq := &RemoteReq{}
	// err := c.ShouldBindJSON(remoteReq)
	// if err != nil {
	// 	log.Error("ShouldBindJSON err:", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Service Error"})
	// 	return
	// }

	// userType := c.GetString(middleware.UTypeKey)
	// if userType != UserTypeOper && userType != UserTypeConfig {
	// 	log.Error("DeviceRemote reject. userType is :", userType, ",gatewaySn:", gatewaySn)
	// 	c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: userType + " not allowed"})
	// 	return
	// }

	// uName := c.GetString(middleware.UNameKey)
	// if data, ok := OnlineUsers.Load(uName); ok {
	// 	online, match := data.(*OnlineUser)
	// 	if !match || remoteReq.Checkpwd != online.Pwd {
	// 		log.Error("DeviceRemote reject. user pwd invalid, remoteReq.Checkpwd:", remoteReq.Checkpwd)
	// 		c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "pwd invalid"})
	// 		return
	// 	}
	// }

	// log.Debug("DeviceRemote pwd check OK. method:", method, ",remoteReq:", remoteReq, ",gateway_sn:", gatewaySn, ",device_sn:", deviceSn)

	// // get state value
	// contentMap, ok := remoteReq.Content.(map[string]interface{})
	// if !ok {
	// 	log.Error("DeviceRemote remoteReq contect is not map:", remoteReq.Content)
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "contect is not map"})
	// 	return
	// }
	// stateValue, exists := contentMap["state"]
	// if !exists {
	// 	log.Error("DeviceRemote remoteReq contect map state not exist:", remoteReq.Content)
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest,
	// 		Message: "contect key 'state' not exist"})
	// 	return
	// }
	// state, ok := stateValue.(float64)
	// if !ok {
	// 	log.Error("DeviceRemote remoteReq contect stateValue not num:", remoteReq.Content)
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest,
	// 		Message: "contect key 'state' not num value"})
	// 	return
	// }

	// code, pType, tableName, err := emsdb.GetCodeName(deviceSn, method)
	// if err != nil {
	// 	log.Error("DeviceRemote GetCodeName failed:", err.Error())
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest,
	// 		Message: "remoteReq GetCodeName failed"})
	// }
	// log.Infof("DeviceRemote Send code:%s, pType:%s, tableName:%s, value:%f", code, pType, tableName, state)
	// respChan := make(chan int, 1)
	// if pType == "Yx" {
	// 	scdnet.SendYkCmdReq(code, tableName, int(state), respChan)
	// } else {
	// 	scdnet.SendYtCmdReq(code, float32(state), respChan)
	// }

	// select {
	// case respState := <-respChan:
	// 	log.Info("DeviceRemote get resp value :", respState)
	// 	if respState == 6 {
	// 		c.JSON(http.StatusOK, emsclient.PushResp{Topic: method, Content: strconv.Itoa(respState)})
	// 		return
	// 	}
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: respState, Message: ""})
	// 	return
	// case <-time.After(time.Second * 15):
	// 	log.Error("DeviceRemote 15 second timeout")
	// 	c.JSON(http.StatusRequestTimeout, models.CommonResp{Code: http.StatusRequestTimeout, Message: ""})
	// 	return
	// }
}

// @Summary		查询点位历史数据
// @Description	查询点位历史数据
// @ID				mqttid-his
// @Tags			gateways
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"				default(<token>)
// @Param			gateway_sn		path		string				true	"网关sn, ems设备写ems即可"		Example("esscab1")
// @Param			device_sn		path		string				true	"子设备sn, ems设备写ems即可"	example("esscab1ac1")
// @Param			mqttid			path		string				true	"mqttid点名"
// @Param			content			body		interface{}			true	"请求消息体 内容是{date: 2024-01-26}格式的json结构，指定历史记录的日期"
// @Success		200				{object}	interface{}			"成功 ，例：{\"date\":\"2024-01-26\",\"list_data\":[{\"data\":100,\"time\":\"00:00\"},{\"data\":100,\"time\":\"00:05\"},{\"data\":100,\"time\":\"00:10\"}...]"
// @Failure		400				{object}	models.CommonResp	"参数错误 {\"code\": 400, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/gateways/{gateway_sn}/devices/{device_sn}/history/{mqttid} [post]
func MqttIDHis(c *gin.Context) {
	log.Info("MqttIDHis in ...")
	// gatewaySn := c.Param("gateway_sn")
	// deviceSn := c.Param("device_sn")
	// mqttid := c.Param("mqttid")

	// body, err := io.ReadAll(c.Request.Body)
	// if err != nil {
	// 	log.Error("MqttIDHis read body err:", err.Error())
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "body err"})
	// 	return
	// }

	// log.Debug("MqttIDHis mqttid:", mqttid, ",body:", string(body), ",gatewaySn:", gatewaySn, ",deviceSn:", deviceSn)

	// seq := GetSeq()
	// pointCode, pointType, _, err := emsdb.GetCodeName(deviceSn, mqttid)
	// if err != nil {
	// 	log.Info("MqttIDHis GetCodeName failed:", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "GetCodeName err"})
	// 	return
	// }
	// pubRmsg := &HisReqMsg{
	// 	Rmsg: gormsg.Rmsg{
	// 		Method:  "AcquisitionHisData" + pointType,
	// 		Payload: string(body),
	// 		Seq:     seq,
	// 	},
	// 	CodeName: pointCode,
	// }
	// pubdata, err := json.Marshal(pubRmsg)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Service Error"})
	// 	log.Info("EmsRemote json.Marshal failed:", err.Error())
	// 	return
	// }

	// redis, redisRrr := gormsg.NewScdRedis()
	// if redisRrr != nil {
	// 	log.Info("MqttIDHis NewScdRedis failed:", redisRrr.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "NewScdRedis err"})
	// 	return
	// }

	// agvcRetCh := make(chan *gormsg.Rmsg, 0)
	// subID, err := redis.Subscribe(func(topic string, data []byte) {
	// 	log.Info("get topic:", topic, ", data:", string(data))
	// 	if topic == LW_HIS_TOPIC_RET {
	// 		agvcRet := &gormsg.Rmsg{}
	// 		json.Unmarshal(data, &agvcRet)
	// 		if agvcRet.Seq != seq {
	// 			log.Info("seq not equal. requset seq:", seq, ", recv seq:", agvcRet.Seq)
	// 			return
	// 		}
	// 		agvcRetCh <- agvcRet
	// 	}
	// }, LW_HIS_TOPIC_RET)

	// if err != nil {
	// 	log.Error("redis Subscribe failed:", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "Subscribe err"})
	// 	return
	// }

	// log.Info("MqttIDHis publish data:", string(pubdata))
	// err = redis.Publish(LW_HIS_TOPIC, pubdata)
	// if err != nil {
	// 	redis.Unsubscribe(subID)
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Service Error"})
	// 	log.Info("EmsRemote redis.Publish failed:", err.Error())
	// 	return
	// }

	// select {
	// case resp := <-agvcRetCh:
	// 	redis.Unsubscribe(subID)
	// 	var result interface{}
	// 	err := json.Unmarshal([]byte(resp.Payload), &result)
	// 	if err != nil {
	// 		log.Error("MqttIDHis Unmarshal resp content failed:", err, ",seq:", seq, ",resp:", resp)
	// 		break
	// 	}
	// 	log.Info("MqttIDHis result:", resp, ",seq:", seq)
	// 	c.JSON(http.StatusOK, result)
	// 	return
	// case <-time.After(time.Second * 10):
	// 	redis.Unsubscribe(subID)
	// 	log.Error("MqttIDHis 10 second timeout")
	// 	break
	// }
	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
		Message: "Internal Service Error"})
}
