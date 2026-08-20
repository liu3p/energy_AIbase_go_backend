package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"webserver-arm/cgolibparse"
	"webserver-arm/emsxmldb"
	"webserver-arm/log"
	"webserver-arm/middleware"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// StartWSReader 启动一个 WebSocket 读取协程
// conn: WebSocket 连接
// cancel: 用于通知主循环退出的 context.CancelFunc（通常是主函数的 cancel）
// onMessage: 可选的回调函数，在收到非关闭消息时调用。如果为 nil 则忽略消息体。
// readTimeout: 读取超时时间，传 0 表示不设置读截止时间
func startWSReader(ctx context.Context, conn *websocket.Conn, cancel context.CancelFunc,
	onMessage func(messageType int, data []byte), readTimeout time.Duration) {

	go func() {
		defer cancel() // 读 goroutine 退出时，通知主循环退出

		if readTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(readTimeout))
		}

		for {
			// 检查 context 是否已取消
			select {
			case <-ctx.Done():
				log.Info("WS reader: context done, exit")
				return
			default:
			}

			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Info("WS reader read message exit:", err.Error())
				return
			}

			if messageType == websocket.CloseMessage {
				log.Info("WS reader received CloseMessage")
				conn.Close() // 主动关闭连接
				return
			}

			// 需要处理客户端消息时，调用外部回调函数
			if onMessage != nil {
				onMessage(messageType, p)
			}
		}
	}()
}

// @Summary		获取指定rtu绑定的通道
// @Description	获取指定rtu绑定的通道
// @ID				datamonitor-get-channels
// @Tags			datamonitor
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	emsxmldb.ChannelGroup	"检查成功 如果err为空，result值为计算结果；如果err不为空，则表示公式错误，内容为错误信息。"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"没有绑定通道 {\"code\": 404, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/datamonitor/rtu/{rid}/channels [get]
func GetRtuChannels(c *gin.Context) {
	rid := c.Param("rid")
	log.Info("GetRtuChannels in ... rid:", rid)
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("GetRtuChannels GetFeConfigXMLManager failed:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "GetRtuChannels failed"})
		return
	}
	channelGroup, err := fecfg.GetChannelsByRtuID(rid)
	if err != nil {
		log.Error("GetRtuChannels GetChannelsByRtuID failed:", err)
		c.JSON(http.StatusNotFound, models.CommonResp{Code: http.StatusNotFound, Message: "GetRtuChannels not found"})
		return
	}
	log.Info("GetRtuChannels success, channelGroup:", channelGroup)
	c.JSON(http.StatusOK, sortChannelGroupPlugins(&channelGroup)) // 拷贝+排序，不修改原始数据
}

// @Summary		[websocket]获取指定通道的状态
// @Description	[websocket]获取指定通道的状态
// @ID				datamonitor-get-channel-status
// @Tags			datamonitor
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			cgid				path		string				true	"通道组id"
// @Param			cid				path		string				true	"通道id"
// @Success		200				{object}	cgolibparse.FeChannelStatus	"获取成功 {\"connected\":1}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/datamonitor/cgroup/{cgid}/channel/{cid}/status [get]
func GetRtuChannelStatus(c *gin.Context) {
	cgid := c.Param("cgid")
	cid := c.Param("cid")
	log.Info("GetRtuChannelStatus in ... channel group ID:", cgid, ", channel ID:", cid)

	conn, err := dataUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("GetFeChannelStatus Upgrade failed:", err.Error())
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	defer conn.Close()

	chanGroupID, err := strconv.ParseUint(cgid, 10, 16)
	if err != nil {
		log.Errorf("GetRtuChannelStatus: cgid %s ParseUint failed: '%w'", cgid, err)
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 创建一个带取消的 context，用于通知所有 goroutine 退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动读消息 goroutine
	startWSReader(ctx, conn, cancel, nil, 0)

	// 获取前置数据管理类
	feParseIns, err := cgolibparse.GetFeParseManagerInstance()
	if err != nil {
		log.Error("GetAllRtuChannelsStatus GetFeChannelsStatus failed")
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			status := feParseIns.GetFeChannelStatus(uint16(chanGroupID))
			// 设置写截止时间，避免 WriteJSON 阻塞过久
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(status); err != nil {
				log.Error("GetAllRtuChannelsStatus ws WriteJSON err:", err.Error())
				return // 写入失败，退出函数，defer 和 cancel 会清理资源
			}
		case <-ctx.Done():
			// 读 goroutine 已退出（通常因为连接关闭或出错），本函数也应退出
			log.Info("GetAllRtuChannelsStatus context done, exit")
			return
		}
	}
}

