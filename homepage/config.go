package homepage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"webserver-arm/emsxmldb"
	//"webserver-arm/log"
)

// ConfigItem 首页配置项（统一结构体，适用于 topology/realtime/basic_info 三个区块）
type ConfigItem struct {
	No         int               `json:"no"`
	ShowName   string            `json:"show_name"`
	ShowNameEn string            `json:"show_name_en"`
	Used       int               `json:"used"`
	OID        string            `json:"oid"`
	Type       int               `json:"type"`
	ShowUnit   string            `json:"show_unit"`
	Table      map[string]string `json:"table"`
	ShowValue  string            `json:"show_value"`
	Name       string            `json:"name"`
	Unit       string            `json:"unit"`
}

// HomepageConfig 首页配置完整结构
type HomepageConfig struct {
	Topology   []ConfigItem `json:"topology"`
	Realtime   []ConfigItem `json:"realtime"`
	RealPower  []ConfigItem `json:"real_power"`
	BasicInfo  []ConfigItem `json:"basic_info"`
	PowerLevel []ConfigItem `json:"power_level"`
}

// ConfigManager 首页配置管理器（单例）
type ConfigManager struct {
	config   HomepageConfig
	filePath string
	dbmgr    *emsxmldb.DbConfigXMLManager
	mu       sync.RWMutex
}

var (
	configInstance *ConfigManager
	configOnce     sync.Once
	configInitErr  error
)

func getConfigBasePath() string {
	return os.Getenv("ARM_SF")
}

// GetConfigManager 获取单例实例
func GetConfigManager() (*ConfigManager, error) {
	configOnce.Do(func() {
		basePath := getConfigBasePath()
		if basePath == "" {
			configInitErr = fmt.Errorf("env ARM_SF not set")
			return
		}
		filePath := filepath.Join(basePath, "etc/web", "homepage.json")

		dbmgr, err := emsxmldb.GetDbConfigManager()
		if err != nil {
			configInitErr = fmt.Errorf("get db config manager failed: %w", err)
			return
		}

		configInstance = &ConfigManager{
			filePath: filePath,
			dbmgr:    dbmgr,
		}
		if err := configInstance.load(); err != nil {
			configInitErr = fmt.Errorf("init homepage config failed: %w", err)
		}
	})
	return configInstance, configInitErr
}

// load 从 JSON 文件加载配置到内存，加载后为有 OID 的条目补充 name/unit
func (cm *ConfigManager) load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			cm.config = HomepageConfig{
				Topology:  []ConfigItem{},
				Realtime:  []ConfigItem{},
				BasicInfo: []ConfigItem{},
			}
			return cm.save()
		}
		return fmt.Errorf("read homepage.json failed: %w", err)
	}

	if len(data) == 0 {
		cm.config = HomepageConfig{
			Topology:  []ConfigItem{},
			Realtime:  []ConfigItem{},
			BasicInfo: []ConfigItem{},
		}
		return nil
	}

	var cfg HomepageConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse homepage.json failed: %w", err)
	}

	cm.config = cfg
	cm.enrichAll()
	return nil
}

// save 将内存配置持久化到 JSON 文件（写文件，不加锁）
func (cm *ConfigManager) save() error {
	if err := os.MkdirAll(filepath.Dir(cm.filePath), 0755); err != nil {
		return fmt.Errorf("create directory failed: %w", err)
	}
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal homepage config failed: %w", err)
	}
	if err := os.WriteFile(cm.filePath, data, 0644); err != nil {
		return fmt.Errorf("write homepage.json failed: %w", err)
	}
	return nil
}

// enrichAll 遍历所有区块，为有 OID 的条目从 XML 索引补充 name 和 unit
func (cm *ConfigManager) enrichAll() {
	cm.enrichItems(cm.config.Topology)
	cm.enrichItems(cm.config.Realtime)
	cm.enrichItems(cm.config.BasicInfo)
	cm.enrichItems(cm.config.PowerLevel)
	cm.enrichItems(cm.config.RealPower)
}

// enrichItems 为一批 ConfigItem 补充 name/unit
func (cm *ConfigManager) enrichItems(items []ConfigItem) {
	for i := range items {
		if items[i].OID == "" {
			continue
		}
		point := cm.findPointByOID(items[i].OID)
		if point == nil {
			continue
		}
		items[i].Name = point.GetName()
		if p, ok := point.(*emsxmldb.Analog); ok {
			items[i].Unit = p.Unit
		}
	}
}

