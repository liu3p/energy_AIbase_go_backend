package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"webserver-arm/emsdb"
	"webserver-arm/homepage"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
)

const (
	EmsGateway = "ems"
	EmsDevice  = "ems"

	LOCALWEB2AGVC = "localweb2agvc"
	AGVC2LOCALWEB = "agvc2localweb"
)

const (
	// pointIDMinParts 测点ID格式 xxx-xxxxx-xxx-xxx，至少 4 段（与 /his/query 的 validatePointIDs 口径一致）
	pointIDMinParts = 4
	// maxExportSpan 单次导出的最大时间跨度，防止无聚合查询把内存打满
	maxExportSpan = 48 * time.Hour
)

//nolint:govet
type QueryEmsResp struct {
	Topic   string              `json:"topic"`
	Content map[string]*EmsInfo `json:"content"`
}

type EmsInfo struct {
	Sn     string `json:"sn"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

// overViewData

type DevInfo struct {
	YcData []string `json:"yc"`
	YxData []string `json:"yx"`
	PpData []string `json:"prop"`
}

type modelInfo map[string]DevInfo             // "dev_code": xxx
var AllModelInfo = make(map[string]modelInfo) // "model" : xxx

func cutPrefixDevCode(devCode string) (string, error) {
	isfound := false
	secondDot := strings.IndexFunc(devCode, func(r rune) bool {
		if r == '.' {
			if isfound {
				return true
			}
			isfound = true
		}
		return false
	})
	if secondDot == -1 {
		return "", fmt.Errorf("invalid devCode format:%s", devCode)
	}
	return devCode[:secondDot], nil
}

// type teleData map[string]map[string]float32      // yc: key: value
type teleData map[string]map[string]interface{}  // yc: key: value
type overViewData map[string]map[string]teleData // devcode: devsn: teleData

// @Summary		查询概览信息
// @Description	查询概览信息
// @ID				query-overview
// @Tags			ems
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			model			path		string				true    "概览的产品型号"
// @Success		200				{object}	overViewData		"查询成功 {\"ess.bms\":{\"esscab1bms1\":{\"yc\":{\"Idc\":32.400024,\"Udc\":750.10004}}},\"ess.pcs\":{\"esscab1pcs1\":{\"yc\":{\"PChgMax\":100,\"RunState\":1}}}}"
// @Success		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没有找到model {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/overview/{model} [get]
func OverView(c *gin.Context) {
	result := make(overViewData)
	model := c.Param("model")

	log.Info("OverView in ... model:", model)

	// get model from config
	info, ok := AllModelInfo[model]
	if !ok {
		info, ok = AllModelInfo["DEFAULT"]
		if !ok {
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "no model found"})
			return
		}
	}

	allDeviceInfo, _, err := getAllDevice()
	if err != nil {
		log.Error("OverView getAllDevice failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Server Error"})
		return
	}

	allDevByDevCode := make(map[string][]models.DeviceInfo)

	for _, devinfo := range allDeviceInfo {
		prefixDevCode, err := cutPrefixDevCode(devinfo.DevCode)
		if err != nil {
			log.Warn("OverView cutPrefixDevCode failed:", err.Error())
			continue
		}
		allDevByDevCode[prefixDevCode] = append(allDevByDevCode[prefixDevCode], devinfo)
	}

	log.Debug("OverView allDevByDevCode:", allDevByDevCode)

	for devCode, devConfig := range info { // 遍历每一种设备类型
		// dev_code like "ess.ac"

		// devConfig like "yc["",""]  yx["",""]"
		result[devCode] = make(map[string]teleData)

		ycMqttSet := make(map[string]any)
		for _, mqttID := range devConfig.YcData {
			ycMqttSet[mqttID] = struct{}{}
		}

		yxMqttSet := make(map[string]any)
		for _, mqttID := range devConfig.YxData {
			yxMqttSet[mqttID] = struct{}{}
		}

		ppMqttSet := make(map[string]any)
		for _, mqttID := range devConfig.PpData {
			ppMqttSet[mqttID] = struct{}{}
		}

		devList, ok := allDevByDevCode[devCode] // 取出devCode类型下的全部设备
		if !ok {
			log.Warn("OverView devCode not exist in allDevByDevCode:", devCode)
			continue
		}

		log.Info("devCode:", devCode, ",devList:", devList, ",devConfig:", devConfig)
		for _, dev := range devList { // 针对每一个设备，取出yc/yx

			result[devCode][dev.Sn] = make(teleData)

			// yc
			if len(ycMqttSet) > 0 {

				teleData, err := emsdb.GetDevicesYc([]string{dev.Sn}, false)
				if err != nil {
					log.Error("OverView GetDevicesYc err:", err)
					c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
						Message: "Internal Service Error"})
					return
				}

				for dev_sn, ycList := range teleData.Content {
					// dataSet := make(map[string]float32)
					dataSet := make(map[string]interface{})
					for _, yc := range ycList {
						if _, ok := ycMqttSet[yc.K]; ok {
							dataSet[yc.K] = yc.V
						}
					}
					result[devCode][dev_sn]["yc"] = dataSet
				}
			}

			// yx
			if len(yxMqttSet) > 0 {
				teleData, err := emsdb.GetDevicesYx([]string{dev.Sn}, false)
				if err != nil {
					log.Error("OverView GetDevicesYx err:", err)
					c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
						Message: "Internal Service Error"})
					return
				}

				for dev_sn, yxList := range teleData.Content {
					dataSet := make(map[string]interface{})
					for _, yx := range yxList {
						if _, ok := yxMqttSet[yx.K]; ok {
							dataSet[yx.K] = yx.V
						}
					}
					result[devCode][dev_sn]["yx"] = dataSet
				}
			}

			// // prop
			// if len(ppMqttSet) > 0 {
			// 	propData, err := emsdatahub.GetDevicesProp([]string{dev.Sn})
			// 	if err != nil {
			// 		log.Error("OverView GetDevicesProp err:", err)
			// 		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			// 			Message: "Internal Service Error"})
			// 		return
			// 	}

			// 	for dev_sn, ppList := range propData.Content {
			// 		dataSet := make(map[string]interface{})
			// 		for _, pp := range ppList {
			// 			if _, ok := ppMqttSet[pp.K]; ok {
			// 				dataSet[pp.K] = pp.V
			// 			}
			// 		}
			// 		result[devCode][dev_sn]["prop"] = dataSet
			// 	}

			// 	// ppReq := &emsclient.QueryPropReq{dev.Sn}
			// 	// ppResp, err := client.QueryProp(ppReq)
			// 	// if err != nil {
			// 	// 	log.Errorf("OverView QueryProp for dev %s Failed", dev.Sn)
			// 	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			// 	// 		Message: "Internal Server Error"})
			// 	// 	return
			// 	// }
			// 	// log.Info("ppReq:", ppReq, ",ppResp:", ppResp)
			// 	// for dev_sn, ppList := range ppResp.Data.Content {
			// 	// 	// dataSet := make(map[string]float32)
			// 	// 	dataSet := make(map[string]interface{})
			// 	// 	for _, pp := range ppList {
			// 	// 		if _, ok := ppMqttSet[pp.K]; ok {
			// 	// 			dataSet[pp.K] = pp.V
			// 	// 		}
			// 	// 	}
			// 	// 	result[devCode][dev_sn]["prop"] = dataSet
			// 	// }
			// }

		}
	}
	// return ok
	log.Info("OverView result:", result)
	c.JSON(http.StatusOK, &result)
}

// @Summary		查询ems设备
// @Description	查询ems设备信息
// @ID				query-ems
// @Tags			ems
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	QueryEmsResp		"查询成功 {\"topic\":\"ems\",\"content\":{\"ems\":{\"sn\":\"ems\",\"name\":\"ems\", \"status\":2}}}"
// @Success		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没有找到ems {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/ems [get]
func QueryEms(c *gin.Context) {
	log.Info("QueryEms in ...")
	// gwInfos := ClientMgr.GetGatewayInfo(EmsGateway)
	result := QueryEmsResp{Topic: "ems", Content: make(map[string]*EmsInfo)}
	// if gwInfo, ok := gwInfos[EmsGateway]; ok {
	// 	result.Content[EmsGateway] = &EmsInfo{
	// 		Sn:     gwInfo.Sn,
	// 		Name:   gwInfo.Name,
	// 		Status: gwInfo.Status,
	// 	}
	// } else {
	// 	c.JSON(http.StatusNotFound, models.CommonResp{Code: http.StatusNotFound, Message: "Not Found"})
	// 	return
	// }

	c.JSON(http.StatusOK, result)
}

// @Summary		查询ems遥测
// @Description	查询ems的遥测数据
// @ID				query-ems-yc
// @Tags			ems
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"telemetry\",\"content\":{\"11223355ems\":[{\"at\":1699263620341,\"k\":\"Uab\",\"mt\":1699263620341,\"v\":400}]}}"
// @Success		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/ems/telemetry [get]
func QueryEmsTelemetry(c *gin.Context) {
	log.Info("QueryEmsTelemetry in ...")
	/*
		var reqInfo emsclient.QueryYCReq
		reqInfo = append(reqInfo, EmsDevice)
		client := ClientMgr.GetClient(EmsGateway)
		if client == nil {
			log.Error("GetClient for ems Failed")
			c.JSON(http.StatusServiceUnavailable, models.CommonResp{Code: http.StatusServiceUnavailable,
				Message: "Service Unavailable"})
			return
		}

		resp, err := client.QueryYC(&reqInfo)
		if err != nil {
			log.Debug("QueryEmsTelemetry QueryYC err", err.Error())
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, resp.Data)*/
	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}

// @Summary		查询ems遥信
// @Description	查询ems的遥信数据
// @ID				query-ems-yx
// @Tags			ems
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"telesignal\",\"content\":{\"ems\":[{\"at\":1699263620341,\"k\":\"Uab\",\"mt\":1699263620341,\"v\":400}]}}"
// @Success		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/ems/telesignal [get]
func QueryEmsTelesignal(c *gin.Context) {
	log.Info("QueryEmsTelesignal in ...")
	/*
		var reqInfo emsclient.QueryYXReq
		reqInfo = append(reqInfo, EmsDevice)
		client := ClientMgr.GetClient(EmsGateway)
		if client == nil {
			log.Error("GetClient for ems Failed")
			c.JSON(http.StatusServiceUnavailable, models.CommonResp{Code: http.StatusServiceUnavailable,
				Message: "Service Unavailable"})
			return
		}

		resp, err := client.QueryYX(&reqInfo)
		if err != nil {
			log.Debug("QueryEmsTelemetry QueryYX err", err.Error())
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, resp.Data)*/
	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}

// @Summary		查询子设备属性
// @Description	查询指定网关下某子设备属性
// @ID				query-ems-prop
// @Tags			ems
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	models.TeleData		"查询成功 {\"topic\":\"property\",\"content\":{\"11223355ems\":[{\"at\":1699263620341,\"k\":\"Runstate\",\"mt\":1699263620341,\"v\":1}]}}"
// @Success		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/ems/prop [get]
func QueryEmsProp(c *gin.Context) {
	log.Info("QueryEmsProp in ...")
	/*
		var reqInfo emsclient.QueryPropReq
		reqInfo = append(reqInfo, EmsDevice)
		client := ClientMgr.GetClient(EmsGateway)
		if client == nil {
			log.Error("GetClient for ems Failed")
			c.JSON(http.StatusServiceUnavailable, models.CommonResp{Code: http.StatusServiceUnavailable,
				Message: "Service Unavailable"})
			return
		}

		resp, err := client.QueryProp(&reqInfo)
		if err != nil {
			log.Debug("QueryEmsProp QueryProp err", err.Error())
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, resp.Data)*/
	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}

//nolint:govet
type EmsRemoteResp struct {
	Topic   string      `json:"topic"`
	Content interface{} `json:"content"`
}

// @Summary		ems遥控/调
// @Description	ems遥控或遥调下发
// @ID				query-ems-remote
// @Tags			ems
// @Produce		json
// @Param			Authorization	header		string				true																																											"JWT Token"																																																											default(<token>)
// @Param			method			path		string				true																																											"方法名"																																																												example( 1>ems策略下发：PolicyConfig,																																																																																	2>手动/智能策略下发:	PlanSet,	3>AGC对象投入/退出:	AGCExitCtrl,	4>AGC对象远方/就地:AGCRemoteCtrl,	5>储能一体机投入/退出:CabExitCtrl,	6>AGC策略修改:AGCStrategyMod,	7>并网点增删改:ParallelBranch)
// @Param			RemoteReq-情况1	body		string				true																																											"ems策略下发"																																																											default({"content": "body", "sn": "项目sn号"})	Example({"content":[{\"type\":1,\"bus\":\"bus1\",\"date\":\"2006-01-02\",\"config_data\":{\"plans\":[{\"p\":100.45,\"bt\":\"00:00\",\"et\":\"01:21\"},{\"p\":0,\"bt\":\"23:21\",\"et\":\"24:00\"},{\"p\":-100.45,\"bt\":\"01:21\",\"et\":\"23:21\"}],\"soc_up\":100,\"soc_dn\":48.5,\"interval\":15}},{\"type\":1,\"bus\":\"bus2\",\"date\":\"2006-01-02\",\"config_data\":{\"plans\":[{\"p\":110.45,\"bt\":\"00:00\",\"et\":\"01:21\"},{\"p\":10,\"bt\":\"23:21\",\"et\":\"24:00\"},{\"p\":-100.45,\"bt\":\"01:21\",\"et\":\"23:21\"}],\"soc_up\":100,\"soc_dn\":28,\"interval\":15}}], "sn": "项目sn号"})
// @Param			RemoteReq-情况2	body		string				true																																											"手动/智能策略下发 (mode：类型0-手动策略，2-智能策略)"																																																					default({"content": "", "sn": "项目sn号"})		Example({"content"：[{\"mode\":0,\"bus\":\"bus1\",\"date\":\"2006-01-02\",\"p\":100.0，\"max_p\":322.1\"soc_up\":85,\"soc_dn\":15，\"chg_max\":100,\"dchg_max\":200，\"chg_eff\":80,\"dchg_eff\":90}], "sn": "项目sn号"})
// @Param			RemoteReq-情况3	body		string				true																																											"AGC对象投入/退出  (0-退出；1-投入)"																																																							default({"content": "", "sn": "项目sn号"})		Example({"content": {\"state\": 0}, "sn": "项目sn号"})
// @Param			RemoteReq-情况4	body		string				true																																											"AGC对象远方/就地  (0-就地；1-远方)"																																																							default({"content": "", "sn": "项目sn号"})		Example({"content": {\"state\": 0}, "sn": "项目sn号"})
// @Param			RemoteReq-情况5	body		string				true																																											"储能一体机投入/退出  (0-退出；1-投入)"																																																							default({"content": "", "sn": "项目sn号"})		Example({"content": {\"state\": 0}, "sn": "项目sn号"})
// @Param			RemoteReq-情况6	body		string				true																																											"AGC策略修改  (0-功率设定；1-计划曲线；2-智能模式)"																																																					default({"content": "", "sn": "项目sn号"})		Example({"content": {\"state\": 0}, "sn": "项目sn号"})
// @Param			RemoteReq-情况7	body		string				true																																											"并网点增删改"																																																											default({"content": "", "sn": "项目sn号"})		Example({"content"：[{\"bus\":\"bus1\",\"demand\":1000.2,\"trans_cap\":800.3,\"config_data\":[{\"sn\":\"esscab1\"},{\"sn\":\"esscab2\"}]},{\"bus\":\"bus2\",\"demand\":1000.2,\"trans_cap\":800.3,\"config_data\":[{\"sn\":\"esscab3\"},{\"sn\":\"esscab4\"}]}]", "sn": "项目sn号"})
// @Success		1				{object}	models.CommonResp	"CODE 200.  ems策略下发 下发成功 {\"topic\": \"LocalPolicyUpAuto\", \"content\": [{\"type\":1,\"bus\":\"bus1\",\"status\":0,\"exec_t\":\"2023-09-15										14:17:00\",\"from\":2,\"plans\":[{\"p\":100.45,\"bt\":\"00:00\",\"et\":\"01:21\"},{\"p\":0,\"bt\":\"23:21\",\"et\":\"24:00\"},{\"p\":-100.45,\"bt\":\"01:21\",\"et\":\"23:21\"}]},{\"type\":1,\"bus\":\"bus2\",\"status\":0,\"exec_t\":\"2023-09-15	14:17:00\",\"from\":2,\"plans\":[{\"p\":100.45,\"bt\":\"00:00\",\"et\":\"01:21\"},{\"p\":0,\"bt\":\"23:21\",\"et\":\"24:00\"},{\"p\":-100.45,\"bt\":\"01:21\",\"et\":\"23:21\"}]}]}"
// @Success		2				{object}	models.CommonResp	"CODE 200.  手动/智能策略下发  手动策略 下发成功 {\"topic\": \"LocalPolicyUpHand\", \"content\": [{\"type\":1,\"bus\":\"bus1\",\"status\":0,\"exec_t\":\"2023-09-15								14:17:00\",\"soc_up\":90,\"soc_dn\":15,\"p\":100.1,\"from\":0}]}"
// @Success		3				{object}	models.CommonResp	"CODE 200.  手动/智能策略下发  智能策略 下发成功 {\"topic\": \"LocalPolicyUpSmart\", \"content\": [{\"max_p\":233.2,\"bus\":\"bus1\",\"type\":1,\"status\":0,\"from\":2,\"exec_t\":\"2023-09-15	14:17:00\",\"chg_max\":40.2,\"dchg_max\":40.2,\"chg_eff\":89.9,\"dchg_eff\":89.9,\"ele_state\":0,\"chg_cpt\":100,\"dchg_cpt\":20,\"soc\":25,\"soc_up\":85,\"soc_dn\":15}]"
// @Success		4				{object}	models.CommonResp	"CODE 200.  AGC对象投入/退出 下发成功 {\"topic\": \"AGCExitCtrlReport\", \"content\": {\"ack:\": true}}"
// @Success		5				{object}	models.CommonResp	"CODE 200.  AGC对象远方/就地 下发成功 {\"topic\": \"AGCRemoteCtrlReport\", \"content\": {\"ack:\": true}}"
// @Success		6				{object}	models.CommonResp	"CODE 200.  储能一体机投入/退出 下发成功  {\"topic\": \"CabExitCtrlReport\", \"content\": {\"ack:\": true}}"
// @Success		7				{object}	models.CommonResp	"CODE 200.  AGC策略修改 下发成功 {\"topic\": \"AGCStrategyModReport\", \"content\": {\"ack:\": true}}"
// @Success		8				{object}	models.CommonResp	"CODE 200.  并网点增删改 下发成功 {\"topic\": \"ParallelBranchReport\", \"content\": [{\"k	\":\"20231128e	ms	\",\"v\":[{\"from	\":1,\"bus\":\"bus1\",\"demand\":1000.2,\"trans_cap\":800.3,\"config_data\":[{\"sn\":\"esscab1\"},{\"sn\":\"esscab2\"}]},{\"from	\":1,\"bus\":\"bus2\",\"demand\":1000.2,\"trans_cap\":800.3,\"config_data\":[{\"sn\":\"esscab3\"},{\"sn\":\"esscab4\"}]}]}]}"
// @Failure		9				{object}	models.CommonResp	"CODE 401.  认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		10				{object}	models.CommonResp	"CODE 403.  拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		11				{object}	models.CommonResp	"CODE 503.  服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/ems/remote/{method} [post]
func EmsRemote(c *gin.Context) {
	log.Info("EmsRemote in ...")
	/*
		method := c.Param("method")
		remoteReq := &RemoteReq{}
		err := c.ShouldBindJSON(remoteReq)
		if err != nil {
			log.Error("ShouldBindJSON err:", err.Error())
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			return
		}

		// check sn
		projInfo, err := emsdb.GetProjectSn()
		if err != nil {
			log.Error("EmsRemote getProjInfo err: ", err.Error())
			c.JSON(http.StatusServiceUnavailable, models.CommonResp{Code: http.StatusServiceUnavailable,
				Message: "Service Unavailable"})
			return
		}

		if remoteReq.ProjSn != projInfo.Sn {
			log.Error("EmsRemote Forbidden: remoteReq sn:", remoteReq.ProjSn, " do not match server sn:", projInfo.Sn)
			c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "Forbidden"})
			return
		}

		log.Debug("EmsRemote sn check OK. method:", method, ",rdReq:", remoteReq)

		// do send
		seq := GetSeq()
		content, err := json.Marshal(remoteReq.Content)
		if err != nil {
			log.Error("EmsRemote Marshal remoteReq content failed:", err.Error())
			c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "content invalid"})
			return
		}

		redis, redisRrr := gormsg.NewScdRedis()
		log.Info("tranChangeData.....")
		if redisRrr != nil {
			log.Info("NewScdRedis failed:", redisRrr.Error())
			return
		}

		// make subscribe
		agvcRetCh := make(chan *gormsg.Rmsg, 0)
		subID, err := redis.Subscribe(func(topic string, data []byte) {
			log.Info("get topic:", topic, ", data:", string(data))
			if topic == AGVC2LOCALWEB {
				agvcRet := &gormsg.Rmsg{}
				if agvcRet.Seq != seq {
					log.Info("seq not equal. requset seq:", seq, ", recv seq:", agvcRet.Seq)
					return
				}
				json.Unmarshal(data, &agvcRet)
				agvcRetCh <- agvcRet
			}
		}, AGVC2LOCALWEB)

		// make publish
		pubRmsg := &gormsg.Rmsg{
			Method:  method,
			Payload: string(content),
			Seq:     seq,
		}
		pubdata, err := json.Marshal(pubRmsg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			log.Info("EmsRemote json.Marshal failed:", err.Error())
			return
		}
		err = redis.Publish(LOCALWEB2AGVC, pubdata)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			log.Info("EmsRemote redis.Publish failed:", err.Error())
			return
		}

		log.Info("EmsRemote Publish over. pubRmsg seq:", seq, ", method:", method, ", content:", string(content))

		select {
		case resp := <-agvcRetCh:
			err = redis.Unsubscribe(subID)
			if err != nil {
				log.Error("EmsRemote redis.Unsubscribe failed:", err.Error())
			}
			result := &EmsRemoteResp{Topic: resp.Method}
			err := json.Unmarshal([]byte(resp.Payload), &result.Content)
			if err != nil {
				log.Error("EmsRemote Unmarshal resp content failed:", err, ",seq:", seq, ",resp:", resp)
				break
			}
			log.Info("EmsRemote result:", result, ",seq:", seq)
			c.JSON(http.StatusOK, result)
			return
		case <-time.After(time.Second * 15):
			log.Error("EmsRemote 15 second timeout")
			break
		}
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})*/
	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}

func ExportInfluxDBData(c *gin.Context) {
	log.Info("ExportInfluxDBData in")

	req := emsdb.ExportHistoryReq{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Errorf("UpdateDevicePointsByID: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if err = emsdb.ExportHistoryDataToCSV(req.IDs, req.StartTime, req.EndTime, "analog_data.csv"); err != nil {
		log.Errorf("UpdateDevicePointsByID: ExportHistoryDataToCSV failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}

// QueryHistoryResult 查询历史数据结果（包装结构）
type QueryHistoryResult struct {
	Code int                    `json:"code"`
	Data emsdb.QueryHistoryResp `json:"data"`
	Msg  string                 `json:"msg"`
}

// @Summary     查询历史数据
// @Description 根据测点类型和测点ID查询InfluxDB历史数据，支持时间插值
// @ID          influxdb-query-history
// @Tags        influxdb
// @Param       Authorization header string true "JWT Token"
// @Param       body body emsdb.QueryHistoryReq true "查询参数"
// @Success     200 {object} handlers.QueryHistoryResult "查询成功，返回时间点数组"
// @Failure     400 {object} models.CommonResp "参数错误"
// @Failure     500 {object} models.CommonResp "服务内部错误"
// @Router      /api/v1/log/influxdb/his/query [post]
func QueryInfluxDBData(c *gin.Context) {
	log.Info("QueryHistoryData in...")

	req := emsdb.QueryHistoryReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("QueryHistoryData ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	historyData, err := emsdb.QueryHistoryData(&req)
	if err != nil {
		log.Error("QueryHistoryData failed:", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, QueryHistoryResult{
		Code: 200,
		Data: historyData,
		Msg:  "查询成功",
	})
}

// makeDownloadFileName 生成安全的下载文件名
// pointID 来自请求参数，需过滤掉可能污染 Content-Disposition 的字符（引号、分号、路径分隔符、换行等）
func makeDownloadFileName(pointID string, start, stop time.Time) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_', r == '.':
			return r
		default:
			return '_'
		}
	}, pointID)
	return fmt.Sprintf("history_%s_%s_%s.csv",
		safe,
		start.UTC().Format("20060102T150405Z"),
		stop.UTC().Format("20060102T150405Z"))
}

// @Summary     导出单测点历史原始数据
// @Description 导出指定测点在指定时间范围内的 InfluxDB 原始历史数据（不做插值与取整），以 CSV 文件返回；时间列时区跟随 start_time 的偏移
// @ID          influxdb-export-raw-history
// @Tags        influxdb
// @Param       Authorization header string true "JWT Token"
// @Param       start_time query string true "开始时刻，RFC3339"
// @Param       end_time   query string true "结束时刻，RFC3339"
// @Param       id         query string true "测点ID，格式 xxx-xxxxx-xxx-xxx"
// @Success     200 {file} file "CSV 文件"
// @Failure     401 {object} models.CommonResp "认证失败"
// @Failure     400 {object} models.CommonResp "参数错误"
// @Failure     500 {object} models.CommonResp "服务内部错误"
// @Router      /api/v1/log/influxdb/his/export [get]
func ExportRawHistory(c *gin.Context) {
	log.Info("ExportRawHistory in...")

	startStr := c.Query("start_time")
	endStr := c.Query("end_time")
	pointID := c.Query("id")
	if startStr == "" || endStr == "" || pointID == "" {
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: "参数 start_time、end_time 和 id 必填",
		})
		return
	}

	// 1. 解析时间：与 /his/query 一致，RFC3339 原样接入，不做任何时区解释
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "参数 start_time 格式错误，应为 RFC3339"})
		return
	}
	stop, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "参数 end_time 格式错误，应为 RFC3339"})
		return
	}
	if !start.Before(stop) {
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "参数 start_time 必须早于 end_time"})
		return
	}
	// 本接口无聚合，返回量正比于原始记录数且全部堆在内存，需要限制跨度
	if stop.Sub(start) > maxExportSpan {
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "导出时间跨度不能超过 48 小时"})
		return
	}

	// 2. 校验 id 基本格式：与 /his/query 一致要求至少 4 段
	//    （oidTypeFromThirdSegment 只判 >=3 段，单独用它会让 "1-1-102" 这种漏过去）
	if len(strings.Split(pointID, "-")) < pointIDMinParts {
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: "参数 id 格式错误，应为 xxx-xxxxx-xxx-xxx",
		})
		return
	}

	// 3. 由 OID 第三段识别测点类型（即 InfluxDB measurement）
	measurement, ok := oidTypeFromThirdSegment(pointID)
	if !ok {
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: "参数 id 第三段必须为 101(digital) / 102(analog) / 105(pulse)",
		})
		return
	}

	// 4. 查询原始数据
	items, err := emsdb.QueryRawHistoryData(measurement, pointID, start, stop)
	if err != nil {
		log.Errorf("ExportRawHistory QueryRawHistoryData failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 5. 在内存中组装 CSV（先组装再发响应，避免半截响应）
	// 时间列按请求方时区输出：与 /his/query 的规则一致——取 start_time 的固定偏移重新表示时间
	// （start_time 给 +08:00 则输出 2026-09-17T00:00:01+08:00，而不是 UTC 的 ...Z）。
	// 用 FixedZone(偏移) 而不是 start 解析出的 Location：time.Parse 在偏移恰等于服务器本地时区
	// 时会返回 Local，跨夏令时会让输出偏移与请求漂移；固定成请求当时的偏移可保证逐字一致。
	// 偏移为 0（请求给 Z）时传 nil，时间列保持 UTC 的 Z 形式，与旧行为一致。
	// 查询本身不受影响：Flux 用的是时间瞬时，start/stop 带不带偏移拼出来等价。
	var loc *time.Location
	if _, offset := start.Zone(); offset != 0 {
		loc = time.FixedZone("", offset)
	}

	var buf bytes.Buffer
	if err := emsdb.WriteRawHistoryCSV(&buf, items, loc); err != nil {
		log.Errorf("ExportRawHistory WriteRawHistoryCSV failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 6. 直接返回文件流
	fileName := makeDownloadFileName(pointID, start, stop)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+fileName+`"`)
	c.Header("Content-Length", strconv.Itoa(buf.Len()))
	if _, err := c.Writer.Write(buf.Bytes()); err != nil {
		log.Error("ExportRawHistory 写入响应失败:", err)
	}
}

