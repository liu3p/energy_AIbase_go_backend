package systeminfo

import (
	"encoding/json"
	"fmt"
	"time"
	"webserver-arm/log"
	"webserver-arm/zmq"
)

const (
	ZMQ_OP_START   = "start"
	ZMQ_OP_STOP    = "stop"
	ZMQ_OP_RESTART = "restart"
)

type ZMQProcessInfo struct {
	CpuUsage     float32 `json:"cpu_usage"`
	Id           int     `json:"id"`
	MemUsage     float32 `json:"memory_usage_percent"`
	MemUsed      float32 `json:"memory_usage_mb"`
	Name         string  `json:"name"`
	Pid          string  `json:"pid"`
	RestartCount int     `json:"restart_count"`
	Status       string  `json:"status"`
	Threads      int     `json:"threads"`
	Uptime       string  `json:"uptime"`
	Warning      string  `json:"warning"`
}

type ZMQDisk struct {
	Device       string  `json:"device"`
	FreeGB       float32 `json:"free_gb"`
	FsType       string  `json:"fs_type"`
	MountPoint   string  `json:"mount_point"`
	TotalGB      float32 `json:"total_gb"`
	UsagePercent float32 `json:"usage_percent"`
	UsedGB       float32 `json:"used_gb"`
}

type ZMQNet struct {
	Broadcast    string `json:"broadcast"`
	Carrier      string `json:"carrier"`
	Description  string `json:"description"`
	Dns1         string `json:"dns1"`
	Dns2         string `json:"dns2"`
	Duplex       string `json:"duplex"`
	Gateway      string `json:"gateway"`
	Iccid        string `json:"iccid"`
	Imei         string `json:"imei"`
	Imsi         string `json:"imsi"`
	Ipv4         string `json:"ipv4"`
	Mac          string `json:"mac"`
	Mtu          string `json:"mtu"`
	Name         string `json:"name"`
	Netmask      string `json:"netmask"`
	OperatorName string `json:"operator_name"`
	Registration string `json:"registration"`
	Signal       string `json:"signal"`
	Speed        string `json:"speed"`
	UpSpeed      string `json:"tx_rate_mbps"`
	DownSpeed    string `json:"rx_rate_mbps"`
	Status       string `json:"status"`
	Technology   string `json:"technology"`
	Type         string `json:"type"`
}

type ZMQBaseSystemInfo struct {
	Disks    []ZMQDisk `json:"disk"`
	Nets     []ZMQNet  `json:"net"`
	Cpu      float32   `json:"system_cpu"`
	CpuFreq  float32   `json:"system_cpu_freq"`
	MemTotal float32   `json:"system_mem_total"`
	MemUsage float32   `json:"system_mem_usage"`
	MemUsed  float32   `json:"system_mem_used"`
}

type ZMQSystemInfo struct {
	System   ZMQBaseSystemInfo `json:"system"`
	Services []ZMQProcessInfo  `json:"services"`
}

// =========================================
type CpuInfo struct {
	CpuUsage float32 `json:"cpuusage"`
	CpuFreq  float32 `json:"cpufreq"`
}

type MemInfo struct {
	MemUsed  float32 `json:"memused"`
	MemTotal float32 `json:"memtotal"`
	MemUsage float32 `json:"memusage"`
}

type DiskInfo struct {
	DiskUsed  float32 `json:"diskused"`
	DiskTotal float32 `json:"disktotal"`
}
type Net struct {
	Ipv4      string `json:"ipv4"`
	Mac       string `json:"mac"`
	Name      string `json:"name"`
	UpSpeed   string `json:"upspeed"`
	DownSpeed string `json:"downspeed"`
	NetType   string `json:"type"`
}

type SystemInfo struct {
	Cpu  CpuInfo    `json:"cpu"`
	Mem  MemInfo    `json:"mem"`
	Disk []DiskInfo `json:"disk"`
	Nets []Net      `json:"nets"`
}

// ==============================
type ProcessInfo struct {
	Id       int     `json:"id"`
	Name     string  `json:"name"`
	CpuUsage float32 `json:"cpuusage"`
	MemUsed  float32 `json:"memused"`
	Status   string  `json:"status"` // Stopped,Running
}

