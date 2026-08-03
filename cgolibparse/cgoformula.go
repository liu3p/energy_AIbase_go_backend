package cgolibparse

/*
#cgo LDFLAGS: -lequation
#include <stdlib.h>       // for C.free
#include "equation_c.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// export LD_LIBRARY_PATH=/root/arm/lib/scada:/root/zxq/webserver_arm/cgolibparse:$LD_LIBRARY_PATH
// build: CGO_LDFLAGS="-L${ARM_SF}/lib/scada -L${ARM_SF}/lib" go build

// --------------- 封装 eq_validate 函数 ---------------
// EqValidate 验证公式是否合法（自动分解变量名）
// 参数：expr - 待验证的公式字符串
// 返回：
//
//	error - 空表示合法，非空表示不合法（包含错误信息）
func EqValidate(expr string) error {
	if expr == "" {
		return errors.New("formula string can not be empty")
	}

	// 1. Go string 转 C char*
	cExpr := C.CString(expr)
	defer C.free(unsafe.Pointer(cExpr)) // 释放C字符串内存

	// 2. 定义C错误信息指针（由C库分配，需释放）
	var cErrMsg *C.char

	// 3. 调用C函数
	ret := C.eq_validate(cExpr, &cErrMsg)

	// 4. 处理错误信息（必须释放，避免内存泄漏）
	defer func() {
		if cErrMsg != nil {
			C.eq_free_string(cErrMsg) // 调用C库的释放函数
		}
	}()

	// 5. 解析返回结果
	if ret != 0 {
		errMsg := C.GoString(cErrMsg) // C char* 转 Go string
		return errors.New(errMsg)
	}

	return nil
}

// --------------- 封装 eq_validate_with_vars 函数 ---------------
// EqValidateWithVars 使用指定变量名验证公式是否合法
// 参数：
//
//	expr - 待验证的公式字符串
//	varNames - 变量名数组
//
// 返回：
//
//	error - 空表示合法，非空表示不合法（包含错误信息）
func EqValidateWithVars(expr string, varNames []string) error {
	if expr == "" {
		return errors.New("formula string can not be empty")
	}
	if len(varNames) == 0 {
		return errors.New("varNames can not be empty")
	}

	// 1. 转换公式字符串
	cExpr := C.CString(expr)
	defer C.free(unsafe.Pointer(cExpr))

	// 2. 转换变量名数组：Go []string → C const char**
	// 步骤2.1：分配C字符指针数组（长度=varCount）
	varCount := C.int(len(varNames))
	cVarNames := (**C.char)(C.malloc(C.size_t(varCount) * C.size_t(unsafe.Sizeof(uintptr(0)))))
	defer C.free(unsafe.Pointer(cVarNames)) // 释放数组内存

	// 步骤2.2：将Go字符串转为C字符串，存入数组
	cVarNamesSlice := unsafe.Slice(cVarNames, len(varNames))
	for i, name := range varNames {
		cName := C.CString(name)
		defer C.free(unsafe.Pointer(cName)) // 释放单个变量名的C字符串
		cVarNamesSlice[i] = cName
	}

	// 3. 定义错误信息指针
	var cErrMsg *C.char
	defer func() {
		if cErrMsg != nil {
			C.eq_free_string(cErrMsg)
		}
	}()

	// 4. 调用C函数
	ret := C.eq_validate_with_vars(cExpr, (**C.char)(unsafe.Pointer(cVarNames)), varCount, &cErrMsg)

	// 5. 解析结果
	if ret != 0 {
		errMsg := C.GoString(cErrMsg)
		return errors.New(errMsg)
	}

	return nil
}

// --------------- 封装 eq_eval 函数 ---------------
// EqEval 计算公式的值
// 参数：
//
//	expr - 待计算的公式字符串
//	varNames - 变量名数组
//	varValues - 变量值数组（需与varNames长度一致）
//
// 返回：
//
//	float64 - 计算结果
//	error - 空表示成功，非空表示失败（包含错误信息）
func EqEval(expr string, varNames []string, varValues []float64) (float64, error) {
	// 参数校验
	if expr == "" {
		return 0, errors.New("formula string can not be empty")
	}
	if len(varNames) == 0 || len(varValues) == 0 {
		return 0, errors.New("varNnames or values can not be empty")
	}
	if len(varNames) != len(varValues) {
		return 0, errors.New("varNames and values must have same length")
	}

	// 1. 转换公式字符串
	cExpr := C.CString(expr)
	defer C.free(unsafe.Pointer(cExpr))

	// 2. 转换变量名数组（同 EqValidateWithVars）
	varCount := C.int(len(varNames))
	cVarNames := (**C.char)(C.malloc(C.size_t(varCount) * C.size_t(unsafe.Sizeof(uintptr(0)))))
	defer C.free(unsafe.Pointer(cVarNames))

	cVarNamesSlice := unsafe.Slice(cVarNames, len(varNames))
	for i, name := range varNames {
		cName := C.CString(name)
		defer C.free(unsafe.Pointer(cName))
		cVarNamesSlice[i] = cName
	}

	// 3. 转换变量值数组：Go []float64 → C double*
	// 注意：Go float64 与 C double 内存布局完全一致，可直接转换
	var cVarValues *C.double
	if len(varValues) > 0 {
		cVarValues = (*C.double)(unsafe.Pointer(&varValues[0]))
	}

	// 4. 定义计算结果和错误信息
	var cResult C.double
	var cErrMsg *C.char
	defer func() {
		if cErrMsg != nil {
			C.eq_free_string(cErrMsg)
		}
	}()

	// 5. 调用C函数
	ret := C.eq_eval(cExpr, (**C.char)(unsafe.Pointer(cVarNames)), cVarValues, varCount, &cResult, &cErrMsg)

	// 6. 解析结果
	if ret != 0 {
		errMsg := C.GoString(cErrMsg)
		return 0, errors.New(errMsg)
	}

	return float64(cResult), nil
}

// --------------- 封装 eq_free_string 函数（备用）---------------
// EqFreeString 释放C库分配的字符串（一般无需外部调用，封装内已自动释放）
func EqFreeString(s string) {
	cStr := C.CString(s)
	defer C.free(unsafe.Pointer(cStr))
	C.eq_free_string(cStr)
}
