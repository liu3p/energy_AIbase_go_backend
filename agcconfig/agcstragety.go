package agcconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
	"webserver-arm/cgolibparse"
	"webserver-arm/log"
)

// -------------------------- 1. 定义JSON对应的结构体 --------------------------

type LibStrategy cgolibparse.StrategyLibInfo

type Strategy struct {
	Name     string                    `json:"name"` // 策略名称
	Desc     string                    `json:"desc,omitempty"`
	Priority int                       `json:"priority"` // 策略优先级
	Setting  []cgolibparse.SettingItem `json:"setting"`  // 配置参数列表
	DynPara  []string                  `json:"-"`
	CalcPara []cgolibparse.SettingItem `json:"calcpara"`
}

func isAllowedType(settingType string) bool {
	if settingType == "CONTROL" || settingType == "REGULATE" || settingType == "ATTRIBUTE" {
		return true
	} else {
		return false
	}
}

func (s Strategy) MarshalJSON() ([]byte, error) {
	// 1. 过滤 Setting
	filteredSetting := make([]cgolibparse.SettingItem, 0, len(s.Setting))
	for _, item := range s.Setting {
		if isAllowedType(item.Type) {
			filteredSetting = append(filteredSetting, item)
		}
	}

	// 2. 过滤 CalcPara
	filteredCalcPara := make([]cgolibparse.SettingItem, 0, len(s.CalcPara))
	for _, item := range s.CalcPara {
		if isAllowedType(item.Type) {
			filteredCalcPara = append(filteredCalcPara, item)
		}
	}

	result := make(map[string]interface{})
	result["name"] = s.Name
	result["desc"] = s.Desc
	result["priority"] = s.Priority
	result["setting"] = filteredSetting
	result["calcpara"] = filteredCalcPara

	return json.MarshalIndent(result, "", "    ")
}

func (s *Strategy) String() string {
	if s == nil {
		return "<nil>"
	}
	// 自定义打印内容，包含所有关键字段（按需调整）
	return fmt.Sprintf(
		"Strategy{Name:%s, Desc:%s, Setting:%v, DynPara:%v}",
		s.Name, s.Desc, s.Setting, s.DynPara,
	)
}

// StrategyRoot JSON根结构
type StrategyRoot struct {
	Strategy []Strategy `json:"strategy"`
}

// -------------------------- 2. 类无关的函数 --------------------------

// mergeSettings 合并配置参数
func mergeSettings(oldSettings []cgolibparse.SettingItem, newTemplates []cgolibparse.SettingItem) []cgolibparse.SettingItem {
	newSettingsMap := make(map[string]cgolibparse.SettingItem)
	for _, d := range newTemplates {
		newSettingsMap[d.Name] = d
	}

	merged := make([]cgolibparse.SettingItem, 0, len(newTemplates))
	// 保留旧DatabaseID
	for _, old := range oldSettings {
		if newTemp, ok := newSettingsMap[old.Name]; ok {
			newTemp.Value = old.Value // 保留参数值
			merged = append(merged, newTemp)
			delete(newSettingsMap, old.Name)
		}
	}

	// 添加模板新增的动态参数
	for _, d := range newSettingsMap {
		merged = append(merged, d)
	}

	return merged
}

// SetStrategyStatus 调用HTTP接口设置C库状态
func SetStrategySettingParams(paraType, path, value, valueType string) error {
	// 构建请求JSON
	reqBody := map[string]string{
		"type":      paraType,
		"path":      path,
		"value":     value,
		"data_type": valueType,
	}

	// 序列化请求体
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("SetStrategySettingParams marshal failed: %w", err)
	}

	// 创建HTTP客户端
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", "http://127.0.0.1:8500/", bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("REQUEST ERROR: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("REQUEST FAILED: %d", resp.StatusCode)
	}

	log.Infof("SetStrategySettingParams %s: %s: %s (%s) success", paraType, path, value, valueType)
	return nil
}

// GetAgcBasePath 获取AGC基础路径（从环境变量）
func GetAgcBasePath() string {
	agcEnv := os.Getenv("ARM_SF")
	if agcEnv == "" {
		log.Error("env ARM_SF not set")
		return ""
	}
	return agcEnv
}

// getStrategyFilePath 获取策略配置文件路径
func getStrategyFilePath() string {
	basePath := GetAgcBasePath()
	if basePath == "" {
		return ""
	}
	return filepath.Join(basePath, "etc/agc/", "strategy.json")
}

// getStrategyLibPath 获取策略动态库路径
func getStrategyLibPath() string {
	basePath := GetAgcBasePath()
	if basePath == "" {
		return ""
	}
	return filepath.Join(basePath, "lib", "strategy")
}