// func GetSystemInfo() (SystemInfo, error) {
// 	clientID := fmt.Sprintf("CallSystemInfo_%d", time.Now().UnixMilli())
// 	zmqClient, err := zmq.NewZmqClient(ZMQ_SYSTEM_INFO_ENDPOINT, clientID)
// 	if err != nil {
// 		return SystemInfo{}, fmt.Errorf("create zmq client failed: %w", err)
// 	}

// 	result, err := zmqClient.SimpleGetMsg("status")
// 	if err != nil {
// 		return SystemInfo{}, fmt.Errorf("send status to server failed: %w", err)
// 	}

// 	// TODO get system info
// 	zmqSysData := result[0]
// 	var zmqSysInfo ZMQSystemInfo
// 	err = json.Unmarshal([]byte(zmqSysData), &zmqSysInfo)
// 	if err != nil {
// 		return SystemInfo{}, fmt.Errorf("unmarshal system info failed: %w", err)
// 	}

// 	sysInfo := SystemInfo{}
// 	sysInfo.Cpu.CpuUsage = zmqSysInfo.System.Cpu
// 	sysInfo.Cpu.CpuSpeed = zmqSysInfo.System.CpuSpeed
// 	sysInfo.Mem.MemUsage = zmqSysInfo.System.MemUsage
// 	sysInfo.Mem.MemTotal = zmqSysInfo.System.MemTotal
// 	sysInfo.Mem.MemUsed = zmqSysInfo.System.MemUsed
// 	sysInfo.Disk = make([]DiskInfo, len(zmqSysInfo.System.Disks))
// 	for i, disk := range zmqSysInfo.System.Disks {
// 		sysInfo.Disk[i].DiskTotal = disk.TotalGB
// 		sysInfo.Disk[i].DiskUsed = disk.UsedGB
// 	}

// 	sysInfo.Nets = make([]Net, len(zmqSysInfo.System.Nets))
// 	for i, net := range zmqSysInfo.System.Nets {
// 		sysInfo.Nets[i].Ipv4 = net.Ipv4
// 		sysInfo.Nets[i].Mac = net.Mac
// 		sysInfo.Nets[i].Name = net.Name
// 		sysInfo.Nets[i].UpSpeed = net.UpSpeed
// 		sysInfo.Nets[i].DownSpeed = net.DownSpeed
// 		sysInfo.Nets[i].NetType = net.Type // wired, wireless, cellular
// 	}
// 	return sysInfo, nil
// }

func PushSystemInfo(onMessage func(data []byte), stopChan <-chan struct{}) error {
	clientID := fmt.Sprintf("CallSystemInfo_%d", time.Now().UnixMilli())
	topic := "status"
	return SubscribeSystemInfo(clientID, topic, func(data string) {
		log.Info("PushSystemInfo SubscribeSystemInfo get message size:", len(data))
		var zmqSysInfo ZMQSystemInfo
		err := json.Unmarshal([]byte(data), &zmqSysInfo)
		if err != nil {
			log.Error("PushSystemInfo unmarshal system info failed:", err)
			return
		}

		sysInfo := SystemInfo{}
		sysInfo.Cpu.CpuUsage = zmqSysInfo.System.Cpu
		sysInfo.Cpu.CpuFreq = zmqSysInfo.System.CpuFreq
		sysInfo.Mem.MemUsage = zmqSysInfo.System.MemUsage
		sysInfo.Mem.MemTotal = zmqSysInfo.System.MemTotal
		sysInfo.Mem.MemUsed = zmqSysInfo.System.MemUsed
		sysInfo.Disk = make([]DiskInfo, len(zmqSysInfo.System.Disks))
		for i, disk := range zmqSysInfo.System.Disks {
			sysInfo.Disk[i].DiskTotal = disk.TotalGB
			sysInfo.Disk[i].DiskUsed = disk.UsedGB
		}

		sysInfo.Nets = make([]Net, len(zmqSysInfo.System.Nets))
		for i, net := range zmqSysInfo.System.Nets {
			sysInfo.Nets[i].Ipv4 = net.Ipv4
			sysInfo.Nets[i].Mac = net.Mac
			sysInfo.Nets[i].Name = net.Name
			sysInfo.Nets[i].UpSpeed = net.UpSpeed
			sysInfo.Nets[i].DownSpeed = net.DownSpeed
			sysInfo.Nets[i].NetType = net.Type // wired, wireless, cellular
		}

		outData, err := json.Marshal(sysInfo)
		if err != nil {
			log.Error("PushSystemInfo marshal system info failed:", err)
			return
		}
		onMessage(outData)
	}, stopChan)
}

