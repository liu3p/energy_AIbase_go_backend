package emsdb

import (
	"fmt"
	"sync"

	//"webserver-arm/db"

	"webserver-arm/models"
)

type MqttDevice struct {
	Sn        string `gorm:"column:SN;primary_key;type:char(30)"`
	Name      string `gorm:"column:NAME;type:char(64)"`
	Scddev    string `gorm:"column:SCD_DEV;type:char(24)"`
	Gw        string `gorm:"column:GATEWAY_SN;type:char(30)"`
	Devcode   string `gorm:"column:DEV_CODE;type:char(30)"`
	Channelno int    `gorm:"column:CHANNEL_NO;type:int(6)"`
}

func (MqttDevice) TableName() string {
	return "mqtt_device"
}

type TransYcParam struct {
	Groupid int    `gorm:"column:组号;primary_key;type:int(6)"`
	Code    string `gorm:"column:代码;primary_key;type:char(24)"`
	//Name      string `gorm:"column:描述;type:char(128)"`
	//ScdCode   string `gorm:"column:设备代码;type:char(24)"`
	MqttGw     string `gorm:"column:MQTT_GATEWAY;type:char(30)"`
	MqttDevice string `gorm:"column:MQTT_DEVICE;type:char(30)"`
	MqttKeyID  string `gorm:"column:MQTT_ID;type:char(30)"`
}

func (TransYcParam) TableName() string {
	return "转发遥测表"
}

type TransYxParam struct {
	Groupid int    `gorm:"column:组号;primary_key;type:int(6)"`
	Code    string `gorm:"column:代码;primary_key;type:char(24)"`
	//Name      string `gorm:"column:描述;type:char(128)"`
	//ScdCode   string `gorm:"column:设备代码;type:char(24)"`
	MqttGw     string `gorm:"column:MQTT_GATEWAY;type:char(30)"`
	MqttDevice string `gorm:"column:MQTT_DEVICE;type:char(30)"`
	MqttKeyID  string `gorm:"column:MQTT_ID;type:char(30)"`
}

func (TransYxParam) TableName() string {
	return "转发遥信表"
}

var (
	transYcParamCache sync.Map
	transYxParamCache sync.Map
	// transYcParamCache map[string][]TransYcParam = make(map[string][]TransYcParam)
	// transYxParamCache map[string][]TransYxParam = make(map[string][]TransYxParam)
)

func getGateways_OldDB() ([]models.GatewayInfo, error) {
	// 1. from stationarea
	// 2. from ScdDeviceParam
	devParams, error := getAllDevices_OldDB()
	if error != nil {
		return nil, error
	}
	gwInfo := make([]models.GatewayInfo, 0)
	gwSn := make(map[string]struct{})
	for _, devParam := range devParams {
		if _, ok := gwSn[devParam.Gw]; !ok {
			gwInfo = append(gwInfo, models.GatewayInfo{Name: devParam.Gw, Sn: devParam.Gw})
			gwSn[devParam.Gw] = struct{}{}
		}
	}

	return gwInfo, nil
}

func getAllDevices_OldDB() ([]MqttDevice, error) {

	var allDevice []MqttDevice
	return allDevice, nil
}

func getDevices_OldDB(gatewaySn string) (*models.DeviceData, error) {
	// allDevice, err := getAllDevices_OldDB()
	// if err != nil {
	// 	return nil, err
	// }
	// deviceStatus := db.SelectChannelStatusMap()
	// devData := &models.DeviceData{Content: make(map[string]models.DeviceInfo), Topic: "devices"}
	// for _, device := range allDevice {
	// 	if device.Gw != gatewaySn {
	// 		continue
	// 	}
	// 	// deviceInfo = append(deviceInfo, models.DeviceInfo{DevCode: device.MqttDevCode, Gateway: device.MqttGateway, Name: device.Name, Sn: device.MqttDevice})
	// 	status := 1
	// 	if st, ok := deviceStatus[device.Channelno]; ok {
	// 		if st == 2 { // 2 online; others offline
	// 			status = 2
	// 		}
	// 	}
	// 	devData.Content[device.Sn] = models.DeviceInfo{DevCode: device.Devcode, Gateway: device.Gw,
	// 		Name: device.Name, Sn: device.Sn, Status: status}
	// }

	// return devData, nil

	return &models.DeviceData{}, nil
}

func getTransYcParamByDevSn(deviceSn string, updateCache bool) ([]TransYcParam, error) {

	return nil, fmt.Errorf("getYcParamByScdCode Find err")
}

func getTransYxParamByDevSn(deviceSn string, updateCache bool) ([]TransYxParam, error) {

	return nil, fmt.Errorf("getTransYxParamByDevSn Find err")
}

func getAllTransYcParam() ([]TransYcParam, error) {

	return nil, fmt.Errorf("getAllTransYcParam open db err")
}

func getAllTransYxParam() ([]TransYxParam, error) {

	return nil, fmt.Errorf("getAllTransYxParam open db err")
}

// func getAllRdbStreamParams_OldDB() ([]db.StreamParam, error) {

// 	return nil, fmt.Errorf("getAllRdbStreamParams open db err:%v", err)
// }

// func getDevicesYc_OldDB(deviceSn []string, updateCache bool) (*models.TeleData, error) {
// 	// make map for rdb code-value
// 	ycValueMap := db.SelectRdbYcDataMap()
// 	teledata := &models.TeleData{Topic: "tm", Content: nil}

// 	content := make(map[string][]models.KeyValueInfo)
// 	for _, devSn := range deviceSn {
// 		// 1. get yc params for each deviceSn
// 		ycParams, err := getTransYcParamByDevSn(devSn, updateCache)
// 		if err != nil {
// 			log.Error("GetDevicesYc getTransYcParamByDevSn failed:", err.Error(), ", devSn:", devSn)
// 			continue
// 		}

