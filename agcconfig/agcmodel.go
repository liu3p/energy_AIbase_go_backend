package agcconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
	"webserver-arm/cgolibparse"
	"webserver-arm/log"

	"github.com/fsnotify/fsnotify"
)

// ############################## 基础结构体定义 ##############################

// DynPara 动态参数
type DynPara struct {
	DatabaseID string `json:"database_id"`
	Desc       string `json:"desc,omitempty"` // 空值不输出
	Name       string `json:"name"`
	Type       string `json:"type"` // ANALOG/DIGITAL/PULSE
	Path       string `json:"-"`    // 不参与默认序列化，用于在点表配置中配置agc设备
}

// Para 静态参数
type Para struct {
	Desc  string `json:"desc"`
	Max   string `json:"max"`
	Min   string `json:"min"`
	Name  string `json:"name"`
	Type  string `json:"type"` // INT/STRING等
	Value string `json:"value"`
	Path  string `json:"-"` // 不参与默认序列化，用于在点表配置中配置agc设备
}

// Device 设备主体（支持多层级嵌套）
type Device struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	DynPara    []DynPara `json:"dyn_para"`
	Para       []Para    `json:"para,omitempty"` // 空切片不输出
	SubDevices []*Device `json:"-"`              // 不参与默认序列化，仅内存维护子设备
	Path       string    `json:"-"`              // 不参与默认序列化，用于在点表配置中配置agc设备
}

// ############################## JSON 自定义编解码 ##############################

// MarshalJSON 自定义序列化：子设备按类型分组转为数组
func (d Device) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})
	// 基础固定字段
	result["name"] = d.Name
	result["type"] = d.Type
	result["dyn_para"] = d.DynPara
	para := make([]Para, 0)
	for i := range d.Para {
		if isAllowedType(d.Para[i].Type) {
			para = append(para, d.Para[i])
		}
	}
	result["para"] = para

	// 子设备按类型分组生成数组
	for i := range d.SubDevices {
		if d.SubDevices[i] == nil {
			continue
		}
		key := d.SubDevices[i].Type
		if _, ok := result[key]; !ok {
			result[key] = []*Device{}
		}
		result[key] = append(result[key].([]*Device), d.SubDevices[i])
	}

	return json.MarshalIndent(result, "", "    ")
}

// UnmarshalJSON 自定义反序列化：递归解析数组格式子设备
func (d *Device) UnmarshalJSON(data []byte) error {
	// 获取json根结构里有哪些字段
	var tempMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &tempMap); err != nil {
		return fmt.Errorf("解析根结构失败: %w", err)
	}

	// 解析基础字段
	if err := json.Unmarshal(tempMap["name"], &d.Name); err != nil {
		return fmt.Errorf("解析name失败: %w", err)
	}
	if err := json.Unmarshal(tempMap["type"], &d.Type); err != nil {
		return fmt.Errorf("解析type失败: %w", err)
	}

	// 忽略空值错误
	_ = json.Unmarshal(tempMap["dyn_para"], &d.DynPara)
	_ = json.Unmarshal(tempMap["para"], &d.Para)

	// 固定字段白名单，其余均为子设备
	fixedFields := map[string]bool{"name": true, "type": true, "dyn_para": true, "para": true}
	for key, val := range tempMap {
		if fixedFields[key] {
			continue
		}
		// 解析子设备数组
		var subList []*Device
		if err := json.Unmarshal(val, &subList); err != nil {
			return fmt.Errorf("解析子设备[%s]失败: %w", key, err)
		}
		d.SubDevices = append(d.SubDevices, subList...)
	}

	return nil
}

// ############################## 设备工具方法 ##############################