// HomeHistoryReq 首页历史数据查询请求
type HomeHistoryReq struct {
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Interval   int       `json:"interval"`    // 时间间隔（秒），最小300秒；传 -1(emsdb.IntervalAll) 表示不插值，返回区间内全部原始数据
	IncludeEnd *bool     `json:"include_end"` // 同 emsdb.QueryHistoryReq.IncludeEnd；仅插值模式生效，默认 true
}

// HomeHistoryResp 首页历史数据查询响应
type HomeHistoryResp struct {
	Topology   emsdb.QueryHistoryResp `json:"topology"`
	PowerLevel emsdb.QueryHistoryResp `json:"power_level"`
	RealPower  emsdb.QueryHistoryResp `json:"real_power"`
}

// HomeHistoryResult 接口返回包装
type HomeHistoryResult struct {
	Code int             `json:"code"`
	Data HomeHistoryResp `json:"data"`
	Msg  string          `json:"msg"`
}

// oidTypeFromThirdSegment 从 OID 第三段识别 InfluxDB 类型
// OID 格式: xxx-xxxxx-xxx-xxx，第三段 101=digital, 102=analog, 105=pulse
func oidTypeFromThirdSegment(oid string) (string, bool) {
	parts := strings.Split(oid, "-")
	if len(parts) < 3 {
		log.Warnf("oidTypeFromThirdSegment: OID %s has less than 3 parts", oid)
		return "", false
	}
	log.Infof("oidTypeFromThirdSegment: OID=%s, parts=%v, third=%s", oid, parts, parts[2])
	switch parts[2] {
	case "101":
		return "digital", true
	case "102":
		return "analog", true
	case "105":
		return "pulse", true
	default:
		log.Warnf("oidTypeFromThirdSegment: OID %s third part %s is not 101/102/105", oid, parts[2])
		return "", false
	}
}

