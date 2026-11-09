package handlers

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"webserver-arm/emsdb"
	"webserver-arm/emsxmldb"
	"webserver-arm/log"
	"webserver-arm/middleware"
	"webserver-arm/models"
	"webserver-arm/systeminfo"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

const (
	SnOperGet = 1
	SnOperSet = 2

	NetworkInterfaceDev = "enp4s0"
)

var timezoneMap = map[string]string{
	"UTC-12:00": "Antarctica/McMurdo",
	"UTC-11:00": "Pacific/Midway",
	"UTC-10:00": "Pacific/Honolulu",
	"UTC-09:30": "Pacific/Marquesas",
	"UTC-09:00": "Pacific/Gambier",
	"UTC-08:00": "America/Los_Angeles",
	"UTC-07:00": "America/Denver",
	"UTC-06:00": "America/Chicago",
	"UTC-05:00": "America/New_York",
	"UTC-04:00": "America/Caracas",
	"UTC-03:30": "America/St_Johns",
	"UTC-03:00": "America/Sao_Paulo",
	"UTC-02:00": "America/Noronha",
	"UTC-01:00": "Atlantic/Azores",
	"UTC+00:00": "Europe/London",
	"UTC+01:00": "Europe/Berlin",
	"UTC+02:00": "Europe/Helsinki",
	"UTC+03:00": "Europe/Moscow",
	"UTC+03:30": "Asia/Tehran",
	"UTC+04:00": "Asia/Dubai",
	"UTC+04:30": "Asia/Kabul",
	"UTC+05:00": "Asia/Karachi",
	"UTC+05:30": "Asia/Kolkata",
	"UTC+05:45": "Asia/Kathmandu",
	"UTC+06:00": "Asia/Dhaka",
	"UTC+06:30": "Asia/Yangon",
	"UTC+07:00": "Asia/Bangkok",
	"UTC+08:00": "Asia/Shanghai",
	"UTC+08:45": "Australia/Eucla",
	"UTC+09:00": "Asia/Tokyo",
	"UTC+09:30": "Australia/Darwin",
	"UTC+10:00": "Australia/Sydney",
	"UTC+10:30": "Australia/Lord_Howe",
	"UTC+11:00": "Pacific/Noumea",
	"UTC+12:00": "Pacific/Fiji",
	"UTC+12:45": "Pacific/Chatham",
	"UTC+13:00": "Pacific/Tongatapu",
	"UTC+14:00": "Pacific/Kiritimati",
}

var SystemConfigData SystemConfig

func SelectGroupId(groupName string) int32 {
	// log.Info("Select group ID :", groupName)
	// var ret int32
	// ret = -1
	// retModbus, err := db.NewMsDB()
	// if err != nil {
	// 	log.Info("SelectGroupId open db err:", err)
	// 	return ret
	// }

	// var modbusTransrv []db.ModbusTransrv
	// tx := retModbus.Where("组名 = ?", groupName).Find(&modbusTransrv)
	// if tx.Error != nil {
	// 	log.Error("SelectGroupId Find err:", tx.Error)
	// 	return ret
	// }
	// if len(modbusTransrv) == 0 {
	// 	log.Error("SelectGroupId from ModbusZHUAN-FA-ZU-BIAO Find groupName empty:", groupName)
	// 	return ret
	// }
	// ret = modbusTransrv[0].GroupId
	// log.Info("SelectGroupId groupId:", ret)
	// return ret

	return 0
}

func GetSysmanConfigPath() string {
	var path string

	if masenv := os.Getenv("MASENV"); masenv != "" {
		path = filepath.Join(masenv, "ini/man.conf")
	} else if ecoems := os.Getenv("ECOEMS"); ecoems != "" {
		path = filepath.Join(ecoems, "ini/man.conf")
	} else {
		return ""
	}

	return path
}
func updateSysmanConfigFile(updates map[string]string) error {
	filename := GetSysmanConfigPath()
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("updateSysmanConfigFile OpenFile failed:%v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// replace exist key=value
			if newValue, ok := updates[key]; ok {
				value = newValue
			}
			line = fmt.Sprintf("%s=%s", key, value)
			delete(updates, key)
		}
		lines = append(lines, line)
	}

	// add new key=value
	for key, value := range updates {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// clear file and write
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("updateSysmanConfigFile Truncate file failed:%v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("updateSysmanConfigFile Seek file failed:%v", err)
	}
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("updateSysmanConfigFile WriteString failed:%v", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("updateSysmanConfigFile Flush file failed:%v", err)
	}

	return nil
}

/*
// @Summary		配置项目sn
// @Description	配置项目sn
// @ID				set-sn
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			SnData			body		emsclient.SnData	true	"配置项目sn请求结构 {\"addr\":\"某地\",\"cabnum\":1,\"name\":\"测试项目\",\"sn\":\"98798798\"}"
// @Success		200				{object}	models.CommonResp	"配置项目sn成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/project/sn [post]
*/

func SetProjectSn(c *gin.Context) {
	log.Info("SetProjectSn in...")
	sndata := &emsdb.SnData{}
	err := c.ShouldBindJSON(&sndata)
	if err != nil {
		log.Error("ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	log.Info("set project:", sndata)

	userType := c.GetString(middleware.UTypeKey)
	if userType != UserTypeConfig {
		log.Error("ResetPwd unaccessable for user:", c.GetString(middleware.UNameKey), ", should be configure")
		c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "Configure access required"})
		return
	}

	err = emsdb.UpdateProjectSn(emsdb.ProjectSn{Sn: sndata.Sn, ProjName: sndata.Name, ProjAddr: sndata.Addr})
	if err != nil {
		log.Error("UpdateProjectSn failed:", err.Error())
		c.JSON(http.StatusServiceUnavailable, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Server Error"})
		return
	}

	upVkey := sndata.Sn + "esscab1"
	log.Info("set vkey:", upVkey)
	updateVkey := map[string]string{
		"vkey": upVkey,
	}
	err = updateSysmanConfigFile(updateVkey)
	if err != nil {
		log.Error("updateSysmanConfigFile failed:", err.Error())
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: 0, Message: ""})
}

// @Summary		查询项目sn
// @Description	查询项目sn
// @ID				get-sn
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	emsclient.SnData	"查询成功 {\"addr\":\"某地\",\"cabnum\":1,\"name\":\"测试项目\",\"sn\":\"98798798\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/project/sn [get]