// 新增：递归设置子设备Path（全层级生效，支持无限嵌套）
func (d *Device) setSubDevicePaths() {
	if d.Type == "STATION" {
		d.Path = "/station/" + d.Name
	}
	for i := range d.SubDevices {
		// 子设备Path = 父Path + / + 子设备Name
		d.SubDevices[i].Path = d.Path + "/" + d.SubDevices[i].Name
		// 递归给子设备的Path赋值
		d.SubDevices[i].setSubDevicePaths()
	}
	for i := range d.Para {
		// 参数Path = 父Path + / + 参数Type + / + 参数Name
		d.Para[i].Path = d.Path + "/" + d.Para[i].Type + "/" + d.Para[i].Name
	}
	for i := range d.DynPara {
		// 参数Path = 父Path + / + 参数Type + / + 参数Name
		d.DynPara[i].Path = d.Path + "/" + d.DynPara[i].Type + "/" + d.DynPara[i].Name
	}
}

func (d *Device) printPath() {
	fmt.Println(d.Name, d.Path)
	for i := range d.Para {
		fmt.Println(d.Para[i].Name, d.Para[i].Path)
	}
	for i := range d.DynPara {
		fmt.Println(d.DynPara[i].Name, d.DynPara[i].Path)
	}
	for i := range d.SubDevices {
		d.SubDevices[i].printPath()
	}
}

// isNameExists 递归检查全树Name是否重复（全局唯一）
func (d *Device) isNameExists(name string) bool {
	if d.Name == name {
		return true
	}
	for i := range d.SubDevices {
		if d.SubDevices[i].isNameExists(name) {
			return true
		}
	}
	return false
}

// findDeviceByName 递归查找指定Name的设备节点
func (d *Device) findDeviceByName(name string) *Device {
	if d.Name == name {
		return d
	}
	for i := range d.SubDevices {
		if found := d.SubDevices[i].findDeviceByName(name); found != nil {
			return found
		}
	}
	return nil
}

// ############################## 设备 CRUD 方法 ##############################

// addSubDevice 添加子设备（强制Name全局唯一）
func (d *Device) addSubDevice(dev *Device) error {
	if dev == nil {
		return fmt.Errorf("设备不能为空")
	}
	if d.isNameExists(dev.Name) {
		return fmt.Errorf("设备名称[%s]已存在，不允许重复", dev.Name)
	}
	dev.Path = d.Path + "/" + dev.Name
	d.SubDevices = append(d.SubDevices, dev)
	return nil
}

// addSubDeviceToNodeByName 按父节点Name递归添加子设备
func (d *Device) addSubDeviceToNodeByName(parentName string, dev *Device) error {
	parent := d.findDeviceByName(parentName)
	if parent == nil {
		return fmt.Errorf("未找到父节点[%s]", parentName)
	}
	return parent.addSubDevice(dev)
}

// removeSubDeviceByName 递归删除指定Name的子设备
func (d *Device) removeSubDeviceByName(name string) bool {
	for i := range d.SubDevices {
		if d.SubDevices[i].Name == name {
			d.SubDevices = append(d.SubDevices[:i], d.SubDevices[i+1:]...) // ...表示把切片拆成独立元素进行传参
			return true
		}
		// 递归删除子节点
		if d.SubDevices[i].removeSubDeviceByName(name) {
			return true
		}
	}
	return false
}

// updateSubDeviceByName 递归修改指定Name的设备
func (d *Device) updateSubDeviceByName(name string, newDev *Device) bool {
	if newDev == nil {
		return false
	}
	// 检查新Name是否重复（排除自身）
	if name != newDev.Name && d.isNameExists(newDev.Name) {
		return false
	}
	for i := range d.SubDevices {
		if d.SubDevices[i].Name == name {
			d.SubDevices[i] = newDev
			return true
		}
		// 递归修改子节点
		if d.SubDevices[i].updateSubDeviceByName(name, newDev) {
			return true
		}
	}
	return false
}

// getSubDeviceByName 递归查找指定Name的设备
func (d *Device) getSubDeviceByName(name string) (*Device, bool) {
	target := d.findDeviceByName(name)
	return target, target != nil
}

// clearSubDevices 清空当前节点所有子设备
func (d *Device) clearSubDevices() {
	d.SubDevices = nil
}

// ############################## 单例设备管理器 ##############################

