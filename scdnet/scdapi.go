package scdnet

// /*
// #cgo LDFLAGS: -lcgo_scdnet -L/home/prs3000/eco_ems/lib
// #cgo CFLAGS: -I/home/prs3000/eco_ems/source/include
// #include <stdlib.h>
// #include "local_web/ScdMan_cgo.h"

// #define  SCADA_YK_PRESET            0		// 遥控预置
// #define  SCADA_YK_CHECK_OK          1		// 遥控返校正确
// #define  SCADA_YK_CHECK_ERR         2		// 遥控返校错误
// #define	 SCADA_YK_CHECK_TIMEOUT	    3		// 遥控返校超时
// #define  SCADA_YK_COMFIRM           4		// 遥控确认
// #define	 SCADA_YK_COMFIRM_TIMEOUT	5		// 遥控确认发令超时
// #define  SCADA_YK_EXEC              6		// 遥控执行
// #define  SCADA_YK_EXEC_TIMEOUT	    7		// 遥控执行发令超时
// #define  SCADA_YK_CANCEL            8		// 遥控撤消
// #define  SCADA_YK_SUCCESS           9		// 遥控成功
// #define  SCADA_YK_EXEC_FAIL         10		// 遥控拒动
// #define  SCADA_YK_CMD_ERR		    126		// 遥控命令错误
// #define  SCADA_YK_NULL              127		// 空：过程已终结

// */
// import (
// 	"C"
// )

// import (
// 	"fmt"
// 	"sync"
// 	"time"
// 	"unsafe"
// 	"webserver-arm/log"
// )

// var (
// 	once         sync.Once
// 	iScdCallback *scdCallBack
// )

// const (
// 	YkYtFailed        = 5
// 	YkYtSuccess       = 6
// 	YkYtSendTimeout   = 7
// 	YkYtRevoke        = 8
// 	YkYtRecvTimeout   = 9
// 	YkYtSysErr        = 10
// 	YkYtPointNotExist = 11
// )

// type callState struct {
// 	state    int
// 	expired  time.Time
// 	resultCh chan<- int
// }

// type scdCallBack struct {
// 	callStateMap map[string]*callState
// 	mutex        sync.Mutex
// 	stopCh       chan struct{}
// }

// func newScdCallBack() *scdCallBack {
// 	once.Do(func() {
// 		iScdCallback = &scdCallBack{
// 			callStateMap: make(map[string]*callState),
// 			stopCh:       make(chan struct{}),
// 		}
// 		go iScdCallback.run()
// 	})
// 	return iScdCallback
// }

// func (scd *scdCallBack) registCall(codeName string, timeOut time.Duration, resultCh chan<- int) {
// 	scd.mutex.Lock()
// 	defer scd.mutex.Unlock()
// 	scd.callStateMap[codeName] = &callState{
// 		state:    -1,
// 		resultCh: resultCh,
// 		expired:  time.Now().Add(timeOut)}
// }

// func (scd *scdCallBack) Stop() {
// 	close(scd.stopCh)
// }

// func (scd *scdCallBack) cleanupTimeout() {
// 	scd.mutex.Lock()
// 	defer scd.mutex.Unlock()

// 	now := time.Now()
// 	for key, st := range scd.callStateMap {
// 		if now.After(st.expired) {
// 			if st.resultCh != nil {
// 				st.resultCh <- YkYtRecvTimeout
// 				close(st.resultCh)
// 			}
// 			delete(scd.callStateMap, key)
// 			log.Infof("scdCallBack key %s expired, removed.", key)
// 		}
// 	}
// }