func GetProjectSn(c *gin.Context) {
	log.Info("GetProjectSn in...")
	dbProjInfo, err := emsdb.GetProjectSn()
	if err != nil {
		log.Error("GetProjectSn err: ", err.Error())
		c.JSON(http.StatusServiceUnavailable, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Server Error"})
		return
	}
	projInfo := &emsdb.SnData{
		Sn:   dbProjInfo.Sn,
		Name: dbProjInfo.ProjName,
		Addr: dbProjInfo.ProjAddr,
	}

	c.JSON(http.StatusOK, projInfo)
}

type ProcessOper struct {
	Operation string `json:"op"`       // start,stop,restart
	CheckPwd  string `json:"checkpwd"` // 校验密码
}

// @Summary	 	操作指定的本地线程
// @Description	操作指定的本地线程
// @ID				oper-process
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			procid			path		string				true	"请求的进程id"
// @Param			ProcessOper		body		ProcessOper			true	"线程操作参数 op：start/stop/restart"
// @Success		200				{object}	models.CommonResp	"操作成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"进程id不存在 {\"code\": 404, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/process/{procid} [post]
func OperProcess(c *gin.Context) {
	log.Info("OperProcess in...")
	processID := c.Param("procid")
	op := &ProcessOper{}
	err := c.ShouldBindJSON(&op)
	if err != nil {
		log.Error("ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	uName := c.GetString(middleware.UNameKey)
	if data, ok := OnlineUsers.Load(uName); ok {
		online, match := data.(*OnlineUser)
		if !match || op.CheckPwd != online.Pwd {
			log.Error("pwd check failed")
			c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "pwd invalid"})
			return
		}
	}

	err = systeminfo.OperateProcess(op.Operation, processID)
	if err != nil {
		log.Error("OperProcess err:", err.Error())
		c.JSON(http.StatusServiceUnavailable, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Server Error"})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: 0, Message: ""})
}

//nolint:govet
type ProcessInfo struct {
	Name   string `json:"name"`
	ProcID string `json:"pid"`
	Status int    `json:"status"`
	Desc   string `json:"describe"`
	Cpu    string `json:"cpu"`
	Mem    string `json:"mem"`
}

func QueryLocalProcesses(c *gin.Context) {
	log.Info("QueryLocalProcesses in...")
	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
		Message: "Internal Service Error"})
	// procs, err := scdnet.GetSysProc()
	// if err != nil {
	// 	log.Error("QueryLocalProcesses GetSysProc failed:", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Service Error"})
	// 	return
	// }

	// procInfos := make([]ProcessInfo, 0)

	// for _, proc := range procs {
	// 	if proc.Name == "massrv" {
	// 		// massrv maybe should be restart by "systemctl restart massrv"
	// 		continue
	// 	}
	// 	var cpuLoad string
	// 	if proc.Pid != "" {
	// 		pid, err := strconv.Atoi(proc.Pid)
	// 		if err != nil {
	// 			log.Error("QueryLocalProcesses parse pid err:", proc.Pid)
	// 			continue
	// 		}
	// 		p, err := process.NewProcess(int32(pid))
	// 		if err != nil {
	// 			log.Error("QueryLocalProcesses getting process failed:", err)
	// 			continue
	// 		}

	// 		cpuUsage, err := p.CPUPercent()
	// 		if err != nil {
	// 			log.Error("QueryLocalProcesses getting CPU usage failed:", err)
	// 			continue
	// 		}
	// 		cpuLoad = fmt.Sprintf("%.2f", cpuUsage)
	// 	}

	// 	procInfo := ProcessInfo{
	// 		Name:   proc.Name,
	// 		ProcID: proc.Pid,
	// 		Status: proc.State,
	// 		Desc:   proc.Desc,
	// 		Cpu:    cpuLoad,
	// 		Mem:    proc.Memusage,
	// 	}

	// 	procInfos = append(procInfos, procInfo)
	// }

	// log.Info("QueryLocalProcesses:", procInfos)
	// c.JSON(http.StatusOK, procInfos)
	//c.jSON(http.StatusOK, []ProcessInfo{})
}

func HandleLocalProcesses(c *gin.Context) {
	log.Info("HandleProcesses in...")
	// procOper := &ProcessOper{}
	// err := c.ShouldBindJSON(&procOper)
	// if err != nil {
	// 	log.Error("HandleProcesses ShouldBindJSON err:", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Service Error"})
	// 	return
	// }
	// procid := c.Param("procid")

	// uName := c.GetString(middleware.UNameKey)
	// if data, ok := OnlineUsers.Load(uName); ok {
	// 	online, match := data.(*OnlineUser)
	// 	if !match || procOper.CheckPwd != online.Pwd {
	// 		log.Warn("HandleLocalProcesses pwd not match, rejected. pwd len:", len(procOper.CheckPwd))
	// 		c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "pwd invalid"})
	// 		return
	// 	}
	// }

	// log.Debug("procOper:", procOper, ", procid:", procid)

	// if procOper.Operation != 1 {
	// 	log.Warn("HandleLocalProcesses operation invalid, operation:", procOper.Operation)
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "operation invalid"})
	// 	return
	// }

	// procs, err := scdnet.GetSysProc()
	// if err != nil {
	// 	log.Error("HandleProcesses GetSysProc err:", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Service Error"})
	// 	return
	// }

	// procName := ""
	// for _, proc := range procs {
	// 	if proc.Pid == procid {
	// 		procName = proc.Name
	// 	}
	// }

	// if procName == "" {
	// 	log.Error("HandleProcesses procName not exist:", procName)
	// 	c.JSON(http.StatusNotFound, models.CommonResp{Code: http.StatusNotFound,
	// 		Message: "Status Not Found"})
	// 	return
	// }

	// err = scdnet.KillAProc(procName)
	// if err != nil {
	// 	log.Error("HandleProcesses KillAProc err:", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	// 		Message: "Internal Service Error"})
	// 	return
	// }

	c.JSON(http.StatusOK, models.CommonResp{Code: 0, Message: ""})
}

// type QueryI18nResp struct {
// 	Version string                       `json:"version"`
// 	Content map[string]map[string]string `json:"content"`
// }

var I18nData = &models.QueryI18nResp{}

// @Summary		查询国际化数据
// @Description	查询国际化数据
// @ID				get-i8n
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string					true	"JWT Token"	default(<token>)
// @Param			current_version	query		string					false	"客户端当前的版本号"
// @Success		200				{object}	models.QueryI18nResp	"查询成功 {\"version\":\"1\",\"content\":{\"zh\":{\"Uab\":\"aaa\",\"Uac\":\"bbb\"},\"en\":{\"Uab\":\"ccc\",\"Uac\":\"ddd\"}}}"
// @Success		208				{object}	models.CommonResp		"版本无更新 {\"code\": 204, \"msg\": \"NoContent\"}"
// @Failure		401				{object}	models.CommonResp		"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp		"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp		"服务内部错误 {\"code\": 500, \"msg\": \"InternalServerError\"}"
// @Router			/api/v1/system/i18n [get]