// Singleton variables
var (
	modelInstance *AgcModelManager
	modelOnce     sync.Once
	modelInitErr  error
)

// AgcModelManager AGC model manager
type AgcModelManager struct {
	filePath    string       // Model file path (model.json)
	agcBasePath string       // libagc_base.so  path (libagc_base.json)
	mu          sync.RWMutex // Read-write lock for in-memory data
	fileMu      sync.RWMutex // Read-write lock for file operations
	station     *Device      // In-memory station model
	//modify by zhh 2026.4.3 取消使用模板文件，改为从C库函数中读取静态参数
	templateCache map[string][]Para // Template cache (key: type, value: template data)
	isSelfWriting bool              // 标记是否为程序自身写入文件
}

// GetInstance Get singleton instance
func GetModelInstance() (*AgcModelManager, error) {
	modelOnce.Do(func() {
		// 1. 获取基础路径
		basePath := GetAgcBasePath()
		if basePath == "" {
			modelInitErr = errors.New("AGC environment variable not set, cannot get base path")
			return // 初始化失败，退出
		}

		// 2. 构造管理器对象
		m := &AgcModelManager{
			filePath:      filepath.Join(basePath, "etc/agc/", "model.json"),
			agcBasePath:   filepath.Join(basePath, "lib", "libagc_base.so"),
			templateCache: make(map[string][]Para),
			mu:            sync.RWMutex{}, //必须初始化锁！
		}

		// 3. 加载模型文件
		if err := m.loadModel(); err != nil {
			modelInitErr = fmt.Errorf("failed to Load model file: %w", err)
			return
		}

		// 4. 加载静态参数
		parasMap, err := cgolibparse.GetDeviceStaticParas()
		if err != nil {
			modelInitErr = fmt.Errorf("failed to Load template files: %w", err)
			return
		}

		// 5. 解析参数
		for key, value := range parasMap {
			var para []Para
			if err := json.Unmarshal([]byte(value), &para); err != nil {
				modelInitErr = fmt.Errorf("Failed to unmarshal static paras : %s, err : %w", &value, err)
				return
			}
			m.templateCache[key] = para
		}

		// 更新动态参数和静态参数
		m.UpdateParaAndDynPara()
		// Path字段赋值
		m.station.setSubDevicePaths()

		// 创建文件监听器
		_ = WatchFile(m.filePath, func() {
			if m.isSelfWriting {
				m.isSelfWriting = false
				return
			}
			// model.json文件更新后重新加载其中内容
			if err := m.loadModel(); err != nil {
				log.Errorf("reload model.json 失败: %v", err)
			} else {
				log.Infof("model.json reload success")
			}
		})

		modelInstance = m
		log.Info("Model Instance success")
	})

	// 最终返回
	return modelInstance, modelInitErr
}

// ############################## 设备管理器内部方法 ##############################

// WatchModelFile 监听 model.json 文件变化，自动重新加载
// filePath: 要监听的文件完整路径
// callback: 文件变化后要执行的函数
func WatchFile(filePath string, callback func()) error {
	// 1. 创建监听器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// 2. 监听文件所在目录
	dir := filepath.Dir(filePath)
	if err := watcher.Add(dir); err != nil {
		return err
	}

	log.Infof("start listen file : %s", filePath)

	// 后台协程处理事件
	go func() {
		var debounceTimer *time.Timer

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// 只匹配目标文件
				if filepath.Clean(event.Name) != filepath.Clean(filePath) {
					continue
				}

				// 只处理创建/写入事件
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					if debounceTimer != nil {
						debounceTimer.Stop()
					}

					// 300ms 防抖
					debounceTimer = time.AfterFunc(300*time.Millisecond, func() {
						log.Infof("file update: %s", filePath)
						// 执行传入的回调函数
						if callback != nil {
							callback()
						}
					})
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Errorf("file listen failed: %v", err)
			}
		}
	}()

	return nil
}