// 		// 2. get each value for each yc param
// 		keyValueInfos := make([]models.KeyValueInfo, 0)
// 		for _, ycParam := range ycParams {
// 			value, ok := ycValueMap[ycParam.Code]
// 			if !ok {
// 				log.Error("GetDevicesYc ycParam:", ycParam, " not found in rdb ycValueMap")
// 				continue
// 			}
// 			// TODO:X if MqttKeyID empty, use ycParam.Code instead
// 			keyValueInfos = append(keyValueInfos, models.KeyValueInfo{K: ycParam.MqttKeyID, V: value})
// 		}

// 		if len(keyValueInfos) > 0 {
// 			// if no data, do not set, then web-pages not show
// 			content[devSn] = keyValueInfos
// 		}
// 	}
// 	if len(content) != 0 {
// 		teledata.Content = content
// 	}

// 	return teledata, nil
// }

// func getDevicesYx_OldDB(deviceSn []string, updateCache bool) (*models.TeleData, error) {
// 	// make map for rdb code-value
// 	yxValueMap := db.SelectRdbYxDataMap()
// 	teledata := &models.TeleData{Topic: "ts", Content: nil}

// 	content := make(map[string][]models.KeyValueInfo)
// 	for _, devSn := range deviceSn {
// 		// 1. get yx params for each deviceSn
// 		yxParams, err := getTransYxParamByDevSn(devSn, updateCache)
// 		if err != nil {
// 			log.Error("GetDevicesYc getTransYxParamByDevSn failed:", err.Error(), ", devSn:", devSn)
// 			continue
// 		}

// 		// 2. get each value for each yc param
// 		keyValueInfos := make([]models.KeyValueInfo, 0)
// 		for _, yxParam := range yxParams {
// 			value, ok := yxValueMap[yxParam.Code]
// 			if !ok {
// 				log.Error("GetDevicesYc yxParam:", yxParam, " not found in rdb yxValueMap")
// 				continue
// 			}
// 			// TODO:X if MqttKeyID empty, use yxParam.Code instead
// 			keyValueInfos = append(keyValueInfos, models.KeyValueInfo{K: yxParam.MqttKeyID, V: float32(value)})
// 		}

// 		if len(keyValueInfos) > 0 {
// 			// if no data, do not set, then web-pages not show
// 			content[devSn] = keyValueInfos
// 		}
// 	}
// 	if len(content) != 0 {
// 		teledata.Content = content
// 	}

// 	return teledata, nil
// }

// func getDevicesYt_OldDB(deviceSn string, updateCache bool) (map[string]float32, error) {
// 	mydb, err := db.NewMsDB()
// 	if err != nil {
// 		log.Error("getDevicesYt_OldDB NewMsDB err:", err)
// 		return nil, fmt.Errorf("getDevicesYt_OldDB open db err:%v", err)
// 	}

// 	var devAdj []db.DevAdjust
// 	// mydb.Find(&devAdj, "MQTT_GATEWAY=? and MQTT_DEVICE=?", gatewaySn, deviceSn)
// 	mydb.Find(&devAdj, "MQTT_DEVICE=?", deviceSn) // only devicesn is enough to filter data

// 	ycValueMap := db.SelectRdbYcDataMap()

// 	result := make(map[string]float32)
// 	for _, v := range devAdj {
// 		value, ok := ycValueMap[v.Code]
// 		if ok {
// 			result[v.MqttID] = value
// 		} else {
// 			result[v.MqttID] = 0
// 		}
// 	}
// 	return result, nil
// }

// func getDevicesYk_OldDB(deviceSn string, updateCache bool) (map[string]int16, error) {
// 	mydb, err := db.NewMsDB()
// 	if err != nil {
// 		log.Error("getDevicesYk_OldDB NewMsDB err:", err)
// 		return nil, fmt.Errorf("getDevicesYk_OldDB open db err:%v", err)
// 	}

// 	var devRemote []db.DevRemote
// 	// mydb.Find(&devRemote, "MQTT_GATEWAY=? and MQTT_DEVICE=?", gatewaySn, deviceSn)
// 	mydb.Find(&devRemote, "MQTT_DEVICE=?", deviceSn) // only devicesn is enough to filter data

// 	yxValueMap := db.SelectRdbYxDataMap()

// 	result := make(map[string]int16, 0)
// 	for _, v := range devRemote {
// 		value, ok := yxValueMap[v.MqttID]
// 		if ok {
// 			result[v.MqttID] = value
// 		} else {
// 			result[v.MqttID] = 0
// 		}
// 	}
// 	return result, nil
// }

// func getCodeName_OldDB(deviceSn string, mqttID string) (code, pointType, tableName string, err error) {
// 	ycParams, err := getTransYcParamByDevSn(deviceSn, false)
// 	if err != nil {
// 		return "", "", "", err
// 	}
// 	for _, ycParam := range ycParams {
// 		if ycParam.MqttKeyID == mqttID {
// 			return ycParam.Code, "Yc", "analog", nil
// 		}
// 	}

// 	yxParams, err := getTransYxParamByDevSn(deviceSn, false)
// 	if err != nil {
// 		return "", "", "", err
// 	}
// 	for _, yxParam := range yxParams {
// 		if yxParam.MqttKeyID == mqttID {
// 			tableName = func(code string) string {
// 				if strings.HasSuffix(code, "Kg") {
// 					return "break"
// 				}
// 				// end with Bh
// 				return "prot"
// 			}(yxParam.Code)
// 			return yxParam.Code, "Yx", tableName, nil
// 		}
// 	}
// 	return "", "", "", fmt.Errorf("not found")
// }
