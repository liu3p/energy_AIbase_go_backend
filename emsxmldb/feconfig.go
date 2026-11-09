package emsxmldb

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
	"webserver-arm/cgolibparse"
	"webserver-arm/log"
)

// -------------------------- 1. XML 结构体映射（与XML节点一一对应） --------------------------
// Parameter XML节点：通用参数（NAME/VALUE键值对）
type Parameter struct {
	Name      string `xml:"NAME,attr" json:"name"`
	Value     string `xml:"VALUE,attr" json:"value"`
	ValueList string `xml:"-" json:"valuelist"`
}

// Plugin XML节点：CHANNEL下的PLUGIN（含多个PARAMETER）
type Plugin struct {
	ID         int         `xml:"ID,attr" json:"id"`
	Name       string      `xml:"NAME,attr" json:"name"`
	Type       string      `xml:"TYPE,attr" json:"type"` // 层号字符串 "1"/"2"/...，由 ID 推导，写入 XML
	Parameters []Parameter `xml:"PARAMETER" json:"parameters"`
}

// PluginLayer 某一层的可选协议集合（用于 /fecfg/plugins 按层返回）
type PluginLayer struct {
	Type    string   `json:"type"`    // 层号字符串 "1"/"2"/...
	Plugins []Plugin `json:"plugins"` // 该层可选协议
}

// pluginLayer 由插件 ID 推导层号：0-999→1, 1000-1999→2, ...
func pluginLayer(id int) int {
	if id < 0 {
		return 0 // 非法
	}
	return id/1000 + 1
}

// layerString 层号的字符串形式（写入 Plugin.Type / XML TYPE 属性）
func layerString(id int) string {
	return strconv.Itoa(pluginLayer(id))
}

// ErrPluginLayer 层协议选择非法（跳层、同层重复、ID 非法）
var ErrPluginLayer = errors.New("plugin layer invalid")

// ValidatePluginLayers 校验通道协议选择：
//   - 允许为空
//   - 插件 ID 合法（>=0）
//   - 每层至多一个
//   - 层号从第 1 层起连续，无跳层
func ValidatePluginLayers(plugins []Plugin) error {
	if len(plugins) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(plugins))
	layers := make([]int, 0, len(plugins))
	for _, p := range plugins {
		if p.ID < 0 {
			return fmt.Errorf("%w: 插件 ID %d 无效", ErrPluginLayer, p.ID)
		}
		l := pluginLayer(p.ID)
		if _, dup := seen[l]; dup {
			return fmt.Errorf("%w: 第 %d 层只能选择一个协议", ErrPluginLayer, l)
		}
		seen[l] = struct{}{}
		layers = append(layers, l)
	}
	sort.Ints(layers)
	for i, l := range layers {
		if l != i+1 {
			return fmt.Errorf("%w: 协议必须从第 1 层起连续选择，缺少第 %d 层", ErrPluginLayer, i+1)
		}
	}
	return nil
}

// sortPluginsByLayer 设置 Type 并返回按层升序排列的副本。
// 按数值层号排序（非字符串字典序），否则 "10" 会排在 "2" 前。
func sortPluginsByLayer(plugins []Plugin) []Plugin {
	cp := make([]Plugin, len(plugins))
	copy(cp, plugins)
	for i := range cp {
		cp[i].Type = layerString(cp[i].ID)
	}
	sort.SliceStable(cp, func(i, j int) bool {
		return pluginLayer(cp[i].ID) < pluginLayer(cp[j].ID)
	})
	return cp
}


// PossibleOwner XML节点：CHANNEL下的POSSIBLE_OWNER
type PossibleOwner struct {
	ID int `xml:"ID,attr" json:"id"`
}

// Channel XML节点：CHANNEL_GROUP下的CHANNEL
type Channel struct {
	Comment        string        `xml:"COMMENT,attr" json:"comment"`
	WorkOnDutyNode int           `xml:"WORK_ON_DUTY_NODE,attr" json:"onduty"`
	ID             int           `xml:"ID,attr" json:"id"`
	ServerGroupID  int           `xml:"SERVER_GROUP_ID,attr" json:"servergroup"`
	Name           string        `xml:"NAME,attr" json:"name"`
	Plugins        []Plugin      `xml:"PLUGIN" json:"plugins"`
	PossibleOwner  PossibleOwner `xml:"POSSIBLE_OWNER" json:"possibleowner"`
}

// Model XML节点：CHANNEL_GROUP下的MODEL（含多个PARAMETER）
type Model struct {
	ID         int         `xml:"ID,attr"`
	Parameters []Parameter `xml:"PARAMETER"`
}