func GetI18N(c *gin.Context) {
	log.Info("GetI18N in...")
	currentVersion := c.Query("current_version")

	// file, err := os.Open("data/clientlan.txt")
	// if err != nil {
	// 	log.Error("open data/clientlan.txt failed. err:", err.Error())
	// 	return
	// }
	// defer file.Close()

	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	log.Error("read data/clientlan.txt failed. err:", err.Error())
	// 	return
	// }

	// resp := &QueryI18nResp{}
	// err = json.Unmarshal(data, resp)
	// if err != nil {
	// 	log.Error("read data/clientlan.txt failed. err:", err.Error())
	// 	return
	// }

	if I18nData.Content == nil {
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "Internal Service Error"})
		return
	}

	if I18nData.Version == currentVersion {
		c.JSON(http.StatusNoContent, models.CommonResp{Code: http.StatusNoContent, Message: "NoContent"})
		return
	}

	log.Debug("currentVersion:", currentVersion)

	c.JSON(http.StatusOK, I18nData)
}

var DataUnit = &models.QueryDataUnitResp{}

var DataUnitEx = &models.QueryDataUnit{}

// @Summary		查询单位数据
// @Description	查询单位数据
// @ID				get-dataunit
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			current_version	query		string						false	"客户端当前的版本号"
// @Success		200				{object}	models.QueryDataUnitResp	"查询成功 {"version":"1","content":{"PRated":"kW","QRated":"kVar"}}"
// @Success		208				{object}	models.CommonResp			"版本无更新 {\"code\": 204, \"msg\": \"NoContent\"}"
// @Failure		401				{object}	models.CommonResp			"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp			"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp			"服务内部错误 {\"code\": 500, \"msg\": \"InternalServerError\"}"
// @Router			/api/v1/system/dataunit [get]

func GetDataUnit(c *gin.Context) {
	log.Info("GetDataUnit in...")
	currentVersion := c.Query("current_version")
	if DataUnit.Content == nil {
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "Internal Service Error"})
		return
	}

	if DataUnit.Version == currentVersion {
		c.JSON(http.StatusNoContent, models.CommonResp{Code: 204, Message: "NoContent"})
		return
	}

	log.Debug("currentVersion:", currentVersion)

	c.JSON(http.StatusOK, DataUnit)
}

type TransEntry map[string]map[string]string // "modbus-key": "trans"
var I18nDataModbus = make(map[string]TransEntry)

// @Summary		查询modbus国际化数据
// @Description	查询modbus国际化数据
// @ID				get-i8n-for-modbus
// @Tags			system
// @Accept			json
// @Produce			json
// @Param			model	query		string					false	"查询产品型号"
// @Success		200				{object}	TransEntry	"查询成功 {\"zh\":{\"DCcn01bms#01102Bh\":\"告警状态\",\"DCcn01bms#01105Bh\":\"故障状态\"},\"en\":{\"DCcn01bms#01102Bh\":\"Alarm Status\",\"DCcn01bms#01105Bh\":\"Fault Status\"}}"
// @Failure		400				{object}	models.CommonResp		"认证失败 {\"code\": 400, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp		"认证失败 {\"code\": 404, \"msg\": \"model not found\"}"
// @Router			/api/v1/system/i18nmb [get]

func GetI18NModbus(c *gin.Context) {
	log.Info("GetI18N4Modbus in...")
	model := c.Query("model")
	if model == "" {
		log.Error("params model empty")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "params model invalid"})
		return
	}

	data, ok := I18nDataModbus[model]
	if !ok {
		log.Errorf("no model %s in I18nDataModbus, try DEFAULT", model)
		data, ok = I18nDataModbus["DEFAULT"]
		if !ok {
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "no model found"})
			return
		}
	}

	c.JSON(http.StatusOK, data)
}

type DataUnitEntry map[string]map[string]string

var DataUnitModbus = make(map[string]DataUnitEntry)

// @Summary		查询modbus单位数据
// @Description	查询modbus单位数据
// @ID				get-dataunit-for-modbus
// @Tags			system
// @Accept			json
// @Produce			json
// @Param			model	query		string						false	"查询产品型号"
// @Success		200				{object}	DataUnitEntry	"查询成功 {\"DCcn01bms#010Yc\":{\"unit\": \"V\"}, \"DCcn01bms#01108Yc\":{\"unit\": \"kWh\"}}"
// @Failure		400				{object}	models.CommonResp		"认证失败 {\"code\": 400, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp		"认证失败 {\"code\": 404, \"msg\": \"model not found\"}"
// @Router			/api/v1/system/dataunitmb [get]

func GetDataUnitModbus(c *gin.Context) {
	log.Info("GetDataUnit4Modbus in...")
	model := c.Query("model")
	if model == "" {
		log.Error("params model empty")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "params model invalid"})
		return
	}

	data, ok := DataUnitModbus[model]
	if !ok {
		log.Errorf("no model %s in DataUnitModbus, try DEFAULT", model)
		data, ok = DataUnitModbus["DEFAULT"]
		if !ok {
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "no model found"})
			return
		}
	}

	c.JSON(http.StatusOK, data)
}

//nolint:govet
type ZoneResp struct {
	// 服务器当前时间
	Now string `json:"now"`

	// 服务器当前时区偏移量
	Offset int `json:"offset"`

	// 服务器当前时区
	Zone string `json:"zone"`
}

func getTimezoneFromTimedatectl() (string, error) {
	cmd := exec.Command("timedatectl")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Time zone:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2], nil
			}
		}
	}
	return "", fmt.Errorf("could not find time zone in timedatectl output")
}

// @Summary		查询服务器时区
// @Description	查询服务器时区
// @ID				get-zoneinfo
// @Tags			system
// @Accept			json
// @Produce		json
// @Success		200	{object}	ZoneResp			"查询成功 {"now":"2024-01-02T14:22:31+08:00","offset":28800,"zone":"Asia/Shanghai"}"
// @Failure		401	{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500	{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"InternalServerError\"}"
// @Router			/api/v1/system/zone [get]

