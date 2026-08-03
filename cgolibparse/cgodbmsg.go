package cgolibparse

/*
#cgo LDFLAGS: -ldb_client_cgo -lscada_client
#include <stdlib.h>       // for C.free
#include "cgo_dbkmsgclient.h"
#include "cgo_scadaclient.h"
*/
import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
	"webserver-arm/log"
)

type PointValue struct {
	OID          uint64 `json:"oid"`
	PointID      string `json:"pointID"`
	Name         string `json:"name"`
	CurrentValue string `json:"currvalue"`
	RawValue     string `json:"rawvalue"`
	CtlValue     string `json:"ctlvalue"` // 遥调遥控前置预置值
	Dead         string `json:"dead"`
	HisPeriod    string `json:"hisperiod"`
	Quality      string `json:"quality"`
	SendTime     string `json:"sendtime"`
}

var pointKeys = []string{"RawValue", "CurrValue", "CtlValue", "Dead", "HisPeriod", "Quality", "SendTime"}
var clientSeq uint64

type jsonResult struct {
	OID    uint64   `json:"oid"`
	Values []string `json:"values"`
}

func genUniqueClientID() string {
	// 原子自增，每次调用+1
	seq := atomic.AddUint64(&clientSeq, 1)
	// 毫秒时间戳，保证不同时间的seq不会重复
	milli := time.Now().UnixMilli()
	return fmt.Sprintf("client-%d-%d", milli, seq)
}

var (
	dbMsgInstance    *DBMsgManager
	dbMsgManagerOnce sync.Once
	dbMsgInitErr     error
)

type DBMsgManager struct {
	pointsValueMap      map[string]PointValue
	pointsValueMapMutex sync.RWMutex
	allPointsID         map[uint64]string
	pointsIDCount       map[uint64]uint32
	allPointsIDMutex    sync.RWMutex
	pointsKeys          []string
	cFields             **C.char
	cFieldCount         C.int
}

func GetDBMsgManagerInstance() (*DBMsgManager, error) {
	dbMsgManagerOnce.Do(func() {
		m := &DBMsgManager{
			pointsValueMap:      make(map[string]PointValue),
			pointsValueMapMutex: sync.RWMutex{},
			allPointsID:         make(map[uint64]string),
			pointsIDCount:       make(map[uint64]uint32),
			allPointsIDMutex:    sync.RWMutex{},
		}

		m.pointsKeys = []string{"RawValue", "CurrValue", "CtlValue", "Dead", "HisPeriod", "Quality", "SendTime"}
		// 将points转换为C++字符串数组
		fCount := len(m.pointsKeys)
		// 创建字符串指针数组
		m.cFields = (**C.char)(C.malloc(C.size_t(fCount) * C.size_t(unsafe.Sizeof(uintptr(0)))))
		if m.cFields == nil {
			dbMsgInitErr = fmt.Errorf("failed to allocate global C fields")
			log.Error(dbMsgInitErr.Error())
			return
		}
		// 将每个字段转换为C字符串
		for i, key := range pointKeys {
			cField := C.CString(key)
			// 使用 unsafe.Slice 赋值
			slice := unsafe.Slice(m.cFields, fCount)
			slice[i] = cField
		}
		m.cFieldCount = C.int(fCount)

		dbMsgInstance = m

		go m.startGetPointsValue()

		// 初始化C库
		C.DBCGOInit()

		log.Info("DBMsgManager Instance success")
	})
	return dbMsgInstance, dbMsgInitErr
}

// 辅助函数：转换点 ID 字符串列表
func convertPointIDs(pointIDs map[string]struct{}) ([]uint64, map[uint64]string, error) {
	oids := make([]uint64, 0, len(pointIDs))
	oidMap := make(map[uint64]string, len(pointIDs))
	for id, _ := range pointIDs {
		oid, err := pOIDToUint64(id)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid pointID %q: %w", id, err)
		}
		oids = append(oids, oid)
		oidMap[oid] = id
	}
	return oids, oidMap, nil
}

// getErrorCodeMsg 转换C接口的错误码为中文描述，方便排查
func getErrorCodeMsg(code int) string {
	switch code {
	case 0:
		return "operation success"
	case -1:
		return "operation db failed" //"操作数据库错误"
	case -2:
		return "memory alloc failed" //"内存分配错误"
	case -3:
		return "params error" //"参数错误"
	case -4:
		return "request timeout" //"请求超时"
	case -5:
		return "network send failed" //"网络发送错误"
	default:
		return fmt.Sprintf("unknown error code(%d)", code)
	}
}