// ChannelGroup XML节点：FE_DEFINE下的CHANNEL_GROUP
type ChannelGroup struct {
	Comment        string    `xml:"COMMENT,attr" json:"comment"`
	WorkOnDutyNode int       `xml:"WORK_ON_DUTY_NODE,attr" json:"onduty"`
	ID             int       `xml:"ID,attr" json:"id"`
	Name           string    `xml:"NAME,attr" json:"name"`
	UpdateTime     int       `xml:"UPDATE_TIME,attr" json:"updatetime"`
	Channels       []Channel `xml:"CHANNEL" json:"channel"`
	Model          Model     `xml:"MODEL" json:"model"`
}

// Host XML节点：FE_DEFINE下的HOST
type Host struct {
	ID         int    `xml:"ID,attr"`
	Name       string `xml:"NAME,attr"`
	FullServer int    `xml:"FULL_SERVER,attr"`
}

// FeDefine XML根节点：FE_DEFINE
type FeDefine struct {
	XMLName       xml.Name       `xml:"FE_DEFINE"`
	Comment       string         `xml:"COMMENT,attr"`
	SwitchCGID    int            `xml:"SWITCH_CG_ID,attr"`
	Parameters    []Parameter    `xml:"PARAMETER"`
	Host          Host           `xml:"HOST"`
	ChannelGroups []ChannelGroup `xml:"CHANNEL_GROUP"`
}

// -------------------------- 2. 单例类 FeConfigXMLManager 定义 --------------------------
// FeConfigXMLManager 单例类：维护XML配置文件的核心管理器
type FeConfigXMLManager struct {
	xmlPath  string   // XML文件路径
	backDir  string   // XML备份文件存放路径
	feDefine FeDefine // 解析后的XML数据

	mutex sync.RWMutex // 读写锁（保证并发安全）
}

// 单例实例 + sync.Once保证只初始化一次
var (
	insFeConfig     *FeConfigXMLManager
	onceFeConfig    sync.Once
	initFeConfigErr error
)

var (
	defaultFeXMLFilePath       = filepath.Join(getConfigBasePath(), "/data/fe_config.xml") //"/data/feconfig.xml"
	defaultBackDir             = filepath.Join(getConfigBasePath(), "data", "fe_config_bak")
	defaultFeDllTypeXMLFileDir = filepath.Join(getConfigBasePath(), "/lib/fe_plugin.xml") //"/root/arm/lib/fe_plugin.xml"
)

// GetFeConfigXMLManager 获取单例实例（需传入XML文件路径）
func GetFeConfigXMLManager() (*FeConfigXMLManager, error) {
	onceFeConfig.Do(func() {
		insFeConfig = &FeConfigXMLManager{
			xmlPath: defaultFeXMLFilePath,
			backDir: defaultBackDir,
		}
		// 初始化时加载XML文件
		if initFeConfigErr := insFeConfig.loadXML(); initFeConfigErr != nil {
			log.Error("feConfig init load xml failed:", initFeConfigErr.Error())
		}

		// 创建备份文件目录
		os.MkdirAll(insFeConfig.backDir, 0755)
	})
	return insFeConfig, initFeConfigErr
}

// -------------------------- 3. 核心辅助方法：XML加载/保存 --------------------------
// LoadXML 从文件加载XML并解析到feDefine
func (m *FeConfigXMLManager) loadXML() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	initDB := func() error {
		hoseName, err := os.Hostname()
		if err != nil {
			log.Error("get hostName failed:", err)
		}
		m.feDefine = FeDefine{
			Comment: "",
			XMLName: xml.Name{Local: "FE_DEFINE"},
			Parameters: []Parameter{
				{Name: "NET_A_MOXA_BASE_IP", Value: "127.0.0.1"},
				{Name: "NET_B_MOXA_BASE_IP", Value: "127.0.0.1"},
			},
			Host:       Host{FullServer: 1, ID: 1, Name: hoseName},
			SwitchCGID: -1,
		}
		return m.saveXML()
	}

	// 读取XML文件
	data, err := os.ReadFile(m.xmlPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Errorf("ReadFile %s not exist, initDB", m.xmlPath)
			return initDB()
		}
		return fmt.Errorf("read XML failed:%w", err)
	}

	// 解析XML到结构体
	var feDefine FeDefine
	if err := xml.Unmarshal(data, &feDefine); err != nil {
		return fmt.Errorf("parse XML failed:%w", err)
	}

	m.feDefine = feDefine

	// 修复存量数据中 Plugin.Type：一律以 ID 推断的层号为准（非数字 / 矛盾均不报错）
	m.repairPluginLayers()
	return nil
}

