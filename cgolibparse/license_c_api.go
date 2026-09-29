package cgolibparse

/*
#cgo LDFLAGS: -L/root/arm/lib -llicense_c_api
#include <stdlib.h>       // for C.free
#include "license_c_api.h"
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

// lmMutex 串行化所有 license C 调用（header 未声明线程安全，防御性串行化）
var lmMutex sync.Mutex

// LicenseInfo 许可证信息（Go 版本）
type LicenseInfo struct {
	IsValid       bool   `json:"is_valid"`
	RemainingDays int    `json:"remaining_days"`
	SerialNumber  string `json:"serial_number"`
	LicenseKey    string `json:"license_key"`
	ExpireDate    string `json:"expire_date"`
}

func cStringOrEmpty(p *C.char) string {
	if p == nil {
		return ""
	}
	return C.GoString(p)
}

// freeLicenseInfo 释放 C 结构体内的字段（header 内存约定：必须逐个 lm_free）
func freeLicenseInfo(info C.LMLicenseInfo) {
	C.lm_free(info.serial_number)
	C.lm_free(info.license_key)
	C.lm_free(info.expire_date)
}

// GenerateMachineSerial 查询本机机器码
func GenerateMachineSerial() (string, error) {
	lmMutex.Lock()
	defer lmMutex.Unlock()

	cSerial := C.lm_generate_machine_serial()
	if cSerial == nil {
		return "", errors.New("generate machine serial failed")
	}
	defer C.lm_free(cSerial)
	return C.GoString(cSerial), nil
}

// ValidateLicense 校验注册码；有效/无效由 IsValid 区分。
// 按 header 契约：RemainingDays 与 ExpireDate 仅在 IsValid 时有效，无效时为 0 / 空串。
// lm_validate_license 按值返回结构体，无失败信号，故本函数不返回 error。
func ValidateLicense(key string) LicenseInfo {
	lmMutex.Lock()
	defer lmMutex.Unlock()

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	info := C.lm_validate_license(cKey)
	defer freeLicenseInfo(info)

	return LicenseInfo{
		IsValid:       info.is_valid == 1,
		RemainingDays: int(info.remaining_days),
		SerialNumber:  cStringOrEmpty(info.serial_number),
		LicenseKey:    cStringOrEmpty(info.license_key),
		ExpireDate:    cStringOrEmpty(info.expire_date),
	}
}

// SaveLicenseToFile 保存注册码到文件；path 传空串使用 C 库默认 "license.dat"
func SaveLicenseToFile(key, path string) error {
	lmMutex.Lock()
	defer lmMutex.Unlock()

	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	var cPath *C.char
	if path != "" {
		cPath = C.CString(path)
		defer C.free(unsafe.Pointer(cPath))
	}
	if C.lm_save_license_to_file(cKey, cPath) != 1 {
		return errors.New("save license to file failed")
	}
	return nil
}
