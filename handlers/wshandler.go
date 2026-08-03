package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"webserver-arm/emsclient"
	"webserver-arm/homepage"
	"webserver-arm/log"
	"webserver-arm/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// @Summary		在线连接
// @Description	在线连接
// @ID				ws-conn
// @Tags			websocket
// @Param			Authorization	header		string	true	"JWT Token或者放到url"		default(<token>)
// @Param			authorization	query		string	true	"JWT Token 或者放到Header"	default(<token>)
// @success		200				{object}	string	"推送遥测，遥信，属性，设备状态。 遥测，遥信，属性 参照其查询接口返回。 设备状态示例(1-离线 2-在线)：{\"topic\":\"status\",\"content\":{\"11223355esscab1pcs1\": 2, \"11223355esscab1ac1\": 1}}"
// @Router			/api/v1/log/ws/push [get]
func WebSocketHandler(c *gin.Context) {
	log.Debug("WebSocketHandler in")
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to upgrade to WebSocket",
		})
		return
	}
	defer conn.Close()

	uname := c.GetString(middleware.UNameKey)
	log.Infof("user %s ws connnected", uname)
	if data, ok := OnlineUsers.Load(uname); ok {
		// 更新用户状态
		onlines, ok := data.(*OnlineUser)
		if ok {
			onlines.State = 2
			onlines.PushConn = conn
			onlines.LastUpdateCache = 0 // make sure to update
			OnlineUsers.Store(uname, onlines)
			// 添加推送
			ClientMgr.PushConn(uname, conn)
		}
	}

	defer func() {
		// 清理首页订阅
		homepage.GetHomepageManager(nil).Unsubscribe()
		// 去除推送
		ClientMgr.PopConn(uname)
		if data, ok := OnlineUsers.Load(uname); ok {
			// 更新用户状态
			onlines, ok := data.(*OnlineUser)
			if ok {
				onlines.State = 1
				onlines.PushConn = nil
				OnlineUsers.Store(uname, onlines)
			}
		}
		log.Infof("user %s ws disconnect", uname)
	}()

	getPong := make(chan int)
	conn.SetPongHandler(func(appData string) error {
		getPong <- 1
		return nil
	})

	conn.SetCloseHandler(func(code int, text string) error {
		log.Warnf("user %s ws Received close: %d, data:%s", uname, code, text)
		return nil
	})

	go func() {
		noPongTimes := 0
		for {
			select {
			case <-getPong:
				noPongTimes = 0
			case <-time.After(10 * time.Second):
				ClientMgr.PushMsg(&emsclient.PushResp{Topic: "ping", Content: uname})
				if noPongTimes > 2 {
					//ClientMgr.PushMsg(&emsclient.PushResp{Topic: "close", Content: uname})
					log.Errorf("user %s ws noPongTimes %d timeout. ws will close", uname, noPongTimes)
					err = conn.Close()
					if err != nil {
						log.Errorf("user %s ws close conn failed: %s", uname, err.Error())
					}
					return
				}
				noPongTimes++
			}
		}
	}()

	for {
		// ReadMessage just read text and binary message
		_, p, err := conn.ReadMessage()
		if err != nil {
			// if err, return to PopConn
			log.Errorf("user %s ws ReadMessage err: %s", uname, err.Error())
			return
		}

		// maybe do something
		log.Debug("user %s recv message: %s", uname, string(p))

		// 解析消息 topic，分发处理
		var msg struct {
			Topic string `json:"topic"`
		}
		if err := json.Unmarshal(p, &msg); err != nil {
			continue
		}
		switch msg.Topic {
		case "homepage_subscribe":
			homepage.GetHomepageManager(nil).Subscribe()
		case "homepage_unsubscribe":
			homepage.GetHomepageManager(nil).Unsubscribe()
		}
	}
}

func PushWs(uName string, messageType int, message []byte) {
	if data, ok := OnlineUsers.Load(uName); ok {
		onlines, match := data.(*OnlineUser)
		if match {
			if onlines.PushConn != nil {
				err := onlines.PushConn.WriteMessage(messageType, message)
				if err != nil {
					log.Error("PushWs WriteMessage failed:", err.Error())
				}
			} else {
				log.Warnf("user %s is not online, state: %d", uName, onlines.State)
			}
		}
	} else {
		log.Error("do not find uName in OnlineUsers: ", uName)
	}
}

func KickOutWs(conn *websocket.Conn) {
	if conn != nil {
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Error("KickOutWs WriteMessage failed:", err.Error())
		}
		err = conn.Close()
		if err != nil {
			log.Error("KickOutWs Close conn failed:", err.Error())
		}
	}
}

func KickOutWsByUname(uName string) {
	ClientMgr.PushMsg(&emsclient.PushResp{Topic: "close", Content: uName})
}