// groupOidsByType 将 ConfigItem 列表按 OID 类型分组
// 返回 map[类型字符串][]OID
func groupOidsByType(items []homepage.ConfigItem) map[string][]string {
	result := make(map[string][]string)
	for _, item := range items {
		if item.OID == "" {
			continue
		}
		typ, ok := oidTypeFromThirdSegment(item.OID)
		if !ok {
			continue
		}
		result[typ] = append(result[typ], item.OID)
	}
	return result
}

// dedupOIDs 对 OID 列表去重
func dedupOIDs(oids []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, oid := range oids {
		if _, ok := seen[oid]; ok {
			continue
		}
		seen[oid] = struct{}{}
		result = append(result, oid)
	}
	return result
}

// mergeResults 将按类型查询的结果合并为按 OID 过滤的时间序列
// groups: map[类型][]OID — 该维度下需要的 OID
// results: map[类型]QueryHistoryResp — 按类型查询的结果
func mergeResults(groups map[string][]string, results map[string]emsdb.QueryHistoryResp) emsdb.QueryHistoryResp {
	// 收集该维度下所有 OID
	oidSet := make(map[string]struct{})
	for _, ids := range groups {
		for _, id := range ids {
			oidSet[id] = struct{}{}
		}
	}
	if len(oidSet) == 0 {
		return emsdb.QueryHistoryResp{}
	}

	// 按时间点合并，只保留该维度下的 OID
	timeMap := make(map[time.Time]map[string]interface{})
	for _, resp := range results {
		for _, item := range resp {
			data := make(map[string]interface{})
			for k, v := range item.Data {
				if _, ok := oidSet[k]; ok {
					data[k] = v
				}
			}
			if len(data) > 0 {
				if _, exists := timeMap[item.Time]; !exists {
					timeMap[item.Time] = make(map[string]interface{})
				}
				for k, v := range data {
					timeMap[item.Time][k] = v
				}
			}
		}
	}

	// 转为有序数组
	resp := make(emsdb.QueryHistoryResp, 0, len(timeMap))
	for t, data := range timeMap {
		resp = append(resp, emsdb.QueryHistoryDataItem{Time: t, Data: data})
	}
	// 按时间升序
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Time.Before(resp[j].Time)
	})
	return resp
}

