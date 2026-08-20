package homepage

import (
	"context"
	"sync"
	"time"

	"webserver-arm/cgolibparse"
	"webserver-arm/log"
)

// HomepagePointData 推送给前端的单条测点实时数据
type HomepagePointData struct {
	OID      string `json:"oid"`
	Value    string `json:"value"`
	Quality  string `json:"quality"`
	SendTime string `json:"sendtime"`
	Dead     string `json:"dead"`
}

// PushFunc 推送回调，由 handlers 包注入，避免循环依赖
type PushFunc func(topic string, data interface{})

// HomepageManager 首页实时数据全局推送管理器（单例）
type HomepageManager struct {
	subscribeCount int
	registeredOids []string
	getValue       func() []cgolibparse.PointValue
	unRegister     func()
	pushStop       context.CancelFunc
	pushFunc       PushFunc
	mu             sync.Mutex
}

var (
	homepageMgr  *HomepageManager
	homepageOnce sync.Once
)

// GetHomepageManager 获取单例，首次调用时注入 pushFunc
func GetHomepageManager(pushFn PushFunc) *HomepageManager {
	homepageOnce.Do(func() {
		homepageMgr = &HomepageManager{
			pushFunc: pushFn,
		}
	})
	if pushFn != nil {
		homepageMgr.pushFunc = pushFn
	}
	return homepageMgr
}

// Subscribe 订阅首页实时数据推送
func (m *HomepageManager) Subscribe() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cm, err := GetConfigManager()
	if err != nil {
		log.Error("homepage subscribe: GetConfigManager failed:", err.Error())
		return
	}
	cfg := cm.GetConfig()
	newOids, newIdNameMap := collectOids(cfg)

	if m.subscribeCount > 0 && equalOids(m.registeredOids, newOids) {
		m.subscribeCount++
		log.Debug("homepage subscribe: config unchanged, count:", m.subscribeCount)
		return
	}

	if m.subscribeCount > 0 {
		m.stopPush()
	}

	if err := m.startPush(newOids, newIdNameMap); err != nil {
		log.Error("homepage startPush failed:", err.Error())
		return
	}
	m.registeredOids = newOids
	m.subscribeCount++
	log.Info("homepage subscribe: started, oids:", len(newOids), "count:", m.subscribeCount)
}

// Unsubscribe 取消订阅首页实时数据推送
func (m *HomepageManager) Unsubscribe() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribeCount <= 0 {
		return
	}
	m.subscribeCount--
	if m.subscribeCount == 0 {
		m.stopPush()
		log.Info("homepage unsubscribe: stopped")
	}
}

// startPush 注册 Cgo OID 并启动推送协程（需持锁调用）
func (m *HomepageManager) startPush(oids []string, idNameMap map[string]string) error {
	if len(oids) == 0 {
		return nil
	}

	mgr, err := cgolibparse.GetDBMsgManagerInstance()
	if err != nil {
		return err
	}

	getValue, unRegister, err := mgr.RegisterPointsID(idNameMap, oids)
	if err != nil {
		return err
	}

	m.getValue = getValue
	m.unRegister = unRegister

	ctx, cancel := context.WithCancel(context.Background())
	m.pushStop = cancel

	go m.pushLoop(ctx)
	return nil
}

// stopPush 停止推送协程并取消 Cgo 注册（需持锁调用）
func (m *HomepageManager) stopPush() {
	if m.pushStop != nil {
		m.pushStop()
		m.pushStop = nil
	}
	if m.unRegister != nil {
		m.unRegister()
		m.unRegister = nil
	}
	m.getValue = nil
	m.registeredOids = nil
}

// pushLoop 定时读取实时数据并广播
func (m *HomepageManager) pushLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 启动后立即推送一次，不等首个 ticker
	m.pushOnce()

	for {
		select {
		case <-ticker.C:
			m.pushOnce()
		case <-ctx.Done():
			return
		}
	}
}

// pushOnce 读取一次数据并广播
func (m *HomepageManager) pushOnce() {
	if m.getValue == nil || m.pushFunc == nil {
		return
	}
	values := m.getValue()
	if len(values) == 0 {
		return
	}
	data := transformToHomepageData(values)
	m.pushFunc("homepage_data", data)
}

// transformToHomepageData 将 Cgo PointValue 转换为前端所需格式
func transformToHomepageData(values []cgolibparse.PointValue) []HomepagePointData {
	result := make([]HomepagePointData, len(values))
	for i, v := range values {
		result[i] = HomepagePointData{
			OID:      v.PointID,
			Value:    v.CurrentValue,
			Quality:  v.Quality,
			SendTime: v.SendTime,
			Dead:     v.Dead,
		}
	}
	return result
}

// collectOids 从配置中收集所有需要订阅的 OID（去重，type≠3 且 oid≠""）
func collectOids(cfg HomepageConfig) ([]string, map[string]string) {
	seen := make(map[string]struct{})
	var oids []string
	idNameMap := make(map[string]string)

	addItems := func(items []ConfigItem) {
		for _, item := range items {
			if item.Type == 3 || item.OID == "" {
				continue
			}
			if _, ok := seen[item.OID]; ok {
				continue
			}
			seen[item.OID] = struct{}{}
			oids = append(oids, item.OID)
			name := item.ShowName
			if name == "" {
				name = item.Name
			}
			idNameMap[item.OID] = name
		}
	}

	addItems(cfg.Topology)
	addItems(cfg.Realtime)
	addItems(cfg.BasicInfo)
	addItems(cfg.PowerLevel)
	addItems(cfg.RealPower)
	return oids, idNameMap
}

// equalOids 比较两个 OID 列表是否相同（忽略顺序）
func equalOids(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[id]; !ok {
			return false
		}
	}
	return true
}
