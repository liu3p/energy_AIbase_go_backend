package handlers

import (
	"net/http"
	"time"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
)

const (
	LwSysQuerySysInfo  = "QUERY_SYSINFO"
	LwSysQueryProcInfo = "QUERY_PROCINFO"
	LwSysCtrlProc      = "CTRL_PROC"
	LwSysCtrlSys       = "CTRL_SYS"
	LwSysGetLog        = "GET_LOG"
)

func GetSeq() int64 {
	return time.Now().UnixMilli()
}

type RdData struct {
	Content string `json:"content"`
}

/*  redis直接查不到发布与订阅数据，暂不实现
// @Summary		定制查询
// @Description	定制查询接口
// @ID				rd-query
// @Tags			ems
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			method			path		string				true	"方法名"
// @Success		200				{object}	RdData				"查询成功 {\"content\": \"test\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/rd_query/{method} [get]
func RdQuery(c *gin.Context) {
	log.Info("RdQuery in ...")
	method := c.Param("method")

	// TODO  query redis

	log.Debug("method:", method)
	c.JSON(http.StatusOK, RdData{Content: "testdata"})
}*/

// @Summary		定制转发
// @Description	定制转发接口
// @ID				rd-post
// @Tags			ems
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			method			path		string				true	"方法名"
// @Param			RdData			body		RdData				true	"转发消息体"
// @Success		200				{object}	models.CommonResp	"成功 {\"code\": 200, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/rd/{method} [post]
func RdPost(c *gin.Context) {
	log.Info("RdPost in ...")
	method := c.Param("method")
	rdPost := &RdData{}
	err := c.ShouldBindJSON(rdPost)
	if err != nil {
		log.Error("ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	log.Info("method:", method, ",rdReq:", rdPost)
	/*
	   client := ClientMgr.GetAvailableClient()

	   	if client == nil {
	   		log.Error("no client available")
	   		loginResp := &models.CommonResp{Code: http.StatusServiceUnavailable, Message: "Service Unavailable"}
	   		c.JSON(http.StatusServiceUnavailable, loginResp)
	   		return
	   	}

	   seq := GetSeq()
	   respChan := make(chan emsclient.PushResp)
	   ClientMgr.RedisReq.Store(seq, respChan)
	   req := &emsclient.RedisCallReq{Method: method, Payload: rdPost.Content, Seq: seq}
	   resp, err := client.RedisCall(req)

	   	if err != nil {
	   		log.Error("RedisCall err:", err.Error())
	   		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	   			Message: "Internal Service Error"})
	   		return
	   	}

	   log.Info("RedisCall resp:", resp, ",seq:", seq)

	   select {
	   case resp := <-respChan:

	   	c.JSON(http.StatusOK, resp)
	   	return

	   case <-time.After(time.Second * 10):

	   		log.Error("10 second timeout")
	   		break
	   	}

	   c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,

	   	Message: "Internal Service Error"})

	   //c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
	*/
	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}

type LogDumpReq struct {
	GatewaySn string `json:"gatewaysn"`
	//DumpDuration int    `json:"duration"`
	Operation int   `json:"operation"`
	TaskID    int64 `json:"taskid"`
}

type LogDumpResp struct {
	TaskID int64 `json:"taskid"`
}

// @Summary		启动/停止收集日志(localapp)
// @Description	控制收集日志的启停。返回内容taskid是收集日志任务执行的唯一标识。收集完成（或者出错）会有相应的推送消息，topic类型：logdump。收集成功后，可以通过GET /api/v1/log/rd/logdump/{taskid}来下载收集的日志文件。
// @ID				log-task-post
// @Tags			system
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			LogDumpReq		body		LogDumpReq			true	"请求消息体 operation：操作 1 开始收集， 2 停止收集；taskid：停止日志（operation为2）的时候，需要传递taskid；gatewaysn：网关sn，用于指定收集日志的机柜。"
// @Success		200				{object}	LogDumpResp			"成功 {\"taskid\": 1708481804673}。 执行完成后会推送：{\"topic\":\"logdump\",\"content\":{\"taskid\":1708481804673,\"result\":0}}, 其中result： 0 成功， 1 生成（部分）失败， 2 因冲突停止 3 失败，0和1的情况可以通过taskid拉取到文件包。"
// @Failure		400				{object}	models.CommonResp	"参数错误 {\"code\": 400, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/rd/logdump [post]
func LogDumpPost(c *gin.Context) {
	dumpReq := &LogDumpReq{}
	err := c.ShouldBindJSON(dumpReq)
	if err != nil {
		log.Error("ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}

	log.Debug("LogDumpPost in ... dumpReq:", dumpReq)
	/*
	   	if dumpReq.Operation != 1 && dumpReq.Operation != 2 {
	   		failedResp := &models.CommonResp{Code: http.StatusBadRequest,
	   			Message: "params error, 'operation' shoule be 1(start logdump) or 2(stop logdump)"}
	   		c.JSON(http.StatusBadRequest, failedResp)
	   		return
	   	}

	   seq := GetSeq()
	   taskid := seq + 1
	   stopOper := 0

	   	if dumpReq.Operation == 2 {
	   		stopOper = 1
	   		taskid = dumpReq.TaskID
	   		if taskid == 0 {
	   			failedResp := &models.CommonResp{Code: http.StatusBadRequest,
	   				Message: "params error, 'operation' is 2(stop) but taskid empty"}
	   			c.JSON(http.StatusBadRequest, failedResp)
	   			return
	   		}
	   	}

	   client := ClientMgr.GetClient(dumpReq.GatewaySn)

	   	if client == nil {
	   		log.Error("LogDumpPost no client available:", dumpReq.GatewaySn)
	   		failedResp := &models.CommonResp{Code: http.StatusNotFound, Message: "gateway not found"}
	   		c.JSON(http.StatusNotFound, failedResp)
	   		return
	   	}

	   hostIP := strings.Split(client.GetServerIP(), ":")[0]
	   respChan := make(chan emsclient.PushRedisResp)

	   ClientMgr.RedisReq.Store(seq, respChan)
	   // default 5min
	   req := &emsclient.RedisCallLogDumpReq{Seq: seq, TaskID: taskid, Duration: 300, HostIP: hostIP, Stop: stopOper}
	   callResp, err := client.RedisCallLogDump(req)

	   	if err != nil {
	   		log.Error("LogDumpPost RedisCallHistory err:", err.Error())
	   		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	   			Message: "Internal Service Error"})
	   		return
	   	}

	   log.Info("LogDumpPost resp:", callResp, ",seq:", seq, ",taskid:", taskid)

	   	if callResp.Code != 0 {
	   		log.Error("LogDumpPost RedisCallLogDump failed. code:", callResp.Code)
	   		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	   			Message: "Internal Service Error"})
	   		return
	   	}

	   select {
	   case resp := <-respChan:

	   	result := &LogDumpResp{TaskID: taskid}
	   	err := json.Unmarshal([]byte(resp.Content), &result)
	   	if err != nil {
	   		log.Error("LogDumpPost Unmarshal resp content failed:", err, ",seq:", seq, ",resp:", resp)
	   		break
	   	}

	   	log.Info("LogDumpPost result:", result, ",seq:", seq)
	   	c.JSON(http.StatusOK, result)
	   	return

	   case <-time.After(time.Second * 10):

	   		log.Error("LogDumpPost 10 second timeout")
	   		break
	   	}

	   c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,

	   	Message: "Internal Service Error"})
	*/
	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}

// @Summary		获取收集的日志(localapp)
// @Description	获取收集的日志
// @ID				log-task-get
// @Tags			system
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			taskid			path		string				true	"任务id"
// @Success		200				{object}	models.CommonResp	"成功 {\"code\": 200, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				string		string				"文件不存在"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/rd/logdump/{taskid} [get]
func LogDumpGet(c *gin.Context) {
	log.Info("LogDumpGet in ...")
	taskid := c.Param("taskid")
	pkgFilePath := models.LogDumpLocalPkgFileE(taskid)
	log.Info("LogDumpGet taskid:", taskid, ",pkgFilePath:", pkgFilePath)
	c.File(pkgFilePath)

	//////
	/*
		// Read the content of the result file
		content, err := ioutil.ReadFile(task.FilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		// Return the result file content to the client
		c.Data(http.StatusOK, "text/plain", content)
	*/
	/*
		client := ClientMgr.GetAvailableClient()
		if client == nil {
			log.Error("MqttIdHis no client available")
			loginResp := &models.CommonResp{Code: http.StatusServiceUnavailable, Message: "Service Unavailable"}
			c.JSON(http.StatusServiceUnavailable, loginResp)
			return
		}

		seq := GetSeq()
		content, err := json.Marshal(hisReq.Content)
		if err != nil {
			log.Error("MqttIdHis Marshal hisReq content failed:", err.Error())
			c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "content invalid"})
			return
		}

		respChan := make(chan emsclient.PushRedisResp)
		ClientMgr.RedisReq.Store(seq, respChan)
		req := &emsclient.RedisCallHisReq{Mqttid: mqttid, Payload: string(content), Seq: seq}
		resp, err := client.RedisCallHistory(req)
		if err != nil {
			log.Debug("MqttIdHis RedisCallHistory err:", err.Error())
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			return
		}
		log.Info("MqttIdHis resp:", resp, ",seq:", seq)

		select {
		case resp := <-respChan:
			result := &MqttIdHisResp{}
			err := json.Unmarshal([]byte(resp.Content), &result.Content)
			if err != nil {
				log.Error("MqttIdHis Unmarshal resp content failed:", err, ",seq:", seq, ",resp:", resp)
				break
			}
			log.Info("MqttIdHis result:", result, ",seq:", seq)
			c.JSON(http.StatusOK, result)
			return
		case <-time.After(time.Second * 10):
			log.Error("MqttIdHis 10 second timeout")
			break
		}
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
	*/
}

// @Summary		查询进程信息(localapp)
// @Description	查询进程信息
// @ID				proc-get
// @Tags			system
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			taskid			path		string				true	"任务id"
// @Success		200				{object}	models.CommonResp	"成功 {\"code\": 200, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				string		string				"文件不存在"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/rd/proclist/{gatewaysn} [get]
func ProcListGet(c *gin.Context) {
	gatewaysn := c.Param("gatewaysn")

	log.Info("ProcListGet in ... gatewaysn:", gatewaysn)
	/*
	   client := ClientMgr.GetClient(gatewaysn)

	   	if client == nil {
	   		log.Error("ProcListGet no client available:", gatewaysn)
	   		failedResp := &models.CommonResp{Code: http.StatusNotFound, Message: "gateway not found"}
	   		c.JSON(http.StatusNotFound, failedResp)
	   		return
	   	}

	   seq := GetSeq()
	   method := LwSysQueryProcInfo
	   body := ""

	   	req := &emsclient.RedisCallProcReq{
	   		Seq:    seq,
	   		Method: method,
	   		Body:   body,
	   	}

	   respChan := make(chan emsclient.PushRedisResp)
	   ClientMgr.RedisReq.Store(seq, respChan)
	   callResp, err := client.RedisCallProc(req)

	   	if err != nil {
	   		log.Error("ProcListGet RedisCallProc err:", err.Error())
	   		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	   			Message: "Internal Service Error"})
	   		return
	   	}

	   log.Info("ProcListGet resp:", callResp, ",seq:", seq)

	   	if callResp.Code != 0 {
	   		log.Error("ProcListGet RedisCallProc failed. code:", callResp.Code)
	   		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
	   			Message: "Internal Service Error"})
	   		return
	   	}

	   select {
	   case resp := <-respChan:

	   	var result interface{}
	   	err := json.Unmarshal([]byte(resp.Content), &result)
	   	if err != nil {
	   		log.Error("ProcListGet Unmarshal resp content failed:", err, ",seq:", seq, ",resp:", resp)
	   		break
	   	}

	   	log.Info("ProcListGet result:", result, ",seq:", seq)
	   	c.JSON(http.StatusOK, result)
	   	return

	   case <-time.After(time.Second * 10):

	   		log.Error("ProcListGet 10 second timeout")
	   		break
	   	}

	   c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,

	   	Message: "Internal Service Error"})
	*/
	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}

//nolint:govet,tagliatelle
type ProcListReq struct {
	ProcPid  int    `json:"ProcPid"`
	ProcName string `json:"ProcName"`
	Action   string `json:"Action"`
}

// @Summary		进程管理控制(localapp)
// @Description	进程管理控制
// @ID				proc-post
// @Tags			system
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			taskid			path		string				true	"任务id"
// @Success		200				{object}	models.CommonResp	"成功 {\"code\": 200, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		404				string		string				"文件不存在"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/rd/proclist/{gatewaysn} [post]
func ProcListPost(c *gin.Context) {
	log.Info("ProcListPost in ...")
	/*
		gatewaysn := c.Param("gatewaysn")

		proclistReq := &ProcListReq{}
		err := c.ShouldBindJSON(proclistReq)
		if err != nil {
			log.Error("ShouldBindJSON err:", err.Error())
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			return
		}
		body, err := json.Marshal(proclistReq)
		if err != nil {
			log.Error("ProcListPost marshal failed:", err.Error())
			return
		}

		log.Info("ProcListPost gatewaysn:", gatewaysn, ",body:", body)

		client := ClientMgr.GetClient(gatewaysn)
		if client == nil {
			log.Error("ProcListPost no client available:", gatewaysn)
			failedResp := &models.CommonResp{Code: http.StatusNotFound, Message: "gateway not found"}
			c.JSON(http.StatusNotFound, failedResp)
			return
		}

		seq := GetSeq()
		method := LwSysCtrlProc
		req := &emsclient.RedisCallProcReq{
			Seq:    seq,
			Method: method,
			Body:   string(body),
		}

		respChan := make(chan emsclient.PushRedisResp)
		ClientMgr.RedisReq.Store(seq, respChan)
		callResp, err := client.RedisCallProc(req)
		if err != nil {
			log.Error("ProcListPost RedisCallProc err:", err.Error())
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			return
		}
		log.Info("ProcListPost resp:", callResp, ",seq:", seq)

		if callResp.Code != 0 {
			log.Error("ProcListPost RedisCallProc failed. code:", callResp.Code)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			return
		}

		select {
		case resp := <-respChan:
			log.Info("ProcListPost OK ,seq:", seq)
			if resp.Topic == "failed" {
				log.Error("ProcListPost push err. content:", resp.Content)
				c.JSON(http.StatusInternalServerError, resp.Content)
				return
			}
			c.JSON(http.StatusOK, resp.Content)
			return
		case <-time.After(time.Second * 10):
			log.Error("ProcListPost 10 second timeout")
			break
		}
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})*/

	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: ""})
}