// findDevicesByType 递归查找【所有指定类型】的设备（用于批量更新参数）
func (m *AgcModelManager) findDevicesByType(devType string) []*Device {
	var result []*Device
	var traverse func(d *Device)

	traverse = func(d *Device) {
		if d == nil {
			return
		}
		if d.Type == devType {
			result = append(result, d)
		}
		for i := range d.SubDevices {
			traverse(d.SubDevices[i])
		}
	}

	traverse(m.station)
	return result
}

func genParamKey(param Para) string {
	return param.Name + "_" + param.Type
}

// mergePara 合并静态参数：保留原有Value，用新模板覆盖Desc/Max/Min/Type
// 规则：
// 1. 按Name匹配
// 2. 存在则更新描述/上下限，保留Value
// 3. 模板新增则添加
// 4. 旧参数不在模板中则删除
func mergePara(oldParas []Para, newTemplates []Para) []Para {
	newParaMap := make(map[string]Para)
	for i := range newTemplates {
		newParaMap[genParamKey(newTemplates[i])] = newTemplates[i]
	}

	merged := make([]Para, 0, len(newTemplates))
	// 保留旧Value
	for i := range oldParas {
		oldParamKey := genParamKey(oldParas[i])
		if newTemp, ok := newParaMap[oldParamKey]; ok {
			newTemp.Value = oldParas[i].Value // 保留用户配置的Value
			newTemp.Path = oldParas[i].Path   // 保留Path字段
			merged = append(merged, newTemp)
			delete(newParaMap, oldParamKey)
		}
	}

	// 添加模板新增的参数
	for i := range newParaMap {
		merged = append(merged, newParaMap[i])
	}

	return merged
}

func genDynParamKey(param DynPara) string {
	return param.Name + "_" + param.Type
}

// mergeDynPara 合并动态参数：保留原有DatabaseID，其他用新模板覆盖
func mergeDynPara(oldDyns []DynPara, newTemplates []DynPara) []DynPara {
	newDynMap := make(map[string]DynPara)
	for i := range newTemplates {
		newDynMap[genDynParamKey(newTemplates[i])] = newTemplates[i]
	}

	merged := make([]DynPara, 0, len(newTemplates))
	// 保留旧DatabaseID
	for i := range oldDyns {
		oldDynParamKey := genDynParamKey(oldDyns[i])
		if newTemp, ok := newDynMap[oldDynParamKey]; ok {
			newTemp.DatabaseID = oldDyns[i].DatabaseID // 保留数据库ID
			newTemp.Path = oldDyns[i].Path             // 保留Path字段
			merged = append(merged, newTemp)
			delete(newDynMap, oldDynParamKey)
		}
	}

	// 添加模板新增的动态参数
	for i := range newDynMap {
		merged = append(merged, newDynMap[i])
	}

	return merged
}

func (m *AgcModelManager) loadModel() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		// Initialize empty station if file not exists
		m.station = &Device{Type: "STATION", Name: "station", Path: "/station/station/"}
		return m.saveModel()
	}

	// Read file content
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to read model file: %w", err)
	}

	// Deserialize JSON
	var station Device
	if err := json.Unmarshal(data, &station); err != nil {
		return fmt.Errorf("failed to parse model JSON: %w", err)
	}

	m.station = &station
	return nil
}

