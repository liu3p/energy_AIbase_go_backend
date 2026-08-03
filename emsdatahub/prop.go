package emsdatahub

// import (
// 	"errors"
// 	"fmt"
// 	"strconv"
// 	"sync"
// 	"time"
// 	"webserver-arm/emsclient"
// 	"webserver-arm/emsdb"
// 	"webserver-arm/log"
// 	"webserver-arm/models"
// )

// var PropMng *PropertyManager

// func (sh *StreamHub) WatchPropertyValues() error {
// 	// load data

// 	propdata, err := emsdb.GetAllProperties()
// 	if err != nil {
// 		return fmt.Errorf("GetAllRDBStreamParams error: %v", err)
// 	}

// 	// regist PropertyManager
// 	PropMng = NewPropertyManager()

// 	for _, prop := range propdata {
// 		propValue := NewPropertyValue(prop.TransCode, prop.MqttID, prop.DeviceSn, float64(prop.Coef), float64(prop.Threshold))
// 		var updater PropUpdater
// 		switch prop.TransType {
// 		case 1: // from yc
// 		// no use
// 		case 2: // from yx
// 		// no use
// 		case 3: // from kwh
// 		// no use
// 		case 4: // from rdb
// 		// no use
// 		case 5: // from sql
// 			updater = NewDBSqlUpdater(propValue)
// 		case 6: // from function
// 		case 7: // fixed data
// 			updater = newDBTableUpdater(propValue)
// 		}
// 		PropMng.Register(propValue, updater)
// 	}

// 	// run once
// 	PropMng.Start()

// 	go func() {
// 		for prop := range PropMng.Changes() {
// 			log.Infof("Property %s changed:%v", prop.name, prop.GetValue())
// 			content := ContentBody{
// 				prop.deviceSN: []models.KeyValueInfoEx{
// 					{K: prop.mqttID, V: prop.GetValue(), At: time.Now().UnixMilli(), Mt: time.Now().UnixMilli()},
// 				},
// 			}
// 			pushData := &emsclient.PushResp{ // 推送数据
// 				Topic:   "prop",
// 				Content: content,
// 			}
// 			sh.clientMgr.PushMsg(pushData)
// 		}
// 	}()

// 	return nil
// }

// //////////////////////////属性值封装///////////////////////////////////////

// type PropertyValue struct {
// 	name      string      // 属性名称（如"电池电压"、"充放电电流"）
// 	coef      float64     // 斜率（原始值 -> 实际值的转换系数）
// 	threshold float64     // 公差（变化超过此值才视为变化，字符串用是否相等判断）
// 	rawValue  interface{} // 原始值（未乘斜率）
// 	mqttID    string
// 	deviceSN  string
// 	mu        sync.RWMutex // 并发安全锁
// }

// func applyCoef(raw interface{}, coef float64) interface{} {
// 	switch v := raw.(type) {
// 	case int:
// 		return float64(v) * coef
// 	case float32:
// 		return float64(v) * coef
// 	case float64:
// 		return v * coef
// 	case string:
// 		return v // 字符串不应用斜率
// 	}
// 	return raw
// }

// func (p *PropertyValue) GetValue() interface{} {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()
// 	return applyCoef(p.rawValue, p.coef)
// }
// func (p *PropertyValue) SetValue(value interface{}) {
// 	p.mu.Lock()
// 	defer p.mu.Unlock()
// 	p.rawValue = value
// }

// func (p *PropertyValue) Same(aRawValue interface{}) (bool, error) {
// 	newActual := applyCoef(aRawValue, p.coef)

// 	p.mu.RLock()
// 	defer p.mu.RUnlock()
// 	oldActual := applyCoef(p.rawValue, p.coef)
// 	if oldActual == nil {
// 		// prop value first time is nil
// 		return false, nil
// 	}

