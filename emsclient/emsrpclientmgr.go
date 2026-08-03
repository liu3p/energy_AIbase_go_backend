package emsclient

import (
	"sync"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/goccy/go-json"

	"github.com/gorilla/websocket"
)

const (
	QueryGatewayMethod    = 100
	PushMethod            = 1000
	PushRedisMethod       = 1001
	PushRemoteMethod      = 1002
	PushLogDumpTaskMethod = 1003
)

func NewManager( /*tryEndpoints []string*/ ) (*Manager, error) {
	emsClientMgr := &Manager{
		// tryEndpoints: make([]string, 0),
		// connections:  make(map[string]*EmsClient, 0),
		// gwInfos:      make(map[string]*models.GatewayInfo),
		pushRespChan: make(chan *PushResp, 1024), // set a buf for concurrent push
	}

	//emsClientMgr.tryEndpoints = append(emsClientMgr.tryEndpoints, tryEndpoints...)

	go emsClientMgr.RunPushMsg()
	//emsClientMgr.initConnections()

	return emsClientMgr, nil
}

//nolint:govet
type Manager struct {
	//tryEndpoints []string
	//connections  map[string]*EmsClient // [gatewaySn] client

	//gwInfos map[string]*models.GatewayInfo

	//stop atomic.Bool

	clientConns sync.Map // 就地app客户端连接

	RedisReq sync.Map // 暂存redis 请求

	pushRespChan chan *PushResp
}

// if same uname existed. old one will be replaced
func (mgr *Manager) PushConn(uname string, conn *websocket.Conn) {
	mgr.clientConns.Store(uname, conn)
}

func (mgr *Manager) PopConn(uname string) {
	mgr.clientConns.Delete(uname)
}

/*
	func (mgr *Manager) initConnections() {
		var gateways map[string]GatewayInfo
		for { // get gateways util success
			for _, endpoint := range mgr.tryEndpoints {
				client := NewEmsClient(mgr, endpoint, 10)
				err := client.ConnectOnce()
				if err != nil {
					log.Error("connect tryendpoint:", endpoint, " failed:", err.Error())
					continue
				}

				resp, err := client.QueryGateway()
				if err != nil {
					log.Error("queryGateway tryendpoint:", endpoint, " failed:", err.Error())
					continue
				}
				log.Debug(">>>>>>>resp:", resp)
				if resp.Code != 0 {
					log.Error("query gateway failed. code:", resp.Code, ", tryendpoint:", endpoint)
					continue
				}

				gateways = resp.Data.Content
				// for sn, gw := range resp.Data.Content {
				// 	gateways[sn] = gw
				// }
				client.Stop()
				break
			}

			if len(gateways) != 0 {
				break
			}
			time.Sleep(5 * time.Second)
			log.Warn("get gateways failed. contuinue...")
		}

		connmaps := make(map[string]*EmsClient) // endpoint: client
		for gwSn, gwInfo := range gateways {
			mgr.gwInfos[gwSn] = &models.GatewayInfo{Sn: gwSn, Name: gwInfo.Name}

			endpoint := gwInfo.Endpoint
			if client, exist := connmaps[endpoint]; exist {
				mgr.connections[gwSn] = client
				continue
			}
			client := NewEmsClient(mgr, endpoint, 10)
			err := client.Connect()
			if err != nil {
				log.Error("connect failed:", endpoint)
				continue
			}
			client.HandlePush()
			mgr.connections[gwSn] = client
			connmaps[endpoint] = client
		}
	}

	func (mgr *Manager) GetClient(gateway string) *EmsClient {
		if client, exist := mgr.connections[gateway]; exist {
			return client
		}
		return nil
	}

	func (mgr *Manager) GetGatewayInfo(gatewaySn string) map[string]*models.GatewayInfo {
		result := make(map[string]*models.GatewayInfo)

		for sn, gw := range mgr.gwInfos {
			gw.Status = 1
			if mgr.connections[sn].RunOK.Load() {
				gw.Status = 2
			}
			mgr.gwInfos[sn] = gw
		}
		if gatewaySn != "" {
			result[gatewaySn] = mgr.gwInfos[gatewaySn]
		}

		return mgr.gwInfos
	}

	func (mgr *Manager) GetAvailableClient() *EmsClient {
		for _, client := range mgr.connections {
			if client.RunOK.Load() {
				return client
			}
		}
		return nil
	}
*/
func (mgr *Manager) PushMsg(resp *PushResp) {
	mgr.pushRespChan <- resp
}