// saveModel Save model to file
func (m *AgcModelManager) saveModel() error {
	m.isSelfWriting = true
	// Serialize with indentation
	data, err := json.MarshalIndent(m.station, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize model: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	return nil
}

// 定义参数类型白名单（非设备类型，仅作为参数类型）
var paramTypeWhitelist = map[string]bool{
	"ANALOG":    true,
	"DIGITAL":   true,
	"PLUSE":     true,
	"CONTROL":   true,
	"REGULATE":  true,
	"ATTRIBUTE": true,
	"analog":    true,
	"digital":   true,
	"pluse":     true,
	"control":   true,
	"regulate":  true,
	"attribute": true,
}

// 获取设备的动态参数列表
func getDeviceDynParams(deviceType string) ([]DynPara, error) {
	if strings.TrimSpace(deviceType) == "" {
		return nil, fmt.Errorf("deviceType can not be empty")
	}

	pathList := make([]string, 0)
	ins, err := GetStrategyInstance()
	if err != nil {
		return nil, fmt.Errorf("getDeviceDynParams GetStrategyInstance failed")
	}
	stragies, err := ins.GetAllStrategyFromCache()
	if err != nil {
		return nil, fmt.Errorf("failed to get all strategy from lib: %w", err)
	}

	for i := range stragies {
		pathList = append(pathList, stragies[i].DynPara...)
	}

	var result []DynPara
	paraMap := make(map[string]struct{})
	for _, path := range pathList {
		// 过滤空路径
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		// 拆分路径为层级切片（按/分割，过滤空字符串）
		levels := strings.Split(path, "/")

		for i := range levels {
			level := strings.TrimSpace(levels[i])
			if level == deviceType {
				// 检查：设备类型后必须有参数类型（下一层）+ 参数名（下两层）
				// i+1: 参数类型层级；i+2: 参数名层级
				if i+1 >= len(levels) || !paramTypeWhitelist[levels[i+1]] {
					continue // 设备类型后无参数类型，或不是合法参数类型，跳过
				}
				if i+2 >= len(levels) {
					continue // 参数类型后无参数名，跳过
				}

				// 提取参数信息，构造结构体，同时排除同名同类型参数
				paramKey := levels[i+1] + "_" + levels[i+2]
				if _, has := paraMap[paramKey]; !has {
					paraMap[paramKey] = struct{}{}
					param := DynPara{
						Name: levels[i+2],
						Type: levels[i+1],
					}
					result = append(result, param)
				}
			}
		}
	}
	return result, nil
}

// updateDevice 递归反射增量更新，支持修改 Para[].Value、DynPara[].DatabaseID 等嵌套字段
func (m *AgcModelManager) updateDevice(device interface{}, updateMap map[string]interface{}) error {
	val := reflect.ValueOf(device)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("model must be a pointer to a struct")
	}
	// 递归更新顶层结构体
	return m.updateStruct(val.Elem(), updateMap)
}

// updateStruct 递归更新结构体字段（支持切片/嵌套结构体）
func (m *AgcModelManager) updateStruct(structVal reflect.Value, updateMap map[string]interface{}) error {
	for key, newValue := range updateMap {
		fmt.Println(key)
		// 字段名映射：json小驼峰 → 大驼峰
		fieldName := toStructFieldName(key)
		fieldVal := structVal.FieldByName(fieldName)

		// 字段不存在/不可修改 → 跳过
		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			continue
		}

		// 新字段为空 → 跳过
		if newValue == nil {
			log.Warnf("字段 %s 的更新值为 nil, 跳过更新", fieldName)
			continue
		}

		// 处理逻辑：
		// 1. 如果是普通字段（string/int等）→ 直接赋值
		// 2. 如果是切片（如[]Para、[]DynPara）→ 递归处理内部元素
		// 3. 如果是嵌套结构体 → 递归处理

		switch fieldVal.Kind() {
		case reflect.Slice:
			// 处理切片：Para、DynPara
			if err := m.updateSlice(fieldVal, newValue); err != nil {
				log.Warnf("update slice %s failed: %v", fieldName, err)
			}
		case reflect.Struct:
			// 处理嵌套结构体
			if nestedMap, ok := newValue.(map[string]interface{}); ok {
				_ = m.updateStruct(fieldVal, nestedMap)
			}
		default:
			// 普通字段直接赋值
			newVal := reflect.ValueOf(newValue)
			if newVal.Type().AssignableTo(fieldVal.Type()) {
				fieldVal.Set(newVal)
			}
		}
	}
	return nil
}