// 	// 按类型判断是否在公差范围内
// 	switch old := oldActual.(type) {
// 	case float64:
// 		new, ok := newActual.(float64)
// 		if !ok {
// 			return false, errors.New("type mismatch")
// 		}
// 		diff := old - new
// 		if diff < 0 {
// 			diff = -diff
// 		}
// 		return diff <= p.threshold, nil
// 	case string:
// 		new, ok := newActual.(string)
// 		if !ok {
// 			return false, errors.New("type mismatch")
// 		}
// 		return old == new, nil // 字符串直接比较
// 	default:
// 		return false, fmt.Errorf("unsupported type for comparison:%T", oldActual)
// 	}
// }

// // NewProperty 创建新属性
// func NewPropertyValue(name, mqttID, devSn string, coef, threshold float64) *PropertyValue {
// 	return &PropertyValue{
// 		name:      name,
// 		mqttID:    mqttID,
// 		deviceSN:  devSn,
// 		coef:      coef,
// 		threshold: threshold,
// 	}
// }

// //////////////////////////////属性管理//////////////////////////////////////////

// // PropertyManager 管理属性集合与更新逻辑
// type PropertyManager struct {
// 	properties map[string]*PropertyValue // 所有属性
// 	updaters   map[string]PropUpdater    // 属性对应的更新器
// 	changes    chan *PropertyValue       // 变化通知通道
// 	stopChan   chan struct{}             // 停止信号
// 	wg         sync.WaitGroup            // 协程等待组
// 	mu         sync.RWMutex              // 保护properties和updaters
// }

// func NewPropertyManager() *PropertyManager {
// 	return &PropertyManager{
// 		properties: make(map[string]*PropertyValue),
// 		updaters:   make(map[string]PropUpdater),
// 		changes:    make(chan *PropertyValue, 100), // 缓冲通道避免阻塞
// 		stopChan:   make(chan struct{}),
// 	}
// }

// // Register 注册属性及其更新器
// func (m *PropertyManager) Register(p *PropertyValue, u PropUpdater) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
// 	m.properties[p.name] = p
// 	m.updaters[p.name] = u
// }

// // Start 启动监控协程（周期性检查更新）
// func (m *PropertyManager) Start() {
// 	m.wg.Add(1)
// 	go func() {
// 		m.checkAndUpdate()
// 		defer m.wg.Done()
// 		ticker := time.NewTicker(10 * time.Second)
// 		defer ticker.Stop()

// 		for {
// 			select {
// 			case <-ticker.C:
// 				m.checkAndUpdate()
// 			case <-m.stopChan:
// 				return
// 			}
// 		}
// 	}()
// }

// // 检查并更新所有属性，判断是否变化
// func (m *PropertyManager) checkAndUpdate() {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	for name, p := range m.properties {
// 		updater := m.updaters[name]
// 		if updater == nil {
// 			log.Errorf("checkAndUpdate: no updater for property %s\n", name)
// 			continue
// 		}

// 		// update to get new value
// 		newValue, err := updater.Update()
// 		if err != nil {
// 			log.Errorf("checkAndUpdate: update property %s failed: %v\n", name, err)
// 			continue
// 		}

// 		// compare value
// 		same, err := p.Same(newValue.GetValue())
// 		if err != nil {
// 			log.Errorf("checkAndUpdate: compare property %s failed: %v\n", name, err)
// 			continue
// 		}

// 		if !same { // if changed, send notification
// 			log.Infof("checkAndUpdate: value of property %s changed, old: %v, new: %v", name, p.GetValue(), newValue.GetValue())
// 			p.SetValue(newValue.GetValue())
// 			select {
// 			case m.changes <- p: // 发送变化通知
// 			default:
// 				log.Warn("checkAndUpdate: changes channel is full, drop notification")
// 			}
// 		}
// 	}
// }

// // Changes 返回变化通知通道（供外部监听）
// func (m *PropertyManager) Changes() <-chan *PropertyValue {
// 	return m.changes
// }