// @Summary     查询首页历史数据
// @Description 自动从 HomepageConfig 获取 Topology、PowerLevel 和 RealPower 的 OID，按类型批量查询历史数据
// @ID          influxdb-query-home-history
// @Tags        influxdb
// @Param       Authorization header string true "JWT Token"
// @Param       body body handlers.HomeHistoryReq true "查询参数"
// @Success     200 {object} handlers.HomeHistoryResult "查询成功"
// @Failure     400 {object} models.CommonResp "参数错误"
// @Failure     500 {object} models.CommonResp "服务内部错误"
// @Router      /api/v1/log/influxdb/his/home [post]
func QueryHomeHistory(c *gin.Context) {
	req := HomeHistoryReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: "参数 start_time 和 end_time 必填",
		})
		return
	}
	if !req.StartTime.Before(req.EndTime) {
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: "start_time 必须早于 end_time",
		})
		return
	}
	// 参数校验：-1 表示"全部原始数据"，不参与最小 300 秒的归一
	if req.Interval != emsdb.IntervalAll && req.Interval < 300 {
		req.Interval = 300
	}

	// 1. 获取 HomepageConfig
	cm, err := homepage.GetConfigManager()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.CommonResp{
			Code:    http.StatusInternalServerError,
			Message: "获取首页配置失败: " + err.Error(),
		})
		return
	}
	cfg := cm.GetConfig()

	// 2. 分别收集 Topology、PowerLevel 和 RealPower 的 OID，按类型分组
	topologyGroups := groupOidsByType(cfg.Topology)
	powerLevelGroups := groupOidsByType(cfg.PowerLevel)
	realPowerGroups := groupOidsByType(cfg.RealPower)

	// 3. 合并所有需要查询的类型（去重）
	allTypes := make(map[string][]string)
	for typ, ids := range topologyGroups {
		allTypes[typ] = append(allTypes[typ], ids...)
	}
	for typ, ids := range powerLevelGroups {
		allTypes[typ] = append(allTypes[typ], ids...)
	}
	for typ, ids := range realPowerGroups {
		allTypes[typ] = append(allTypes[typ], ids...)
	}
	log.Infof("QueryHomeHistory: allTypes=%v", allTypes)

	// 4. 对每种类型调用 QueryHistoryData
	queryResults := make(map[string]emsdb.QueryHistoryResp)
	for typ, ids := range allTypes {
		ids = dedupOIDs(ids)
		log.Infof("QueryHomeHistory: query type=%s, ids=%v", typ, ids)
		queryReq := &emsdb.QueryHistoryReq{
			Type:       typ,
			IDs:        ids,
			StartTime:  req.StartTime,
			EndTime:    req.EndTime,
			Interval:   req.Interval,
			IncludeEnd: req.IncludeEnd, // 指针直接透传，nil 语义一致
		}
		resp, err := emsdb.QueryHistoryData(queryReq)
		if err != nil {
			log.Errorf("QueryHomeHistory: query type=%s failed: %v", typ, err)
			// 单类型查询失败不阻塞其他类型
			continue
		}
		queryResults[typ] = resp
	}

	// 5. 按 Topology、PowerLevel 和 RealPower 维度组装响应
	topologyResp := mergeResults(topologyGroups, queryResults)
	powerLevelResp := mergeResults(powerLevelGroups, queryResults)
	realPowerResp := mergeResults(realPowerGroups, queryResults)

	c.JSON(http.StatusOK, HomeHistoryResult{
		Code: 200,
		Data: HomeHistoryResp{
			Topology:   topologyResp,
			PowerLevel: powerLevelResp,
			RealPower:  realPowerResp,
		},
		Msg: "查询成功",
	})
}