// updateSlice 按 Name 匹配更新
func (m *AgcModelManager) updateSlice(sliceVal reflect.Value, newValue interface{}) error {
	newSlice, ok := newValue.([]interface{})
	if !ok {
		return fmt.Errorf("更新值不是切片类型，实际类型: %T", newValue)
	}

	for _, item := range newSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			log.Warnf("切片项不是 map 类型，跳过: %T", item)
			continue
		}

		targetName, ok := itemMap["name"].(string)
		if !ok || targetName == "" {
			log.Warnf("切片项缺少 name 字段，跳过: %+v", itemMap)
			continue
		}

		// 遍历原切片，按 name 匹配
		for i := 0; i < sliceVal.Len(); i++ {
			elem := sliceVal.Index(i)
			if !elem.IsValid() || elem.Kind() != reflect.Struct {
				continue
			}

			nameField := elem.FieldByName("Name")
			if !nameField.IsValid() || nameField.String() != targetName {
				continue
			}

			// 找到匹配项，递归更新
			if err := m.updateStruct(elem, itemMap); err != nil {
				log.Warnf("更新切片项 %s 失败: %v", targetName, err)
			}
			break
		}
	}
	return nil
}

// toStructFieldName 保持不变
func toStructFieldName(jsonKey string) string {
	switch jsonKey {
	case "name":
		return "Name"
	case "type":
		return "Type"
	case "dyn_para":
		return "DynPara"
	case "para":
		return "Para"
	case "value":
		return "Value"
	case "desc":
		return "Desc"
	case "max":
		return "Max"
	case "min":
		return "Min"
	case "database_id":
		return "DatabaseID"
	default:
		return strings.ToUpper(jsonKey[:1]) + jsonKey[1:]
	}
}

// buildDeviceNameTree 递归构建纯设备名称树形结构（内部工具，小写）
func buildDeviceNameTree(dev *Device) *DeviceNameNode {
	if dev == nil {
		return nil
	}

	node := &DeviceNameNode{
		Name: dev.Name,
	}

	// 递归处理所有子设备
	for i := range dev.SubDevices {
		if subNode := buildDeviceNameTree(dev.SubDevices[i]); subNode != nil {
			node.SubDevices = append(node.SubDevices, subNode)
		}
	}

	return node
}

// ############################## 设备管理器对外方法实现 ##############################

// 判断静态参数和动态参数是否存在修改，如果修改则更新现有结构
func (m *AgcModelManager) UpdateParaAndDynPara() error {
	// 1. 遍历所有设备类型的动静态参数模板
	for deviceType, templateParas := range m.templateCache {
		// 2. 递归找到内存中【所有该类型】的设备
		devices := m.findDevicesByType(deviceType)
		if len(devices) == 0 {
			continue
		}

		newDynParas, err := getDeviceDynParams(deviceType)
		if err != nil {
			log.Warnf("update [%s] dyn para failed: %v", deviceType, err)
			continue
		}

		// 3. 逐个更新设备的静态参数和动态参数
		for i := range devices {
			devices[i].Para = mergePara(devices[i].Para, templateParas)
			devices[i].DynPara = mergeDynPara(devices[i].DynPara, newDynParas)
		}
	}

	return m.saveModel()
}

// AddAgcDevice 在指定父设备下添加子设备
//
// 参数:
// parentName - 父设备名称
// device     - 待新增子设备（需要Name和Type）
//
// 返回值:
// error - 调用失败时返回错误
func (m *AgcModelManager) AddAgcDevice(parentName string, device *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 检查子设备名称是否重复（全局唯一）
	if m.station.isNameExists(device.Name) {
		return fmt.Errorf("AddAgcDevice : sub device '%s' is already exists", device.Name)
	}

	// 2. 查找父设备是否存在
	parent, ok := m.station.getSubDeviceByName(parentName)
	if !ok {
		return fmt.Errorf("AddAgcDevice : parient device '%s' not found", parentName)
	}

	// 3. 从模板缓存获取对应类型的静态参数
	para, ok := m.templateCache[device.Type]
	if !ok {
		return fmt.Errorf("AddAgcDevice: static para not found for type '%s'", device.Type)
	}
	device.Para = para

	// 4. 获取动态参数
	dynPara, err := getDeviceDynParams(device.Type)
	if err != nil {
		return fmt.Errorf("AddAgcDevice: failed to get dyn parameters for '%s' : '%w'", device.Type, err)
	}
	device.DynPara = dynPara

	// 5. 调用设备CRUD方法添加子设备
	if err := parent.addSubDevice(device); err != nil {
		return fmt.Errorf("AddAgcDevice: add failed: %w", err)
	}

	// 7. 保存到文件
	return m.saveModel()
}

