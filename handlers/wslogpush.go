package handlers

import (
	"net/http"
	"strconv"
	"webserver-arm/log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var logWsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const timeLayout = "2006-01-02 15:04:05"

type LogType struct {
	LogType int `json:"logtype"`
}

// @Summary		控制台日志推送websocket接口
// @Description	控制台日志推送websocket接口
// @ID				ws-conn-logpush
// @Tags			websocket
// @Param			Authorization	header		string	true	"JWT Token或者放到url"		default(<token>)
// @Param			authorization	query		string	true	"JWT Token 或者放到Header"	default(<token>)
// @Param			procid			path		string	true	"日志请求的进程id"
// @Param			logtypeid		path		string	true	"日志请求的日志类型id"
// @success		200				{object}	string	"直接推送日志文本"
// // @Router			/api/v1/log/system/process/{procid}/logdump/{logtypeid} [get]
func LogPush(c *gin.Context) {
	procID := c.Param("procid")
	logTypeID := c.Param("logtypeid")
	log.Info("LogPush websocket in. procid:", procID, ", logtypeid:", logTypeID)

	conn, err := logWsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("LogPush Upgrade failed:", err.Error())
		return
	}
	defer conn.Close()

	go func() {
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				// if err, return to PopConn
				log.Error("LogPush ws ReadMessage err:", err.Error())
				return
			}
			log.Debug("LogPush get message:", string(p))

			if messageType == websocket.CloseMessage {
				log.Info("LogPush ws received CloseMessage")
				conn.Close()
				return
			}
		}
	}()

	// get pid
	logPid, err := strconv.Atoi(procID)
	if err != nil {
		log.Error("LogPush ws ReadMessage not pid:", procID)
		return
	}

	logType, err := strconv.Atoi(logTypeID)
	if err != nil {
		log.Error("LogPush ws ReadMessage not pid:", procID)
		return
	}

	err = writeProcLog(logPid, logType, conn)
	if err != nil {
		log.Error("writeProcLog err:", err.Error())
		return
	}
}

func writeProcLog(procPid, logType int, conn *websocket.Conn) error {
	// logProcs, err := scdnet.GetLogProcs()
	// if err != nil {
	// 	return fmt.Errorf("GetSysProc failed: %w", err)
	// }

	// log_idx := -1
	// for index, logProc := range logProcs {
	// 	if logProc.Pid == procPid {
	// 		log_idx = index
	// 	}
	// }
	// if log_idx == -1 {
	// 	return fmt.Errorf("no match pid found: %+v", logProcs)
	// }

	// logTypes, err := scdnet.GetLogTypes(log_idx)
	// if err != nil {
	// 	return fmt.Errorf("GetLogTypes failed: %w", err)
	// }

	// logTypeMap := make(map[int]string)
	// for _, logType := range logTypes {
	// 	logTypeMap[logType.TypeIdx] = logType.TypeDesc
	// }

	// err = scdnet.SetLogProc(log_idx)
	// if err != nil {
	// 	return fmt.Errorf("SetLogProc failed: %w", err)
	// }

	// for {
	// 	logTypeIdx, logdata, err := scdnet.GetLog(logType)
	// 	if err != nil {
	// 		return fmt.Errorf("GetLog failed: %w", err)
	// 	}
	// 	for idx, logStr := range logdata {
	// 		log.Debug("logContent:", logdata, logTypeIdx, logTypes)
	// 		logDesc := logTypeMap[logTypeIdx[idx]]
	// 		logContent := fmt.Sprintf("%s %s %s", time.Now().Format(timeLayout), logDesc, logStr)
	// 		err := conn.WriteMessage(websocket.TextMessage, []byte(logContent))
	// 		if err != nil {
	// 			return fmt.Errorf("WriteMessage failed: %w", err)
	// 		}
	// 	}
	// }
	return nil
}

// @Summary		localweb下载指定进程的日志文件
// @Description	下载指定进程的日志文件，进程启动或未启动都可以获取
// @ID				log-file
// @Tags			system
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			procid			path		string				true	"日志请求的进程id"
// @Success		200				{file}		octet-stream		"操作成功 返回文件"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"文件不存在"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/system/process/{procid}/logfile [get]
func GetLogFile(c *gin.Context) {
	// procid := c.Param("procid")

	// log.Info("GetLogFile in ... procid:", procid)

	// procs, err := scdnet.GetSysProc()
	// if err != nil {
	// 	log.Error("LocalProcListGet GetSysProc failed:", err.Error())
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
	// 	log.Errorf("GetLogType no match pid %d found: %+v", procid, procs)
	// 	failedResp := &models.CommonResp{Code: http.StatusNotFound, Message: "procid not found"}
	// 	c.JSON(http.StatusNotFound, failedResp)
	// 	return
	// }

	// sourceFilePath := models.SysLogFilePath
	// targetFilePath := procName + ".tgz"

	// utils.TarGzDirectory(sourceFilePath, targetFilePath, func(path string, file os.FileInfo) bool {
	// 	if strings.HasPrefix(file.Name(), procName) {
	// 		return true
	// 	}
	// 	return false
	// })

	// //c.JSON(http.StatusOK, models.CommonResp{Code: 0, Message: ""})
	// log.Info("GetLogFile created target:", targetFilePath)

	// c.Header("Content-Disposition", "attachment; filename="+procName+".tgz")
	// c.File(targetFilePath)
}