type pushAllChanStatus struct {
	RtuID     string   `json:"rtuid"`
	RtuName   string   `json:"rtuname"`
	Connected int      `json:"connected"`
	Device    []string `json:"device"`
}

// @Summary		[websocket]获取全部通道的状态
// @Description	[websocket]获取全部通道的状态
// @ID				datamonitor-get-allchannel-status
// @Tags			datamonitor
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	[]pushAllChanStatus	"获取成功 {\"connected\":1}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/datamonitor/channels/status [get]
func GetAllRtuChannelsStatus(c *gin.Context) {
	log.Info("GetAllRtuChannelsStatus in ...")

	conn, err := dataUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("GetAllRtuChannelsStatus Upgrade failed:", err.Error())
		return
	}
	defer conn.Close()

	// 创建一个带取消的 context，用于通知所有 goroutine 退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动读消息 goroutine
	startWSReader(ctx, conn, cancel, nil, 0)

	// 获取前置数据管理类
	feParseIns, err := cgolibparse.GetFeParseManagerInstance()
	if err != nil {
		log.Error("GetAllRtuChannelsStatus GetFeChannelsStatus failed")
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// 获取所有rtu id和通道id的对应关系
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("GetAllRtuChannelsStatus GetFeConfigManager failed")
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	rtuChannelMap := fecfg.GetAllChennalStationID()

	// 调用dbxml管理类获取所有rtu和device
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Error("GetAllRtuChannelsStatus GetDbConfigManager failed")
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	rtus := dbcfg.GetAllRTUsExceptPoints()
	allStatus := make([]pushAllChanStatus, len(rtus))
	for i := range rtus {
		allStatus[i].RtuID = rtus[i].ID
		allStatus[i].RtuName = rtus[i].Name
		for j := range rtus[i].Devices {
			allStatus[i].Device = append(allStatus[i].Device, rtus[i].Devices[j].Name)
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			allStatusMap := feParseIns.GetFeChannelsStatus()
			// 更新连接状态
			for i := range allStatus {
				if chanGroupID, has := rtuChannelMap[allStatus[i].RtuID]; has {
					if status, has := allStatusMap[uint16(chanGroupID)]; has {
						allStatus[i].Connected = status.Connected
					}
				}
			}
			// 设置写截止时间，避免 WriteJSON 阻塞过久
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(allStatus); err != nil {
				log.Error("GetAllRtuChannelsStatus ws WriteJSON err:", err.Error())
				return // 写入失败，退出函数，defer 和 cancel 会清理资源
			}
		case <-ctx.Done():
			// 读 goroutine 已退出（通常因为连接关闭或出错），本函数也应退出
			log.Info("GetAllRtuChannelsStatus context done, exit")
			return
		}
	}
}

