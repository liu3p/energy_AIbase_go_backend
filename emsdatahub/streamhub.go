package emsdatahub

import (
	"webserver-arm/emsclient"
	"webserver-arm/models"
)

// type PushMessage struct {
// 	Topic   string                     `json:"topic"`   // 主题字段,"prop","ts","tm","state"
// 	Content map[string][]PointDataItem `json:"content"` // 内容字段，键为设备标识（如"esscab1"），值为属性数组
// }

type ContentBody map[string][]models.KeyValueInfoEx

// type PointDataItem struct {
// 	K  string      `json:"k"`  // 属性键（如"CabIP"、"CabMac"）
// 	V  interface{} `json:"v"`  // 属性值（如IP地址、Mac地址）
// 	At int64       `json:"at"` // 数据产生时间戳（毫秒级）
// 	Mt int64       `json:"mt"` // 数据修改时间戳（毫秒级）
// }

type StreamHub struct {
	clientMgr   *emsclient.Manager
	streamOrder int
}

func NewStreamHub(c *emsclient.Manager) *StreamHub {
	return &StreamHub{streamOrder: 1, clientMgr: c}
}

// func (sh *StreamHub) processRealTimeData(datas map[string]interface{}, allStreamData map[string]db.StreamParam) {
// 	for objName, data := range datas {
// 		pushData := emsclient.PushResp{}

// 		streamData, ok := allStreamData[objName]
// 		if !ok {
// 			log.Error("GetStreamParam obj", objName, " not found")
// 			continue
// 		}

// 		if streamData.TabName == "analog" {
// 			pushData.Topic = "tm"
// 		} else {
// 			pushData.Topic = "ts"
// 		}

// 		pushData.Content = ContentBody{
// 			streamData.DeviceSn: []models.KeyValueInfoEx{
// 				{K: streamData.MqttID, V: data, At: time.Now().UnixMilli(), Mt: time.Now().UnixMilli()},
// 			},
// 		}
// 		log.Info("preprocessRealTimeData obj:", objName, " streamData:", streamData, " pushData:", pushData)
// 		sh.clientMgr.PushMsg(&pushData)
// 	}
// }

func (sh *StreamHub) RegistPush() error {
	// // get all data
	// streamData, err := emsdb.GetAllRDBStreamParams()
	// if err != nil {
	// 	log.Error("RegistPush GetAllRDBStreamParams error:", err)
	// 	return err
	// }

	// if len(streamData) == 0 {
	// 	log.Warn("RegistPush no stream data")
	// 	return nil
	// }

	// // regist push
	// streamDataMap := make(map[string]db.StreamParam)
	// for _, data := range streamData {
	// 	streamDataMap[data.ObjName] = data
	// }

	// callback := func(data map[string]interface{}) {
	// 	sh.processRealTimeData(data, streamDataMap)
	// }

	// // regist to rdb
	// return db.SetStreamParam(sh.streamOrder, streamData, callback)
	return nil
}