func timestampToLocalString(timestampStr string) string {
	if timestampStr == "" {
		log.Error("timestampToLocalString timestampStr is empty")
		return ""
	}

	if len(timestampStr) >= 13 {
		// 截断最后三位
		timestampStr = timestampStr[:len(timestampStr)-3]
	}
	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		log.Error("timestampToLocalString parse timestamp failed: %s", err.Error())
		return ""
	}

	localLoc, err := time.LoadLocation("Local")
	if err != nil {
		localLoc = time.UTC
		log.Error("load local timezone failed: %s, use default UTC", err.Error())
	}

	t := time.Unix(ts, 0).In(localLoc)
	return t.Format(time.RFC3339)
}

func (m *DBMsgManager) CallCFuncQueryPointsValue(ids []uint64) ([]jsonResult, error) {
	cOids := (*C.uint64_t)(unsafe.Pointer(&ids[0]))
	cOidCount := C.int(len(ids))
	// 生成唯一clientID
	clientID := genUniqueClientID()
	cClientID := C.CString(clientID)

	// 初始化C侧JsonResult结构体，用于接收返回值
	cJsonResult := C.JsonResult{data: nil, error_code: C.int(0), length: C.int(0)}

	// 调用C核心接口GetFieldsValues
	cRet := C.GetFieldsValues(cClientID, cOids, cOidCount, (**C.char)(unsafe.Pointer(m.cFields)), m.cFieldCount, &cJsonResult)
	C.free(unsafe.Pointer(cClientID))
	// 处理C接口调用错误
	if cRet != 0 || cJsonResult.error_code != 0 {
		errMsg := getErrorCodeMsg(int(cJsonResult.error_code))
		if cJsonResult.data != nil {
			C.free(unsafe.Pointer(cJsonResult.data))
		}
		return nil, fmt.Errorf("GetFieldsValues failed, error_code: %d: %s", cJsonResult.error_code, errMsg)
	}
	if cJsonResult.data == nil || cJsonResult.length <= 0 {
		return nil, fmt.Errorf("GetFieldsValues return cJsonResult.data invalid")
	}

	// C字符串转为go字符串，解析JSON数据
	jsonStr := C.GoStringN(cJsonResult.data, C.int(cJsonResult.length)) // 按实际长度解析，避免截断
	C.free(unsafe.Pointer(cJsonResult.data))
	var result []jsonResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse C called GetFieldsValues result failed:%w, json:%s", err, jsonStr)
	}
	return result, nil
}

// startGetPointsValue 调用C接口查询数据的线程
func (m *DBMsgManager) startGetPointsValue() {
	for true {
		time.Sleep(1000 * time.Millisecond)

		m.allPointsIDMutex.RLock()
		if len(m.allPointsID) == 0 {
			m.allPointsIDMutex.RUnlock()
			continue
		}
		// 获取所有待查询的测点pid, 按照表分组（遥测、遥信...)
		idsTableMap := make(map[uint64][]uint64)
		for id := range m.allPointsID {
			tableNo := (id >> 24) & 0xFF
			if _, has := idsTableMap[tableNo]; !has {
				idsTableMap[tableNo] = make([]uint64, 0)
			}
			idsTableMap[tableNo] = append(idsTableMap[tableNo], id)
		}
		m.allPointsIDMutex.RUnlock()

		for _, ids := range idsTableMap {
			result, err := m.CallCFuncQueryPointsValue(ids)
			if err != nil {
				log.Error(err.Error())
				continue
			}

			m.pointsValueMapMutex.Lock()
			// 解析结果并赋值
			for i := range result {
				if result[i].OID == 0 {
					log.Error("C called GetFieldsValues returned invalid oid 0")
					continue
				}
				if len(result[i].Values) != len(m.pointsKeys) {
					log.Errorf("GetFieldsValues returned invalid, need %d, actual %d", len(m.pointsKeys), len(result[i].Values))
					continue
				}
				var pValue PointValue
				pValue.OID = result[i].OID
				pValue.PointID = m.allPointsID[pValue.OID]
				pValue.RawValue = result[i].Values[0]
				pValue.CurrentValue = result[i].Values[1]
				pValue.CtlValue = result[i].Values[2]
				pValue.Dead = result[i].Values[3]
				pValue.HisPeriod = result[i].Values[4]
				pValue.Quality = result[i].Values[5]
				pValue.SendTime = timestampToLocalString(result[i].Values[6])
				m.pointsValueMap[pValue.PointID] = pValue
			}
			m.pointsValueMapMutex.Unlock()
		}
		// 1s查询一次
		time.Sleep(1000 * time.Millisecond)
	}
}

