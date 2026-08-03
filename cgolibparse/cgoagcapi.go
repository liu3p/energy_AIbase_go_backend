package cgolibparse

/*
#cgo CFLAGS: -I/root/arm/src/agc/include -I/usr/local/include/Poco
#cgo LDFLAGS: -L/root/arm/lib -lagc_base
#include <stdlib.h>       // for C.free
#include "agc_c_api.h"
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"unsafe"
	"webserver-arm/log"
)

// GetDeviceChildren 获取指定设备类型的子设备类型列表
// 功能：调用AGC底层C接口，获取某类设备可挂载的子设备类型，返回字符串数组
// 参数：
// deviceType - 设备类型（如STATION/INLINE/TRANSFORMER/PCS/BMS等）
//
// 返回值：
// []string - 子设备类型列表
// error    - 调用C接口失败/JSON解析失败时返回错误
func GetDeviceChildren(deviceType string) ([]string, error) {
	// 将Go字符串转为C语言字符串，用于C接口传参
	var cType *C.char
	if deviceType != "" {
		cType = C.CString(deviceType)
		defer C.free(unsafe.Pointer(cType))
	}

	// 调用AGC C接口：获取子设备类型的JSON字符串
	cJSON := C.agc_get_device_children_json(cType)
	if cJSON == nil {
		return nil, fmt.Errorf("GetDeviceChildren: agc_get_device_children_json returned nil")
	}
	defer C.agc_free_string(cJSON)

	// 将C语言字符串转为Go字符串
	jsonStr := C.GoString(cJSON)

	// 解析JSON字符串为Go字符串切片
	var children []string
	if err := json.Unmarshal([]byte(jsonStr), &children); err != nil {
		return nil, fmt.Errorf("GetDeviceChildren: unmarshal children failed: %w, raw=%s", err, jsonStr)
	}

	return children, nil
}

// GetDeviceStaticParas 获取所有设备的静态参数
// 功能：调用AGC底层C接口，获取所有设备的静态参数，返回设备-参数映射关系
//
// 返回值：
// map[string]string - 设备-参数映射关系
// error    		  - 调用C接口失败/JSON解析失败时返回错误
func GetDeviceStaticParas() (parasMap map[string]string, err error) {
	parasMap = make(map[string]string)

	// 获取所有设备类型
	deviceTypes, err := GetDeviceChildren("")
	log.Info("all deviceTypes: ", deviceTypes)
	if err != nil {
		return nil, err
	}

	// 遍历所有设备类型获取静态参数
	for _, deviceType := range deviceTypes {
		cType := C.CString(deviceType)

		// 调用AGC C接口：获取当前设备的静态参数
		cJSON := C.agc_get_static_dev_para_json_by_type(cType)
		C.free(unsafe.Pointer(cType))

		if cJSON == nil {
			return nil, fmt.Errorf("GetDeviceStaticParas: agc_get_static_dev_para_json_by_type returned nil")
		}

		// 将C语言字符串转为Go字符串
		jsonStr := C.GoString(cJSON)
		C.agc_free_string(cJSON)

		parasMap[deviceType] = jsonStr
	}

	return parasMap, nil
}

// -------------------------- 第一步：定义与JSON对应的Go结构体（严格映射JSON格式） --------------------------
// StrategyLibInfo 对应C接口返回的完整JSON结构
type StrategyLibInfo struct {
	Name     string        `json:"name"` // 策略名称
	Desc     string        `json:"desc,omitempty"`
	Priority int           `json:"priority"`
	Setting  []SettingItem `json:"setting"` // 配置参数列表
	DynPara  []string      `json:"dynpara"`
	CalcPara []SettingItem `json:calcpara`
}

//  -lagc_base -lPocoJSON -lPocoNet -lPocoUtil  -lPocoData -lPocoFoundation -lstdc++ -lpthread

// SettingItem 对应setting数组中的单个元素（包含value字段）
type SettingItem struct {
	Name  string `json:"name"`  // 参数名
	Max   string `json:"max"`   // 最大值（允许为空字符串）
	Min   string `json:"min"`   // 最小值（允许为空字符串）
	Type  string `json:"type"`  // 参数类型（INT/FLOAT）
	Value string `json:"value"` // 参数值
	Desc  string `json:"desc"`
	Path  string `json:"-"` // 用于在点表配置中配置agc设备，默认不参与初始化
}

// -------------------------- 第二步：cgo调用C接口，获取JSON并转换为Go结构体 --------------------------
// GetStrategyLibInfo 封装cgo调用，接收库文件路径，返回解析后的Go结构体
// 参数：libPath - Go字符串，对应C接口的char* libpath（文件路径）
// 返回：
//   - *StrategyLibInfo：解析后的策略信息结构体指针
//   - error：调用失败或解析失败的错误信息
func GetStrategyLibInfo(libPath string) (*StrategyLibInfo, error) {
	// 1. 前置校验：入参不能为空
	if libPath == "" {
		return nil, errors.New("libpath can not be empty")
	}

	// 2. Go字符串 → C字符串（C.CString分配C堆内存，后续必须用C.free释放）
	cLibPath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cLibPath))

	// 3. 调用C接口，获取返回的JSON C字符串（const char*）
	cJsonStr := C.agc_get_strategy_lib_info(cLibPath)
	// 校验C接口返回结果：nil表示调用失败
	if cJsonStr == nil {
		return nil, errors.New("C-func agc_get_strategy_lib_info call failed")
	}
	defer C.agc_free_string(cJsonStr)

	// 4. C字符串 → Go字符串（C.GoString会拷贝数据到Go堆，安全无依赖）
	jsonStr := C.GoString(cJsonStr)
	if jsonStr == "" {
		return nil, errors.New("C-func getStrategyLibInfo return empty json")
	}

	// 5. JSON字符串 → Go结构体（反序列化）
	var strategyInfo StrategyLibInfo
	err := json.Unmarshal([]byte(jsonStr), &strategyInfo)
	if err != nil {
		return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
	}

	// 6. 给参数Path字段赋值
	prefix := "/strategy/" + strategyInfo.Name + "/"
	for i := range strategyInfo.Setting {
		strategyInfo.Setting[i].Path = prefix + strategyInfo.Setting[i].Type + "/" + strategyInfo.Setting[i].Name
	}
	for i := range strategyInfo.CalcPara {
		strategyInfo.CalcPara[i].Path = prefix + strategyInfo.CalcPara[i].Type + "/" + strategyInfo.CalcPara[i].Name
	}

	// 7. 返回解析结果
	return &strategyInfo, nil
}