// repairPluginLayers 用 ID 推断的层号修正每个 Plugin.Type。
// 存量数据中 TYPE 可能缺失、非数字（"APP"/"LINK"）或与 ID 矛盾——一律以 ID 为准。
func (m *FeConfigXMLManager) repairPluginLayers() {
	for i := range m.feDefine.ChannelGroups {
		channels := m.feDefine.ChannelGroups[i].Channels
		for j := range channels {
			for k := range channels[j].Plugins {
				p := &channels[j].Plugins[k]
				if p.ID < 0 { // 无法推导层号
					log.Warnf("repairPluginLayers: CHANNEL_GROUP %d CHANNEL %d PLUGIN 非法 ID=%d，跳过",
						m.feDefine.ChannelGroups[i].ID, channels[j].ID, p.ID)
					continue
				}
				expected := layerString(p.ID)
				if p.Type != expected { // 非数字 / 与 ID 矛盾 / 缺失 → 以 ID 为准
					log.Warnf("repairPluginLayers: CHANNEL_GROUP %d CHANNEL %d PLUGIN ID=%d TYPE=%q 与推断层号 %q 不一致，修正为 %q",
						m.feDefine.ChannelGroups[i].ID, channels[j].ID, p.ID, p.Type, expected, expected)
					p.Type = expected
				}
			}
		}
	}
}

// SaveXML 将feDefine写入XML文件（新增自动备份旧文件逻辑）
func (m *FeConfigXMLManager) saveXML() error {
	// m.mutex.Lock()
	// defer m.mutex.Unlock()

	// -------------------------- 步骤1：备份旧文件 --------------------------
	xmlPath := m.xmlPath
	// 检查旧文件是否存在
	if _, err := os.Stat(xmlPath); err == nil {
		// 解析文件名：拆分前缀、后缀（如 FECONFIG.xml → 前缀FECONFIG，后缀xml）
		filename := filepath.Base(xmlPath)          // 文件名（含后缀）
		ext := filepath.Ext(filename)               // 后缀（.xml）
		prefix := filename[:len(filename)-len(ext)] // 前缀（FECONFIG）
		extWithoutDot := ext[1:]                    // 去掉点的后缀（xml）

		// 生成时间戳：20060102_150405（Go的时间格式固定参考）
		timestamp := time.Now().Format("20060102_150405")
		// 备份文件名：FECONFIG_20260101_010203.bak.xml
		backupFilename := fmt.Sprintf("%s_%s.bak.%s", prefix, timestamp, extWithoutDot)
		backupPath := filepath.Join(m.backDir, backupFilename)

		// 复制旧文件到备份文件（保留原文件权限）
		oldFile, err := os.Open(xmlPath)
		if err != nil {
			log.Error("feConfig open xml file failed:", err.Error())
		} else {
			defer oldFile.Close()
			backupFile, err := os.Create(backupPath)
			if err != nil {
				log.Error("feConfig create xml backup file failed:", err.Error())
			} else {
				defer backupFile.Close()
				// 复制文件内容
				if _, err := io.Copy(backupFile, oldFile); err != nil {
					log.Error("feConfig copy xml backup to new file failed:", err.Error())
				} else {
					// 同步文件到磁盘，确保备份完成
					backupFile.Sync()
					log.Info("feConfig save xml backup old file success:", backupPath)
				}
			}
		}
	} else if !os.IsNotExist(err) {
		// 非「文件不存在」的错误，打印警告
		log.Error("feConfig get xml file failed:", err.Error())
	}

	// -------------------------- 步骤2：写入新内容到原文件 --------------------------
	// 结构体序列化为XML（带缩进，可读性强）
	data, err := xml.MarshalIndent(m.feDefine, "", "  ")
	if err != nil {
		return fmt.Errorf("XML marshal failed:%w", err)
	}

	// 补充XML头（xml.Marshal不会自动加）
	xmlHeader := []byte(xml.Header)
	data = append(xmlHeader, data...)

	// 写入文件（覆盖原文件）
	if err := os.WriteFile(xmlPath, data, 0644); err != nil {
		return fmt.Errorf("write xml file failed:%w", err)
	}

	log.Info("feConfig save xml success:", xmlPath)
	return nil
}

// -------------------------- 4. CHANNEL_GROUP 增删改查接口 --------------------------
// -------------------------- 4.1 新增CHANNEL_GROUP --------------------------
// CreateChannelGroupParam 新增CHANNEL_GROUP的入参（含MODEL必要参数）
type ChannelGroupParam struct {
	ID             int    `json:"id"`
	Comment        string `json:"comment"`
	WorkOnDutyNode int    `json:"onduty"`
	Name           string `json:"name"`
	UpdateTime     int    `json:"updatetime"`
	// MODEL必要参数
	ModelID       int    `json:"modelid"`
	ForTransfer   string `json:"fortransfer"`
	StationBaseID string `json:"stationid"` // rtu ID
	// DeviceBaseID  string `json:"deviceid"`
	// StationN      string `json:"stationn"`
	// DeviceN       string `json:"devicen"`
}