// RegisterPointsID 注册需要订阅数据的点
func (m *DBMsgManager) RegisterPointsID(idNameMap map[string]string, registerIDs []string) (func() []PointValue, func(), error) {
	m.allPointsIDMutex.Lock()
	defer m.allPointsIDMutex.Unlock()

	for _, pointID := range registerIDs {
		pOIDUint64, err := pOIDToUint64(pointID)
		if err != nil {
			return nil, nil, fmt.Errorf("pOID '%s' is invalid: %s", pointID, err.Error())
		}
		m.allPointsID[pOIDUint64] = pointID
		m.pointsIDCount[pOIDUint64]++
	}

	// 获取所有测点函数
	getValue := func() []PointValue {
		m.pointsValueMapMutex.RLock()
		defer m.pointsValueMapMutex.RUnlock()

		pointsValue := make([]PointValue, 0, len(registerIDs))
		for _, id := range registerIDs {
			pValue, has := m.pointsValueMap[id]
			if !has {
				continue
			}
			pValue.Name = idNameMap[id]
			pointsValue = append(pointsValue, pValue)
		}
		return pointsValue
	}

	// 清理函数：删除已注册的点位
	unRegister := func() {
		m.allPointsIDMutex.Lock()
		defer m.allPointsIDMutex.Unlock()

		for _, pointID := range registerIDs {
			pOIDUint64, _ := pOIDToUint64(pointID)
			m.pointsIDCount[pOIDUint64]--
			if m.pointsIDCount[pOIDUint64] == 0 {
				delete(m.allPointsID, pOIDUint64)
			}
		}
	}
	return getValue, unRegister, nil
}

func pOIDToUint64(pOIDStr string) (uint64, error) {
	var rtuID, deviceID, tableNo, gin uint64
	n, err := fmt.Sscanf(pOIDStr, "%d-%d-%d-%d", &rtuID, &deviceID, &tableNo, &gin)
	if err != nil || n != 4 {
		return 0, fmt.Errorf("invalid pOID format: %s", pOIDStr)
	}
	// 使用掩码截断超出范围的值
	rtuID &= 0xFFFF
	deviceID &= 0xFFFF
	tableNo &= 0xFF
	gin &= 0xFFFFFF
	return (rtuID << 48) | (deviceID << 32) | (tableNo << 24) | gin, nil
}

// ===================== C Enum Mappings (Go Constants) =====================
const (
	// ScadaControlSelect: Select phase for remote control/adjustment
	ScadaControlSelect = C.SCADA_CONTROL_SELECT
	// ScadaControlCancel: Cancel remote control/adjustment operation
	ScadaControlCancel = C.SCADA_CONTROL_CANCEL
	// ScadaControlExecute: Execute remote control/adjustment command
	ScadaControlExecute = C.SCADA_CONTROL_EXECUTE
	// ScadaControlPause: Pause remote control/adjustment operation
	ScadaControlPause = C.SCADA_CONTROL_PAUSE
)

const (
	// ScadaActionOn: Remote control - ON action
	ScadaActionOn = C.SCADA_ACTION_ON
	// ScadaActionOff: Remote control - OFF action
	ScadaActionOff = C.SCADA_ACTION_OFF
	// ScadaActionError: Remote control - Error state
	ScadaActionError = C.SCADA_ACTION_ERROR
	// ScadaActionStop: Remote control - Stop action
	ScadaActionStop = C.SCADA_ACTION_STOP
)

const (
	// ScadaSetAnalog: Manual set - Analog type
	ScadaSetAnalog = C.SCADA_SET_ANALOG
	// ScadaSetDigital: Manual set - Digital type
	ScadaSetDigital = C.SCADA_SET_DIGITAL
	// ScadaSetPulse: Manual set - Pulse type
	ScadaSetPulse = C.SCADA_SET_PULSE
)

// ===================== Core Interfaces (No Client Handle Required) =====================

// SendYtCmd sends remote adjustment (YT) command
// Parameters:
//
//	poid: Remote adjustment object ID
//	cmd: Control phase (ScadaControlSelect/ScadaControlExecute etc.)
//	value: Adjustment value (float)
//	isFastYt: 1 = fast adjustment, 0 = normal adjustment
//
// Returns: error (nil if success)
func SendYtCmd(pointID string, cmd C.scada_control_type_t, value float32, isFastYt int) error {
	// 1. Create client internally
	client, err := createScadaClient()
	if err != nil {
		return fmt.Errorf("failed to create SCADA client: %w", err)
	}
	// Ensure client is destroyed after command execution
	defer func() {
		if err := destroyScadaClient(client); err != nil {
			log.Errorf("failed to destroy SCADA client: %v", err)
		}
	}()

	// 2. Parameter validation
	if isFastYt != 0 && isFastYt != 1 {
		return fmt.Errorf("invalid fast adjustment flag: %d (only 0/1 allowed)", isFastYt)
	}

	// 3. Convert Go types to C types
	cPoid, err := pOIDToUint64(pointID) //C.uint64_t(poid)
	if err != nil {
		return fmt.Errorf("invalid point ID: %w", err)
	}
	cCmd := C.scada_control_type_t(cmd)
	cValue := C.float(value)
	cIsFastYt := C.int(isFastYt)

	// 4. Call C API
	cRet := C.scada_send_yt_cmd(
		C.scada_client_t(client),
		C.uint64_t(cPoid),
		cCmd,
		cValue,
		cIsFastYt,
	)

	// 5. Handle result
	if cRet != 0 {
		return fmt.Errorf("YT command failed (poid: %s(%d), error code: %d)", pointID, cPoid, cRet)
	}

	log.Infof("YT command sent successfully (poid: %s(%d))", pointID, cPoid)
	return nil
}

