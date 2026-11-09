package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"webserver-arm/cgolibparse"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
)

// 注册码/机器码文件复用部署根环境变量 ARM_SF（与 db_config.xml / homepage.json / web.log 同一部署根，
// 目录由部署保证存在且 web 进程可写，码写进 $ARM_SF/etc/license/ 避免污染既有目录）。
// 注意：这里用 var 而非 const，因为路径依赖运行期环境变量。
var (
	// licenseFilePath 注册码保存路径；C 库默认相对 CWD 的 "license.dat" 不可用，必须显式绝对路径。
	licenseFilePath = filepath.Join(getLicenseBasePath(), "etc/license", "license.dat")
	// machineSerialFile 机器码持久化文件：首次生成后写盘，之后直接读文件，
	// 避免 C 库指纹采集（网卡/内存）变化导致前端展示不一致。
	machineSerialFile = filepath.Join(getLicenseBasePath(), "etc/license", "machine_serial")
)

// getLicenseBasePath 部署根目录。包级 var 初始化期即被调用（见上方 licenseFilePath），
// 故必须无副作用、不打印日志：log 包的 logger 初值为 nil，要等 main 里 InitLog() 后才非空，
// 此处若调用 log.Error 会 nil 指针 panic（log/log.go）。
// 未设置时返回空，路径退化为相对路径（与现有 getEnvParam 惯例一致，运行期不追加日志）。
func getLicenseBasePath() string {
	return os.Getenv("ARM_SF")
}

// ensureLicenseDir 写文件前确保父目录存在（与 homepage/config.go save() 的 MkdirAll 兜底同款）
func ensureLicenseDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

// MachineSerialResp 机器码响应
type MachineSerialResp struct {
	SerialNumber string `json:"serial_number"`
}

// RegisterLicenseReq 提交注册码请求
type RegisterLicenseReq struct {
	LicenseKey string `json:"license_key" binding:"required"`
}

// LicenseStatusResp 许可证状态响应
type LicenseStatusResp struct {
	IsValid       bool   `json:"is_valid"`
	IsExpired     bool   `json:"is_expired"`
	RemainingDays int    `json:"remaining_days"`
	ExpireDate    string `json:"expire_date"`
	LicenseKey    string `json:"license_key"`
}

// readMachineSerial returns the persisted machine serial.
// An empty file is treated as missing so the caller can regenerate it.
func readMachineSerial() (string, error) {
	data, err := os.ReadFile(machineSerialFile)
	if err != nil {
		return "", err
	}
	serial := strings.TrimSpace(string(data))
	if serial == "" {
		return "", os.ErrNotExist
	}
	return serial, nil
}

// resolveSystemSerial keeps the system information API aligned with the
// machine serial used for license registration, while retaining the stored
// value as a fallback for deployments without a machine serial file.
func resolveSystemSerial(storedSerial string) string {
	serial, err := readMachineSerial()
	if err != nil || serial == "" {
		return storedSerial
	}
	return serial
}

func persistMachineSerial(serial string) error {
	if err := ensureLicenseDir(machineSerialFile); err != nil {
		return err
	}
	return os.WriteFile(machineSerialFile, []byte(serial), 0o644)
}

func loadOrCreateMachineSerial() (string, error) {
	if serial, err := readMachineSerial(); err == nil {
		return serial, nil
	}

	// Remove empty or unreadable remnants before generating a replacement.
	if err := os.Remove(machineSerialFile); err != nil && !os.IsNotExist(err) {
		log.Warn("remove machine serial before generate failed:", err)
	}

	serial, err := cgolibparse.GenerateMachineSerial()
	if err != nil {
		return "", err
	}

	if err := persistMachineSerial(serial); err != nil {
		log.Warn("persist machine serial failed:", err)
	}
	return serial, nil
}

// @Summary		查询机器码
// @Description	返回本机序列号，用于向厂商申请注册码
// @Tags			license
// @Produce		json
// @Success		200	{object}	MachineSerialResp
// @Router			/api/v1/log/license/machine_serial [get]
func GetMachineSerial(c *gin.Context) {
	serial, err := loadOrCreateMachineSerial()
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, MachineSerialResp{SerialNumber: serial})
}