// findPointByOID 通过 OID（rtuid-devid-tableNo-gin）在层级结构中查找测点
func (cm *ConfigManager) findPointByOID(oid string) emsxmldb.Pointer {
	var rtuID, deviceID, tableNo, gin int
	n, _ := fmt.Sscanf(oid, "%d-%d-%d-%d", &rtuID, &deviceID, &tableNo, &gin)
	if n != 4 {
		return nil
	}

	db := cm.dbmgr.GetDB()
	rtuIDStr := fmt.Sprintf("%d", rtuID)
	deviceIDStr := fmt.Sprintf("%d", deviceID)
	for _, rtu := range db.Hierarchy.RTUs {
		if rtu.GetID() != rtuIDStr {
			continue
		}
		for _, dev := range rtu.Devices {
			if dev.GetID() != deviceIDStr {
				continue
			}
			return findPointInDevice(dev, tableNo, gin)
		}
	}
	return nil
}

// findPointInDevice 在 Device 中按 tableNo 和 gin 查找测点
func findPointInDevice(dev *emsxmldb.Device, tableNo, gin int) emsxmldb.Pointer {
	points := getPointsByTableNo(dev, tableNo)
	targetID := fmt.Sprintf("%s-%s-%d-%d", "", "", tableNo, gin)
	_ = targetID // ID 由 emsxmldb 在 add 时自动拼接，此处直接用 gin 匹配
	for _, p := range points {
		if p.GetGin() == fmt.Sprintf("%d", gin) {
			return p
		}
	}
	return nil
}

// getPointsByTableNo 根据 tableNo 返回 Device 中对应的测点切片
func getPointsByTableNo(dev *emsxmldb.Device, tableNo int) []emsxmldb.Pointer {
	switch tableNo {
	case 101, 106:
		ps := make([]emsxmldb.Pointer, len(dev.YX.Digitals))
		for i := range dev.YX.Digitals {
			ps[i] = dev.YX.Digitals[i]
		}
		return ps
	case 102, 107:
		ps := make([]emsxmldb.Pointer, len(dev.YC.Analogs))
		for i := range dev.YC.Analogs {
			ps[i] = dev.YC.Analogs[i]
		}
		return ps
	case 103, 108:
		ps := make([]emsxmldb.Pointer, len(dev.YK.Controls))
		for i := range dev.YK.Controls {
			ps[i] = dev.YK.Controls[i]
		}
		return ps
	case 104, 109:
		ps := make([]emsxmldb.Pointer, len(dev.YT.Regulates))
		for i := range dev.YT.Regulates {
			ps[i] = dev.YT.Regulates[i]
		}
		return ps
	case 105, 110:
		ps := make([]emsxmldb.Pointer, len(dev.YM.Pulses))
		for i := range dev.YM.Pulses {
			ps[i] = dev.YM.Pulses[i]
		}
		return ps
	case 115, 116:
		ps := make([]emsxmldb.Pointer, len(dev.SX.Attributes))
		for i := range dev.SX.Attributes {
			ps[i] = dev.SX.Attributes[i]
		}
		return ps
	default:
		return nil
	}
}

// deduplicateNo 对 items 按 no 去重，重复时自动分配下一个可用编号
func deduplicateNo(items []ConfigItem) []ConfigItem {
	seen := make(map[int]bool)
	nextNo := 1
	for i := range items {
		if !seen[items[i].No] {
			seen[items[i].No] = true
		} else {
			for seen[nextNo] {
				nextNo++
			}
			items[i].No = nextNo
			seen[nextNo] = true
			nextNo++
		}
	}
	return items
}

// GetConfig 返回完整配置（读锁，内存直接返回）
func (cm *ConfigManager) GetConfig() HomepageConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// UpdateSection 更新指定区块（写锁 + no 去重 + 补充 name/unit + 写文件）
func (cm *ConfigManager) UpdateSection(section string, items []ConfigItem) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	items = deduplicateNo(items)
	cm.enrichItems(items)

	switch section {
	case "topology":
		cm.config.Topology = items
	case "realtime":
		cm.config.Realtime = items
	case "basic_info":
		cm.config.BasicInfo = items
	case "power_level":
		cm.config.PowerLevel = items
	case "real_power":
		cm.config.RealPower = items
	default:
		return fmt.Errorf("unknown section: %s", section)
	}

	return cm.save()
}