// // Stop 停止监控协程
// func (m *PropertyManager) Stop() {
// 	close(m.stopChan)
// 	m.wg.Wait()
// 	close(m.changes)
// }

// func (m *PropertyManager) GetAllPropertiesContent(deviceSn []string) ContentBody {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	filterDevsnMap := make(map[string]bool)
// 	for _, devsn := range deviceSn {
// 		filterDevsnMap[devsn] = true
// 	}

// 	var props []*PropertyValue
// 	for _, p := range m.properties {
// 		props = append(props, p)
// 	}
// 	result := make(ContentBody)
// 	for _, p := range props {
// 		if _, ok := filterDevsnMap[p.deviceSN]; !ok {
// 			continue
// 		}
// 		if p.GetValue() == nil {
// 			log.Infof("GetAllPropertiesContent: property %s value is nil", p.name)
// 			continue
// 		}
// 		log.Infof("GetAllPropertiesContent: property %s value: %v", p.name, p.GetValue())
// 		if _, ok := result[p.deviceSN]; !ok {
// 			result[p.deviceSN] = []models.KeyValueInfoEx{
// 				{K: p.mqttID, V: p.GetValue(), At: time.Now().UnixMilli(), Mt: time.Now().UnixMilli()},
// 			}
// 		} else {
// 			result[p.deviceSN] = append(result[p.deviceSN], models.KeyValueInfoEx{K: p.mqttID, V: p.GetValue(), At: time.Now().UnixMilli(), Mt: time.Now().UnixMilli()})
// 		}

// 	}

// 	return result
// }

// func GetDevicesProp(deviceSn []string) (*models.PropData, error) {
// 	propContent := PropMng.GetAllPropertiesContent(deviceSn)
// 	if len(propContent) == 0 {
// 		return &models.PropData{Topic: "prop", Content: nil}, nil
// 	}

// 	return &models.PropData{Topic: "prop", Content: propContent}, nil
// }

// // ////////////////////////各种 Updater 实现//////////////////////////////////////////////
// type DBDataType string

// const (
// 	DBTypeString  DBDataType = "1"
// 	DBTypeInt     DBDataType = "2"
// 	DBTypeFloat32 DBDataType = "3"
// 	DBTypeFloat64 DBDataType = "4"
// )

// type PropUpdater interface {
// 	// Update 更新指定属性的原始值
// 	Update() (*PropertyValue, error)
// }

// type DBSqlUpdater struct {
// 	value    PropertyValue
// 	querySql string
// 	sqldb    *db.MsDB
// 	dataType DBDataType
// }

// func NewDBSqlUpdater(value *PropertyValue) *DBSqlUpdater {
// 	dbsqlUpdater := &DBSqlUpdater{value: *value}
// 	sqldb, err := db.NewMsDB()
// 	if err != nil {
// 		log.Info("NewDBSqlUpdater GetAllProperties open db err:", err)
// 		return nil
// 	}
// 	dbsqlUpdater.sqldb = sqldb

// 	propdata, err := emsdb.GetAllProperties()
// 	if err != nil {
// 		log.Info("NewDBSqlUpdater GetAllProperties error:", err)
// 		return nil
// 	}
// 	for _, prop := range propdata {
// 		if prop.TransCode == dbsqlUpdater.value.name {
// 			dbsqlUpdater.querySql = prop.PropCondition
// 			dbsqlUpdater.dataType = DBDataType(prop.PropField)
// 		}
// 	}

// 	return dbsqlUpdater
// }

// func (d *DBSqlUpdater) Update() (*PropertyValue, error) {

