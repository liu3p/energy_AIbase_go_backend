package emsdb

import (
	"fmt"
	"sort"
	"strings"

	"webserver-arm/log"
	"webserver-arm/models"
)

type StationArea struct {
	Id       int    `gorm:"column:id;primary_key;type:int(6)"`
	Name     string `gorm:"column:name;type:char(64)"`
	MasterId int    `gorm:"column:masterid;type:int(6)"`
}

func (StationArea) TableName() string {
	return "stationarea"
}

type StationParam struct {
	SubID  int    `gorm:"column:subid;type:int"`
	Code   string `gorm:"column:代码;primary_key;type:char(24)"`
	Name   string `gorm:"column:描述;type:char(40)"`
	AreaId int    `gorm:"column:区域序号;type:int(6)"`
}

func (StationParam) TableName() string {
	return "站所线系参数表"
}

type ScdDeviceParam struct {
	Code        string `gorm:"column:代码;primary_key;type:char(24)"`
	Name        string `gorm:"column:描述;type:char(40)"`
	StationCode string `gorm:"column:站所线系代码;type:char(24)"`
	MqttGateway string `gorm:"column:MQTT_GATEWAY;type:char(30)"`
	MqttDevice  string `gorm:"column:MQTT_DEVICE;type:char(30)"`
	MqttDevCode string `gorm:"column:MQTT_DEV_CODE;type:char(30)"`
}

func (ScdDeviceParam) TableName() string {
	return "scd设备参数表"
}

type YcParam struct {
	Code       string `gorm:"column:代码;primary_key;type:char(24)"`
	Name       string `gorm:"column:描述;type:char(128)"`
	TerminalNo int    `gorm:"column:终端序号;type:int(6)"`
	No         int    `gorm:"column:遥测序号;type:int(6)"`
	ScdCode    string `gorm:"column:设备代码;type:char(24)"`
	MqttKeyID  string `gorm:"column:MqttKeyId;type:char(64)"`
}

func (YcParam) TableName() string {
	return "遥测参数表"
}

type YxParam struct {
	Code       string `gorm:"column:代码;primary_key;type:char(24)"`
	Name       string `gorm:"column:描述;type:char(128)"`
	Type       int    `gorm:"column:类型;type:int(6)"`
	TerminalNo int    `gorm:"column:终端序号;type:int(6)"`
	No         int    `gorm:"column:遥信序号;type:int(6)"`
	isControl  int    `gorm:"column:是否遥控;type:int(4)"`
	ScdCode    string `gorm:"column:设备代码;type:char(24)"`
	MqttKeyID  string `gorm:"column:MqttKeyId;type:char(64)"`
}

func (YxParam) TableName() string {
	return "遥信参数表"
}

type YtParam struct {
	Code       string `gorm:"column:代码;primary_key;type:char(24)"`
	ScdCode    string `gorm:"column:设备代码;type:char(24)"`
	MqttDevice string `gorm:"column:MqttDeviceId;type:char(64)"`
	MqttKeyID  string `gorm:"column:MqttKeyId;type:char(64)"`
}

func (YtParam) TableName() string {
	return "设点控制参数表"
}

type PropParam struct {
	GroupNo       int     `gorm:"column:GROUP_NO;type:int(6)"`
	TransCode     string  `gorm:"column:TRANS_CODE;type:char(24)"`
	TransType     int     `gorm:"column:TRANS_TYPE;type:int(6)"`
	GatewaySn     string  `gorm:"column:GATEWAY_SN;type:char(30)"`
	DeviceSn      string  `gorm:"column:DEVICE_SN;type:char(30)"`
	MqttID        string  `gorm:"column:MQTT_ID;type:char(30)"`
	PropTable     string  `gorm:"column:PROP_TABLE;type:char(30)"`
	PropField     string  `gorm:"column:PROP_FIELD;type:char(30)"`
	PropCondition string  `gorm:"column:PROP_CONDITION;type:char(255)"`
	PropFunc      string  `gorm:"column:PROP_FUNC_NAME;type:char(30)"`
	Coef          float32 `gorm:"column:COEF;type:float"`
	Threshold     float32 `gorm:"column:THRESHOLD;type:float"`
}

func (PropParam) TableName() string {
	return "MQTT_PROPERTY"
}