func (mgr *Manager) RunPushMsg() {
	for resp := range mgr.pushRespChan {
		if resp.Topic == "close" {
			uname, ok := resp.Content.(string)
			if !ok {
				log.Error("close push but content not string")
				continue
			}

			value, ok := mgr.clientConns.LoadAndDelete(uname)
			if !ok {
				log.Error("close push but user not exist, user:", uname)
				continue
			}
			conn, ok := value.(*websocket.Conn)
			if !ok {
				log.Error("close push but not websocket.Conn, user:", uname)
				continue
			}
			log.Info("PUSH close message to user:", uname)
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Error("close push WriteMessage failed:", err.Error())
			}
			err = conn.Close()
			if err != nil {
				log.Error("close push Close conn failed:", err.Error())
			}
			log.Info("PUSH close message OVER:", uname)
			continue
		}

		if resp.Topic == "ping" {
			uname, ok := resp.Content.(string)
			if !ok {
				log.Error("ping push but content not string")
				continue
			}
			value, ok := mgr.clientConns.Load(uname)
			if !ok {
				log.Error("ping push but user not exist, user:", uname)
				continue
			}
			conn, ok := value.(*websocket.Conn)
			if !ok {
				log.Error("ping push but not websocket.Conn, user:", uname)
				continue
			}
			err := conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				log.Error("ping push WriteMessage failed:", err.Error(), ", user:", uname)
			}
			continue
		}

		mgr.clientConns.Range(func(key, value interface{}) bool {
			conn, ok := value.(*websocket.Conn)
			if !ok {
				log.Error("not websocket.Conn")
				return true
			}

			msg, err := json.Marshal(resp)
			if err != nil {
				log.Error("Marshal resp failed:", err.Error())
				return true
			}
			// TODO do some message trans
			log.Info("PUSH message to user:", key)
			err = conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Error("WriteMessage resp failed:", err.Error())
			}
			log.Info("PUSH message OVER:", key)
			return true
		})
	}
	log.Error("RunPushMsg Exit !!")
}

type GatewayInfo struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Sn       string `json:"sn"`
}

//nolint:govet
type DataStruct struct {
	Topic   string                 `json:"topic"`
	Content map[string]GatewayInfo `json:"content"`
}

//nolint:govet
type QueryGatewayResp struct {
	// gateway: endpoint
	//Gateways map[string]string `json:"gateways"`
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    DataStruct `json:"data"`
}

func (g *QueryGatewayResp) Marshal() ([]byte, error) {
	return json.Marshal(g)
}
func (g *QueryGatewayResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, g)
}

func (g *QueryGatewayResp) Clone() models.IEmsRPCResp {
	return &QueryGatewayResp{g.Code, g.Message, g.Data}
}