// 	var queryResult []interface{}
// 	txerr := d.sqldb.Raw(d.querySql).Scan(&queryResult)
// 	if txerr.Error != nil {
// 		return &d.value, fmt.Errorf("DBSqlUpdater error. exec sql: %s failed: %v", d.querySql, txerr.Error)
// 	}
// 	if d.dataType == DBTypeString { // string
// 		if len(queryResult) == 0 {
// 			return &d.value, fmt.Errorf("DBSqlUpdater error. exec sql: %s get value '%v' empty.", d.querySql, queryResult)
// 		}
// 		if strValue, ok := queryResult[0].(string); ok {
// 			d.value.SetValue(strValue)
// 		} else {
// 			return &d.value, fmt.Errorf("DBSqlUpdater error. exec sql: %s get value '%v' not string. type:%T", d.querySql, queryResult, queryResult[0])
// 		}

// 	} else if d.dataType == DBTypeInt { // int
// 		if len(queryResult) == 0 {
// 			return &d.value, fmt.Errorf("DBSqlUpdater error. exec sql: %s get value '%v' empty.", d.querySql, queryResult)
// 		}
// 		if intValue, ok := queryResult[0].(int); ok {
// 			d.value.SetValue(intValue)
// 		} else {
// 			return &d.value, fmt.Errorf("DBSqlUpdater error. exec sql: %s get value '%v' not int. type:%T", d.querySql, queryResult, queryResult[0])
// 		}

// 	} else if d.dataType == DBTypeFloat32 || d.dataType == DBTypeFloat64 { // double&float
// 		if len(queryResult) == 0 {
// 			return &d.value, fmt.Errorf("DBSqlUpdater error. exec sql: %s get value '%v' empty.", d.querySql, queryResult)
// 		}
// 		switch floatValue := queryResult[0].(type) {
// 		case float32:
// 			d.value.SetValue(floatValue)
// 		case float64:
// 			d.value.SetValue(floatValue)
// 		case []uint8:
// 			// 字节切片转字符串（如 []uint8{48,46,48,53} → "0.05"）
// 			var floatVal float64
// 			strVal := string(floatValue)
// 			if _, err := fmt.Sscanf(strVal, "%f", &floatVal); err != nil {
// 				return &d.value, fmt.Errorf("DBSqlUpdater error. parse float from string failed: %v", err)
// 			}
// 			d.value.SetValue(floatVal)
// 		default:
// 			return &d.value, fmt.Errorf("DBSqlUpdater error. gexec sql: %s et value '%v' not float. type:%T", d.querySql, queryResult, queryResult[0])
// 		}
// 	}

// 	return &d.value, nil
// }

// //==================

// type DBTableUpdater struct { // 从属性表表读取固定字段
// 	value PropertyValue
// }

// func newDBTableUpdater(value *PropertyValue) *DBTableUpdater {
// 	return &DBTableUpdater{value: *value}
// }

// func (d *DBTableUpdater) Update() (*PropertyValue, error) {
// 	propdata, err := emsdb.GetAllProperties()
// 	if err != nil {
// 		return &d.value, fmt.Errorf("GetAllRDBStreamParams error: %v", err)
// 	}

// 	for _, prop := range propdata {
// 		if prop.TransCode == d.value.name {
// 			if prop.PropField == string(DBTypeString) { // string
// 				d.value.SetValue(prop.PropCondition)
// 			} else if prop.PropField == string(DBTypeInt) { // int
// 				intValue, err := strconv.Atoi(prop.PropCondition)
// 				if err != nil {
// 					return &d.value, fmt.Errorf("parseInt error: %v", err)
// 				}
// 				d.value.SetValue(intValue)
// 			} else if prop.PropField == string(DBTypeFloat32) || prop.PropField == string(DBTypeFloat64) { // double&float
// 				floatValue, err := strconv.ParseFloat(prop.PropCondition, 64)
// 				if err != nil {
// 					return &d.value, fmt.Errorf("parseFloat error: %v", err)
// 				}
// 				d.value.SetValue(floatValue)
// 			} else {
// 				return &d.value, fmt.Errorf("unknown property field value type: %s", prop.PropField)
// 			}
// 		}
// 	}
// 	return &d.value, nil
// }