func GetAllProperties() ([]PropParam, error) {
	// sqldb, err := db.NewMsDB()
	// if err != nil {
	// 	log.Info("GetAllProperties open db err:", err)
	// 	return nil, fmt.Errorf("GetAllProperties open db err:%v", err)
	// }

	var allProp []PropParam
	// tx := sqldb.Find(&allProp)
	// if tx.Error != nil {
	// 	log.Error("GetAllProperties Find err:", tx.Error)
	// 	return nil, fmt.Errorf("GetAllProperties Find err:%v", tx.Error)
	// }
	// log.Info("GetAllProperties:", len(allProp))

	return allProp, nil
}

func GetGateways() ([]models.GatewayInfo, error) {
	//return getGateways_OldDB()
	return getGateways()
}
func getGateways() ([]models.GatewayInfo, error) {
	// 1. from stationarea
	// 2. from ScdDeviceParam
	devParams, error := GetAllDevices()
	if error != nil {
		return nil, error
	}
	gwInfo := make([]models.GatewayInfo, 0)
	gwSn := make(map[string]struct{})
	for _, devParam := range devParams {
		if devParam.MqttGateway == "" {
			log.Info("filter gateways from device list: dev ", devParam.Code, " MqttGateway is empty, continue")
			continue
		}
		if _, ok := gwSn[devParam.MqttGateway]; !ok {
			gwInfo = append(gwInfo, models.GatewayInfo{Name: devParam.MqttGateway, Sn: devParam.MqttGateway})
			gwSn[devParam.MqttGateway] = struct{}{}
		}
	}

	return gwInfo, nil
}

func GetAllDevices() ([]ScdDeviceParam, error) {

	var allDevice []ScdDeviceParam

	return allDevice, nil
}

func GetAllStationParam() ([]StationParam, error) {

	var allstParams []StationParam

	return allstParams, nil
}

func GetDevices(gatewaySn string) (*models.DeviceData, error) {
	//return getDevices_OldDB(gatewaySn)
	return getDevices(gatewaySn)
}

func getDevices(gatewaySn string) (*models.DeviceData, error) {
	// allDevice, err := GetAllDevices()
	// if err != nil {
	// 	return nil, fmt.Errorf("GetDevices failed:%v", err)
	// }

	// // make map to termianl no
	// stCode2RtuNO := make(map[string]int)
	// allStParams, err := GetAllStationParam()
	// if err != nil {
	// 	return nil, fmt.Errorf("GetDevices failed:%v", err)
	// }
	// for _, stParam := range allStParams {
	// 	stCode2RtuNO[stParam.Code] = stParam.SubID
	// }

	// rtuChStatus := db.SelectTerminalStatusMap()
	// devData := &models.DeviceData{Content: make(map[string]models.DeviceInfo), Topic: "devices"}
	// for _, device := range allDevice {
	// 	deviceKey := device.MqttDevice
	// 	if deviceKey == "" {
	// 		deviceKey = device.Code
	// 		// TODO:X if MqttDevice empty, use device.Code instead
	// 		continue
	// 	}
	// 	if device.MqttGateway != gatewaySn {
	// 		continue
	// 	}

	// 	status := 1
	// 	if rtuNo, ok := stCode2RtuNO[device.StationCode]; ok {
	// 		if st, ok := rtuChStatus[rtuNo]; ok {
	// 			if st == 2 { // 2 online; others offline
	// 				status = 2
	// 			}
	// 			log.Info("GetDevices rbd channel %d status:%d (2-online, 3 offline)", rtuNo, st)
	// 		} else {
	// 			log.Errorf("GetDevices rtuNo %d not found in rdb channel:%v", rtuNo, rtuChStatus)
	// 		}
	// 	} else {
	// 		log.Errorf("GetDevices device.StationCode %s not found in StationParam:%v", device.StationCode, allStParams)
	// 	}

	// 	// deviceInfo = append(deviceInfo, models.DeviceInfo{DevCode: device.MqttDevCode, Gateway: device.MqttGateway, Name: device.Name, Sn: device.MqttDevice})
	// 	devData.Content[deviceKey] = models.DeviceInfo{
	// 		DevCode: device.MqttDevCode, Gateway: device.MqttGateway, Name: device.Name, Sn: device.MqttDevice, Status: status}
	// }

	// return devData, nil

	return &models.DeviceData{}, nil
}

func getYcParamByScdCode(scdCode string) ([]YcParam, error) {
	var ycParams []YcParam
	return ycParams, nil
}

func getYxParamByScdCode(scdCode string) ([]YxParam, error) {

	var yxParams []YxParam
	return yxParams, nil
}

func getYkParamByScdCode(scdCode string) ([]YxParam, error) {

	var yxParams []YxParam
	return yxParams, nil
}