// refreshMachineSerial 删除并重建机器码文件。先生成后删除，生成失败时旧文件不受影响。
func refreshMachineSerial(serial string) error {
	if err := os.Remove(machineSerialFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return persistMachineSerial(serial)
}

// @Summary		提交注册码
// @Description	校验并保存注册码到本地文件；有效期按提交的注册码生效，同时重建机器码文件
// @Tags			license
// @Accept			json
// @Produce		json
// @Param			RegisterLicenseReq	body	RegisterLicenseReq	true	"注册码"
// @Success		200	{object}	LicenseStatusResp
// @Failure		400	{object}	models.CommonResp	"注册码无效"
// @Router			/api/v1/log/license/register [post]
func RegisterLicense(c *gin.Context) {
	req := &RegisterLicenseReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest,
			models.CommonResp{Code: http.StatusBadRequest, Message: "license_key required"})
		return
	}

	info := cgolibparse.ValidateLicense(req.LicenseKey)
	// is_valid 已含"已过期视为无效"，签名错误/机码不匹配也返回 invalid
	if !info.IsValid {
		c.JSON(http.StatusBadRequest,
			models.CommonResp{Code: http.StatusBadRequest, Message: "invalid license key"})
		return
	}
	// 先确保目录存在（调用 C 库保存文件前必须保证目标目录存在，目录缺失会导致保存失败），再写入
	if err := ensureLicenseDir(licenseFilePath); err != nil {
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if err := cgolibparse.SaveLicenseToFile(req.LicenseKey, licenseFilePath); err != nil {
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// 有效期随新码落盘即生效；随后删除并重建机器码文件，使其与当前硬件一致。
	// 重建失败不影响注册结果，仅记录日志（前端不感知）。
	if serial, err := cgolibparse.GenerateMachineSerial(); err != nil {
		log.Warn("refresh machine serial: generate failed:", err)
	} else if serial == "" {
		log.Warn("refresh machine serial: generated serial is empty")
	} else if err := refreshMachineSerial(serial); err != nil {
		log.Warn("refresh machine serial: persist failed:", err)
	}
	c.JSON(http.StatusOK, LicenseStatusResp{
		IsValid:       true,
		IsExpired:     false,
		RemainingDays: info.RemainingDays,
		ExpireDate:    info.ExpireDate,
		LicenseKey:    info.LicenseKey,
	})
}

// GetLicenseStatusData 计算许可证状态（/status 接口与登录接口共用）。
// 语义：读文件原文 + lm_validate_license 实时校验三态（有效/无效（含过期）/未注册）。
func GetLicenseStatusData() LicenseStatusResp {
	// 1. 读注册码文件原文，不依赖 C 库进程内状态（跨重启一致）
	data, err := os.ReadFile(licenseFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn("read license file failed:", err)
		}
		// 文件不存在 / 不可读 → 未注册（不是"过期"）
		return LicenseStatusResp{IsValid: false, IsExpired: false}
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return LicenseStatusResp{IsValid: false, IsExpired: false}
	}

	// 2. 实时校验。header 契约：is_valid=1 有效；is_valid=0 无效（含已过期/签名错误/机码不匹配），
	//    RemainingDays/ExpireDate 仅在有效时返回；「已过期」与「无效」C API 无法区分。
	info := cgolibparse.ValidateLicense(key)
	if info.IsValid {
		return LicenseStatusResp{
			IsValid:       true,
			IsExpired:     false,
			RemainingDays: info.RemainingDays,
			ExpireDate:    info.ExpireDate,
			LicenseKey:    info.LicenseKey,
		}
	}
	// 文件存在但校验失败 → 无效（含已过期）
	return LicenseStatusResp{IsValid: false, IsExpired: true}
}

// @Summary		查询许可证状态
// @Description	登录后校验注册码是否有效（header 契约：有效 / 无效（含已过期））
// @Tags			license
// @Produce		json
// @Success		200	{object}	LicenseStatusResp
// @Router			/api/v1/log/license/status [get]
func GetLicenseStatus(c *gin.Context) {
	c.JSON(http.StatusOK, GetLicenseStatusData())
}