// CreateChannelGroup 新增CHANNEL_GROUP（自动创建MODEL结构）
func (m *FeConfigXMLManager) CreateChannelGroup(param ChannelGroupParam) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	newID := 0
	// check exist conflict

	for _, cg := range m.feDefine.ChannelGroups {
		if cg.ID >= newID {
			newID = cg.ID + 1
		}
	}

	if newID >= math.MaxInt32 {
		return fmt.Errorf("CHANNEL_GROUP ID range (0-2^31) reached its upper limit")
	}

	if param.ModelID == 0 {
		param.ModelID = 20000
	}
	// if param.DeviceBaseID == "" {
	// 	param.DeviceBaseID = "0"
	// }
	// if param.StationN == "" {
	// 	param.StationN = "1"
	// }
	// if param.DeviceN == "" {
	// 	param.DeviceN = "1"
	// }

	model := Model{
		ID: param.ModelID,
		Parameters: []Parameter{
			{Name: "FOR_TRANSFER", Value: param.ForTransfer},
			{Name: "STATION_BASE_ID", Value: param.StationBaseID},
			// {Name: "DEVICE_BASE_ID", Value: param.DeviceBaseID},
			// {Name: "STATION_N", Value: param.StationN},
			// {Name: "DEVICE_N", Value: param.DeviceN},
		},
	}

	// 3. 构建CHANNEL_GROUP结构体
	newCG := ChannelGroup{
		Comment:        param.Comment,
		WorkOnDutyNode: param.WorkOnDutyNode,
		ID:             newID,
		Name:           param.Name,
		UpdateTime:     param.UpdateTime,
		Model:          model,
		Channels:       []Channel{}, // 初始无CHANNEL
	}

	// 4. 添加到切片
	m.feDefine.ChannelGroups = append(m.feDefine.ChannelGroups, newCG)

	return m.saveXML()
}

// -------------------------- 4.2 删除CHANNEL_GROUP --------------------------
// DeleteChannelGroup 根据ID删除CHANNEL_GROUP
func (m *FeConfigXMLManager) DeleteChannelGroup(cgID int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, cg := range m.feDefine.ChannelGroups {
		if cg.ID == cgID {
			m.feDefine.ChannelGroups = append(m.feDefine.ChannelGroups[:i], m.feDefine.ChannelGroups[i+1:]...)
			return m.saveXML()
		}
	}

	return fmt.Errorf("CHANNEL_GROUP ID %d not exist", cgID)
}

// -------------------------- 4.3 修改CHANNEL_GROUP --------------------------
// UpdateChannelGroupParam 修改CHANNEL_GROUP的入参
type UpdateChannelGroupParam struct {
	ID             int    // 要修改的CHANNEL_GROUP ID（必填）
	Comment        string // 可选：新的COMMENT
	WorkOnDutyNode int    // 可选：新的WORK_ON_DUTY_NODE
	Name           string // 可选：新的NAME
	UpdateTime     int    // 可选：新的UPDATE_TIME
	// MODEL可选参数（为空则不修改）
	ModelForTransfer string // MODEL的FOR_TRANSFER新值
	StationBaseID    string // MODEL的STATION_BASE_ID参数值   // rtu ID
	DeviceBaseID     string // MODEL的DEVICE_BASE_ID参数值
	StationN         string // MODEL的STATION_N参数值
}