func getYtParamByMqttDev(deviceSn string) ([]YtParam, error) {

	var ytParams []YtParam
	return ytParams, nil
}

func GetDevicesYc(deviceSn []string, updateCache bool) (*models.TeleData, error) {
	//return getDevicesYc_OldDB(deviceSn, updateCache)
	return getDevicesYc(deviceSn)
}

func getAllDevCodesMap() (map[string][]string, error) {

	devCodeMap := make(map[string][]string)
	return devCodeMap, nil
}

func getDevicesYc(deviceSn []string) (*models.TeleData, error) {
	// devCodeMap, err := getAllDevCodesMap()
	// if err != nil {
	// 	log.Error("GetDevicesYc getDevCodes err:", err)
	// 	return nil, fmt.Errorf("GetDevicesYc getDevCodes err:%v", err)
	// }

	// // make map for rdb code-value
	// ycValueMap := db.SelectRdbYcDataMap()

	// teledata := &models.TeleData{Topic: "tm", Content: nil}

	// content := make(map[string][]models.KeyValueInfo)
	// for _, devSn := range deviceSn {
	// 	// 1. get yc params for each deviceSn
	// 	scdCodes, ok := devCodeMap[devSn]
	// 	if !ok {
	// 		log.Error("GetDevicesYc deviceSn:", devSn, " not found")
	// 		continue
	// 	}
	// 	ycParams := make([]YcParam, 0)
	// 	for _, scdCode := range scdCodes {
	// 		ycParam, err := getYcParamByScdCode(scdCode)
	// 		if err != nil {
	// 			log.Error("GetDevicesYc getYcParamByScdCode failed:", err.Error(), ", scdCode:", scdCode)
	// 			continue
	// 		}
	// 		ycParams = append(ycParams, ycParam...)
	// 	}

	// 	// 2. get each value for each yc param
	// 	keyValueInfos := make([]models.KeyValueInfo, 0)
	// 	for _, ycParam := range ycParams {
	// 		value, ok := ycValueMap[ycParam.Code]
	// 		if !ok {
	// 			log.Error("GetDevicesYc ycParam:", ycParam, " not found in rdb ycValueMap")
	// 			continue
	// 		}

	// 		pointKey := ycParam.MqttKeyID
	// 		if pointKey == "" {
	// 			pointKey = ycParam.Name
	// 			continue // TODO:May comment out the continue if show All data. [if MqttKeyID empty, use ycParam.Name instead]
	// 		}
	// 		keyValueInfos = append(keyValueInfos, models.KeyValueInfo{K: pointKey, V: value})
	// 	}

	// 	if len(keyValueInfos) > 0 {
	// 		// if no data, do not set, then web-pages not show
	// 		content[devSn] = keyValueInfos
	// 	}
	// }

	// if len(content) != 0 {
	// 	teledata.Content = content
	// }

	// return teledata, nil

	return &models.TeleData{}, nil
}

func GetDevicesYx(deviceSn []string, updateCache bool) (*models.TeleData, error) {
	//return getDevicesYx_OldDB(deviceSn, updateCache)
	return getDevicesYx(deviceSn)
}

func getDevicesYx(deviceSn []string) (*models.TeleData, error) {

	// devCodeMap, err := getAllDevCodesMap()
	// if err != nil {
	// 	log.Error("getDevicesYx getDevCodes err:", err)
	// 	return nil, fmt.Errorf("getDevicesYx getDevCodes err:%v", err)
	// }

	// // make map for rdb code-value
	// yxValueMap := db.SelectRdbYxDataMap()

	// teledata := &models.TeleData{Topic: "ts", Content: nil}

	// content := make(map[string][]models.KeyValueInfo)
	// for _, devSn := range deviceSn {
	// 	// 1. get yx params for each deviceSn
	// 	scdCodes, ok := devCodeMap[devSn]
	// 	if !ok {
	// 		log.Error("getDevicesYx deviceSn:", devSn, " not found")
	// 		continue
	// 	}
	// 	yxParams := make([]YxParam, 0)
	// 	for _, scdCode := range scdCodes {
	// 		yxParam, err := getYxParamByScdCode(scdCode)
	// 		if err != nil {
	// 			log.Error("getDevicesYx getYxParamByScdCode failed:", err.Error(), ", scdCode:", scdCode)
	// 			continue
	// 		}
	// 		yxParams = append(yxParams, yxParam...)
	// 	}

	// 	// 2. get each value for each yc param
	// 	keyValueInfos := make([]models.KeyValueInfo, 0)
	// 	for _, yxParam := range yxParams {
	// 		value, ok := yxValueMap[yxParam.Code]
	// 		if !ok {
	// 			log.Error("getDevicesYx yxParam:", yxParam, " not found in rdb yxValueMap")
	// 			continue
	// 		}
	// 		pointKey := yxParam.MqttKeyID
	// 		if pointKey == "" {
	// 			pointKey = yxParam.Name
	// 			continue // TODO: May comment out the continue if show All data. [if MqttKeyID empty, use yxParam.Name instead]
	// 		}

	// 		keyValueInfos = append(keyValueInfos, models.KeyValueInfo{K: pointKey, V: float32(value)})
	// 	}

	// 	if len(keyValueInfos) > 0 {
	// 		// if no data, do not set, then web-pages not show
	// 		content[devSn] = keyValueInfos
	// 	}
	// }
	// if len(content) != 0 {
	// 	teledata.Content = content
	// }

	// return teledata, nil

	return &models.TeleData{}, nil
}