// -------------------------- 3. 单例策略管理器 --------------------------

// AgcStrategyManager 策略管理器
type AgcStrategyManager struct {
	filePath         string                  // 配置文件路径
	mu               sync.RWMutex            // 读写锁（读多写少优化）
	root             *StrategyRoot           // 内存中的配置数据
	libStrategyCache map[string]*LibStrategy // 动态库策略缓存
	isSelfWriting    bool                    // 标记是否为程序自身写入文件
}

var (
	strategyInstance *AgcStrategyManager
	strategyOnce     sync.Once
	strategyInitErr  error
)

func GetStrategyInstance() (*AgcStrategyManager, error) {
	strategyOnce.Do(func() {
		// 获取配置文件路径
		filePath := getStrategyFilePath()
		if filePath == "" {
			strategyInitErr = errors.New("get strategy file path failed")
			return
		}

		// 初始化管理器
		m := &AgcStrategyManager{
			filePath:         filePath,
			libStrategyCache: make(map[string]*LibStrategy),
		}

		// 加载配置文件
		if err := m.load(); err != nil {
			strategyInitErr = fmt.Errorf("load strategy file failed: %w", err)
			return
		}

		// 预加载动态库策略
		if err := m.loadLibStrategies(); err != nil {
			log.Warn("loadLibStrategies err: %v", err)
		}

		// 用索引遍历，直接操作原切片数据
		for i := range m.root.Strategy {
			origStrategy := &m.root.Strategy[i]
			// 从动态库缓存匹配策略
			if newStrategy, ok := m.libStrategyCache[origStrategy.Name]; ok {
				origStrategy.Priority = newStrategy.Priority
				// 合并参数，直接修改原数据
				origStrategy.Setting = mergeSettings(origStrategy.Setting, newStrategy.Setting)
			}
		}

		_ = WatchFile(m.filePath, func() {
			if m.isSelfWriting {
				m.isSelfWriting = false
				return
			}
			// model.json文件更新后重新加载其中内容
			if err := m.load(); err != nil {
				log.Errorf("reload model.json 失败: %v", err)
			} else {
				log.Infof("model.json reload success")
			}
		})

		strategyInstance = m
	})

	return strategyInstance, strategyInitErr
}

// -------------------------- 4. 策略管理器内部方法 --------------------------
// load 从文件加载配置
func (m *AgcStrategyManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		// 文件不存在，初始化空结构
		m.root = &StrategyRoot{Strategy: []Strategy{}}
		return m.save()
	}

	// 读取文件内容
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	// 反序列化JSON
	var root StrategyRoot
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse JSON failed: %w", err)
	}

	// 策略的Path字段赋值
	for i := range root.Strategy {
		prefix := "/strategy/" + root.Strategy[i].Name + "/"
		for j := range root.Strategy[i].Setting {
			root.Strategy[i].Setting[j].Path = prefix + root.Strategy[i].Setting[j].Type + "/" + root.Strategy[i].Setting[j].Name
		}
		for j := range root.Strategy[i].CalcPara {
			root.Strategy[i].CalcPara[j].Path = prefix + root.Strategy[i].CalcPara[j].Type + "/" + root.Strategy[i].CalcPara[j].Name
		}
	}

	m.root = &root
	return nil
}