// UpdateChannelGroup 修改CHANNEL_GROUP（支持部分字段更新）
func (m *FeConfigXMLManager) UpdateChannelGroup(chGroupID int, param ChannelGroupParam) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if param.ModelID == 0 {
		param.ModelID = 20000
	}
	// if param.DeviceBaseID == "" {
	// 	param.DeviceBaseID = "0"
	// }
	// if param.StationN == "" {
	// 	param.StationN = "1"
	// }
	// if param.DeviceN == "" {
	// 	param.DeviceN = "1"
	// }

	// 1. 查找目标CHANNEL_GROUP
	for i, cg := range m.feDefine.ChannelGroups {
		if cg.ID == chGroupID {
			// do not update ID
			m.feDefine.ChannelGroups[i].Comment = param.Comment
			m.feDefine.ChannelGroups[i].WorkOnDutyNode = param.WorkOnDutyNode
			m.feDefine.ChannelGroups[i].Name = param.Name
			m.feDefine.ChannelGroups[i].UpdateTime = param.UpdateTime
			m.feDefine.ChannelGroups[i].Model.ID = param.ModelID

			// update Model parameters
			for j, p := range m.feDefine.ChannelGroups[i].Model.Parameters {
				if p.Name == "FOR_TRANSFER" {
					m.feDefine.ChannelGroups[i].Model.Parameters[j].Value = param.ForTransfer
				} else if p.Name == "STATION_BASE_ID" {
					m.feDefine.ChannelGroups[i].Model.Parameters[j].Value = param.StationBaseID
				}
				// else if p.Name == "DEVICE_BASE_ID" {
				// 	m.feDefine.ChannelGroups[i].Model.Parameters[j].Value = param.DeviceBaseID
				// } else if p.Name == "STATION_N" {
				// 	m.feDefine.ChannelGroups[i].Model.Parameters[j].Value = param.StationN
				// } else if p.Name == "DEVICE_N" {
				// 	m.feDefine.ChannelGroups[i].Model.Parameters[j].Value = param.DeviceN
				// }
			}

			return m.saveXML()
		}
	}

	return fmt.Errorf("CHANNEL_GROUP ID %d not exist", param.ID)
}

// -------------------------- 4.4 查询CHANNEL_GROUP --------------------------
// GetChannelGroupByID 根据ID查询CHANNEL_GROUP
func (m *FeConfigXMLManager) GetChannelGroupByID(cgID int) (*ChannelGroup, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, cg := range m.feDefine.ChannelGroups {
		if cg.ID == cgID {
			return &cg, nil
		}
	}

	return nil, fmt.Errorf("CHANNEL_GROUP ID %d not exist", cgID)
}

// ListAllChannelGroups 查询所有CHANNEL_GROUP
func (m *FeConfigXMLManager) ListAllChannelGroups() []ChannelGroup {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 返回副本，避免外部修改原数据
	copyCG := make([]ChannelGroup, len(m.feDefine.ChannelGroups))
	copy(copyCG, m.feDefine.ChannelGroups)
	return copyCG
}

// GetAllChennalStationID 获取所有rtu和通道的对应关系
func (m *FeConfigXMLManager) GetAllChennalStationID() map[string]int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	rtuChannelMap := make(map[string]int)
	// 遍历所有通道组
	for i := range m.feDefine.ChannelGroups {
		// 遍历所有通道，找到名称为STATION_BASE_ID的参数
		for j := range m.feDefine.ChannelGroups[i].Model.Parameters {
			if m.feDefine.ChannelGroups[i].Model.Parameters[j].Name == "STATION_BASE_ID" {
				rtuID := m.feDefine.ChannelGroups[i].Model.Parameters[j].Value
				rtuChannelMap[rtuID] = m.feDefine.ChannelGroups[i].ID
				continue
			}
		}
	}
	return rtuChannelMap
}

// GetRelatedRtuID 查询指定通道组关联的rtu
func (m *FeConfigXMLManager) GetRelatedRtuID(channelGroupID string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cgid, err := strconv.Atoi(channelGroupID)
	if err != nil {
		return ""
	}
	// 遍历所有通道组
	for i := range m.feDefine.ChannelGroups {
		if m.feDefine.ChannelGroups[i].ID == cgid {
			// 遍历所有通道，找到名称为STATION_BASE_ID的参数
			for j := range m.feDefine.ChannelGroups[i].Model.Parameters {
				if m.feDefine.ChannelGroups[i].Model.Parameters[j].Name == "STATION_BASE_ID" {
					rtuID := m.feDefine.ChannelGroups[i].Model.Parameters[j].Value
					return rtuID
				}
			}
		}
	}
	return ""
}

// -------------------------- 5. CHANNEL 增删改查接口（归属CHANNEL_GROUP） --------------------------
// -------------------------- 5.1 新增CHANNEL --------------------------
// CreateChannelParam 新增CHANNEL的入参（含PLUGIN必要参数）
type ChannelParam struct {
	Comment         string `json:"comment"`
	WorkOnDutyNode  int    `json:"onduty"`
	ChannelID       int    `json:"id"`
	ServerGroupID   int    `json:"servergroup"`
	Name            string `json:"name"`
	PossibleOwnerID int    `json:"possibleownerid"`

	Plugins []Plugin `json:"plugins"` // 多层协议选择；每层至多一个、从第1层起连续，可为空
}

