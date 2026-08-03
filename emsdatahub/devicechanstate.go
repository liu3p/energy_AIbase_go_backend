package emsdatahub

import (
	"fmt"
	"time"
	"webserver-arm/emsclient"
	"webserver-arm/emsdb"
	"webserver-arm/log"
)

// tmp FAKE struct
type rtuStatus struct {
	RtuNo   int
	Status  int16
	Time    int64
	DevSn   string
	DevType int16
	DevName string
}

func (sh *StreamHub) WatchDevChannelState() error {
	allDevice, err := emsdb.GetAllDevices()
	if err != nil {
		return fmt.Errorf("WatchDevChanState GetAllDevices failed:%v", err)
	}

	// make map to termianl no
	stCode2RtuNO := make(map[string]int)
	rtuNo2DevSn := make(map[int]string)
	allStParams, err := emsdb.GetAllStationParam()
	if err != nil {
		return fmt.Errorf("WatchDevChanState GetAllStationParam failed:%v", err)
	}
	for _, stParam := range allStParams {
		stCode2RtuNO[stParam.Code] = stParam.SubID
	}
	for _, device := range allDevice {
		if device.MqttDevice == "" {
			// TODO:X if MqttDevice empty, use device.Code instead
			continue
		}

		if rtuNo, ok := stCode2RtuNO[device.StationCode]; ok {
			rtuNo2DevSn[rtuNo] = device.MqttDevice
		}
	}
	log.Infof("WatchDevChanState rtuNo2DevSn:%v", rtuNo2DevSn)

	go func() {
		timer := time.NewTicker(time.Second * 3)
		defer timer.Stop()

		devStatusMap := make(map[int]int16) // rtuno: chanstate
		for {
			select {
			case <-timer.C:
				pushMap := make(map[string]int16)

				rtuStatus := make([]rtuStatus, 0) // db.SelectChannelStatus()
				log.Infof("WatchDevChanState rtuStatus:%v, devStatusMap:%v", rtuStatus, devStatusMap)
				for _, status := range rtuStatus {
					var st int16
					ok := false
					if st, ok = devStatusMap[status.RtuNo]; ok {
						if st == status.Status { // no change
							continue
						}
					}

					devStatusMap[status.RtuNo] = status.Status

					// push
					if devSn, ok := rtuNo2DevSn[status.RtuNo]; !ok {
						log.Errorf("WatchDevChanState rtuno %d devSn not found in rtuNo2DevSn:%v", status.RtuNo, rtuNo2DevSn)
						continue
					} else {
						log.Infof("WatchDevChanState devSn %s(%d) chanstate change from %d to %d, push devSn %s", devSn, status.RtuNo, st, status.Status)
						pushMap[devSn] = status.Status
					}
				}

				if len(pushMap) > 0 {
					pushData := emsclient.PushResp{Topic: "status", Content: pushMap}
					log.Info("WatchDevChanState pushData:", pushData)
					sh.clientMgr.PushMsg(&pushData)
				}
			}
		}
	}()

	return nil
}
