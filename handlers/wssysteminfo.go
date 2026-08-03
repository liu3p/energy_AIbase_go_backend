package handlers

import (
	"net/http"
	"webserver-arm/log"
	"webserver-arm/systeminfo"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var sysinfoWsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// func WsGetSystemInfo(c *gin.Context) {
// 	conn, err := sysinfoWsUpgrader.Upgrade(c.Writer, c.Request, nil)
// 	if err != nil {
// 		log.Error("PushSystemInfo Upgrade failed:", err.Error())
// 		return
// 	}
// 	defer conn.Close()

// 	go func() {
// 		for {
// 			messageType, p, err := conn.ReadMessage()
// 			if err != nil {
// 				// if err, return to PopConn
// 				log.Error("PushSystemInfo ws ReadMessage err:", err.Error())
// 				return
// 			}
// 			log.Debug("PushSystemInfo get message:", string(p))

// 			if messageType == websocket.CloseMessage {
// 				log.Info("PushSystemInfo ws received CloseMessage")
// 				conn.Close()
// 				return
// 			}
// 		}
// 	}()

// 	for {
// 		info, err := systeminfo.GetSystemInfo()
// 		if err != nil {
// 			log.Error("GetSystemInfo err:", err.Error())
// 			return
// 		}
// 		log.Debug("PushSystemInfo send message:", info)
// 		retData, err := json.Marshal(info)
// 		if err != nil {
// 			log.Error("PushSystemInfo json.Marshal err:", err.Error())
// 			return
// 		}
// 		err = conn.WriteMessage(websocket.TextMessage, retData)
// 		if err != nil {
// 			log.Error("PushSystemInfo ws WriteMessage err:", err.Error())
// 			return
// 		}
// 		time.Sleep(time.Second * 1)
// 	}
// }

// @Summary		推送系统信息接口
// @Description	推送系统信息接口
// @ID				ws-conn-systeminfo
// @Tags			websocket
// @Param			Authorization	header		string	true	"JWT Token或者放到url"		default(<token>)
// @Param			authorization	query		string	true	"JWT Token 或者放到Header"	default(<token>)
// @success		200				{object}	systeminfo.SystemInfo	"推送系统信息"
// @Router			/api/v1/log/system/ws/sysinfo [get]
func WsGetSystemInfo(c *gin.Context) {
	log.Info("WsGetSystemInfo websocket in")
	conn, err := sysinfoWsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WsGetSystemInfo Upgrade failed:", err.Error())
		return
	}
	defer conn.Close()

	stopChan := make(chan struct{})
	defer func() {
		close(stopChan)
	}()

	err = systeminfo.PushSystemInfo(func(data []byte) {
		log.Debug("WsGetSystemInfo get message:", string(data))
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Error("WsGetSystemInfo ws WriteMessage err:", err.Error())
			return
		}
	}, stopChan)
	if err != nil {
		log.Error("WsGetSystemInfo SubscribeSystemInfo err:", err.Error())
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Error("WsGetSystemInfo ws ReadMessage err:", err.Error())
			return
		}
		log.Debug("WsGetSystemInfo get message:", string(p))

		if messageType == websocket.CloseMessage {
			log.Info("WsGetSystemInfo ws received CloseMessage")
			return
		}
	}
}

// @Summary		推送进程信息接口
// @Description	推送进程信息接口
// @ID				ws-conn-procinfo
// @Tags			websocket
// @Param			Authorization	header		string	true	"JWT Token或者放到url"		default(<token>)
// @Param			authorization	query		string	true	"JWT Token 或者放到Header"	default(<token>)
// @success		200				{object}	[]systeminfo.ProcessInfo	"推送进程信息"
// @Router			/api/v1/log/system/ws/process [get]
func WsGetProcessInfo(c *gin.Context) {
	log.Info("WsGetProcessInfo websocket in")
	conn, err := sysinfoWsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WsGetProcessInfo Upgrade failed:", err.Error())
		return
	}
	defer conn.Close()

	stopChan := make(chan struct{})
	defer func() {
		close(stopChan)
	}()

	err = systeminfo.PushProcessInfo(func(data []byte) {
		log.Debug("WsGetProcessInfo get message:", string(data))
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Error("WsGetProcessInfo ws WriteMessage err:", err.Error())
			return
		}
	}, stopChan)
	if err != nil {
		log.Error("WsGetProcessInfo SubscribeSystemInfo err:", err.Error())
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Error("WsGetProcessInfo ws ReadMessage err:", err.Error())
			return
		}
		log.Debug("WsGetProcessInfo ws ReadMessage:", string(p))

		if messageType == websocket.CloseMessage {
			log.Info("WsGetProcessInfo ws received CloseMessage")
			return
		}
	}
}

// @Summary		控制台日志推送websocket接口
// @Description	控制台日志推送websocket接口
// @ID				ws-conn-logdump
// @Tags			websocket
// @Param			Authorization	header		string	true	"JWT Token或者放到url"		default(<token>)
// @Param			authorization	query		string	true	"JWT Token 或者放到Header"	default(<token>)
// @Param			procid			path		string	true	"日志请求的进程id"
// @success		200				{object}	string	"直接推送日志文本"
// @Router			/api/v1/log/system/process/{procid}/logdump [get]
func WsGetProcessLog(c *gin.Context) {
	processID := c.Param("procid")

	log.Info("WsGetProcessLog websocket in. procid:", processID)
	conn, err := sysinfoWsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WsGetProcessLog Upgrade failed:", err.Error())
		return
	}
	defer conn.Close()

	stopChan := make(chan struct{})
	defer func() {
		close(stopChan)
	}()

	err = systeminfo.PushProcessLog(processID, func(data []byte) {
		log.Debug("WsGetProcessLog get message:", string(data))
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Error("WsGetProcessLog ws WriteMessage err:", err.Error())
			return
		}
	}, stopChan)
	if err != nil {
		log.Error("WsGetProcessLog SubscribeSystemInfo err:", err.Error())
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Error("WsGetProcessLog ws ReadMessage err:", err.Error())
			return
		}
		log.Debug("WsGetProcessLog get message:", string(p))

		if messageType == websocket.CloseMessage {
			log.Info("WsGetProcessLog ws received CloseMessage")
			return
		}
	}
}