// CreateChannel 新增CHANNEL到指定CHANNEL_GROUP（自动创建PLUGIN结构）
func (m *FeConfigXMLManager) CreateChannel(chGroupID int, param ChannelParam) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 协议层校验（先于任何修改）
	if err := ValidatePluginLayers(param.Plugins); err != nil {
		return err
	}

	// get channelgroup
	var cgIndex int
	var targetCG *ChannelGroup
	found := false
	for i, cg := range m.feDefine.ChannelGroups {
		if cg.ID == chGroupID {
			cgIndex = i
			targetCG = &m.feDefine.ChannelGroups[i]
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("CHANNEL_GROUP ID %d not exist", chGroupID)
	}

	// generate ChannelID
	newID := 0
	for _, ch := range targetCG.Channels {
		if newID <= ch.ID {
			newID = ch.ID + 1
		}
	}
	param.ChannelID = newID

	// create plugin（按层升序，Type 由 ID 推导）
	plugins := sortPluginsByLayer(param.Plugins)

	if param.PossibleOwnerID == 0 {
		param.PossibleOwnerID = m.feDefine.Host.ID
	}

	// make channel struct
	newChannel := Channel{
		Comment:        param.Comment,
		WorkOnDutyNode: param.WorkOnDutyNode,
		ID:             param.ChannelID,
		ServerGroupID:  chGroupID + 1,
		Name:           param.Name,
		Plugins:        plugins,
		PossibleOwner: PossibleOwner{
			ID: param.PossibleOwnerID,
		},
	}

	// add to channelgroup
	m.feDefine.ChannelGroups[cgIndex].Channels = append(m.feDefine.ChannelGroups[cgIndex].Channels, newChannel)

	return m.saveXML()
}

// -------------------------- 5.2 删除CHANNEL --------------------------
// DeleteChannel 删除指定CHANNEL_GROUP下的CHANNEL
func (m *FeConfigXMLManager) DeleteChannel(cgID, channelID int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 1. 查找归属的CHANNEL_GROUP
	for i, cg := range m.feDefine.ChannelGroups {
		if cg.ID == cgID {
			// 2. 查找并删除目标CHANNEL
			for j, ch := range cg.Channels {
				if ch.ID == channelID {
					m.feDefine.ChannelGroups[i].Channels = append(m.feDefine.ChannelGroups[i].Channels[:j], m.feDefine.ChannelGroups[i].Channels[j+1:]...)
					return nil
				}
			}
			return fmt.Errorf("CHANNEL_GROUP %d CHANNEL ID %d not exist", cgID, channelID)
		}
	}

	return fmt.Errorf("CHANNEL_GROUP ID %d not exist", cgID)
}

// -------------------------- 5.3 修改CHANNEL --------------------------
// UpdateChannelParam 修改CHANNEL的入参
type UpdateChannelParam struct {
	CGID           int    // 归属的CHANNEL_GROUP ID
	ChannelID      int    // 要修改的CHANNEL ID
	Comment        string // 可选：新的COMMENT
	WorkOnDutyNode int    // 可选：新的WORK_ON_DUTY_NODE
	Name           string // 可选：新的NAME
	// PLUGIN参数（可选：键值对，为空则不修改）
	//PluginParams     map[string]string // PLUGIN的PARAMETER新值（如TCP_PORT:503）
	AppPluginID      int               // PLUGIN的ID
	AppPluginParams  map[string]string // PLUGIN的PARAMETER键值对（如PORT_TYPE:COM, TCP_IP:10.2.5.40）
	LinkPluginID     int               // PLUGIN的ID
	LinkPluginParams map[string]string // PLUGIN的PARAMETER键值对（如PORT_TYPE:COM, TCP_IP:10.2.5.40）

}

// UpdateChannel 修改指定CHANNEL_GROUP下的CHANNEL
func (m *FeConfigXMLManager) UpdateChannel(chGroupID int, param ChannelParam) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 协议层校验（先于任何修改）
	if err := ValidatePluginLayers(param.Plugins); err != nil {
		return err
	}

	// 1. 查找归属的CHANNEL_GROUP
	for i, cg := range m.feDefine.ChannelGroups {
		if cg.ID == chGroupID {
			// 2. 查找目标CHANNEL
			for j, ch := range cg.Channels {
				if ch.ID == param.ChannelID {
					if param.PossibleOwnerID == 0 {
						param.PossibleOwnerID = m.feDefine.Host.ID
					}

					m.feDefine.ChannelGroups[i].Channels[j].Comment = param.Comment
					m.feDefine.ChannelGroups[i].Channels[j].WorkOnDutyNode = param.WorkOnDutyNode
					m.feDefine.ChannelGroups[i].Channels[j].Name = param.Name
					m.feDefine.ChannelGroups[i].Channels[j].ServerGroupID = chGroupID + 1
					m.feDefine.ChannelGroups[i].Channels[j].PossibleOwner = PossibleOwner{ID: param.PossibleOwnerID}

					// 按层升序替换，Type 由 ID 推导
					m.feDefine.ChannelGroups[i].Channels[j].Plugins = sortPluginsByLayer(param.Plugins)

					return m.saveXML()
				}
			}
			return fmt.Errorf("CHANNEL_GROUP %d CHANNEL ID %d not exist", chGroupID, param.ChannelID)
		}
	}
	return fmt.Errorf("CHANNEL_GROUP ID %d not exist", chGroupID)
}