func GetDevicesYt(deviceSn string, updateCache bool) (map[string]float32, error) {
	// return getDevicesYt_OldDB(deviceSn, updateCache)
	return getDevicesYt(deviceSn)
}

func getDevicesYt(deviceSn string) (map[string]float32, error) {

	ytValueMap := make(map[string]float32)
	return ytValueMap, nil
}

func GetDevicesYk(deviceSn string, updateCache bool) (map[string]int16, error) {
	//return getDevicesYk_OldDB(deviceSn, updateCache)
	return getDevicesYk(deviceSn)
}

func getDevicesYk(deviceSn string) (map[string]int16, error) {

	result := make(map[string]int16)

	return result, nil
}

func sortYcParams(params []YcParam) {
	sort.Slice(params, func(i, j int) bool {
		a, b := params[i], params[j]
		if a.TerminalNo != b.TerminalNo {
			return a.TerminalNo < b.TerminalNo
		}
		return a.No < b.No
	})
}

func GetAllYcParams() ([]YcParam, error) {

	var ycParams []YcParam
	return ycParams, nil
}

func sortYxParams(params []YxParam) {
	sort.Slice(params, func(i, j int) bool {
		a, b := params[i], params[j]
		if a.TerminalNo != b.TerminalNo {
			return a.TerminalNo < b.TerminalNo
		}
		return a.No < b.No
	})
}

func GetAllYxParams() ([]YxParam, error) {
	var yxParams []YxParam
	return yxParams, nil
}

// func getAllRdbStreamParams() ([]db.StreamParam, error) {
// 	result := make([]db.StreamParam, 0)
// 	return result, nil
// }

// func GetAllRDBStreamParams() ([]db.StreamParam, error) {
// 	return getAllRdbStreamParams()
// 	//return getAllRdbStreamParams_OldDB()
// }

func getCodeName(deviceSn string, mqttID string) (code, pointType, tableName string, err error) {
	// get code from yaocecanshubiao, but no mqtt_device in it
	// so get station code from scd by mqtt_device , then get station id by station code
	// then get code from yaocecanshubiao by station id(same as terminal id)

	scddev, err := GetAllDevices()
	if err != nil {
		return "", "", "", fmt.Errorf("getCodeName GetAllDevices err:%v", err)
	}
	scdDevCode := ""
	for _, dev := range scddev {
		if dev.MqttDevice == deviceSn {
			scdDevCode = dev.Code
		}
	}

	ycParams, err := getYcParamByScdCode(scdDevCode)
	if err != nil {
		return "", "", "", fmt.Errorf("getCodeName getYcParamByScdCode err:%v", err)
	}
	for _, ycParam := range ycParams {
		if ycParam.MqttKeyID == mqttID {
			return ycParam.Code, "Yc", "analog", nil
		}
	}

	yxParams, err := getYxParamByScdCode(scdDevCode)
	if err != nil {
		return "", "", "", fmt.Errorf("getCodeName getYxParamByScdCode err:%v", err)
	}
	for _, yxParam := range yxParams {
		if yxParam.MqttKeyID == mqttID {
			tableName = func(code string) string {
				if strings.HasSuffix(code, "Kg") {
					return "break"
				}
				// end with Bh
				return "prot"
			}(yxParam.Code)
			return yxParam.Code, "Yx", tableName, nil
		}
	}

	return "", "", "", fmt.Errorf("not found")
}

func GetCodeName(deviceSn string, mqttID string) (code, pointType, tableName string, err error) {
	//return getCodeName_OldDB(deviceSn, mqttID)

	return getCodeName(deviceSn, mqttID)
}