// save 将配置保存到文件
func (m *AgcStrategyManager) save() error {
	m.isSelfWriting = true

	// 序列化JSON（带缩进，便于阅读）
	data, err := json.MarshalIndent(m.root, "", "    ")
	if err != nil {
		return fmt.Errorf("save strategy marshal json failed: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("save strategy write in file failed: %w", err)
	}

	return nil
}

// loadLibStrategies 从动态库加载策略（预留接口，可通过purego实现）
func (m *AgcStrategyManager) loadLibStrategies() error {
	// 此处为遍历getStrategyLibPath接口返回的目录，获取全部的so为扩展名的lib文件，
	// 然后遍历这些lib文件，调用cgolibparse.GetStrategyLibInfo获取策略信息，并缓存起来，下次调用时直接从缓存中获取。
	files, err := filepath.Glob(filepath.Join(getStrategyLibPath(), "lib*.so*"))
	if err != nil {
		return fmt.Errorf("getStrategyLibPath failed: %w", err)
	}
	for _, file := range files {
		log.Infof("loadLibStrategies load lib: %s", file)
		libInfo, err := cgolibparse.GetStrategyLibInfo(file)
		if err != nil {
			log.Warnf("getStrategyLibInfo err: %v", err)
			continue
		}
		// fmt.Println(libInfo)
		m.libStrategyCache[libInfo.Name] = &LibStrategy{
			Name:     libInfo.Name,
			Desc:     libInfo.Desc,
			Priority: libInfo.Priority,
			Setting:  libInfo.Setting,
			DynPara:  libInfo.DynPara,
		}
	}

	return nil
}

// getStrategyFromCache 从缓存中获取单个策略
func (m *AgcStrategyManager) getStrategyFromCache(sname string) (*Strategy, error) {
	if s, ok := m.libStrategyCache[sname]; ok {
		s := &Strategy{Name: s.Name, Desc: s.Desc, Setting: s.Setting}
		return s, nil
	}
	return nil, fmt.Errorf("strategy %s not exist", sname)
}

// getStrategyByName 根据策略名称查找，返回策略指针
// 入参：sname 策略名称
// 返回值：匹配到的策略指针，未找到则返回error
func (m *AgcStrategyManager) getStrategyByName(sname string) (*Strategy, error) {
	// 遍历策略列表，按名称匹配
	for i := range m.root.Strategy {
		if m.root.Strategy[i].Name == sname {
			// 直接返回切片中元素的指针（避免值拷贝）
			return &m.root.Strategy[i], nil
		}
	}

	// 未找到对应策略，返回标准错误
	return nil, fmt.Errorf("strategy %s not exist", sname)
}

// getAllStrategyPriorities 获取所有策略的优先级
func (m *AgcStrategyManager) getAllStrategyPriorities() map[string]int {
	result := make(map[string]int)
	for _, st := range m.libStrategyCache {
		result[st.Name] = st.Priority
	}
	return result
}

// -------------------------- 5. 策略管理器对外接口实现 --------------------------

// GetAllStrategyFromCache 从缓存获取所有已添加的策略
func (m *AgcStrategyManager) GetAllStrategyFromCache() ([]*Strategy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var strategies []*Strategy
	for _, strategy := range m.root.Strategy {
		if s, ok := m.libStrategyCache[strategy.Name]; ok {
			strategies = append(strategies, &Strategy{Name: s.Name, Desc: s.Desc, Setting: s.Setting, DynPara: s.DynPara})
		}
	}

	return strategies, nil
}

// GetAllStrategies 获取所有策略
func (m *AgcStrategyManager) GetAllStrategies() ([]*Strategy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var strategies []*Strategy
	for _, s := range m.libStrategyCache {
		strategies = append(strategies, &Strategy{Name: s.Name, Desc: s.Desc, Setting: s.Setting, DynPara: s.DynPara})
	}

	return strategies, nil
}

// GetStrategies 获取当前配置的策略列表（含激活状态，返回JSON字符串）

type ResultStrategies struct {
	Name   string `json:"name"`
	Desc   string `json:"desc,omitempty"`
	Active int    `json:"active"` // 1:运行，0:停止
}

func (m *AgcStrategyManager) GetStrategies() ([]ResultStrategies, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 构造返回数据

	var result []ResultStrategies

	for _, s := range m.root.Strategy {
		// 检查激活状态
		var active = 0
		for _, setting := range s.Setting {
			if setting.Name == "Activate" && setting.Value == "1" {
				active = 1
				break
			}
		}

		result = append(result, ResultStrategies{Name: s.Name, Active: active})
	}

	return result, nil
}

// SetStrategyParams 更新策略参数
func (m *AgcStrategyManager) SetStrategyParams(sname string, settings []cgolibparse.SettingItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 查找策略是否存在
	strategy, err := m.getStrategyByName(sname)
	if err != nil {
		return err
	}

	settingMap := make(map[string]cgolibparse.SettingItem)
	for _, seeting := range settings {
		settingMap[seeting.Name] = seeting
	}
	// 更新参数
	for i := range strategy.Setting {
		if s, ok := settingMap[strategy.Setting[i].Name]; ok {
			// 前端传入的值不一样时才进行修改
			if strategy.Setting[i].Value != s.Value {
				// 调用http接口更新配置参数
				paraPath := "/" + sname + "/" + s.Name
				err := SetStrategySettingParams("strategy_para", paraPath, s.Value, s.Type)
				if err != nil {
					log.Errorf("SetStrategyParams SetStrategySettingParams failed: %v", err)
					return err
				}
				strategy.Setting[i].Value = s.Value
			}
			delete(settingMap, strategy.Setting[i].Name)
		}
	}

	// 打印不存在的参数（警告）
	for name := range settingMap {
		log.Warnf("strategy %s not exist params %s, Setting:%+v", sname, name, strategy.Setting)
	}

	return nil
}

// AddStrategy 添加策略到配置文件
func (m *AgcStrategyManager) AddStrategy(sname string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 查找策略是否已存在
	strategy, err := m.getStrategyByName(sname)
	if strategy != nil {
		return fmt.Errorf("AddStrategy strategy %s exist", sname)
	}

	// 从缓存获取策略
	strategy, err = m.getStrategyFromCache(sname)
	if err != nil {
		return err
	}

	// 添加到配置
	m.root.Strategy = append(m.root.Strategy, *strategy)

	// 保存文件
	return m.save()
}

// RemoveStrategy 从配置文件中删除策略（遍历重建切片，无索引append删除）
func (m *AgcStrategyManager) RemoveStrategy(sname string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 首先查找是否策略存在
	if _, err := m.getStrategyByName(sname); err != nil {
		return err
	}

	// 新建空切片，只保留「不是目标名称」的策略
	newStrategies := make([]Strategy, 0, len(m.root.Strategy))
	for _, s := range m.root.Strategy {
		if s.Name == sname {
			continue
		}
		newStrategies = append(newStrategies, s)
	}

	// 用新切片替换原切片（完成删除）
	m.root.Strategy = newStrategies

	// 保存文件
	return m.save()
}

// GetStrategyParams 获取策略的配置参数（排除Activate，返回JSON字符串）
func (m *AgcStrategyManager) GetStrategyParams(sname string) ([]cgolibparse.SettingItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 查找策略是否存在
	strategy, err := m.getStrategyByName(sname)
	if err != nil {
		return nil, err
	}

	var settings []cgolibparse.SettingItem
	for _, s := range strategy.Setting {
		// 过滤Activate项
		if s.Name != "Activate" {
			// 仅保留遥控和遥调参数
			if s.Type == "CONTROL" || s.Type == "REGULATE" || s.Type == "ATTRIBUTE" {
				settings = append(settings, s)
			}
		}
	}

	return settings, nil
}

// CheckPriorityConflict 检查优先级冲突（返回冲突列表JSON字符串）
type ConflictItem struct {
	Name string `json:"name"`
}

func (m *AgcStrategyManager) CheckPriorityConflict(sname string) ([]ConflictItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取策略优先级
	priorities := m.getAllStrategyPriorities()

	// 获取目标策略优先级
	targetPriority, ok := priorities[sname]
	if !ok {
		return nil, fmt.Errorf("strategy %s priority not exist", sname)
	}

	// 查找冲突策略
	conflicts := make([]ConflictItem, 0)

	for _, s := range m.root.Strategy {
		// 获取策略优先级
		priority, ok := priorities[s.Name]
		if !ok {
			continue
		}

		// 优先级相同时检查是否已激活
		if priority == targetPriority {
			for _, setting := range s.Setting {
				if setting.Name == "Activate" && setting.Value == "1" {
					conflicts = append(conflicts, ConflictItem{Name: s.Name})
					break
				}
			}
		}
	}
	return conflicts, nil
}

func (m *AgcStrategyManager) GetAgcRtuModelPoints(name, paraType, dataType string, exceptPaths map[string]struct{}) (points []*AgcRtuModelPoints) {
	// 读操作使用读锁，提升并发性能
	m.mu.RLock()
	defer m.mu.RUnlock()

	processItems := func(items []cgolibparse.SettingItem, prefix string) {
		for j := range items {
			// 筛选遥测、遥信等
			if dataType == "all" || dataType == items[j].Type {
				point := &AgcRtuModelPoints{
					Name: prefix + items[j].Name,
					Desc: items[j].Desc,
					Type: items[j].Type,
					Max:  items[j].Max,
					Min:  items[j].Min,
					Path: items[j].Path,
				}
				if _, has := exceptPaths[point.Path]; !has {
					points = append(points, point)
				}
			}
		}
	}

	for strategyName, strategy := range m.libStrategyCache {
		if name == "all" || name == strategyName {
			if paraType == "all" || paraType == "para" {
				processItems(strategy.Setting, strategyName+"/")
			}
			if paraType == "all" || paraType == "calc_para" {
				processItems(strategy.CalcPara, strategyName+"/")
			}
		}
	}
	return points
}

// if paraType == "all" || paraType == "dyn_para" {
// for j := range m.root.Strategy[i].DynPara {
// 	var dynPara DynPara
// 	err := json.Unmarshal([]byte(m.root.Strategy[i].DynPara[j]), &dynPara)
// 	if err != nil {
// 		continue
// 	}
// 	point := &AgcRtuModelPoints{
// 		Name: m.root.Strategy[i].Name + "/" dynPara.Name,
// 		Desc: dynPara.Desc,
// 		Path: dynPara.Path,
// 		Type: dynPara.Type,
// 	}
// }
// }