func PushProcessInfo(onMessage func(data []byte), stopChan <-chan struct{}) error {
	clientID := fmt.Sprintf("CallProcessInfo_%d", time.Now().UnixMilli())
	topic := "status"
	return SubscribeSystemInfo(clientID, topic, func(data string) {
		log.Debug("PushProcessInfo SubscribeSystemInfo get message size:", len(data))
		var zmqSysInfo ZMQSystemInfo
		err := json.Unmarshal([]byte(data), &zmqSysInfo)
		if err != nil {
			log.Error("PushProcessInfo unmarshal system info failed:", err)
			return
		}

		processInfo := make([]ProcessInfo, len(zmqSysInfo.Services))
		for i, process := range zmqSysInfo.Services {
			processInfo[i].Id = process.Id
			processInfo[i].Name = process.Name
			processInfo[i].CpuUsage = process.CpuUsage
			processInfo[i].MemUsed = process.MemUsed
			processInfo[i].Status = process.Status
		}

		outData, err := json.Marshal(processInfo)
		if err != nil {
			log.Error("PushProcessInfo marshal system info failed:", err)
			return
		}
		onMessage(outData)
	}, stopChan)
}

func PushProcessLog(processID string, onMessage func(data []byte), stopChan <-chan struct{}) error {
	clientID := fmt.Sprintf("CallProcessLog_%d", time.Now().UnixMilli())
	topic := fmt.Sprintf("log_%s", processID)
	return SubscribeSystemInfo(clientID, topic, func(data string) {
		log.Debug("PushProcessLog SubscribeSystemInfo get message size:", len(data))
		// var zmqSysInfo ZMQSystemInfo
		// err := json.Unmarshal([]byte(data), &zmqSysInfo)
		// if err != nil {
		// 	log.Error("WsGetProcessInfo unmarshal system info failed:", err)
		// 	return
		// }

		// processInfo := make([]ProcessInfo, len(zmqSysInfo.Services))
		// for i, process := range zmqSysInfo.Services {
		// 	processInfo[i].Id = process.Id
		// 	processInfo[i].CpuUsage = process.CpuUsage
		// 	processInfo[i].MemUsed = process.MemUsed
		// 	processInfo[i].Status = process.Status
		// }

		// outData, err := json.Marshal(processInfo)
		// if err != nil {
		// 	log.Error("WsGetSystemInfo marshal system info failed:", err)
		// 	return
		// }
		onMessage([]byte(data))
	}, stopChan)
}

func SubscribeSystemInfo(clientID, topic string, onMessage func(data string), stopChan <-chan struct{}) error {
	subClient, err := zmq.NewZmqSub(zmq.ZMQ_SYSTEM_INFO_SUB_ENDPOINT, clientID, topic) // log_1
	if err != nil {
		return fmt.Errorf("create zmq client failed: %w", err)
	}
	log.Infof("zmq Subscribe system info, topic: %s, clientID:%s", topic, clientID)
	go func() {
		for {
			select {
			case <-stopChan:
				log.Info("Stop subscribe system info")
				subClient.Close()
				return
			default:
				msg, err := subClient.Recv()
				if err != nil {
					log.Errorf("Receive message failed: %v", err)
					continue
				}
				log.Infof("Receive message: %v", msg)
				if len(msg) < 2 {
					log.Warn("invalid message, length less than 2")
					continue
				}
				processJSON := msg[1]
				onMessage(processJSON)
			}
		}
	}()
	return nil
}

func OperateProcess(op, id string) error {
	clientID := fmt.Sprintf("CallProcess_%d", time.Now().UnixMilli())
	zmqClient, err := zmq.NewZmqClient(zmq.ZMQ_SYSTEM_INFO_REQ_ENDPOINT, clientID)
	if err != nil {
		return fmt.Errorf("create zmq client failed: %w", err)
	}

	// op start/stop/restart
	result, err := zmqClient.SimpleGetMsg(op, id)
	if err != nil {
		return fmt.Errorf("send start to server failed: %w", err)
	}
	log.Info("OperateProcess result: ", result)
	return nil
}