func GetZoneInfo(c *gin.Context) {
	log.Info("GetZoneInfo in...")

	zoneAddr, err := getTimezoneFromTimedatectl()
	if err != nil {
		log.Error("GetZoneInfo getTimezoneFromTimedatectl failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	loc, err := time.LoadLocation(zoneAddr)
	if err != nil {
		log.Error("GetZoneInfo LoadLocation failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	current := time.Now().In(loc)
	nowStr := current.Format(time.RFC3339)
	_, offset := current.Zone()

	// if zone == "CST" {
	// 	switch offset {
	// 	case -21600:
	// 		zone = "America/Chicago"
	// 	case 28800:
	// 		zone = "Asia/Shanghai"
	// 	case -14400:
	// 		zone = "America/Havana"
	// 	case 34200:
	// 		zone = "Australia/Adelaide"
	// 	}
	// 	log.Info("GetZoneInfo get zone: CST from system, offset:", offset)
	// }

	c.JSON(http.StatusOK, ZoneResp{Now: nowStr, Offset: offset, Zone: zoneAddr})
}

func getDefaultGw(dev string) (string, error) {
	output, err := exec.Command("ip", "route", "show", "dev", dev).Output()
	if err != nil {
		return "", fmt.Errorf("ip route show dev %s command run failed:%w", dev, err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "default via") {
			parts := strings.Fields(line)
			if len(parts) > 2 {
				gateway := parts[2]
				return gateway, nil
			}
		}
	}
	log.Warnf("Gateway not found. command result:%s", string(output))
	return "", nil
}

func getDNSConfig(dev string) (dns1, dns2 string, err error) {
	filePath := path.Join("/etc/sysconfig/network-scripts/", "ifcfg-"+dev)
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for DNS1 and DNS2
		if strings.HasPrefix(line, "DNS1=") {
			dns1 = strings.Trim(strings.TrimPrefix(line, "DNS1="), "\"")
		} else if strings.HasPrefix(line, "DNS2=") {
			dns2 = strings.Trim(strings.TrimPrefix(line, "DNS2="), "\"")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("failed to read file: %v", err)
	}

	return dns1, dns2, nil
}

func getLocalIP(dev string) (ip, mask string, err error) {
	intf, err := net.InterfaceByName(dev)
	if err != nil {
		return ip, mask, fmt.Errorf("get interface by name %s failed:%w", dev, err)
	}

	addrs, err := intf.Addrs()
	if err != nil {
		return ip, mask, fmt.Errorf("get addrs of dev %s failed:%w", dev, err)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.IP.To4() != nil {
				ip = ipNet.IP.String()
				mask = net.IP(ipNet.Mask).String()
				return ip, mask, nil
			} else {
				err = fmt.Errorf("get addrs of dev %s failed, addr is not ipv4", dev)
			}
		} else {
			err = fmt.Errorf("get addrs of dev %s failed, addr is not net.IPNet:%v", dev, addr)
		}
	}

	return ip, mask, err
}

func setLocalIP(dev, ip, mask, gateway, dns1, dns2 string) error {
	clearCmd := fmt.Sprintf(
		`sed -i '/^IPADDR=/d; /^NETMASK=/d; /^GATEWAY=/d; /^BOOTPROTO=/d; /^DNS1=/d; /^DNS2=/d ' /etc/sysconfig/network-scripts/ifcfg-%s`, dev)
	addCmd := fmt.Sprintf(
		`sed -i '$a\BOOTPROTO=static\nIPADDR=%s\nNETMASK=%s\nGATEWAY=%s\nDNS1=%s\nDNS2=%s' /etc/sysconfig/network-scripts/ifcfg-%s`,
		ip, mask, gateway, dns1, dns2, dev)
	// setCmd := fmt.Sprintf(
	// 	`sed -i 's/^IPADDR=.*/IPADDR=%s/g;
	// 			s/^NETMASK=.*/NETMASK=%s/g;
	// 			s/^GATEWAY=.*/GATEWAY=%s/g'
	// 	 /etc/sysconfig/network-scripts/ifcfg-%s`,
	// 	ip, mask, gateway, dev)
	restartCmd := "systemctl restart network"

	err := execCommand(clearCmd)
	if err != nil {
		return fmt.Errorf("run clearCmd '%s' failed:%w", clearCmd, err)
	}

	err = execCommand(addCmd)
	if err != nil {
		return fmt.Errorf("run addCmd '%s' failed:%w", addCmd, err)
	}

	err = execCommand(restartCmd)
	if err != nil {
		return fmt.Errorf("run restartCmd '%s' failed:%w", restartCmd, err)
	}
	return nil
}

func execCommand(cmd string) error {
	command := exec.Command("sh", "-c", cmd)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error: %w, output: %s", err, string(output))
	}
	return nil
}

type NetworkConfig struct {
	IP      string `json:"ip"`
	Mask    string `json:"mask"`
	Gateway string `json:"gateway"`
	Dns1    string `json:"dns1"`
	Dns2    string `json:"dns2"`
}

// @Summary		locaweb查询本地网络配置
// @Description	查询本地网络配置
// @ID				query-networkconfig
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	NetworkConfig		"成功 {\"ip\":\"192.168.2.27\", \"mask\":\"255.255.254.0\", \"gateway\":\"192.168.2.1\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/network/config [get]

func GetNetworkConfig(c *gin.Context) {
	log.Info("GetNetworkConfig in ...")

	ip, mask, err := getLocalIP(NetworkInterfaceDev)
	if err != nil {
		log.Error("getLocalIP failed:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}
	gw, err := getDefaultGw(NetworkInterfaceDev)
	if err != nil {
		log.Error("getDefaultGw failed:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}

	dns1, dns2, err := getDNSConfig(NetworkInterfaceDev)
	if err != nil {
		log.Error("getDNSConfig failed:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}

	c.JSON(http.StatusOK, NetworkConfig{IP: ip, Mask: mask, Gateway: gw, Dns1: dns1, Dns2: dns2})
}

func validateIP(ipStr string) bool {
	parsedIP := net.ParseIP(ipStr)
	return parsedIP != nil
}

func validateSubnetMask(mask string) bool {
	// 解析子网掩码为 IP 地址
	parsedMask := net.ParseIP(mask)
	if parsedMask == nil {
		return false // 不是有效的 IP 地址
	}

	if len(parsedMask.To4()) != net.IPv4len {
		return false // 不是 IPv4 地址
	}

	// 将子网掩码转换为 32 位整数
	maskInt := ipv4ToInt(parsedMask.To4())

	// 检查子网掩码是否只包含连续的 1 和连续的 0
	if maskInt|(maskInt-1) != 0xFFFFFFFF {
		return false
	}

	return true
}

// 将 IPv4 地址转换为 32 位无符号整数
func ipv4ToInt(ip net.IP) uint32 {
	bits := strings.Split(ip.String(), ".")
	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])

	return uint32(b0)<<24 + uint32(b1)<<16 + uint32(b2)<<8 + uint32(b3)
}

// @Summary		locaweb修改本地网络配置
// @Description	修改本地网络配置
// @ID				config-networkconfig
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			networkConfig	body		NetworkConfig		true	"网络参数配置"
// @Success		200				{object}	NetworkConfig		"成功 {\"ip\":\"192.168.2.27\", \"mask\":\"255.255.254.0\", \"gateway\":\"192.168.2.1\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/network/config [post]

func SetNetworkConfig(c *gin.Context) {
	netCfg := &NetworkConfig{}
	err := c.ShouldBindJSON(&netCfg)
	if err != nil {
		log.Error("SetNetworkConfig ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}
	log.Info("SetNetworkConfig in ...netCfg:", netCfg)
	if netCfg.IP == "" || netCfg.Mask == "" {
		log.Error("SetNetworkConfig params err. netCfg.IP:", netCfg.IP, ", netCfg.Mask:", netCfg.Mask)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Params err"})
		return
	}

	if !validateIP(netCfg.IP) {
		log.Error("invalid IP. netCfg.IP:", netCfg.IP)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Params IP err"})
		return
	}

	if netCfg.Gateway != "" && !validateIP(netCfg.Gateway) { // gateway allowed empty
		log.Error("invalid Gateway. netCfg.Gateway:", netCfg.Gateway)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Params Gateway err"})
		return
	}

	if netCfg.Dns1 != "" && !validateIP(netCfg.Dns1) { // dns1 allowed empty
		log.Error("invalid Dns1. netCfg.Dns1:", netCfg.Dns1)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Params Dns1 err"})
		return
	}

	if netCfg.Dns2 != "" && !validateIP(netCfg.Dns2) { // dns2 allowed empty
		log.Error("invalid Dns2. netCfg.Dns2:", netCfg.Dns2)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Params Dns2 err"})
		return
	}

	if !validateSubnetMask(netCfg.Mask) {
		log.Error("invalid Mask. netCfg.Mask:", netCfg.Mask)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Params Mask err"})
		return
	}

	err = setLocalIP(NetworkInterfaceDev, netCfg.IP, netCfg.Mask, netCfg.Gateway, netCfg.Dns1, netCfg.Dns2)
	if err != nil {
		log.Error("SetNetworkConfig setLocalIP err:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}
	c.JSON(http.StatusOK, netCfg)
}

type RebootReq struct {
	CheckPwd string `json:"checkpwd"`
}

// @Summary		locaweb重启工控机
// @Description	重启工控机
// @ID				reboot-system
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rebootReq	body		RebootReq		true	"重启参数配置"
// @Success		200				{object}	models.CommonResp	"成功 {\"code\":200, \"message\":\"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/reboot [post]
func SystemReboot(c *gin.Context) {
	log.Info("SystemReboot in ...")
	req := &RebootReq{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Error("SystemReboot ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}

	uName := c.GetString(middleware.UNameKey)
	if data, ok := OnlineUsers.Load(uName); ok {
		online, match := data.(*OnlineUser)
		if !match || req.CheckPwd != online.Pwd {
			log.Error("pwd check failed")
			c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "pwd invalid"})
			return
		}
	}

	rebootCmd := "reboot"
	err = execCommand(rebootCmd)
	if err != nil {
		log.Error("run reboot failed:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: ""})
}

// @Summary      EMS热加载
// @Description  通知前端重载配置后，依次停止并重启EMS相关进程，需密码校验
// @Tags         system
// @Accept       json
// @Produce      json
// @Param        body body   RebootReq true  "密码校验"
// @Success      200  {object}  models.CommonResp  "热加载成功"
// @Failure      400  {object}  models.CommonResp  "请求参数错误"
// @Failure      401  {object}  models.CommonResp  "认证失败"
// @Failure      403  {object}  models.CommonResp  "密码校验失败"
// @Failure      500  {object}  models.CommonResp  "服务内部错误"
// @Router       /api/v1/log/system/para_enable [post]
func ParaEnable(c *gin.Context) {
	log.Info("ParaEnable in ...")

	// 1. 密码校验
	req := &RebootReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("ParaEnable ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}

	uName := c.GetString(middleware.UNameKey)
	if data, ok := OnlineUsers.Load(uName); ok {
		online, match := data.(*OnlineUser)
		if !match || req.CheckPwd != online.Pwd {
			log.Error("ParaEnable pwd check failed")
			c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "pwd invalid"})
			return
		}
	}

	// 2. 通知前端服务重载配置
	//if err := cgolibparse.NotifyFeServerReload(); err != nil {
	//	log.Errorf("ParaEnable NotifyFeServerReload error: %v", err)
	//	c.JSON(http.StatusInternalServerError, models.CommonResp{
	//		Code: http.StatusInternalServerError, Message: "NotifyFeServerReload failed",
	//	})
	//	return
	//}
	//log.Info("ParaEnable NotifyFeServerReload success")

	// 3. 获取 ARM_SF/bin 路径
	armSF := os.Getenv("ARM_SF")
	if armSF == "" {
		log.Error("ParaEnable ARM_SF env not set")
		c.JSON(http.StatusInternalServerError, models.CommonResp{
			Code: http.StatusInternalServerError, Message: "ARM_SF env not set",
		})
		return
	}
	binDir := filepath.Join(armSF, "bin")

	// 4. 按序执行 stop 和 restart 命令
	commands := []struct {
		action string
		id     string
	}{
		{"stop", "6"}, {"stop", "3"}, {"stop", "2"}, {"stop", "1"},
		{"restart", "1"}, {"restart", "2"}, {"restart", "3"}, {"restart", "6"},
	}

	for _, cmd := range commands {
		fullCmd := fmt.Sprintf("cd %s && ./arm_ems %s %s", binDir, cmd.action, cmd.id)
		log.Infof("ParaEnable exec: %s", fullCmd)
		if err := execCommand(fullCmd); err != nil {
			log.Errorf("ParaEnable exec '%s' failed: %v", fullCmd, err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{
				Code:    http.StatusInternalServerError,
				Message: fmt.Sprintf("exec '%s %s' failed: %v", cmd.action, cmd.id, err),
			})
			return
		}
		log.Infof("ParaEnable '%s %s' success", cmd.action, cmd.id)
	}

	log.Info("ParaEnable all success")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "ParaEnable success"})
}

type HostNameInfo struct {
	HostName string `json:"hostname"`
}

func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("get hostname failed: %v", err)
	}
	return hostname, nil
}

func setCentOSHostname(newHostname string) error {
	if newHostname == "" {
		return errors.New("hostname can't be empty")
	}

	// temp modify(no need reboot)
	cmdTemp := exec.Command("hostname", newHostname)
	outputTemp, err := cmdTemp.CombinedOutput()
	if err != nil {
		return fmt.Errorf("temp modify hostname failed: %v, output: %s", err, string(outputTemp))
	}

	// modify /etc/hostname(need to reboot)
	cmdPersist := exec.Command("bash", "-c", fmt.Sprintf("echo %s > /etc/hostname", newHostname))
	outputPersist, err := cmdPersist.CombinedOutput()
	if err != nil {
		return fmt.Errorf("modify /etc/hostname failed: %v, output: %s", err, string(outputPersist))
	}

	return nil
}

// @Summary		获取hostname
// @Description	获取hostname
// @ID				get-hostname
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	models.CommonResp	"成功 {\"hostname\":\"ems-node01\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/hostname [get]

func GetHostname(c *gin.Context) {
	log.Info("GetHostname in ...")
	hostname, err := getHostname()
	if err != nil {
		log.Error("GetHostname getHostname err:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}

	hostNameInfo := &HostNameInfo{HostName: hostname}

	c.JSON(http.StatusOK, hostNameInfo)
	log.Info("GetHostname OK:", hostNameInfo)
	return
}

// @Summary		配置hostname
// @Description	配置hostname
// @ID				set-hostname
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			req				body		HostNameInfo		true	"hostname配置"
// @Success		200				{object}	models.CommonResp	"成功 {\"code\":200, \"message\":\"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/hostname [post]

func SetHostname(c *gin.Context) {
	log.Info("SetHostname in ...")
	req := &HostNameInfo{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Error("SetHostname ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}
	log.Info("SetHostname in ... req:", req)
	err = setCentOSHostname(req.HostName)
	if err != nil {
		log.Error("SetHostname setHostname err:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}

	log.Info("SetHostname OK")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: ""})
}

type SystemTime struct {
	DateTime string `json:"datetime"`
	Zone     string `json:"zone"`
}

func getSystemTimezone(offset int) string {
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	return fmt.Sprintf("UTC%s%02d:%02d", sign, hours, minutes)
}

func GetSystemTime(c *gin.Context) {
	zoneAddr, err := getTimezoneFromTimedatectl()
	if err != nil {
		log.Error("GetZoneInfo getTimezoneFromTimedatectl failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	loc, err := time.LoadLocation(zoneAddr)
	if err != nil {
		log.Error("GetZoneInfo LoadLocation failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	current := time.Now().In(loc)
	nowStr := current.Format(time.DateTime)
	_, offset := current.Zone()
	zoneStr := getSystemTimezone(offset)

	log.Info("GetSystemTime in ... nowStr:", nowStr, ",zoneStr:", zoneStr)

	c.JSON(http.StatusOK, SystemTime{DateTime: nowStr, Zone: zoneStr})
}

// @Summary		设置系统时间
// @Description	设置系统时间
// @ID				set-time
// @Tags			system
// @Accept			json
// @Produce			json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			req				body		SystemTime		true	"系统时间,填入datetime部分即可"
// @Success		200				{object}	models.CommonResp	"成功 {\"code\":200, \"message\":\"\"}"
// @Failure		400				{object}	models.CommonResp	"参数错误 {\"code\": 400, \"msg\": \"Bad Request\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/system/time [post]

func SetSystemTime(c *gin.Context) {
	log.Info("SetSystemTime in ...")
	req := &SystemTime{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Error("SetSystemTime ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}

	_, err = time.Parse(time.DateTime, req.DateTime)
	if err != nil {
		log.Error("SetSystemTime datetime layout invalid:", req.DateTime)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request: layout invalid"})
		return
	}
	log.Info("SetSystemTime set system time:", req.DateTime)

	err = exec.Command("date", "-s", req.DateTime).Run()
	if err != nil {
		log.Info("SetSystemTime failed to set system time, err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: ""})
}

// @Summary		设置系统时区
// @Description	设置系统时区
// @ID				set-timezone
// @Tags			system
// @Accept			json
// @Produce			json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			req				body		SystemTime		true	"系统时区,填入zone部分即可"
// @Success		200				{object}	models.CommonResp	"成功 {\"code\":200, \"message\":\"\"}"
// @Failure		400				{object}	models.CommonResp	"参数错误 {\"code\": 400, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/system/timezone [post]

func SetSystemTimeZone(c *gin.Context) {
	log.Info("SetSystemTimeZone in ...")
	req := &SystemTime{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Error("SetSystemTimeZone ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}

	// timezone, exists := timezoneMap[req.Zone]
	// if !exists {
	// 	log.Error("SetSystemTimeZone req Zone not exist:", req.Zone)
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request. invalid timezone param:" + req.Zone})
	// 	return
	// }
	// log.Errorf("SetSystemTimeZone set zone:%s for param: %s", timezone, req.Zone)
	log.Info("SetSystemTimeZone set zone:", req.Zone)

	cmd := exec.Command("timedatectl", "set-timezone", req.Zone)
	err = cmd.Run()
	if err != nil {
		log.Info("SetSystemTimeZone failed to set system timezone:", err.Error(), ",req.Zone:", req.Zone)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	loc, err := time.LoadLocation(req.Zone)
	current := time.Now().In(loc)
	nowStr := current.Format(time.DateTime)
	log.Debug("test nowStr:", nowStr)

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: ""})
}

//nolint:govet
type SystemConfig struct {
	// 设备型号
	ModelNum      string `json:"modelnum"`
	ShowOverView  int    `json:"showoverview"`
	ShowMonitor   int    `json:"showmonitor"`
	ShowTransData int    `json:"showtransdata"`
}

func loadSystemConfigData() bool {
	fileUnit, err := os.Open("data/systemconfig.json")
	if err != nil {
		log.Error("open data/systemconfig.json failed. err:", err.Error())
		return false
	}
	defer fileUnit.Close()

	dataUnit, err := io.ReadAll(fileUnit)
	if err != nil {
		log.Error("read data/systemconfig.json failed. err:", err.Error())
		return false
	}

	err = json.Unmarshal(dataUnit, &SystemConfigData)
	if err != nil {
		log.Error("read data/systemconfig.json failed. err:", err.Error())
		return false
	}

	return true
}

func setSystemConfigData(data *SystemConfig) error {
	SystemConfigData.ModelNum = data.ModelNum
	filePath := "data/systemconfig.json"

	// 要写入的内容
	content, err := json.Marshal(SystemConfigData)
	if err != nil {
		log.Error("setSystemConfigData Marshal failed:", err.Error())
		return err
	}

	// 使用os.WriteFile直接写入内容
	err = os.WriteFile(filePath, content, 0644) // 0644 是文件权限设置
	if err != nil {
		log.Error("setSystemConfigData WriteFile failed:", err.Error())
		return err
	}
	return nil
}

// @Summary		查询设备型号
// @Description	查询设备型号，无需登录
// @ID				get-modelnum
// @Tags			system
// @Accept			json
// @Produce		json
// @Success		200	{object}	SystemConfig		"查询成功 {\"modelnum\": \"ECO-E233LS-G01-EN\"}"
// @Failure		401	{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500	{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"Missing configuration data\"}"
// @Router			/api/v1/system/modelnum [get]

func GetModelNum(c *gin.Context) {
	log.Info("GetModelNum in...")

	// always get config from file, no need reboot
	loadSystemConfigData()

	if len(SystemConfigData.ModelNum) == 0 {
		log.Error("GetModelNum failed: loadSystemConfigData failed.")
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "Missing configuration data"})
		return
	}
	log.Info("GetModelNum OK:", SystemConfigData)
	c.JSON(http.StatusOK, SystemConfigData)
}

// @Summary		设置设备型号
// @Description	设置设备型号，需先登录
// @ID				set-modelnum
// @Tags			system
// @Accept			json
// @Produce			json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			req				body		SystemConfig		true	"{"modelnum":"ECO-E233LS-G01-EN"}"
// @Success		200	{object}	models.CommonResp	"修改成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		400	{object}	models.CommonResp	"认证失败 {\"code\": 400, \"msg\": \"Bad Request\"}"
// @Failure		401	{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500	{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"InternalServerError\"}"
// @Router			/api/v1/log/system/modelnum [post]

func SetModelNum(c *gin.Context) {
	log.Info("SetModelNum in...")

	req := &SystemConfig{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Error("SetModelNum ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}

	if req.ModelNum == "" {
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "model num should not be empty"})
		return
	}

	if err := setSystemConfigData(req); err != nil {
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: 0, Message: ""})
}

// systemGroup.GET("/baseinfo", handlers.GetSysBaseInfo)
// systemGroup.POST("/baseinfo", handlers.SetSysBaseInfo)

type SysTimeInfo struct {
	// 当前时间
	Time string `json:"time"`

	// 服务器当前时区偏移量
	Offset int `json:"offset"`

	// 服务器当前时区
	Zone string `json:"zone"`
}

type SysBaseInfo struct {
	Sn       string      `json:"sn"`
	Name     string      `json:"name"`
	Addr     string      `json:"addr"`
	ModelNum string      `json:"modelnum"`
	HostName string      `json:"hostname"`
	Timeinfo SysTimeInfo `json:"timeinfo"`
}

// @Summary		查询系统基本信息
// @Description	查询系统基本信息
// @ID				get-sysbaseinfo
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200	{object}	SysBaseInfo	"查询成功 {\"sn\": \"9876543210\", \"name\": \"myproject\", \"addr\": \"nanjing\", \"modelnum\": \"ECO-E233LS-G01-EN\", \"hostname\": \"eco-ems\", \"timeinfo\": {\"time\":\"2026-02-04T14:22:31+08:00\",\"offset\": 28800, \"zone\": \"Asia/Shanghai\"}"
// @Failure		500	{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"InternalServerError\"}"
// @Router			/api/v1/log/system/baseinfo [get]
func GetSysBaseInfo(c *gin.Context) {
	log.Info("GetSysBaseInfo in...")

	// 1. get system info
	sysInfoMng, err := emsxmldb.GetSystemInfoManager()
	if err != nil {
		log.Error("GetSysBaseInfo GetSystemInfoManager err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	sysinfo := sysInfoMng.GetAll()
	result := &SysBaseInfo{Sn: sysinfo.Sn, Name: sysinfo.Name, Addr: sysinfo.Addr, ModelNum: sysinfo.ModelNum}
	result.Sn = resolveSystemSerial(result.Sn)

	// 2. get hostname
	hostname, err := getHostname()
	if err != nil {
		log.Error("GetHostname getHostname err:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}
	result.HostName = hostname

	// 3. get timezone
	zoneAddr, err := getTimezoneFromTimedatectl()
	if err != nil {
		log.Error("GetZoneInfo getTimezoneFromTimedatectl failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	loc, err := time.LoadLocation(zoneAddr)
	if err != nil {
		log.Error("GetZoneInfo LoadLocation failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	current := time.Now().In(loc)
	nowStr := current.Format(time.RFC3339)
	_, offset := current.Zone()
	result.Timeinfo = SysTimeInfo{Time: nowStr, Offset: offset, Zone: zoneAddr}

	log.Info("GetSysBaseInfo result:", result)

	c.JSON(http.StatusOK, result)
}

// @Summary		设置系统基本信息
// @Description	设置系统基本信息，需先登录
// @ID				set-sysbaseinfo
// @Tags			system
// @Accept			json
// @Produce			json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			req				body		SysBaseInfo		true	"{\"sn\": \"9876543210\", \"name\": \"myproject\", \"addr\": \"nanjing\", \"modelnum\": \"ECO-E233LS-G01-EN\", \"hostname\": \"eco-ems\", \"timeinfo\": {\"time\":\"2026-02-04T14:22:31+08:00\",\"offset\": 28800, \"zone\": \"Asia/Shanghai\"}"
// @Success		200	{object}	models.CommonResp	"修改成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		400	{object}	models.CommonResp	"认证失败 {\"code\": 400, \"msg\": \"Bad Request\"}"
// @Failure		401	{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500	{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"InternalServerError\"}"
// @Router			/api/v1/log/system/baseinfo [post]
func SetSysBaseInfo(c *gin.Context) {
	log.Info("SetSysBaseInfo in...")
	req := &SysBaseInfo{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Error("SetSysBaseInfo ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}
	log.Info("SetSysBaseInfo req:", req)

	// 1. set system info
	sysInfoMng, err := emsxmldb.GetSystemInfoManager()
	if err != nil {
		log.Error("SetSysBaseInfo GetSystemInfoManager err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// Sn is derived from the license machine serial on read. Preserve the
	// stored XML value here so this endpoint cannot desynchronize it.
	err = sysInfoMng.SetAll(sysInfoMng.GetSn(), req.Name, req.Addr, req.ModelNum)
	if err != nil {
		log.Error("SetSysBaseInfo SetAll err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 2. set hostname
	err = setCentOSHostname(req.HostName)
	if err != nil {
		log.Error("SetSysBaseInfo setCentOSHostname err:", err.Error())
		c.JSON(http.StatusInternalServerError,
			models.CommonResp{Code: http.StatusInternalServerError, Message: "Internal Service Error"})
		return
	}

	// 3. set timezone
	cmd := exec.Command("timedatectl", "set-timezone", req.Timeinfo.Zone)
	err = cmd.Run()
	if err != nil {
		log.Info("SetSysBaseInfo failed to set system timezone:", err.Error(), ",req.Timeinfo.Zone:", req.Timeinfo.Zone)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// TODO 4. set system time
	// _, err = time.Parse(time.DateTime, req.Timeinfo.Time)
	// if err != nil {
	// 	log.Error("SetSystemTime datetime layout invalid:", req.Timeinfo.Time)
	// 	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request: layout invalid"})
	// 	return
	// }
	// log.Info("SetSystemTime set system time:", req.Timeinfo.Time)

	// err = exec.Command("date", "-s", req.Timeinfo.Time).Run()
	// if err != nil {
	// 	log.Info("SetSystemTime failed to set system time, err:", err.Error())
	// 	c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
	// 	return
	// }

	log.Info("SetSysBaseInfo OK")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "success"})
}

// @Summary		获取系统网络配置
// @Description	获取系统网络配置，需先登录
// @ID				get-sysnetconfig
// @Tags			system
// @Accept			json
// @Produce			json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200	{object}	[]systeminfo.NetworkDeviceInfo	"获取成功"
// @Failure		400	{object}	models.CommonResp	"认证失败 {\"code\": 400, \"msg\": \"Bad Request\"}"
// @Failure		401	{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500	{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"InternalServerError\"}"
// @Router			/api/v1/log/system/netconfig [get]
func GetSysNetConfig(c *gin.Context) {
	log.Info("GetSysNetConfig in...")
	netConfig, err := systeminfo.GetAllNetworkDevices()
	if err != nil {
		log.Error("GetSysNetConfig GetAllNetworkDevices err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("GetSysNetConfig result:", netConfig)
	c.JSON(http.StatusOK, netConfig)
}

// @Summary		配置系统网络
// @Description	配置系统网络，需先登录
// @ID				post-sysnetconfig
// @Tags			system
// @Accept			json
// @Produce			json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			req				body		systeminfo.NetworkDeviceInfo	true	"{\"deviceName\": \"eth0\", \"addressMode\": \"dhcp\", \"ipAddress\": \"192.168.1.1\", \"subnetMask\": \"255.255.255.0\", \"gateway\": \"192.168.1.1\", \"dns\": \"192.168.1.1\", \"backupDns\": \"192.168.1.1\"}"
// @Success		200	{object}	models.CommonResp	"修改成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		400	{object}	models.CommonResp	"认证失败 {\"code\": 400, \"msg\": \"Bad Request\"}"
// @Failure		401	{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500	{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"InternalServerError\"}"
// @Router			/api/v1/log/system/netconfig [post]
func SetSysNetConfig(c *gin.Context) {
	log.Info("SetSysNetConfig in...")
	req := &systeminfo.NetworkDeviceInfo{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Error("SetSysNetConfig ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}
	log.Info("SetSysNetConfig req:", req)
	err = systeminfo.ConfigNetworkDevice(req.DeviceName, req.AddressMode, req.IPAddress, req.SubnetMask, req.Gateway, req.DNS, req.BackupDNS)
	if err != nil {
		log.Error("SetSysNetConfig ConfigNetworkDevice err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("SetSysNetConfig OK")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "success"})
}

func GetLogList(c *gin.Context) {
	log.Info("GetLogList in...")

	processname := c.Param("processname")

	logPath := path.Join(getEnvParam("ARM_SF"), "data", "log")

	entries, err := os.ReadDir(logPath)
	if err != nil {
		log.Errorf("DownloadHisDataFile: get file failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	var files []string
	var sizes []int64
	for _, entry := range entries {
		// 只处理普通文件，忽略目录
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), processname) {
			info, err := entry.Info()
			if err != nil {
				log.Errorf("获取 %s 信息失败: %v", entry.Name(), err)
				return
			}
			sizes = append(sizes, info.Size())
			files = append(files, entry.Name())
		}
	}

	result := make(map[string]interface{})
	result["files"] = files
	result["sizes"] = sizes

	c.JSON(http.StatusOK, result)
}

func DownloadLogFile(c *gin.Context) {
	log.Info("DownloadLogFile in...")

	var files []string
	err := c.ShouldBindJSON(&files)
	if err != nil {
		log.Errorf("DownloadLogFile: ShouldBindJSON failed: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "invalid request body"})
		return
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "file list is empty"})
		return
	}

	// 获取日志根目录
	logRoot := path.Join(getEnvParam("ARM_SF"), "data", "log")

	// 创建临时 ZIP 文件 写入内存缓冲区
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, filename := range files {
		// 安全检查：防止路径遍历攻击，只允许文件名，不允许包含路径分隔符
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
			log.Warnf("DownloadLogFile: illegal filename: %s", filename)
			continue // 跳过非法文件名，也可直接返回错误，视需求而定
		}
		// 构建完整路径
		filePath := filepath.Join(logRoot, filename)
		// 确保文件在 logRoot 下（防止通过符号链接等绕过）
		if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(logRoot)+string(os.PathSeparator)) {
			log.Warnf("DownloadLogFile: file outside log root: %s", filePath)
			continue
		}

		// 打开文件
		srcFile, err := os.Open(filePath)
		if err != nil {
			log.Errorf("DownloadLogFile: open file %s failed: %v", filePath, err)
			continue // 跳过无法打开的文件，或者可以返回错误，这里选择跳过继续
		}
		defer srcFile.Close()

		// 获取文件信息（用于写入 ZIP 头）
		info, err := srcFile.Stat()
		if err != nil {
			log.Errorf("DownloadLogFile: stat file %s failed: %v", filePath, err)
			continue
		}

		// 在 ZIP 中创建文件头，使用原始文件名
		header := &zip.FileHeader{
			Name:   filename,
			Method: zip.Deflate, // 使用压缩
		}
		header.SetModTime(info.ModTime())

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			log.Errorf("DownloadLogFile: create zip entry for %s failed: %v", filename, err)
			continue
		}

		// 复制文件内容到 ZIP
		if _, err := io.Copy(writer, srcFile); err != nil {
			log.Errorf("DownloadLogFile: copy file %s to zip failed: %v", filename, err)
			continue
		}
	}

	// 关闭 ZIP Writer，完成压缩
	if err := zipWriter.Close(); err != nil {
		log.Errorf("DownloadLogFile: close zip writer failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "zip creation failed"})
		return
	}

	// 设置响应头，返回 ZIP 文件
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=logs.zip")
	// 返回压缩后的数据
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}
