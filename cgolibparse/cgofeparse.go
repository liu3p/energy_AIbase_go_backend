package cgolibparse

/*
#cgo LDFLAGS:  -L. -L/root/arm/lib -lparse
#cgo CFLAGS: -I./cgolibparse -I/root/arm/src/agc/include
#include <stdlib.h>
#include <stdint.h>
#include "cgo_libparse.h"
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"
	"webserver-arm/log"
)

type Param struct {
	Default   string `json::"default"`
	ValueList string `json::"valuelist"`
}

type PluginLibInfo struct {
	ID     int              `json:"id"`
	Name   string           `json:"name"`
	Params map[string]Param `json:"params"`
}

func GetPluginLibInfo() ([]PluginLibInfo, error) {
	cJsonStr := C.getPluginLibInfo()
	// 校验C接口返回结果：nil表示调用失败
	if cJsonStr == nil {
		return nil, errors.New("C-func GetPluginLibInfo call failed")
	}
	defer C.free(unsafe.Pointer(cJsonStr))
	jsonStr := C.GoString(cJsonStr)
	if jsonStr == "" {
		return nil, errors.New("C-func GetPluginLibInfo return empty json")
	}
	var pluginsInfo []PluginLibInfo
	err := json.Unmarshal([]byte(jsonStr), &pluginsInfo)
	if err != nil {
		return nil, fmt.Errorf("unmarshal JSON failed: %w", err)
	}

	return pluginsInfo, nil
}

func getFeChannelID(groupID, channelID uint16) uint32 {
	// groupID左移16位和channelID拼接成
	return (uint32(groupID&0x0FFF) << 16) | uint32(channelID)
}

func getGroupAndChannelID(feChannelID uint32) (uint16, uint16) {
	groupID := uint16(feChannelID >> 16)
	chennelID := uint16(feChannelID & 0xFFFF)
	return groupID, chennelID
}

var initFeConnection sync.Once

type feRecvMsgItem struct {
	Type string          `json:"type"` // protocol，status
	Data json.RawMessage `json:"data"` // parse to FeChannelStatus FeChannelLog by type
}

type FeChannelStatus struct {
	FeChannelID uint32 `json:"channelid"`
	GroupID     uint16 `json:"group_id"`
	ChannelID   uint16 `json:"channel_id"`
	Durty       int    `json:"duty"`
	Connected   int    `json:"connected"`
	Used        int    `json:"used"`
}

type FeChannelLog struct {
	FeChannelID uint32 `json:"channelid"`
	PluginID    uint32 `json:"pluginid"`
	LogInfo     string `json:"loginfo"`
}

// clientSubscription 记录每个客户端订阅的通道和插件信息
type clientSubscription struct {
	ch       chan<- FeChannelLog
	pluginID uint32
}

var (
	feParseInstance    *FeParseManager
	feParseManagerOnce sync.Once
	feParseInitErr     error
)

type FeParseManager struct {
	allChannelStatus map[uint16]FeChannelStatus // 通道组id -> 通道状态
	statusMapMutex   sync.RWMutex
	allClientChannel map[uint32]map[string]clientSubscription
	channelMapMutex  sync.RWMutex
}

func GetFeParseManagerInstance() (*FeParseManager, error) {
	feParseManagerOnce.Do(func() {
		m := &FeParseManager{
			allChannelStatus: make(map[uint16]FeChannelStatus),
			statusMapMutex:   sync.RWMutex{},
			allClientChannel: make(map[uint32]map[string]clientSubscription),
			channelMapMutex:  sync.RWMutex{},
		}

		// 初次订阅通道状态（立即执行）
		cRet := C.subscribeFeChannelStatus()
		if cRet != 0 {
			feParseInitErr = fmt.Errorf("GetFeChannelStatus call C subscribeFeChannelStatus failed: %d", cRet)
			log.Error(feParseInitErr.Error())
			return
		}

		// 启动接收消息的线程
		go m.startFeRecvMsg()

		// 启动周期性保活 goroutine，每 50 秒调用一次订阅
		go func() {
			ticker := time.NewTicker(50 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				ret := C.subscribeFeChannelStatus()
				if ret != 0 {
					log.Errorf("subscribeFeChannelStatus keepalive failed: %d", ret)
					// 失败只记录日志，不中断循环，继续下一次尝试
				} else {
					log.Debug("subscribeFeChannelStatus keepalive success")
				}
			}
		}()

		feParseInstance = m
		log.Info("FeParseManager Instance success")
	})
	return feParseInstance, feParseInitErr
}

// 开始从共享内存中接收消息
func (m *FeParseManager) startFeRecvMsg() {
	for true {
		var cOutputJson *C.char
		cRet := C.recvFeChannelMsg(&cOutputJson)
		if cRet <= 0 || cOutputJson == nil {
			log.Debugf("call C recvFeChannelMsg failed: %d", cRet)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// ========== C字符串 → Go字符串 ==========
		jsonStr := C.GoString(cOutputJson)
		C.free(unsafe.Pointer(cOutputJson))

		switch cRet {
		case 11003: // FES::MT_DISPLAY_CODE
			m.dispatchFeLogMsg(jsonStr)
		case 11021: // FES::MT_SUBSCRIBE_CHANNEL_STATUS
			m.dispatchFeStatusMsg(jsonStr)
		case 11011: // FES::MT_RELOAD_RESULT
			log.Info("fe reload result:", string(jsonStr))
		default:
			log.Errorf("unknown fe message type: %d, data:%s", cRet, string(jsonStr))
		}
	}
}

// dispatchFeStatusMsg 解析并记录所有通道的状态信息
func (m *FeParseManager) dispatchFeStatusMsg(data string) {
	// 解析原始JSON到FeChannelStatus
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		log.Errorf("parse fe status message failed:%v, data:%s", err, data)
		return
	}
	// 解析所有通道状态
	var statusList []FeChannelStatus
	if err := json.Unmarshal(raw["status_list"], &statusList); err != nil {
		log.Errorf("parse fe status message failed:%v, data:%s", err, data)
		return
	}

	m.statusMapMutex.Lock()
	defer m.statusMapMutex.Unlock()

	// 构建通道组id和通道状态的对应关系
	for i := range statusList {
		feChannelID := statusList[i].FeChannelID
		statusList[i].GroupID, statusList[i].ChannelID = getGroupAndChannelID(feChannelID)
		m.allChannelStatus[statusList[i].GroupID] = statusList[i]
	}
}

// dispatchFeLogMsg 解析并分发FE通道日志消息到所有注册的客户端（非阻塞发送）
func (m *FeParseManager) dispatchFeLogMsg(data string) {
	var logMsg FeChannelLog
	if err := json.Unmarshal([]byte(data), &logMsg); err != nil {
		log.Errorf("parse fe log message failed:%v, data:%s", err, data)
		return
	}

	m.channelMapMutex.RLock()
	channels, has := m.allClientChannel[logMsg.FeChannelID]
	m.channelMapMutex.RUnlock()
	if !has {
		return
	}

	for _, sub := range channels {
		// 按 pluginID 过滤，只发送给订阅了对应插件的客户端
		if sub.pluginID != logMsg.PluginID {
			continue
		}
		// 使用非阻塞发送或带超时的发送，避免阻塞
		select {
		case sub.ch <- logMsg:
		default:
			log.Warnf("client channel full, drop message for feChannelID %d, pluginID %d", logMsg.FeChannelID, logMsg.PluginID)
		}
	}
}

// GetFeChannelsStatus 获取所有通道状态
func (m *FeParseManager) GetFeChannelsStatus() map[uint16]FeChannelStatus {
	m.statusMapMutex.RLock()
	defer m.statusMapMutex.RUnlock()

	return m.allChannelStatus
}

// GetFeChannelStatus 获取指定通道状态
func (m *FeParseManager) GetFeChannelStatus(channelGroupID uint16) FeChannelStatus {
	m.statusMapMutex.RLock()
	defer m.statusMapMutex.RUnlock()

	if status, has := m.allChannelStatus[channelGroupID]; has {
		return status
	} else {
		return FeChannelStatus{}
	}
}

// GetFeChannelLogs 获取通道日志信息
func (m *FeParseManager) GetFeChannelLogs(groupID, channelID uint16, pluginID uint32) (<-chan FeChannelLog, func(), error) {
	m.channelMapMutex.Lock()
	defer m.channelMapMutex.Unlock()

	feChannelID := getFeChannelID(groupID, channelID)
	if _, has := m.allClientChannel[feChannelID]; !has {
		m.allClientChannel[feChannelID] = make(map[string]clientSubscription)
	}
	clientID := time.Now().Format("20060102_150405")
	msgChan := make(chan FeChannelLog, 100)
	m.allClientChannel[feChannelID][clientID] = clientSubscription{ch: msgChan, pluginID: pluginID}

	// 初次订阅
	cRet := C.subscribeFeChannelLogs(C.uint32_t(feChannelID), C.uint32_t(pluginID))
	if cRet != 0 {
		err := fmt.Errorf("call C subscribeFeChannelLogs failed: %d", cRet)
		log.Error(err)
		return nil, nil, err
	}

	// 创建可取消的 context
	ctx, cancel := context.WithCancel(context.Background())

	// 启动周期性保活 goroutine
	go func() {
		ticker := time.NewTicker(50 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ret := C.subscribeFeChannelLogs(C.uint32_t(feChannelID), C.uint32_t(pluginID))
				if ret != 0 {
					log.Errorf("subscribeFeChannelLogs keepalive failed: %d", ret)
				} else {
					log.Debug("subscribeFeChannelLogs keepalive success")
				}
			case <-ctx.Done():
				log.Debugf("stop keepalive goroutine for feChannelID: %d", feChannelID)
				return
			}
		}
	}()

	// 清理函数：停止保活 goroutine 并移除订阅记录
	cleanup := func() {
		cancel() // 停止保活 goroutine

		m.channelMapMutex.Lock()
		defer m.channelMapMutex.Unlock()
		if clientMap, ok := m.allClientChannel[feChannelID]; ok {
			delete(clientMap, clientID)
			if len(clientMap) == 0 {
				delete(m.allClientChannel, feChannelID)
			}
		}
		// 注意：不关闭 msgChan，由外部调用方决定是否关闭
	}

	return msgChan, cleanup, nil
}

func NotifyFeServerReload() error {
	//cRet := C.notifyFeChannelReload()
	cRet := C.notifyFeServerReload(nil, C.int(0), C.int(1))
	if cRet != 0 {
		return fmt.Errorf("call C notifyFeChannelReload failed: %d", cRet)
	}
	return nil
}