// @Summary		[websocket]获取指定通道的报文
// @Description	[websocket]获取指定通道的报文
// @ID				datamonitor-get-channel-logs
// @Tags			datamonitor
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			cgid			path		string				true	"通道组id"
// @Param			cid				path		string				true	"通道id"
// @Param			pluginid		path		string				true	"插件id"
// @Success		200				{object}	cgolibparse.FeChannelLog	"获取成功 {\"loginfo\":\"xxxxx\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/datamonitor/cgroup/{cgid}/channel/{cid}/plugin/{pluginid}/logs [get]
func GetRtuChannelLogs(c *gin.Context) {
	cgid := c.Param("cgid")
	cid := c.Param("cid")
	pid := c.Param("pluginid")
	log.Info("GetRtuChannelLogs in ...channel group ID:", cgid, ", channel ID:", cid, ", pluginID:", pid)

	conn, err := dataUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("GetRtuChannelLogs Upgrade failed:", err.Error())
		return
	}
	defer conn.Close()

	chanGroupID, err := strconv.ParseUint(cgid, 10, 16)
	if err != nil {
		log.Error("GetRtuChannelLogs Parse group ID failed:", err)
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	chanID, err := strconv.ParseUint(cid, 10, 16)
	if err != nil {
		log.Error("GetRtuChannelLogs Parse channel ID failed:", err)
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	pluginID, err := strconv.ParseUint(pid, 10, 32)
	if err != nil {
		log.Error("GetRtuChannelLogs Parse channel ID failed:", err)
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 创建一个带取消的 context，用于通知所有 goroutine 退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动读消息 goroutine
	startWSReader(ctx, conn, cancel, nil, 0)

	// 获取前置数据管理类
	feParseIns, err := cgolibparse.GetFeParseManagerInstance()
	if err != nil {
		log.Error("GetRtuChannelLogs GetFeChannelsStatus failed:", err)
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	msgCh, cleanup, err := feParseIns.GetFeChannelLogs(uint16(chanGroupID), uint16(chanID), uint32(pluginID))
	if err != nil {
		log.Error("GetRtuChannelLogs GetFeChannelLogs failed:", err)
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	defer cleanup()

	for {
		select {
		case logs, ok := <-msgCh:
			if !ok {
				log.Info("msgCh closed, exit")
				return
			}
			err = conn.WriteJSON(logs.LogInfo)
			if err != nil {
				log.Error("GetRtuChannelLogs ws WriteJSON err:", err.Error())
				return
			}
		case <-ctx.Done():
			// 读 goroutine 已退出（通常因为连接关闭或出错），本函数也应退出
			log.Info("GetRtuChannelLogs context done, exit")
			return
		}
	}
}

var dataUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 前端数据监控页面控制指令
type CMD struct {
	Code     int    `json:"code"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pagesize,omitempty"`
	Name     string `json:"name,omitempty"`
	Total    int    `json:"total"`
}

// lastOIDSegment 提取 OID 字符串最后一段的数字值
// OID 格式: "rtuID-deviceID-tableNo-gin"，如 "0-0-0-1"
// 返回 gin 的数值，用于排序；解析失败返回 0
func lastOIDSegment(oid string) uint64 {
	idx := strings.LastIndex(oid, "-")
	if idx == -1 || idx == len(oid)-1 {
		return 0
	}
	val, err := strconv.ParseUint(oid[idx+1:], 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// getPointsAndDBMsg 获取所有测点
func getPointsAndDBMsg(rid, did, ptype string) (map[string]string, []string, *cgolibparse.DBMsgManager, error) {
	// 获取所有测点id和name
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("GetPointsData GetDbConfigManager failed:", err)
	}
	idNameMap, err := dbcfg.GetPointsIDFromDeviceByType(rid, did, ptype)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("GetPointsData GetPointsFromDevice failed:", err)
	}
	ids := make([]string, 0, len(idNameMap))
	for id := range idNameMap {
		ids = append(ids, id)
	}
	// 按 OID 最后一段数字（gin）排序
	sort.Slice(ids, func(i, j int) bool {
		return lastOIDSegment(ids[i]) < lastOIDSegment(ids[j])
	})

	// 获取前置数据管理类
	dbMsg, err := cgolibparse.GetDBMsgManagerInstance()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("GetPointsData GetDBMsgManagerInstance failed:", err)
	}
	return idNameMap, ids, dbMsg, nil
}

// getRegisterPointsID 获取待注册的测点id
func getRegisterPointsID(idNameMap map[string]string, ids, registerIDs []string, cmdData []byte) (int, []string, error) {
	var cmd CMD
	if err := json.Unmarshal(cmdData, &cmd); err != nil {
		return 0, nil, fmt.Errorf("invalid command:", err)
	}

	newRegisterIDs := make([]string, 0)
	switch cmd.Code {
	case 1: // 直接分页
		stratIdx := cmd.Page * cmd.PageSize
		for i := stratIdx; i < stratIdx+cmd.PageSize && i < len(ids); i++ {
			id := ids[i]
			newRegisterIDs = append(newRegisterIDs, id)
		}
	case 2: // 模糊查询
		for id, name := range idNameMap {
			if strings.Contains(name, cmd.Name) {
				newRegisterIDs = append(newRegisterIDs, id)
			}
		}
	case 3: // 按照上次查询结果分页
		stratIdx := cmd.Page * cmd.PageSize
		for i := stratIdx; i < stratIdx+cmd.PageSize && i < len(registerIDs); i++ {
			id := registerIDs[i]
			newRegisterIDs = append(newRegisterIDs, id)
		}
	}
	return cmd.Code, newRegisterIDs, nil
}

// @Summary		[websocket]获取指定设备下的指定类型的点数据
// @Description	[websocket]获取指定设备下的指定类型的点数据
// @ID				datamonitor-get-points-data
// @Tags			datamonitor
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid				path		string				true	"设备所属的rtu id"	default(<rtu id>)
// @Param			did				path		string				true	"设备id"	default(<device id>)
// @Param			ptype			path		string				true	"点类型 analog/digital/pulse/regulate/control"
// @Success		200				{object}	[]cgolibparse.PointValue	"获取成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/datamonitor/rtu/{rid}/device/{did}/points/{ptype} [get]

// formatAnalogValues 对 analog 类型的 PointValue 副本中 rawvalue/currvalue 保留三位小数
// 返回格式化后的新切片，不修改原始数据
func formatAnalogValues(values []cgolibparse.PointValue) []cgolibparse.PointValue {
	result := make([]cgolibparse.PointValue, len(values))
	copy(result, values)
	for i := range result {
		result[i].RawValue = roundToDecimal(result[i].RawValue, 3)
		result[i].CurrentValue = roundToDecimal(result[i].CurrentValue, 3)
	}
	return result
}

// roundToDecimal 将数值字符串保留 n 位小数，解析失败则原样返回
func roundToDecimal(s string, n int) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return fmt.Sprintf("%."+strconv.Itoa(n)+"f", f)
}

func GetPointsData(c *gin.Context) {
	rid := c.Param("rid")
	did := c.Param("did")
	ptype := c.Param("ptype")
	log.Info("GetPointsData in ... rid:", rid, " did:", did, " ptype:", ptype)

	conn, err := dataUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("GetPointsData Upgrade failed:", err.Error())
		return
	}
	defer conn.Close()

	// 创建一个带取消的 context，用于通知所有 goroutine 退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	idNameMap, allIDs, dbMsg, err := getPointsAndDBMsg(rid, did, ptype)
	if err != nil {
		log.Errorf(err.Error())
		conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// 给前端推送总数
	conn.WriteJSON(CMD{Code: 0, Total: len(allIDs)})

	// 用于安全地存储和切换 getValue 函数
	var valueFunc atomic.Value // 存储 func() []PointValue 类型
	// 存储当前 unRegister，以便切换时先清理旧订阅
	var unRegisterMu sync.Mutex
	var currentUnRegister func()
	var registerIDs []string

	registerPointsFunc := func(registerIDs []string) error {
		// 注册待查询的点位
		if getValue, unRegister, err := dbMsg.RegisterPointsID(idNameMap, registerIDs); err != nil {
			return err
		} else {
			// 清理旧的订阅
			unRegisterMu.Lock()
			if currentUnRegister != nil {
				currentUnRegister()
			}
			currentUnRegister = unRegister
			unRegisterMu.Unlock()

			// 更新 getValue 函数
			valueFunc.Store(getValue)
			return nil
		}
	}

	onMessageFunc := func(messageType int, data []byte) {
		// 只处理文本消息，解析为控制指令
		if messageType != websocket.TextMessage {
			return
		}

		var code int
		code, registerIDs, err = getRegisterPointsID(idNameMap, allIDs, registerIDs, data)
		if err != nil {
			conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
			log.Error("getRegisterPointsID failed:", err)
			return
		}
		if code == 2 {
			conn.WriteJSON(CMD{Code: 0, Total: len(registerIDs)})
		} else {
			// 根据指令重新注册点集
			if err := registerPointsFunc(registerIDs); err != nil {
				conn.WriteJSON(models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
				log.Error("registerPoints failed:", err)
			}
		}
	}

	// 启动读消息 goroutine
	startWSReader(ctx, conn, cancel, onMessageFunc, 0)

	// 启动写 goroutine，每秒推送
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				getValue, ok := valueFunc.Load().(func() []cgolibparse.PointValue)
				if !ok || getValue == nil {
					continue
				}
				pointValues := getValue()

				// analog 类型：rawvalue/currvalue 保留三位小数（拷贝后格式化，不修改原始数据）
				if ptype == "analog" {
					pointValues = formatAnalogValues(pointValues)
				}

				if err := conn.WriteJSON(pointValues); err != nil {
					log.Error("WS write error:", err)
					cancel() // 通知其他 goroutine 退出
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 主 goroutine 阻塞等待退出信号（由 reader 的 cancel 触发）
	<-ctx.Done()
	// 清理最后的订阅
	unRegisterMu.Lock()
	if currentUnRegister != nil {
		currentUnRegister()
	}
	unRegisterMu.Unlock()
}

type SetPointReq struct {
	Value    float32 `json:"value"`
	IsFastYt int     `json:"fastyt"`
	Checkpwd string  `json:"checkpwd"`
}

// @Summary		遥控下发
// @Description	遥控下发
// @ID				datamonitor-set-point-control
// @Tags			datamonitor
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid				path		string				true	"设备所属的rtu id"	default(<rtu id>)
// @Param			did				path		string				true	"设备id"	default(<device id>)
// @Param			pid				path		string				true	"点id"	default(<point id>)
// @Param			req				body		SetPointReq		true	"请求参数 value 1 是控合，0是控分；checkpwd 验证密码；fastyt忽略"
// @Success		200				{object}	models.CommonResp	"设置成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"密码验证失败 {\"code\": 403, \"msg\": \"pwd invalid\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/datamonitor/rtu/{rid}/device/{did}/point/{pid}/control [post]
func SetPointControl(c *gin.Context) {
	rid := c.Param("rid")
	did := c.Param("did")
	pid := c.Param("pid")
	log.Info("SetPointControl in ... rid:", rid, " did:", did, " pid:", pid)
	var req SetPointReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("SetPointControl bind json failed:", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "SetPointControl bind json failed"})
		return
	}

	uName := c.GetString(middleware.UNameKey)
	if data, ok := OnlineUsers.Load(uName); ok {
		online, match := data.(*OnlineUser)
		if !match || req.Checkpwd != online.Pwd {
			log.Error("SetPointControl reject. user pwd invalid, remoteReq.Checkpwd:", req.Checkpwd)
			c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "pwd invalid"})
			return
		}
	}

	if req.Value == 0 {
		err := cgolibparse.SendYkCmd(pid, cgolibparse.ScadaControlExecute, cgolibparse.ScadaActionOff)
		if err != nil {
			log.Error("SetPointControl SendYkCmd failed:", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "SetPointControl failed"})
			return
		}
	} else {
		err := cgolibparse.SendYkCmd(pid, cgolibparse.ScadaControlExecute, cgolibparse.ScadaActionOn)
		if err != nil {
			log.Error("SetPointControl SendYkCmd failed:", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "SetPointControl failed"})
			return
		}
	}
	log.Info("SetPointControl success rid:", rid, " did:", did, " pid:", pid)
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "SetPointControl success"})
}