// func (scd *scdCallBack) run() {
// 	ticker := time.NewTicker(time.Millisecond * 200)
// 	for {
// 		select {
// 		case <-scd.stopCh:
// 			log.Warn("scdCallBack-go stop run")
// 			return
// 		case <-ticker.C:
// 			var scdRetInfo C.ScdReturnInfo
// 			ret := C.ScdMan_ScadaReturn(&scdRetInfo)
// 			if ret == 0 { // yt/yk
// 				codeName := C.GoString(&scdRetInfo.objectname[0])
// 				scd.mutex.Lock()
// 				if st, ok := scd.callStateMap[codeName]; ok {
// 					resutState := -1
// 					switch scdRetInfo.resulttype {
// 					case C.SCADA_YK_EXEC_TIMEOUT:
// 						resutState = YkYtSendTimeout
// 					case C.SCADA_YK_CANCEL:
// 						resutState = YkYtRevoke
// 					case C.SCADA_YK_SUCCESS:
// 						resutState = YkYtSuccess
// 					case C.SCADA_YK_EXEC_FAIL:
// 						resutState = YkYtRecvTimeout
// 					case C.SCADA_YK_CMD_ERR:
// 						resutState = YkYtFailed
// 					case C.SCADA_YK_NULL:
// 						log.Errorf("scdCallBack  codeName %s type SCADA_YK_NULL", codeName)
// 					default:
// 						// record state, continue
// 						st.state = int(scdRetInfo.resulttype)
// 					}
// 					log.Infof("scdCallBack-go get ScadaReturn yk/yt code:%s resulttype:%d, to resutState:%d", codeName, int(scdRetInfo.resulttype), resutState)
// 					if resutState != -1 {
// 						if st.resultCh != nil {
// 							st.resultCh <- resutState
// 							close(st.resultCh)
// 						}
// 						delete(scd.callStateMap, codeName)
// 					}
// 				} else {
// 					// log err, continue
// 					log.Errorf("scdCallBack-go get ScadaReturn yk/yt code:%s resulttype:%d, not exist in callStateMap:%v", codeName, int(scdRetInfo.resulttype), scd.callStateMap)
// 				}
// 				scd.mutex.Unlock()
// 			} else {
// 				scd.cleanupTimeout()
// 			}

// 		}
// 	}
// }

// func GetClusterNodesIP() ([]string, int, error) {
// 	var srvInfo C.ScdSrvInfo
// 	ret := C.ScdMan_SrvInfo(&srvInfo)
// 	if ret != 0 {
// 		return nil, 0, fmt.Errorf("ScdMan_SrvInfo failed. ret:%d", ret)
// 	}

// 	fmt.Println("scdnum:", srvInfo.scdnum)
// 	fmt.Println("mainsrvno:", srvInfo.mainsrvno)

// 	ips := make([]string, 0)
// 	for i := 0; i < int(srvInfo.scdnum); i++ {
// 		hostname := C.GoString(&srvInfo.hostnames[i][0])
// 		ip := C.GoString(&srvInfo.ips[i][0])
// 		fmt.Printf("Node %d: hostname=%s, ip=%s\n", i, hostname, ip)
// 		ips = append(ips, ip)
// 	}

// 	return ips, int(srvInfo.mainsrvno), nil
// }

// func SendYkCmdReq(codeName, tableName string, value int, resultCh chan<- int) error {
// 	call := newScdCallBack()
// 	call.registCall(codeName, 40*time.Second, resultCh) // scada timeout is 30s

// 	cCodeName := C.CString(codeName)
// 	defer C.free(unsafe.Pointer(cCodeName))
// 	cTableName := C.CString(tableName)
// 	defer C.free(unsafe.Pointer(cTableName))

// 	result := C.ScdMan_SendYkCmdReq(cCodeName, cTableName, C.int(value))
// 	if result != 1 {
// 		return fmt.Errorf("ScdMan_SendYkCmdReq called failed. result:%d", result)
// 	}
// 	return nil
// }

// func SendYtCmdReq(codeName string, value float32, resultCh chan<- int) error {
// 	call := newScdCallBack()
// 	call.registCall(codeName, 40*time.Second, resultCh) // scada timeout is 30s

// 	cCodeName := C.CString(codeName)
// 	defer C.free(unsafe.Pointer(cCodeName))

// 	result := C.ScdMan_SendYtCmdReq(cCodeName, C.float(value))
// 	if result != 1 {
// 		return fmt.Errorf("ScdMan_SendYtCmdReq called failed. result:%d", result)
// 	}
// 	return nil
// }