// -------------------------- 5.4 查询CHANNEL --------------------------
// GetChannelByID 查询指定CHANNEL_GROUP下的CHANNEL
func (m *FeConfigXMLManager) GetChannelByID(cgID, channelID int) (*Channel, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, cg := range m.feDefine.ChannelGroups {
		if cg.ID == cgID {
			for _, ch := range cg.Channels {
				if ch.ID == channelID {
					return &ch, nil
				}
			}
			return nil, fmt.Errorf("CHANNEL_GROUP %d CHANNEL ID %d not exist", cgID, channelID)
		}
	}

	return nil, fmt.Errorf("CHANNEL_GROUP ID %d not exist", cgID)
}

// ListChannelsByCGID 查询指定CHANNEL_GROUP下的所有CHANNEL
func (m *FeConfigXMLManager) ListChannelsByCGID(cgID int) ([]Channel, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, cg := range m.feDefine.ChannelGroups {
		if cg.ID == cgID {
			// 返回副本，避免外部修改原数据
			copyCh := make([]Channel, len(cg.Channels))
			copy(copyCh, cg.Channels)
			return copyCh, nil
		}
	}

	return nil, fmt.Errorf("CHANNEL_GROUP ID %d not exist", cgID)
}

// GetChannelsByRtuID 获取指定RTU下的所有CHANNEL
func (m *FeConfigXMLManager) GetChannelsByRtuID(rtuID string) (ChannelGroup, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, cg := range m.feDefine.ChannelGroups {
		for _, p := range cg.Model.Parameters {
			if p.Name == "STATION_BASE_ID" && p.Value == rtuID {
				return cg, nil
			}
		}
	}

	return ChannelGroup{}, fmt.Errorf("RTU ID %s not exist", rtuID)
}

// CreateChannelGroupWithChannel 自动创建通道组+通道（1:1），返回 (channelGroupID, channelID, error)
// STATION_BASE_ID 和 PossibleOwner.ID 由调用方决定，此处留空由外部设置
func (m *FeConfigXMLManager) CreateChannelGroupWithChannel(cgParam ChannelGroupParam, chParam ChannelParam) (int, int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 1. 生成 ChannelGroupID
	newCGID := 0
	for _, cg := range m.feDefine.ChannelGroups {
		if cg.ID >= newCGID {
			newCGID = cg.ID + 1
		}
	}

	// 2. 创建 ChannelGroup（含 Model）
	if cgParam.ModelID == 0 {
		cgParam.ModelID = 20000
	}
	model := Model{
		ID: cgParam.ModelID,
		Parameters: []Parameter{
			{Name: "FOR_TRANSFER", Value: cgParam.ForTransfer},
			{Name: "STATION_BASE_ID", Value: cgParam.StationBaseID},
		},
	}
	newCG := ChannelGroup{
		Comment:        cgParam.Comment,
		WorkOnDutyNode: cgParam.WorkOnDutyNode,
		ID:             newCGID,
		Name:           cgParam.Name,
		UpdateTime:     cgParam.UpdateTime,
		Model:          model,
		Channels:       []Channel{},
	}

	// 3. 在 ChannelGroup 下创建唯一 Channel
	newChID := 0
	if chParam.PossibleOwnerID == 0 {
		chParam.PossibleOwnerID = m.feDefine.Host.ID
	}
	newChannel := Channel{
		Comment:        chParam.Comment,
		WorkOnDutyNode: chParam.WorkOnDutyNode,
		ID:             newChID,
		ServerGroupID:  newCGID + 1,
		Name:           chParam.Name,
		Plugins:        []Plugin{},
		PossibleOwner: PossibleOwner{
			ID: chParam.PossibleOwnerID,
		},
	}
	newCG.Channels = append(newCG.Channels, newChannel)

	// 4. 添加到 ChannelGroups
	m.feDefine.ChannelGroups = append(m.feDefine.ChannelGroups, newCG)

	// 5. 保存
	if err := m.saveXML(); err != nil {
		return 0, 0, err
	}

	return newCGID, newChID, nil
}