// SendYkCmd sends remote control (YK) command
// Parameters:
//
//	poid: Remote control object ID
//	cmd: Control phase (ScadaControlSelect/ScadaControlExecute etc.)
//	action: Control action (ScadaActionOn/ScadaActionOff etc.)
//
// Returns: error (nil if success)
func SendYkCmd(pointID string, cmd C.scada_control_type_t, action C.scada_action_type_t) error {
	// 1. Create client internally
	client, err := createScadaClient()
	if err != nil {
		return fmt.Errorf("failed to create SCADA client: %w", err)
	}
	// Ensure client destruction
	defer func() {
		if err := destroyScadaClient(client); err != nil {
			log.Errorf("Warning: failed to destroy SCADA client:%v", err)
		}
	}()

	// 2. Convert types
	cPoid, err := pOIDToUint64(pointID)
	if err != nil {
		return fmt.Errorf("invalid point ID: %w", err)
	}
	cCmd := C.scada_control_type_t(cmd)
	cAction := C.scada_action_type_t(action)

	// 3. Call C API
	cRet := C.scada_send_yk_cmd(
		C.scada_client_t(client),
		C.uint64_t(cPoid),
		cCmd,
		cAction,
	)

	// 4. Handle result
	if cRet != 0 {
		return fmt.Errorf("YK command failed (poid: %s(%d), error code: %d)", pointID, cPoid, cRet)
	}
	log.Infof("YK command sent successfully (poid: %s(%d))", pointID, cPoid)
	return nil
}

// SendManualSetCmd sends manual set command
// Parameters:
//
//	poid: Manual set object ID
//	setFlag: 0 = cancel manual set, 1 = enable manual set
//	value: Manual set value (double precision)
//	setType: Set type (ScadaSetAnalog/ScadaSetDigital etc.)
//
// Returns: error (nil if success)
func SendManualSetCmd(pointID string, setFlag uint8, value float64, setType C.scada_set_type_t) error {
	// 1. Create client internally
	client, err := createScadaClient()
	if err != nil {
		return fmt.Errorf("failed to create SCADA client: %w", err)
	}
	// Ensure client destruction
	defer func() {
		if err := destroyScadaClient(client); err != nil {
			log.Errorf("Warning: failed to destroy SCADA client: %v", err)
		}
	}()

	// 2. Parameter validation
	if setFlag != 0 && setFlag != 1 {
		return fmt.Errorf("invalid manual set flag: %d (only 0/1 allowed)", setFlag)
	}

	// 3. Convert types
	cPoid, err := pOIDToUint64(pointID)
	if err != nil {
		return fmt.Errorf("invalid point ID: %w", err)
	}
	cSetFlag := C.uint8_t(setFlag)
	cValue := C.double(value)
	cSetType := C.scada_set_type_t(setType)

	// 4. Call C API
	cRet := C.scada_send_manual_set_cmd(
		C.scada_client_t(client),
		C.uint64_t(cPoid),
		cSetFlag,
		cValue,
		cSetType,
	)

	// 5. Handle result
	if cRet != 0 {
		return fmt.Errorf("manual set command failed (poid: %s(%d), error code: %d)", pointID, cPoid, cRet)
	}

	log.Infof("Manual set command sent successfully (poid: %s(%d))", pointID, cPoid)
	return nil
}

// ===================== Internal Helper Functions (Hidden from External Callers) =====================

// createScadaClient creates a SCADA client handle
// clientName: Name of the client
// Returns: client handle (unsafe.Pointer) + error
func createScadaClient() (unsafe.Pointer, error) {
	clientName := genUniqueClientID()

	// Convert Go string to C string (need manual free)
	cClientName := C.CString(clientName)
	defer C.free(unsafe.Pointer(cClientName))

	// Create client via C API
	cClient := C.scada_client_create(cClientName)
	if cClient == nil {
		return nil, fmt.Errorf("client creation failed (name: %s)", clientName)
	}

	return unsafe.Pointer(cClient), nil
}

// destroyScadaClient destroys a SCADA client handle
// client: Client handle to destroy
// Returns: error (nil if success)
func destroyScadaClient(client unsafe.Pointer) error {
	if client == nil {
		return errors.New("client handle is nil - no need to destroy")
	}

	// Destroy client via C API
	C.scada_client_destroy(C.scada_client_t(client))
	return nil
}
