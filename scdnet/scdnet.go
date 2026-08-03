package scdnet

// /*
// #cgo LDFLAGS: -lcgo_scdnet -L/home/prs3000/eco_ems/lib
// #cgo CFLAGS: -I/home/prs3000/eco_ems/source/include
// #include <stdlib.h>
// #include "local_web/ProcMan_cgo.h"
// */
// import (
// 	"C"
// )
// import (
// 	"bytes"
// 	"fmt"
// 	"io"
// 	"unsafe"
// 	"webserver-arm/log"

// 	"golang.org/x/text/encoding/simplifiedchinese"
// 	"golang.org/x/text/transform"
// )

// type ProcInfo struct {
// 	Desc     string // max len 24
// 	Name     string // max len 24
// 	State    int
// 	Pid      string // max len 128
// 	Cpuload  string // max len 128
// 	Memusage string // max len 128
// 	Warninfo string // max len 128
// 	Keyproc  int
// }

// type SysProcs struct {
// 	Num  int
// 	Info []ProcInfo
// }

// type Proc struct {
// 	ProcName string // max len 24
// 	Pid      int
// 	Idx      int
// }

// type LogProcs struct {
// 	Num  int
// 	Info []Proc
// }

// type ProcLogType struct {
// 	TypeIdx  int    `json:"typeidx"`
// 	TypeDesc string `json:"typedesc"`
// }

// type ProcLogInfo struct {
// 	Num  int
// 	Logs []string
// }

// func gbk2utf8(gbkStr string) string {
// 	gbkReader := transform.NewReader(bytes.NewReader([]byte(gbkStr)), simplifiedchinese.GBK.NewDecoder())
// 	utf8Data, err := io.ReadAll(gbkReader)
// 	if err != nil {
// 		log.Error("gbk2utf8 failed:", err.Error(), ", ret gbkStr:", gbkStr)
// 		return gbkStr
// 	}
// 	return string(utf8Data)
// }

// func KillAProc(procName string) error {
// 	scdPtr := C.GetProcMan()
// 	cStr := C.CString(procName)
// 	defer C.free(unsafe.Pointer(cStr))
// 	ret := C.ProcMan_killAProc(scdPtr, cStr)
// 	if ret < 0 {
// 		return fmt.Errorf("run failed code(%d)", ret)
// 	}
// 	return nil
// }

// func GetSysProc() ([]*ProcInfo, error) {
// 	scdPtr := C.GetProcMan()
// 	var sysProcs C.SysProcs
// 	ret := C.ProcMan_GetSysProc(scdPtr, &sysProcs)
// 	if ret < 0 {
// 		return nil, fmt.Errorf("run failed code(%d)", ret)
// 	}

// 	procInfos := make([]*ProcInfo, int(sysProcs.num))
// 	for i := 0; i < int(sysProcs.num); i++ {
// 		procInfo := &ProcInfo{
// 			Desc:     gbk2utf8(C.GoString(&sysProcs.procs[i].desc[0])),
// 			Name:     gbk2utf8(C.GoString(&sysProcs.procs[i].name[0])),
// 			State:    int(sysProcs.procs[i].state),
// 			Pid:      C.GoString(&sysProcs.procs[i].pid[0]),
// 			Cpuload:  C.GoString(&sysProcs.procs[i].cpuload[0]),
// 			Memusage: C.GoString(&sysProcs.procs[i].memusage[0]),
// 			Warninfo: C.GoString(&sysProcs.procs[i].warninfo[0]),
// 			Keyproc:  int(sysProcs.procs[i].key_proc),
// 		}
// 		procInfos[i] = procInfo
// 	}
// 	log.Info("ProcMan_GetSysProc sysProcs.num:", int(sysProcs.num), ",len procInfos:", len(procInfos))
// 	return procInfos, nil
// }

// func GetLogProcs() ([]*Proc, error) {
// 	scdPtr := C.GetProcMan()
// 	var logProcs C.LogProcs
// 	ret := C.ProcMan_GetLogProcs(scdPtr, &logProcs)
// 	if ret < 0 {
// 		return nil, fmt.Errorf("run failed code(%d)", ret)
// 	}

// 	procs := make([]*Proc, int(logProcs.num))
// 	for i := 0; i < int(logProcs.num); i++ {
// 		proc := &Proc{
// 			ProcName: gbk2utf8(C.GoString(&logProcs.procs[i].proc_name[0])),
// 			Pid:      int(logProcs.procs[i].pid),
// 			Idx:      int(logProcs.procs[i].idx),
// 		}
// 		procs[i] = proc
// 	}

// 	return procs, nil
// }

// func GetLogTypes(procIdx int) ([]ProcLogType, error) {
// 	scdPtr := C.GetProcMan()
// 	var procLogType C.ProcLogType
// 	ret := C.ProcMan_GetLogTypes(scdPtr, C.int(procIdx), &procLogType)
// 	if ret < 0 {
// 		return nil, fmt.Errorf("run failed code(%d)", ret)
// 	}
// 	logTypes := make([]ProcLogType, int(procLogType.num))
// 	for i := 0; i < int(procLogType.num); i++ {

// 		logType := ProcLogType{
// 			TypeDesc: gbk2utf8(C.GoString(&procLogType.type_desc[i][0])),
// 			TypeIdx:  int(procLogType.type_idx[i]),
// 		}
// 		logTypes[i] = logType
// 	}

// 	return logTypes, nil
// }

// func SetLogProc(procIdx int) error {
// 	scdPtr := C.GetProcMan()
// 	ret := C.ProcMan_SetLogProc(scdPtr, C.int(procIdx))
// 	if ret < 0 {
// 		return fmt.Errorf("run failed code(%d)", ret)
// 	}
// 	return nil
// }

// func GetLog(logType int) ([]int, []string, error) {
// 	scdPtr := C.GetProcMan()
// 	var procLogInfo C.ProcLogInfo
// 	ret := C.ProcMan_GetLog(scdPtr, C.int(logType), &procLogInfo)
// 	if ret != 0 {
// 		return nil, nil, fmt.Errorf("run failed code(%d)", ret)
// 	}

// 	logs := make([]string, int(procLogInfo.num))
// 	logTypeID := make([]int, int(procLogInfo.num))
// 	for i := 0; i < int(procLogInfo.num); i++ {
// 		logs[i] = gbk2utf8(C.GoString(&procLogInfo.logs[i][0]))
// 		logTypeID[i] = int(procLogInfo.logs_type[i])
// 	}
// 	return logTypeID, logs, nil
// }