// DeleteChannelGroupByStationID 按 STATION_BASE_ID 删除 ChannelGroup + Channel
func (m *FeConfigXMLManager) DeleteChannelGroupByStationID(rtuID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, cg := range m.feDefine.ChannelGroups {
		for _, p := range cg.Model.Parameters {
			if p.Name == "STATION_BASE_ID" && p.Value == rtuID {
				m.feDefine.ChannelGroups = append(m.feDefine.ChannelGroups[:i], m.feDefine.ChannelGroups[i+1:]...)
				return m.saveXML()
			}
		}
	}
	// 未找到不报错（可能是无通道 RTU）
	return nil
}

// GetChannelGroupByStationID 按 STATION_BASE_ID 查询 ChannelGroup
func (m *FeConfigXMLManager) GetChannelGroupByStationID(rtuID string) (*ChannelGroup, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for i := range m.feDefine.ChannelGroups {
		for _, p := range m.feDefine.ChannelGroups[i].Model.Parameters {
			if p.Name == "STATION_BASE_ID" && p.Value == rtuID {
				return &m.feDefine.ChannelGroups[i], nil
			}
		}
	}
	return nil, fmt.Errorf("RTU ID %s not found in any channel group", rtuID)
}

// ///////////////////////////// plugin ///////////////////////////////////
type PluginDefine struct {
	XMLName   xml.Name     `xml:"PLUGIN_DEFINE"`
	Model     PluginItem   `xml:"MODEL"`    // 单个MODEL节点
	Protocols []PluginItem `xml:"PROTOCOL"` // 多个PROTOCOL节点（切片）
}
type PluginItem struct {
	DLL  string `xml:"DLL,attr"`  // 映射DLL属性
	Type string `xml:"type,attr"` // 映射type属性（注意XML中是type，小写）
}

func ReadPluginTypeXMLToMap(xmlPath string) (map[string]string, error) {
	// 1. 检查文件是否存在
	if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("XML not exist: %s, err: %w", xmlPath, err)
	}

	// 2. 打开XML文件
	file, err := os.Open(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("open XML failed: %w", err)
	}
	defer file.Close() // 确保文件关闭

	// 3. 解析XML到结构体
	var pluginDef PluginDefine
	decoder := xml.NewDecoder(file)
	// 禁用XML实体解析（避免安全风险，同时兼容普通XML）
	decoder.Entity = xml.HTMLEntity
	if err := decoder.Decode(&pluginDef); err != nil {
		return nil, fmt.Errorf("parse XML failed: %w", err)
	}

	// 4. 生成DLL->type的map
	dllTypeMap := make(map[string]string)

	// 处理MODEL节点（若存在）
	if pluginDef.Model.DLL != "" {
		dllTypeMap[pluginDef.Model.DLL] = pluginDef.Model.Type
	}

	// 处理所有PROTOCOL节点
	for _, protocol := range pluginDef.Protocols {
		if protocol.DLL != "" {
			dllTypeMap[protocol.DLL] = protocol.Type
		}
	}

	// 5. 检查map是否为空（可选，根据业务需求调整）
	if len(dllTypeMap) == 0 {
		return nil, errors.New("parse XML file no dll node")
	}

	return dllTypeMap, nil
}

// GetChannelPlugins 获取可选协议，按层（ID 区间）分组返回；只返回非空层。
func GetChannelPlugins() ([]PluginLayer, error) {
	cgoPlugins, err := cgolibparse.GetPluginLibInfo()
	if err != nil {
		return nil, fmt.Errorf("GetPluginLibInfo failed: %w", err)
	}

	log.Info("GetChannelPlugins cgoPlugins:", cgoPlugins)

	layerMap := make(map[int][]Plugin)
	maxLayer := 0
	for _, plugin := range cgoPlugins {
		item := Plugin{ID: plugin.ID, Name: plugin.Name, Type: layerString(plugin.ID)}
		for name, value := range plugin.Params {
			item.Parameters = append(item.Parameters, Parameter{
				Name:      name,
				Value:     value.Default,
				ValueList: value.ValueList,
			})
		}
		layer := pluginLayer(plugin.ID)
		layerMap[layer] = append(layerMap[layer], item)
		if layer > maxLayer {
			maxLayer = layer
		}
	}

	result := make([]PluginLayer, 0, maxLayer)
	for layer := 1; layer <= maxLayer; layer++ {
		if plugins := layerMap[layer]; len(plugins) > 0 {
			result = append(result, PluginLayer{Type: strconv.Itoa(layer), Plugins: plugins})
		}
	}
	return result, nil
}