// GetAgcDevice 根据设备名称获取设备信息, 不包含子设备
//
// 参数:
// deviceName - 设备名称
//
// 返回值:
// Device - 调用成功时返回查找到的设备信息, 失败时返回空设备
// error  - 调用失败时返回错误
func (m *AgcModelManager) GetAgcDevice(deviceName string) (Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 默认情况下返回所有厂站信息
	if deviceName == "" {
		return *m.station, nil
	}
	// 查找设备是否存在
	device, ok := m.station.getSubDeviceByName(deviceName)
	if !ok {
		return Device{}, fmt.Errorf("GetAgcDevice : device '%s' not found", deviceName)
	}

	dev := *device
	dev.SubDevices = nil

	return dev, nil
}

// UpdateAgcStation 更新指定设备信息
//
// 参数:
// deviceName - 设备名称
// jsonStr	  - 需要更改的设备信息
//
// 返回值:
// error  - 调用失败时返回错误
func (m *AgcModelManager) UpdateAgcDevice(deviceName string, jsonStr string) (Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 查找设备是否存在
	device, ok := m.station.getSubDeviceByName(deviceName)
	if !ok {
		return Device{}, fmt.Errorf("UpdateAgcStation : device '%s' not found", deviceName)
	}

	// 把json字符串转成更新map
	var updateMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &updateMap); err != nil {
		return Device{}, fmt.Errorf("UpdateAgcDevice: parse update data failed: %w", err)
	}

	// 从前端参数中获取静态参数列表
	newPara := make(map[string]string)
	if updatePara, ok := updateMap["para"]; ok {
		// 转成数组
		if paraSlice, ok := updatePara.([]interface{}); ok {
			// 遍历数组
			for i := range paraSlice {
				// 从数组中获取一组静态参数
				if pMap, ok := paraSlice[i].(map[string]interface{}); ok {
					var name, newVal = pMap["name"].(string), pMap["value"].(string)
					newPara[name] = newVal
				}
			}
		}
	}

	for i := range device.Para {
		// 判断值是否更新
		if newVal, has := newPara[device.Para[i].Name]; has {
			if device.Para[i].Value != newVal {
				device.Para[i].Value = newVal
				// 调用http接口更新配置参数
				paraPath := device.Path[8:] + "/para/" + device.Para[i].Name //忽略path前面的"/station"
				if err := SetStrategySettingParams("station_para", paraPath, newVal, device.Para[i].Type); err != nil {
					log.Errorf("SetStrategyParams SetStrategySettingParams failed: %v", err)
				}
			}
		}
	}

	// // 对结构体进行增量更新
	// if err := m.updateDevice(device, updateMap); err != nil {
	// 	return Device{}, fmt.Errorf("UpdateAgcDevice: update failed: %w", err)
	// }

	dev := *device
	dev.SubDevices = nil

	return dev, nil
}

// RemoveAgcDevice 删除指定设备信息
//
// 参数:
// deviceName - 设备名称
//
// 返回值:
// error  - 调用失败时返回错误
func (m *AgcModelManager) RemoveAgcDevice(deviceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果删除的是根节点 → 直接重置
	if m.station.Name == deviceName {
		// 重置为空站点（正确类型）
		m.station = &Device{
			Type: "STATION",
			Name: "station",
			Path: "/station/station",
		}
		return m.saveModel()
	}

	// 否则删除子设备
	if ok := m.station.removeSubDeviceByName(deviceName); !ok {
		return fmt.Errorf("RemoveAgcDevice : device '%s' not found", deviceName)
	}

	return m.saveModel()
}

