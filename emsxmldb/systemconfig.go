package emsxmldb

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

/*
<?xml version="1.0" encoding="UTF-8"?>
<SystemInfo>
    <Sn></Sn>
    <Name></Name>
    <Addr></Addr>
    <ModelNum></ModelNum>
</SystemInfo>
*/

type SystemInfo struct {
	XMLName  xml.Name `xml:"SystemInfo"` // 根节点名称
	Sn       string   `xml:"Sn"`         // 项目序列号
	Name     string   `xml:"Name"`       // 项目名称
	Addr     string   `xml:"Addr"`       // 项目地址
	ModelNum string   `xml:"ModelNum"`   // 设备型号
}

type SystemInfoManager struct {
	filePath string       // XML文件路径
	info     SystemInfo   // 内存中的设备信息
	mu       sync.RWMutex // 读写锁，保护并发访问
}

var (
	systemInfoInstance     *SystemInfoManager                                              // 全局唯一实例
	systemInfoInstanceonce sync.Once                                                       // 保证只初始化一次
	systemInfoXmlPath      = filepath.Join(getConfigBasePath(), "/webdata/systeminfo.xml") //"data/systeminfo.xml"
)

func GetSystemInfoManager() (*SystemInfoManager, error) {
	var initErr error
	systemInfoInstanceonce.Do(func() {
		// 初始化单例
		systemInfoInstance = &SystemInfoManager{
			filePath: systemInfoXmlPath,
			info:     SystemInfo{}, // 默认空初始化
		}
		// 加载XML文件（不存在则创建）
		if err := systemInfoInstance.Load(); err != nil {
			initErr = fmt.Errorf("init device manager failed: %w", err)
			instance = nil // 初始化失败，清空实例
		}
	})
	return systemInfoInstance, initErr
}

func (m *SystemInfoManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := os.Stat(m.filePath)
	if os.IsNotExist(err) {
		return m.Save()
	}
	if err != nil {
		return fmt.Errorf("check file stat failed: %w", err)
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	if err := xml.Unmarshal(data, &m.info); err != nil {
		m.info = SystemInfo{}
		return fmt.Errorf("parse xml failed: %w", err)
	}

	return nil
}

func (m *SystemInfoManager) Save() error {
	data, err := xml.MarshalIndent(&m.info, "", "    ")
	if err != nil {
		return fmt.Errorf("MarshalIndent XML failed: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("WriteFile failed: %w", err)
	}

	return nil
}

// -------------------------- 字段查询方法（Get） --------------------------
// GetSn 获取设备序列号
func (m *SystemInfoManager) GetSn() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.info.Sn
}

// GetName 获取设备名称
func (m *SystemInfoManager) GetName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.info.Name
}

// GetAddr 获取设备地址
func (m *SystemInfoManager) GetAddr() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.info.Addr
}

// GetModelNum 获取设备型号
func (m *SystemInfoManager) GetModelNum() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.info.ModelNum
}

// -------------------------- 字段修改方法（Set） --------------------------
// SetSn 修改设备序列号，修改后自动保存（可根据需求关闭自动保存）
func (m *SystemInfoManager) SetSn(sn string) error {
	m.mu.Lock()
	m.info.Sn = sn
	m.mu.Unlock()
	return m.Save() // 自动持久化，若不需要可注释此行，手动调用Save()
}

// SetName 修改设备名称
func (m *SystemInfoManager) SetName(name string) error {
	m.mu.Lock()
	m.info.Name = name
	m.mu.Unlock()
	return m.Save()
}

// SetAddr 修改设备地址
func (m *SystemInfoManager) SetAddr(addr string) error {
	m.mu.Lock()
	m.info.Addr = addr
	m.mu.Unlock()
	return m.Save()
}

// SetModelNum 修改设备型号
func (m *SystemInfoManager) SetModelNum(modelNum string) error {
	m.mu.Lock()
	m.info.ModelNum = modelNum
	m.mu.Unlock()
	return m.Save()
}

func (m *SystemInfoManager) SetAll(sn, name, addr, modelNum string) error {
	m.mu.Lock()
	// 批量修改内存中的字段
	m.info.Sn = sn
	m.info.Name = name
	m.info.Addr = addr
	m.info.ModelNum = modelNum
	m.mu.Unlock()
	// 统一持久化到文件
	return m.Save()
}

func (m *SystemInfoManager) GetAll() SystemInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.info
}