func (e *EmsClient) QueryGateway() (resp *QueryGatewayResp, err error) {
	method := uint32(QueryGatewayMethod)
	resp = &QueryGatewayResp{}
	err = e.Call(method, &models.CommonEmptyReq{}, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

type PushReq struct {
}

func (e *PushReq) Marshal() ([]byte, error) {
	return []byte{}, nil
}

func (e *PushReq) Unmarshal(data []byte) error {
	return nil
}

//nolint:govet
type PushResp struct {
	// gateway: endpoint
	Topic   string      `json:"topic"`
	Content interface{} `json:"content"`
}

func (g *PushResp) Clone() models.IEmsRPCResp {
	return &PushResp{g.Topic, g.Content}
}

func (g *PushResp) clean() {
	g.Topic = ""
	g.Content = ""
}

func (g *PushResp) Marshal() ([]byte, error) {
	return json.Marshal(g)
}

func (g *PushResp) Unmarshal(data []byte) error {
	g.clean()
	log.Debug(">>>>>>> data:", string(data))
	return json.Unmarshal(data, g)
}

func (e *EmsClient) Push(req *PushReq) (resp *PushResp, err error) {
	method := uint32(QueryGatewayMethod)

	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

type PushRedisResp struct {
	// gateway: endpoint
	Topic   string `json:"topic"`
	Content string `json:"content"`
	Seq     int64  `json:"seq"`
}

func (g *PushRedisResp) Clone() models.IEmsRPCResp {
	return &PushRedisResp{g.Topic, g.Content, g.Seq}
}

func (g *PushRedisResp) clean() {
	g.Topic = ""
	g.Content = ""
	g.Seq = 0
}

func (g *PushRedisResp) Marshal() ([]byte, error) {
	return json.Marshal(g)
}

func (g *PushRedisResp) Unmarshal(data []byte) error {
	g.clean()
	log.Debug(">>>>>>> PushRedisResp:", string(data))
	return json.Unmarshal(data, g)
}

type PushRemoteResp struct {
	// gateway: endpoint
	Result int   `json:"result"`
	Seq    int64 `json:"seq"`
}

func (g *PushRemoteResp) clean() {
	g.Result = 0
	g.Seq = 0
}
func (g *PushRemoteResp) Clone() models.IEmsRPCResp {
	return &PushRemoteResp{g.Result, g.Seq}
}

func (g *PushRemoteResp) Marshal() ([]byte, error) {
	return json.Marshal(g)
}

func (g *PushRemoteResp) Unmarshal(data []byte) error {
	g.clean()
	log.Debug(">>>>>>> PushRemoteResp:", string(data))
	return json.Unmarshal(data, g)
}

/*
func (e *EmsClient) HandlePush() {
	pushResp := &PushResp{}
	e.RegistPushHandler(uint32(PushMethod), pushResp, func(resp models.IEmsRPCResp) {
		push, ok := resp.(*PushResp)
		if !ok { // we may not here
			//err = fmt.Errorf()
			log.Errorf("value not PushResp. method: %d", PushMethod)
			return
		}
		log.Info("push:", push)
		e.mgr.PushMsg(push)
	})

	pushRedisResp := &PushRedisResp{}
	e.RegistPushHandler(uint32(PushRedisMethod), pushRedisResp, func(resp models.IEmsRPCResp) {
		push, ok := resp.(*PushRedisResp)
		if !ok { // we may not here
			//err = fmt.Errorf()
			log.Errorf("redis push. value not PushRedisResp. method: %d", PushRedisMethod)
			return
		}
		log.Info("redis push. data:", push)
		entry, ok := e.mgr.RedisReq.LoadAndDelete(push.Seq)
		if !ok {
			log.Errorf("redis push. seq not exist. Seq: %d", push.Seq)
			return
		}
		pushChan, ok := entry.(chan PushRedisResp)
		if !ok {
			log.Errorf("redis push. entry not PushResp. Seq: %d", push.Seq)
			return
		}
		pushChan <- *push
	})

	pushRemoteResp := &PushRemoteResp{}
	e.RegistPushHandler(uint32(PushRemoteMethod), pushRemoteResp, func(resp models.IEmsRPCResp) {
		push, ok := resp.(*PushRemoteResp)
		if !ok { // we may not here
			//err = fmt.Errorf()
			log.Errorf("remote push. value not pushRemoteResp. method: %d", PushRemoteMethod)
			return
		}
		log.Info("remote push. data:", push)
		entry, ok := e.mgr.RedisReq.LoadAndDelete(push.Seq)
		if !ok {
			log.Errorf("remote push. seq not exist. Seq: %d", push.Seq)
			return
		}
		pushChan, ok := entry.(chan PushRemoteResp)
		if !ok {
			log.Errorf("remote push. entry not PushResp. Seq: %d", push.Seq)
			return
		}
		pushChan <- *push
	})

	e.RegistPushHandler(uint32(PushLogDumpTaskMethod), pushRedisResp, func(resp models.IEmsRPCResp) {
		taskInfo, ok := resp.(*PushRedisResp)
		if !ok { // we may not here
			//err = fmt.Errorf()
			log.Errorf("remote push. value not pushRedisResp. method: %d", PushLogDumpTaskMethod)
			return
		}
		log.Info("recv logdump task push:", taskInfo.Seq)
		go func(taskData string) {
			task := &PushLogDumpTask{}
			err := json.Unmarshal([]byte(taskData), task)
			if err != nil {
				log.Error("Unmarshal task data failed:", err.Error(), ",data:", taskData)
				return
			}

			defer func(task *PushLogDumpTask) {
				pushResp := &PushResp{Topic: "logdump"}
				pushResp.Content = PushLogDumpResp{TaskID: task.TaskID, Result: task.Result}
				e.mgr.PushMsg(pushResp)
			}(task)

			if task.Result == 2 {
				// artificial stoped
				log.Warn("task ", task.TaskID, " result is 2, conflict stoped.")
				return
			}

			localPath := models.LogDumpLocalPath(task.TaskID)
			log.Info("recv PushLogDumpTask. taskid:", task.TaskID, ", result:", task.Result, ", fromIp:", task.HostIP,
				", remotePath:", task.LogPath, ", localPath:", localPath)

			fetcher, err := NewSFTPFileFetcher(task.HostIP, models.LogDumpUser, models.LogDumpPwd)
			if err != nil {
				log.Error("NewSFTPFileFetcher for task ", task.TaskID, " failed:", err.Error())
				task.Result = 3
				return
			}
			defer fetcher.Close()

			err = fetcher.CopyRemoteToLocal(task.LogPath, localPath)
			if err != nil {
				log.Error("CopyRemoteToLocal for task ", task.TaskID, " failed:", err.Error())
				task.Result = 3
				return
			}

			err = fetcher.CopyLocalFileToDir(log.LogFile, localPath)
			if err != nil {
				log.Error("CopyLocalFileToDir for task ", task.TaskID, " failed:", err.Error())
				task.Result = 1
				return
			}

			log.Info("CopyRemoteToLocal and CopyLocalFileToDir for task", task.TaskID, " OK.")
			targetPkgPath := models.LogDumpLocalPkgFile(task.TaskID)

			err = utils.TarGzDirectory(localPath, targetPkgPath, func(filePath string, fileInfo os.FileInfo) bool {
				return true
			})
			if err != nil {
				log.Error("tarGzDirectory for task ", task.TaskID, " failed:", err.Error())
				task.Result = 3
				return
			}

			log.Debug("tarGzDirectory OK, remove localPath:", localPath)
			err = os.RemoveAll(localPath)
			if err != nil {
				log.Error("RemoveAll ", localPath, " for task ", task.TaskID, " failed:", err.Error())
			}

			// remove last 10 files
			log.Debug("remove localPath OK, deleteOldFiles keep latest 10 files in ", models.LogDumpPath)
			err = deleteOldFiles(models.LogDumpPath, 10)
			if err != nil {
				log.Error("deleteOldFiles failed:", err.Error())
			}
		}(taskInfo.Content)
	})
}
*/