// DeviceNameNode 仅包含设备名称的层级结构
type DeviceNameNode struct {
	Name       string            `json:"name"`
	SubDevices []*DeviceNameNode `json:"sub_devices,omitempty"`
}

// GetDeviceNameTree 获取【根厂站】的设备名称层级树（无参、大写开头、直接返回根结构）
func (m *AgcModelManager) GetDeviceNameTree() (*DeviceNameNode, error) {
	// 并发读锁
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 根设备为空则返回错误
	if m.station == nil {
		return nil, fmt.Errorf("根厂站设备不存在")
	}

	// 递归构建纯名称层级
	return buildDeviceNameTree(m.station), nil
}

// agc atu 返回的测点信息
type AgcRtuModelPoints struct {
	Name string `json:"name"`
	Desc string `json:"aliasname"`
	Type string `json:"type"`
	Max  string `json:"upperlimit"`
	Min  string `json:"lowerlimit"`
	Path string `json:"path"`
}

// 递归获取所有可在agc rtu中添加的测点信息
func getAllAgcRtuModelPoints(device *Device, name, paraType, dataType string, exceptPaths map[string]struct{}) (points []*AgcRtuModelPoints) {
	if paraType == "all" || paraType == "para" {
		for i := range device.Para {
			if dataType == "all" || dataType == device.Para[i].Type {
				point := &AgcRtuModelPoints{
					Name: device.Name + "/" + device.Para[i].Name,
					Desc: device.Para[i].Desc,
					Type: device.Para[i].Type,
					Max:  device.Para[i].Max,
					Min:  device.Para[i].Min,
					Path: device.Para[i].Path,
				}
				// 判断当前点位是否已经添加过
				if _, has := exceptPaths[point.Path]; !has {
					points = append(points, point)
				}
			}
		}
	}
	if paraType == "all" || paraType == "dyn_para" {
		for i := range device.DynPara {
			if dataType == "all" || dataType == device.DynPara[i].Type {
				point := &AgcRtuModelPoints{
					Name: device.Name + "/" + device.DynPara[i].Name,
					Desc: device.DynPara[i].Desc,
					Path: device.DynPara[i].Path,
					Type: device.DynPara[i].Type,
				}
				// 判断当前点位是否已经添加过
				if _, has := exceptPaths[point.Path]; !has {
					points = append(points, point)
				}
			}
		}
	}

	// 全量查询时需要遍历子设备
	if name == "all" {
		for i := range device.SubDevices {
			points = append(points, getAllAgcRtuModelPoints(device.SubDevices[i], name, paraType, dataType, exceptPaths)...)
		}
	}
	return points
}

// GetAgcRtuModelPoints 获取模型中关联的AGC RTU测点信息
// 支持按设备名称、参数类型、数据类型过滤查询，返回测点结构化列表
// 参数：
//
//	deviceName - 设备名称（传all表示查询全量设备测点）
//	paraType   - 参数类型（all/para/dyn_para，分别表示全部/静态参数/动态参数）
//	dataType   - 数据类型（all/ANALOG/DIGITAL等，按类型过滤测点）
//
// 返回：
//
//	points - 匹配的AGC RTU测点列表
//	err    - 设备不存在等错误信息
func (m *AgcModelManager) GetAgcRtuModelPoints(name, paraType, dataType string, exceptPaths map[string]struct{}) (points []*AgcRtuModelPoints) {
	// 读操作使用读锁，提升并发性能
	m.mu.RLock()
	defer m.mu.RUnlock()

	// deviceName为all时，获取全量设备的测点
	if name == "all" {
		return getAllAgcRtuModelPoints(m.station, name, paraType, dataType, exceptPaths)
	}

	// 按名称查找目标设备
	device, ok := m.station.getSubDeviceByName(name)
	if !ok {
		return nil
	}

	// 获取指定设备下的测点
	return getAllAgcRtuModelPoints(device, name, paraType, dataType, exceptPaths)
}