// @Summary		遥调下发
// @Description	遥调下发
// @ID				datamonitor-set-point-regulate
// @Tags			datamonitor
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid				path		string				true	"设备所属的rtu id"	default(<rtu id>)
// @Param			did				path		string				true	"设备id"	default(<device id>)
// @Param			pid				path		string				true	"点id"	default(<point id>)
// @Param			req				body		SetPointReq		true	"请求参数  value 是遥调值；fastyt 1快速遥调，0普通遥调；checkpwd 验证密码"
// @Success		200				{object}	models.CommonResp	"设置成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"密码验证失败 {\"code\": 403, \"msg\": \"pwd invalid\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"
// @Router			/api/v1/log/datamonitor/rtu/{rid}/device/{did}/point/{pid}/regulate [post]
func SetPointRegulate(c *gin.Context) {
	log.Info("SetPointRegulate in ...")
	rid := c.Param("rid")
	did := c.Param("did")
	pid := c.Param("pid")
	log.Info("SetPointRegulate in ... rid:", rid, " did:", did, " pid:", pid)
	var req SetPointReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("SetPointRegulate bind json failed:", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "SetPointRegulate bind json failed"})
		return
	}

	uName := c.GetString(middleware.UNameKey)
	if data, ok := OnlineUsers.Load(uName); ok {
		online, match := data.(*OnlineUser)
		if !match || req.Checkpwd != online.Pwd {
			log.Error("SetPointSetPointRegulateControl reject. user pwd invalid, remoteReq.Checkpwd:", req.Checkpwd)
			c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "pwd invalid"})
			return
		}
	}

	err := cgolibparse.SendYtCmd(pid, cgolibparse.ScadaControlExecute, req.Value, req.IsFastYt)
	if err != nil {
		log.Error("SetPointRegulate SendYtCmd failed:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "SetPointRegulate failed"})
		return
	}
	log.Info("SetPointRegulate success rid:", rid, " did:", did, " pid:", pid)
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "SetPointRegulate success"})
}
