package emsxmldb

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"webserver-arm/log"

	"github.com/xuri/excelize/v2"
)

const (
	RtuTypeAcqu    = 1
	RtuTypeTrans   = 2
	RtuTypeAgc     = 3
	RtuTypeMonit   = 4
	RtuTypeFormula = 5
)

// -------------------------- 1. 定义XML对应的结构体（与XML结构严格匹配） --------------------------

// DB 根节点结构体
type DB struct {
	XMLName   xml.Name  `xml:"db"`
	Hierarchy Hierarchy `xml:"Hierarchy"`
	Single    Single    `xml:"Single"`
}

// Hierarchy 层次结构节点
type Hierarchy struct {
	XMLName xml.Name `xml:"Hierarchy"`
	RTUs    []*RTU   `xml:"RTU"` // 多个RTU节点
}

// 基础类型，需实现GetID方法（包括RTU、Device、五遥、属性）
type IDer interface {
	GetID() string
	SetID(string)
	GetName() string
}

// RTU、Device 以及测点类型都包含的字段
type BaseInfo struct {
	ID   string `xml:"ID,attr" json:"id" excel:"-"`
	Name string `xml:"Name,attr" json:"name" excel:"原始名"`
}

func (b *BaseInfo) GetID() string {
	return b.ID
}

func (b *BaseInfo) SetID(id string) {
	b.ID = id
}

func (b *BaseInfo) GetName() string {
	return b.Name
}

// RTU RTU结构体（包含属性）
type RTU struct {
	BaseInfo
	XMLName        xml.Name  `xml:"RTU" json:"-"`
	Type           int       `xml:"Type,attr" json:"type"`
	RtuAddr        string    `xml:"RtuAddr,attr" json:"rtuaddr"`
	MemOfCabinet   string    `xml:"MemOfCabinet,attr" json:"memofcabinet"`
	InService      string    `xml:"InService,attr" json:"inservice"`
	IsConnected    string    `xml:"IsConnected,attr" json:"isconnected"`
	ChannelGroupID int       `xml:"ChannelGroupID,attr" json:"channelgroupid"` // 绑定的通道组 ID，-1 表示无通道
	ChannelID      int       `xml:"ChannelID,attr" json:"channelid"`           // 绑定的通道 ID，-1 表示无通道
	Devices        []*Device `xml:"Device" json:"device"`                      // RTU下的多个Device

	// 内存字段（不写入 XML，启动时填充）
	Channel          *Channel `xml:"-" json:"channel,omitempty"`
	ChannelGroupName string   `xml:"-" json:"channelgroupname,omitempty"` // 通道组名称
	ForTransfer      string   `xml:"-" json:"for_transfer,omitempty"`     // 通道组 FOR_TRANSFER 参数
}

// Device 装置结构体
type Device struct {
	BaseInfo
	XMLName      xml.Name `xml:"Device" json:"-"`
	SourceDevice string   `xml:"SourceDevice,atr" json:"source_device"` // 转发设备所关联的原设备
	DeviceAddr   string   `xml:"DeviceAddr,attr" json:"devaddr"`
	MemOfRtu     string   `xml:"MemOfRtu,attr" json:"memofrtu"`
	InService    string   `xml:"InService,attr" json:"inservice"`
	MqttKey      string   `xml:"MqttKey,attr" json:"mqttkey"`
	TransDevice  string   `xml:"TransDevice,attr"`                 // 标识该设备转发了哪个设备的测点
	YX           *YX      `xml:"YX,omitempty" json:"yx,omitempty"` // 遥信
	YC           *YC      `xml:"YC,omitempty" json:"yc,omitempty"` // 遥测
	YM           *YM      `xml:"YM,omitempty" json:"ym,omitempty"` // 遥脉
	YT           *YT      `xml:"YT,omitempty" json:"yt,omitempty"` // 遥调
	YK           *YK      `xml:"YK,omitempty" json:"yk,omitempty"` // 遥控
	SX           *SX      `xml:"SX,omitempty" json:"sx,omitempty"` // 属性
}

// YX 遥信结构体
type YX struct {
	XMLName  xml.Name   `xml:"YX" json:"-"`
	Digitals []*Digital `xml:"Digital" json:"digital"`
}

// YC 遥测结构体
type YC struct {
	XMLName xml.Name  `xml:"YC" json:"-"`
	Analogs []*Analog `xml:"Analog" json:"analog"`
}

// YM 遥脉结构体
type YM struct {
	XMLName xml.Name `xml:"YM" json:"-"`
	Pulses  []*Pulse `xml:"Pulse" json:"pulse"`
}

// YT 遥调结构体
type YT struct {
	XMLName   xml.Name    `xml:"YT" json:"-"`
	Regulates []*Regulate `xml:"Regulate" json:"regulate"`
}

// YK 遥控结构体
type YK struct {
	XMLName  xml.Name   `xml:"YK" json:"-"`
	Controls []*Control `xml:"Control" json:"control"`
}

// SX 属性结构体
type SX struct {
	XMLName    xml.Name     `xml:"SX" json:"-"`
	Attributes []*Attribute `xml:"Attribute" json:"attribute"`
}

// 测点类型，需要实现GetGin方法
type Pointer interface {
	IDer
	GetGin() string
	SetGin(string)
	GetTrans() string
	GetDataSource() string
	SetDataSource(string)
	Clone() Pointer
	GetPath() string
}

// 定义测点类型的基础结构
type BasePoint struct {
	BaseInfo
	Gin        string `xml:"Gin,attr" json:"gin" excel:"点号"`
	Trans      string `xml:"Trans,attr" json:"trans" excel:"是否转发"`
	MqttKey    string `xml:"MqttKey,attr" json:"mqttkey" excel:"MqttKey"`
	DataSource string `xml:"DataSource,attr" json:"datasource" excel:"-"`
	Path       string `xml:"Path,attr" json:"path" excel:"-"`
}

func (b *BasePoint) GetGin() string {
	return b.Gin
}

func (b *BasePoint) SetGin(gin string) {
	b.Gin = gin
}

func (b *BasePoint) GetTrans() string {
	return b.Trans
}

func (b *BasePoint) GetDataSource() string {
	return b.DataSource
}

func (b *BasePoint) SetDataSource(dataSource string) {
	b.DataSource = dataSource
}

func (b *BasePoint) GetPath() string {
	return b.Path
}

// 包含公式的测点类型，需要实现GetFormula方法
type FormulaPointer interface {
	Pointer
	GetFormula() string
	SetFormula(string)
}

// 定义包含公式的测点结构
type FormulaPoint struct {
	BasePoint
	Formula string `xml:"Formula,attr" json:"formula" excel:"-"`
}

func (f *FormulaPoint) GetFormula() string {
	return f.Formula
}

func (f *FormulaPoint) SetFormula(formula string) {
	f.Formula = formula
}

// Analog 遥测测点结构体
type Analog struct {
	FormulaPoint
	XMLName       string `xml:"Analog" json:"-" excel:"-"`
	Register      string `xml:"Register,attr" json:"register" excel:"原始地址"`
	SubType       string `xml:"SubType,attr" json:"subtype" excel:"遥测子类型"`
	DataType      string `xml:"DataType,attr" json:"datatype" excel:"数据类型"`
	Code          string `xml:"Code,attr" json:"code" excel:"功能码"`
	Offset        string `xml:"Offset,attr" json:"offset" excel:"偏移量"`
	AliasName     string `xml:"AliasName,attr" json:"aliasname" excel:"-"`
	Coeff         string `xml:"Coeff,attr" json:"coeff" excel:"系数"`
	Unit          string `xml:"Unit,attr" json:"unit" excel:"单位"`
	UpperLimit    string `xml:"UpperLimit,attr" json:"upperlimit" excel:"上限"`
	LowerLimit    string `xml:"LowerLimit,attr" json:"lowerlimit" excel:"下限"`
	Calculated    string `xml:"Calculated,attr" json:"calculated" excel:"计算点"`
	Manual        string `xml:"Manual,attr" json:"manual" excel:"人工置数点"`
	Dead          string `xml:"Dead,attr" json:"dead" excel:"-"`
	EnableWarning string `xml:"EnableWarning,attr" json:"enablewarning" excel:"报警允许"`
	RawValue      string `xml:"RawValue,attr" json:"rawvalue" excel:"-"`
	CurrValue     string `xml:"CurrValue,attr" json:"currvalue" excel:"-"`
	HisPeriod     string `xml:"HisPeriod,attr" json:"hisperiod" excel:"历史存盘周期"`
	Quality       string `xml:"Quality,attr" json:"quality" excel:"-"`
	SendTime      string `xml:"SendTime,attr" json:"sendtime" excel:"-"`
}

func (a *Analog) Clone() Pointer {
	if a == nil {
		return nil
	}
	cp := *a
	return &cp
}

// Digital 遥信测点结构体
type Digital struct {
	FormulaPoint
	XMLName       string `xml:"Digital" json:"-" excel:"-"`
	Register      string `xml:"Register,attr" json:"register" excel:"原始地址"`
	SubType       string `xml:"SubType,attr" json:"subtype" excel:"遥信子类型"`
	DataType      string `xml:"DataType,attr" json:"datatype" excel:"数据类型"`
	Code          string `xml:"Code,attr" json:"code" excel:"功能码"`
	BitOffset     string `xml:"BitOffset,attr" json:"bitoffset" excel:"位偏移"`
	AliasName     string `xml:"AliasName,attr" json:"aliasname" excel:"-"`
	Negation      string `xml:"Negation,attr" json:"negation" excel:"取反"`
	Calculated    string `xml:"Calculated,attr" json:"calculated" excel:"计算点"`
	Manual        string `xml:"Manual,attr" json:"manual" excel:"人工置数点"`
	Dead          string `xml:"Dead,attr" json:"dead" excel:"-"`
	EnableWarning string `xml:"EnableWarning,attr" json:"enablewarning" excel:"报警允许"`
	RawValue      string `xml:"RawValue,attr" json:"rawvalue" excel:"-"`
	CurrValue     string `xml:"CurrValue,attr" json:"currvalue" excel:"-"`
	HisPeriod     string `xml:"HisPeriod,attr" json:"hisperiod" excel:"历史存盘周期"`
	Quality       string `xml:"Quality,attr" json:"quality" excel:"-"`
	SendTime      string `xml:"SendTime,attr" json:"sendtime" excel:"-"`
}

func (d *Digital) Clone() Pointer {
	if d == nil {
		return nil
	}
	cp := *d
	return &cp
}

// Pulse 遥脉测点结构体
type Pulse struct {
	FormulaPoint
	XMLName        string `xml:"Pulse" json:"-" excel:"-"`
	Register       string `xml:"Register,attr" json:"register" excel:"原始地址"`
	SubType        string `xml:"SubType,attr" json:"subtype" excel:"遥脉子类型"`
	DataType       string `xml:"DataType,attr" json:"datatype" excel:"数据类型"`
	Code           string `xml:"Code,attr" json:"code" excel:"功能码"`
	AliasName      string `xml:"AliasName,attr" json:"aliasname" excel:"-"`
	FullScaleValue string `xml:"FullScaleValue,attr" json:"fullscalevalue" excel:"满刻度值"`
	Coeff          string `xml:"Coeff,attr" json:"coeff" excel:"系数"`
	Unit           string `xml:"Unit,attr" json:"unit" excel:"单位"`
	Calculated     string `xml:"Calculated,attr" json:"calculated" excel:"计算点"`
	Manual         string `xml:"Manual,attr" json:"manual" excel:"人工置数点"`
	Dead           string `xml:"Dead,attr" json:"dead" excel:"-"`
	RawValue       string `xml:"RawValue,attr" json:"rawvalue" excel:"-"`
	CurrValue      string `xml:"CurrValue,attr" json:"currvalue" excel:"-"`
	TotalValue     string `xml:"TotalValue,attr" json:"totalvalue" excel:"-"`
	HisPeriod      string `xml:"HisPeriod,attr" json:"hisperiod" excel:"历史存盘周期"`
	Quality        string `xml:"Quality,attr" json:"quality" excel:"-"`
	SendTime       string `xml:"SendTime,attr" json:"sendtime" excel:"-"`
}

func (p *Pulse) Clone() Pointer {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

// Regulate 遥调测点结构体
type Regulate struct {
	BasePoint
	XMLName       string `xml:"Regulate" json:"-" excel:"-"`
	Register      string `xml:"Register,attr" json:"register" excel:"原始地址"`
	DataType      string `xml:"DataType,attr" json:"datatype" excel:"数据类型"`
	Code          string `xml:"Code,attr" json:"code" excel:"功能码"`
	AliasName     string `xml:"AliasName,attr" json:"aliasname" excel:"-"`
	CtlMode       string `xml:"CtlMode,attr" json:"ctlmode" excel:"控制模式"`
	RelateAnalog  string `xml:"RelateAnalog,attr" json:"relateanalog" excel:"相关遥测"`
	Coeff         string `xml:"Coeff,attr" json:"coeff" excel:"下发系数"`
	UpLimit       string `xml:"UpLimit,attr" json:"uplimit" excel:"遥调上限"`
	LowLimit      string `xml:"LowLimit,attr" json:"lowlimit" excel:"遥调下限"`
	AdjustTimeout string `xml:"AdjustTimeout,attr" json:"adjusttimeout" excel:"遥调超时时间"`
	RawValue      string `xml:"RawValue,attr" json:"rawvalue" excel:"-"`
	CtlValue      string `xml:"CtlValue,attr" json:"ctlvalue" excel:"-"`
}

func (r *Regulate) Clone() Pointer {
	if r == nil {
		return nil
	}
	cp := *r
	return &cp
}

// Control 遥控测点结构体
type Control struct {
	BasePoint
	XMLName        string `xml:"Control" json:"-" excel:"-"`
	Register       string `xml:"Register,attr" json:"register" excel:"原始地址"`
	DataType       string `xml:"DataType,attr" json:"datatype" excel:"数据类型"`
	Code           string `xml:"Code,attr" json:"code" excel:"功能码"`
	AliasName      string `xml:"AliasName,attr" json:"aliasname" excel:"-"`
	CtlMode        string `xml:"CtlMode,attr" json:"ctlmode" excel:"控制模式"`
	RelateDigital  string `xml:"RelateDigital,attr" json:"relatedigital" excel:"相关遥信"`
	ControlTimeout string `xml:"ControlTimeout,attr" json:"controltimeout" excel:"遥信超时时间"`
	RawValue       string `xml:"RawValue,attr" json:"rawvalue" excel:"-"`
	CtlValue       string `xml:"CtlValue,attr" json:"ctlvalue" excel:"-"`
}

func (c *Control) Clone() Pointer {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

// Attribute 属性结构体
type Attribute struct {
	BasePoint
	XMLName   string `xml:"Attribute" json:"-" excel:"-"`
	ValueType string `xml:"ValueType,attr" json:"valuetype" excel:"值类型"`
	CurrValue string `xml:"CurrValue,attr" json:"currvalue" excel:"-"`
	HisPeriod string `xml:"HisPeriod,attr" json:"hisperiod" excel:"历史存盘周期"`
}

func (a *Attribute) Clone() Pointer {
	if a == nil {
		return nil
	}
	cp := *a
	return &cp
}

// Single 单表结构节点
type Single struct {
	XMLName xml.Name `xml:"Single"`
	Tables  []Table  `xml:"table"` // 多个table节点
}

// Table 单表中的表结构体
type Table struct {
	XMLName xml.Name `xml:"table"`
	Name    string   `xml:"name,attr"`
	TableNo string   `xml:"TableNo,attr"`
	MemOf   string   `xml:"MemOf,attr"`

	// 表中的子项（根据表名不同，包含不同的子节点）
	Analogs        []*Analog        `xml:"Analog"`
	Digitals       []*Digital       `xml:"Digital"`
	Pulses         []*Pulse         `xml:"Pulse"`
	Regulates      []*Regulate      `xml:"Regulate"`
	Controls       []*Control       `xml:"Control"`
	Formulas       []*Formula       `xml:"Formula"`
	FormulaFactors []*FormulaFactor `xml:"FormulaFactor"`
	RTUs           []*RTU           `xml:"RTU"`
	Devices        []*Device        `xml:"Device"`
	Attributes     []*Attribute     `xml:"Attribute"`
}

func (t *Table) addItemsToSingleTable(items interface{}) {
	switch t.Name {
	case "rtu":
		rtus := items.([]*RTU)
		t.RTUs = append(t.RTUs, rtus...)
	case "device":
		devices := items.([]*Device)
		t.Devices = append(t.Devices, devices...)
	case "analog", "trans_analog":
		analogs := items.([]*Analog)
		t.Analogs = append(t.Analogs, analogs...)
	case "digital", "trans_digital":
		digitals := items.([]*Digital)
		t.Digitals = append(t.Digitals, digitals...)
	case "pulse", "trans_pulse":
		pulses := items.([]*Pulse)
		t.Pulses = append(t.Pulses, pulses...)
	case "control", "trans_control":
		controls := items.([]*Control)
		t.Controls = append(t.Controls, controls...)
	case "regulate", "trans_regulate":
		regulates := items.([]*Regulate)
		t.Regulates = append(t.Regulates, regulates...)
	case "formula":
		formulas := items.([]*Formula)
		t.Formulas = append(t.Formulas, formulas...)
	case "formula_factor":
		formulaFactors := items.([]*FormulaFactor)
		t.FormulaFactors = append(t.FormulaFactors, formulaFactors...)
	case "attribute", "trans_attribute":
		attribute := items.([]*Attribute)
		t.Attributes = append(t.Attributes, attribute...)
	}
}

func (t *Table) updateItemsInSingleTable(items interface{}) {
	switch t.Name {
	case "analog", "trans_analog":
		analogs := items.([]*Analog)
		updateSliceItemByID(t.Analogs, analogs)
	case "digital", "trans_digital":
		digitals := items.([]*Digital)
		updateSliceItemByID(t.Digitals, digitals)
	case "pulse", "trans_pulse":
		pulses := items.([]*Pulse)
		updateSliceItemByID(t.Pulses, pulses)
	case "control", "trans_control":
		controls := items.([]*Control)
		updateSliceItemByID(t.Controls, controls)
	case "regulate", "trans_regulate":
		regulates := items.([]*Regulate)
		updateSliceItemByID(t.Regulates, regulates)
	case "attribute", "trans_attribute":
		attribute := items.([]*Attribute)
		updateSliceItemByID(t.Attributes, attribute)
	}
}

func (t *Table) deleteItemsFromSingleTable(delIDs map[string]struct{}) {
	switch t.Name {
	case "rtu":
		deleteSliceItemByIDs(&t.RTUs, delIDs)
	case "device":
		deleteSliceItemByIDs(&t.Devices, delIDs)
	case "analog", "trans_analog":
		deleteSliceItemByIDs(&t.Analogs, delIDs)
	case "digital", "trans_digital":
		deleteSliceItemByIDs(&t.Digitals, delIDs)
	case "pulse", "trans_pulse":
		deleteSliceItemByIDs(&t.Pulses, delIDs)
	case "control", "trans_control":
		deleteSliceItemByIDs(&t.Controls, delIDs)
	case "regulate", "trans_regulate":
		deleteSliceItemByIDs(&t.Regulates, delIDs)
	case "formula":
		deleteSliceItemByIDs(&t.Formulas, delIDs)
	case "formula_factor":
		deleteSliceItemByIDs(&t.FormulaFactors, delIDs)
	case "attribute", "trans_attribute":
		deleteSliceItemByIDs(&t.Attributes, delIDs)
	}
}

// Formula 公式结构体
type Formula struct {
	BaseInfo
	XMLName      string `xml:"Formula" json:"-"`
	Info         string `xml:"Info,attr" json:"info"`                 // 公式内容
	CalcType     string `xml:"CalcType,attr" json:"calctype"`         // 计算方式
	CalcCycle    string `xml:"CalcCycle,attr" json:"calccycle"`       // 计算周期
	TriggerPoint string `xml:"TriggerPoint,attr" json:"triggerpoint"` // 0 变分 1 变合 2 变位
	TriggerType  string `xml:"TriggerType,attr" json:"triggertype"`   // 触发方式
	LastCalcTime string `xml:"LastCalcTime,attr" json:"lastcalctime"`
	ValueType    string `xml:"ValueType,attr" json:"valuetype"`
	Value        string `xml:"Value,attr" json:"value"`
}

// FormulaFactor 公式因子结构体
type FormulaFactor struct {
	BaseInfo
	XMLName      string `xml:"FormulaFactor" json:"-"`
	ValueType    string `xml:"ValueType,attr" json:"valuetype"`
	Value        string `xml:"Value,attr" json:"value"`
	DataSource   string `xml:"DataSource,attr" json:"datasource"` // 绑定的点
	MemofFormula string `xml:"MemofFormula,attr" json:"memofformula"`
}

// -------------------------- 2. 核心管理类 --------------------------

var (
	instance *DbConfigXMLManager // 单例实例
	once     sync.Once           // 保证初始化只执行一次
	initErr  error               // 存储初始化错误
)

func getConfigBasePath() string {
	configEnv := os.Getenv("ARM_SF")
	if configEnv == "" {
		log.Error("env ARM_SF not set")
		return ""
	}
	return configEnv
}

var defaultDBXMLFilePath = filepath.Join(getConfigBasePath(), "/db/db_config.xml")
var defaulExportJsonFilePath = filepath.Join(getConfigBasePath(), "/etc/agc/export.json")

// excel模板附录映射关系
type AppendixMapping struct {
	Headers []string       // 列名顺序
	Data    [][]string     // 行号 -> 各列描述
	ColIdx  map[string]int // 列名 -> 列索引
}

// DbConfigXMLManager XML管理器，提供所有操作接口
type DbConfigXMLManager struct {
	filePath        string           // XML文件路径
	exportJsonPath  string           // agc rtu导出的export.json路径
	mu              sync.Mutex       // 并发安全锁
	db              *DB              // 内存中的XML数据
	rtuID           map[int]struct{} // 记录已存在的rtuID
	deviceID        int              // 记录已存在的最大deviceID
	formulaID       int              // 记录已存在的最大formulaID
	formulaFactorID int              // 记录已存在的最大formulaFactorID
	templateFile    *excelize.File   // 导入导出模板excel文件
	appendixMapping *AppendixMapping // excel模板附录映射
	colPathsCache   sync.Map         // key: reflect.Type, value: map[string][]int
}

func GetDbConfigManager() (*DbConfigXMLManager, error) {
	once.Do(func() {
		// 初始化实例
		log.Info("init DbConfigXMLManager")
		m := &DbConfigXMLManager{
			filePath:       defaultDBXMLFilePath,
			exportJsonPath: defaulExportJsonFilePath,
		}
		m.rtuID = make(map[int]struct{})

		// 加载点表XML文件
		m.loadXML()
		// 加载导入导出模板文件
		m.loadTemplateAndAppendix()

		// 初始化成功，赋值给instance
		instance = m
	})

	// 返回实例和错误
	return instance, initErr
}

// loadXML 加载xml文件并统计各类id信息
func (m *DbConfigXMLManager) loadXML() {
	// 加载XML文件
	if err := m.load(); err != nil {
		initErr = fmt.Errorf("init DbConfigXMLManager load xml failed:%w", err)
		return
	}
	log.Info("init DbConfigXMLManager success")

	// 遍历单表数据, 统计已存在的各类ID信息
	for i := range m.db.Single.Tables {
		if m.db.Single.Tables[i].Name == "rtu" { // rtu用map存储
			for j := range m.db.Single.Tables[i].RTUs {
				id, err := strconv.Atoi(m.db.Single.Tables[i].RTUs[j].ID)
				if err != nil {
					initErr = fmt.Errorf("RTU ID %s is not num", m.db.Single.Tables[i].RTUs[j].ID)
					return
				}
				m.rtuID[id] = struct{}{}
			}
		} else if m.db.Single.Tables[i].Name == "device" { // device仅保留最大值
			for j := range m.db.Single.Tables[i].Devices {
				id, err := strconv.Atoi(m.db.Single.Tables[i].Devices[j].ID)
				if err != nil {
					initErr = fmt.Errorf("Device ID %s is not num", m.db.Single.Tables[i].Devices[j].ID)
					return
				}
				// 过滤60000以上的特殊设备
				if id < 60001 {
					m.deviceID = max(m.deviceID, id)
				}
			}
		} else if m.db.Single.Tables[i].Name == "formula" { // formula需要切割字符串, 仅保留最大值
			for j := range m.db.Single.Tables[i].Formulas {
				parts := strings.Split(m.db.Single.Tables[i].Formulas[j].ID, "-")
				id, err := strconv.Atoi(parts[len(parts)-1])
				if err != nil {
					initErr = fmt.Errorf("Formula ID %s 格式错误", m.db.Single.Tables[i].Formulas[j].ID)
					return
				}
				m.formulaID = max(m.formulaID, id)
			}
		} else if m.db.Single.Tables[i].Name == "formula_factor" { // formula_factor需要切割字符串, 仅保留最大值
			for j := range m.db.Single.Tables[i].FormulaFactors {
				parts := strings.Split(m.db.Single.Tables[i].FormulaFactors[j].ID, "-")
				id, err := strconv.Atoi(parts[len(parts)-1])
				if err != nil {
					initErr = fmt.Errorf("FormulaFactor ID %s 格式错误", m.db.Single.Tables[i].FormulaFactors[j].ID)
					return
				}
				m.formulaFactorID = max(m.formulaFactorID, id)
			}
		}
	}

}

// 加载模板excel文件并解析附录Sheet页
func (m *DbConfigXMLManager) loadTemplateAndAppendix() error {
	templatePath := filepath.Join(getConfigBasePath(), "db", "template.xlsx")
	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return err
	}
	// 深拷贝一份存储，防止后续写操作污染原始模板
	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return err
	}
	m.templateFile, err = excelize.OpenReader(buf)
	if err != nil {
		return err
	}

	// 加载附录Sheet映射
	const appendixSheet = "附录"
	rows, err := m.templateFile.GetRows(appendixSheet)
	if err != nil || len(rows) < 2 {
		// 附录Sheet不存在或数据不足，使用空映射
		m.appendixMapping = &AppendixMapping{
			Headers: []string{},
			Data:    [][]string{},
			ColIdx:  make(map[string]int),
		}
		return nil
	}
	headers := rows[0]
	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		trimmed := strings.TrimSpace(h)
		colIdx[trimmed] = i
	}
	dataRows := rows[1:]
	m.appendixMapping = &AppendixMapping{
		Headers: headers,
		Data:    dataRows,
		ColIdx:  colIdx,
	}
	return nil
}

// getCachedColPaths 获取结构体字段的excel路径映射
func (m *DbConfigXMLManager) getCachedColPaths(slice interface{}) (map[string][]int, error) {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %T", slice)
	}
	if v.Len() == 0 {
		return make(map[string][]int), nil
	}
	elemType := v.Index(0).Type()
	structType := elemType
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("slice element must be struct or *struct, got %s", structType)
	}
	if cached, ok := m.colPathsCache.Load(structType); ok {
		return cached.(map[string][]int), nil
	}
	paths := buildExcelIndexPaths(structType)
	m.colPathsCache.Store(structType, paths)
	return paths, nil
}

// 辅助函数：获取Single中的指定表
func (m *DbConfigXMLManager) getSingleTable(tableName string) (*Table, error) {
	for i := range m.db.Single.Tables {
		if m.db.Single.Tables[i].Name == tableName {
			return &m.db.Single.Tables[i], nil
		}
	}
	return nil, fmt.Errorf("Single table %s not exist", tableName)
}

// ExportAllPointsToBuffer 导出指定RTU+Device下的五遥数据到Excel buffer
func (m *DbConfigXMLManager) ExportAllPointsToBuffer(rtuID, deviceID string) (*bytes.Buffer, error) {
	// 2. 复制模板（深拷贝）
	srcFile := m.templateFile

	var buf bytes.Buffer
	if err := srcFile.Write(&buf); err != nil {
		return nil, err
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// 3. 获取点数据
	allPoints, err := m.GetPointsFromDevice(rtuID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("获取测点数据失败: %w", err)
	}

	// 4. 定义需要导出的sheets
	type sheetConfig struct {
		name string
		data interface{}
	}
	sheets := []sheetConfig{
		{"遥测", allPoints.Analogs},
		{"遥信", allPoints.Digitals},
		{"遥脉", allPoints.Pulses},
		{"遥调", allPoints.Regulates},
		{"遥控", allPoints.Controls},
		{"属性", allPoints.Attributes},
	}

	// 5. 逐个sheet写入
	for _, cfg := range sheets {
		if err := m.writeSheet(f, cfg.name, cfg.data, m.appendixMapping); err != nil {
			return nil, err
		}
	}

	// 6. 输出buffer
	outBuf := new(bytes.Buffer)
	if err := f.Write(outBuf); err != nil {
		return nil, err
	}
	return outBuf, nil
}

// ImportPointsFromExcel 从给定的 Excel 文件路径导入所有 Sheet 的数据到对应的 DevicePoints 字段
// 参数：excelPath - Excel 文件路径，rtuID, deviceID - 用于生成设备前缀
// 返回：*DevicePoints，包含导入后的所有测点切片
func (m *DbConfigXMLManager) ImportPointsFromExcel(reader io.Reader, rtuID, deviceID string) error {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return err
	}
	defer f.Close()

	points := &RtuDevPoints{}

	// 导入各个 Sheet
	if err := importSheetToPoints(f, "遥测", &points.Analogs, m.appendixMapping); err != nil {
		return fmt.Errorf("解析遥测失败: %w", err)
	}
	if err := importSheetToPoints(f, "遥信", &points.Digitals, m.appendixMapping); err != nil {
		return fmt.Errorf("解析遥信失败: %w", err)
	}
	if err := importSheetToPoints(f, "遥脉", &points.Pulses, m.appendixMapping); err != nil {
		return fmt.Errorf("解析遥脉失败: %w", err)
	}
	if err := importSheetToPoints(f, "遥调", &points.Regulates, m.appendixMapping); err != nil {
		return fmt.Errorf("解析遥调失败: %w", err)
	}
	if err := importSheetToPoints(f, "遥控", &points.Controls, m.appendixMapping); err != nil {
		return fmt.Errorf("解析遥控失败: %w", err)
	}
	if err := importSheetToPoints(f, "属性", &points.Attributes, m.appendixMapping); err != nil {
		return fmt.Errorf("解析属性失败: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 创建空设备测试导入点位是否正确
	testDev := &Device{BaseInfo: BaseInfo{ID: "65536", Name: "test"}, DeviceAddr: "65536"}
	initDev(testDev)
	if err := m.checkAddPointsID(rtuID, testDev, points); err != nil {
		return err
	}

	// 删除所有原始点位
	if err := m.deleteAllPointsFromDevice(rtuID, deviceID); err != nil {
		return err
	}
	// 导入新点位
	if err := m.addPointsToDevice(rtuID, deviceID, *points); err != nil {
		return err
	}

	return nil
}

// initDevn 初始化设备五遥属性表
func initDev(dev *Device) {
	dev.YC = &YC{XMLName: xml.Name{Local: "YC"}}
	dev.YX = &YX{XMLName: xml.Name{Local: "YX"}}
	dev.YM = &YM{XMLName: xml.Name{Local: "YM"}}
	dev.YT = &YT{XMLName: xml.Name{Local: "YT"}}
	dev.YK = &YK{XMLName: xml.Name{Local: "YK"}}
	dev.SX = &SX{XMLName: xml.Name{Local: "SX"}}
}

// writeSheet 写入单个sheet
func (m *DbConfigXMLManager) writeSheet(f *excelize.File, sheetName string, dataSlice interface{}, appendix *AppendixMapping) error {
	// 获取表头
	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) == 0 {
		return fmt.Errorf("sheet %s 不存在或无表头", sheetName)
	}
	headers := rows[0]

	// 清空除表头外的所有行（从第2行开始删除）
	for rowIdx := len(rows); rowIdx >= 2; rowIdx-- {
		_ = f.RemoveRow(sheetName, rowIdx) // 忽略删除错误
	}

	// 将数据转为 []map[string]string
	rowMaps, err := m.getExcelRowMaps(dataSlice)
	if err != nil {
		return fmt.Errorf("转换数据失败: %w", err)
	}
	if len(rowMaps) == 0 {
		return nil // 无数据，不写入
	}

	// 预计算特殊列的索引（提升性能）
	relatedYCIdx := -1
	relatedYXIdx := -1
	for i, h := range headers {
		switch h {
		case "相关遥测":
			relatedYCIdx = i
		case "相关遥信":
			relatedYXIdx = i
		}
	}

	// 写入数据行
	for rowIdx, rowMap := range rowMaps {
		rowData := make([]interface{}, len(headers))
		for colIdx, colName := range headers {
			rawVal := rowMap[colName]

			// 附录转换
			if idx, ok := appendix.ColIdx[colName]; ok {
				if n, err := strconv.Atoi(rawVal); err == nil && n >= 0 && n < len(appendix.Data) {
					if idx < len(appendix.Data[n]) {
						rawVal = appendix.Data[n][idx]
					}
				}
			}

			// 相关遥测/遥信只保留点号（Gin中'-'后面的部分）
			if colIdx == relatedYCIdx || colIdx == relatedYXIdx {
				rawVal = getGinByID(rawVal)
			}

			// 数字转数值类型
			if num, ok := isNumeric(rawVal); ok {
				rowData[colIdx] = num
			} else {
				rowData[colIdx] = rawVal
			}
		}
		cell, _ := excelize.CoordinatesToCellName(1, rowIdx+2)
		if err := f.SetSheetRow(sheetName, cell, &rowData); err != nil {
			return fmt.Errorf("写入第%d行失败: %w", rowIdx+2, err)
		}
	}
	return nil
}

// buildExcelIndexPaths 分析结构体类型 t，提取所有带 "excel" 标签的字段，
// 返回标签名到字段索引路径的映射。
// 索引路径是一个 []int，表示从根结构体到达该字段的连续字段索引序列，
// 可用于 reflect.Value.FieldByIndex 快速定位字段。
// 嵌入字段（匿名）会被递归展开；若嵌入字段为指针类型，会自动解引用。
// 若 t 不是结构体类型或指向结构体的指针，函数返回空 map。
func buildExcelIndexPaths(t reflect.Type) map[string][]int {
	// 处理传入的指针类型，统一转为结构体类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		// 非结构体类型无法遍历字段
		return make(map[string][]int)
	}

	paths := make(map[string][]int)

	var collect func(typ reflect.Type, prefix []int)
	collect = func(typ reflect.Type, prefix []int) {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)

			// 构造当前字段的路径：父路径 + 当前字段索引
			// 使用 append 并显式复制，避免意外共享底层数组（prefix 来自上层调用）
			curPath := make([]int, len(prefix), len(prefix)+1)
			copy(curPath, prefix)
			curPath = append(curPath, i)

			if field.Anonymous {
				// 嵌入字段：递归处理其内部字段
				ft := field.Type
				// 如果嵌入的是指针，需要解引用以获取实际的结构体类型
				if ft.Kind() == reflect.Ptr {
					ft = ft.Elem()
				}
				// 只对结构体类型进行递归（指针解引用后可能仍是结构体）
				if ft.Kind() == reflect.Struct {
					collect(ft, curPath)
				}
				// 非结构体的匿名嵌入（如基础类型）没有字段可递归，忽略
			} else {
				// 非匿名字段：检查 excel 标签
				tag := field.Tag.Get("excel")
				if tag != "" && tag != "-" {
					// 标签存在且非排除标记，记录路径
					// 若有重复标签，后面覆盖前面（可根据需求调整策略）
					paths[tag] = curPath
				}
			}
		}
	}

	collect(t, nil)
	return paths
}

// getValueByPath 按索引路径从可能是指针的 reflect.Value 中取值，
// 自动处理指针解引用和 nil 指针（返回空字符串），非 string 字段会通过 fmt.Sprint 转换。
func getValueByPath(v reflect.Value, path []int) string {
	// 如果传入的根元素是指针，先解引用
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	// 按路径逐层进入
	for _, idx := range path {
		// 每一步都检查是否为指针，若是则解引用
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return ""
			}
			v = v.Elem()
		}
		// 取字段
		v = v.Field(idx)
	}
	// 最后再处理一次指针（例如字段本身是指针类型）
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return ""
	}
	// 字符串直接返回，其他类型用 fmt.Sprint 转为字符串
	if v.Kind() == reflect.String {
		return v.String()
	}
	return fmt.Sprint(v.Interface())
}

// setValueByPath 按索引路径给结构体字段赋值，支持字符串自动转换类型
func setValueByPath(v reflect.Value, path []int, val string) error {
	// 如果 v 是指针，先解引用
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	// 进入路径（最后一个索引为字段，前面是父级）
	for i := 0; i < len(path)-1; i++ {
		idx := path[i]
		v = v.Field(idx)
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
	}
	// 最后一个字段
	lastIdx := path[len(path)-1]
	field := v.Field(lastIdx)
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}
	// 根据字段类型转换字符串
	switch field.Kind() {
	case reflect.String:
		field.SetString(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			// 兼容 "是"/"否"
			if val == "是" || val == "true" || val == "1" {
				b = true
			} else {
				b = false
			}
		}
		field.SetBool(b)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// ImportSheetToPoints 从 Excel 的指定 Sheet 导入数据到目标切片
// 参数：
//   - f: 已打开的 excelize.File
//   - sheetName: Sheet 名称
//   - target: 切片指针，如 &[]Analog{}
//   - appendix: 附录映射（复用导出时的附录结构）
func importSheetToPoints(f *excelize.File, sheetName string, target interface{}, appendix *AppendixMapping) error {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("读取sheet失败: %w", err)
	}
	if len(rows) < 2 {
		return nil // 没有数据行，直接返回
	}
	headers := rows[0]
	dataRows := rows[1:]

	// 校验target是否指向结构体切片
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr || targetVal.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("target must be pointer to slice")
	}
	sliceVal := targetVal.Elem()
	elemType := sliceVal.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("slice element must be struct or pointer to struct")
	}

	// 复用 buildExcelIndexPaths 获取标签到索引路径的映射
	colPaths := buildExcelIndexPaths(elemType) // map[excel标签] -> []int

	// 构建列名到 Excel 列索引的映射（表头）
	headerToIdx := make(map[string]int)
	for i, h := range headers {
		trimmed := strings.TrimSpace(h)
		if trimmed != "" {
			headerToIdx[trimmed] = i
		}
	}

	// 预构建每个标签对应的列索引（避免循环内多次查找）
	tagToColIdx := make(map[string]int)
	for tag := range colPaths {
		if idx, ok := headerToIdx[tag]; ok {
			tagToColIdx[tag] = idx
		}
	}

	// 附录反向映射：值 -> 行号（数值）  提前构建，提升性能
	var appendixReverse map[string]string
	if appendix != nil && len(appendix.Data) > 0 {
		appendixReverse = make(map[string]string)
		for rowNum, row := range appendix.Data {
			for colIdx, cellValue := range row {
				// 构造 key: 列名|单元格值
				colName := ""
				if colIdx < len(appendix.Headers) {
					colName = appendix.Headers[colIdx]
				}
				if colName != "" && cellValue != "" {
					key := colName + "|" + cellValue
					appendixReverse[key] = fmt.Sprint(rowNum) // 数值作为字符串
				}
			}
		}
	}

	// 遍历每一行数据
	for rowIdx, rowCells := range dataRows {
		// 创建新的结构体实例（元素是指针方便赋值）
		newElemPtr := reflect.New(elemType)
		newElem := newElemPtr.Elem()

		for tag, path := range colPaths {
			colIdx, ok := tagToColIdx[tag]
			if !ok || colIdx >= len(rowCells) {
				continue
			}
			rawVal := strings.TrimSpace(rowCells[colIdx])
			if rawVal == "" {
				continue
			}

			// 附录反向：将描述文字转回数值
			if appendixReverse != nil {
				key := tag + "|" + rawVal
				if numStr, exists := appendixReverse[key]; exists {
					rawVal = numStr
				}
			}

			// 赋值
			if err := setValueByPath(newElem, path, rawVal); err != nil {
				return fmt.Errorf("第%d行, 列[%s]赋值失败: %v", rowIdx+2, tag, err)
			}
		}
		// 追加到切片
		sliceVal.Set(reflect.Append(sliceVal, newElemPtr))
	}
	return nil
}

// getExcelRowMaps 带缓存的切片转map
func (m *DbConfigXMLManager) getExcelRowMaps(slice interface{}) ([]map[string]string, error) {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %T", slice)
	}
	if v.Len() == 0 {
		return nil, nil
	}
	colPaths, err := m.getCachedColPaths(slice)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		row := make(map[string]string, len(colPaths))
		for tag, path := range colPaths {
			row[tag] = getValueByPath(elem, path)
		}
		result[i] = row
	}
	return result, nil
}

// 基于oid获取点号
func getGinByID(id string) string {
	if i := strings.LastIndex(id, "-"); i != -1 {
		return id[i+1:]
	}
	return ""
}

// 优化后的数字判断（只判断整数/浮点即可）
func isNumeric(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	// 先尝试整数解析（更快）
	if i, err := strconv.Atoi(s); err == nil {
		return float64(i), true
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// load 从文件加载XML到内存
func (m *DbConfigXMLManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	initDB := func() error {
		m.db = &DB{Single: Single{Tables: []Table{
			{XMLName: xml.Name{Local: "table"}, Name: "rtu", TableNo: "201", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "device", TableNo: "202", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "digital", TableNo: "101", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "analog", TableNo: "102", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "control", TableNo: "103", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "regulate", TableNo: "104", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "pulse", TableNo: "105", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "trans_digital", TableNo: "106", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "trans_analog", TableNo: "107", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "trans_control", TableNo: "108", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "trans_regulate", TableNo: "109", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "trans_pulse", TableNo: "110", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "formula", TableNo: "111", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "formula_factor", TableNo: "112", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "attribute", TableNo: "115", MemOf: "1"},
			{XMLName: xml.Name{Local: "table"}, Name: "trans_attribute", TableNo: "116", MemOf: "1"},
		}}}
		return m.save()
	}
	// 读取文件
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		// 文件不存在则初始化空的DB
		log.Error("ReadFile failed, err:", err.Error())
		if errors.Is(err, os.ErrNotExist) {
			if err := initDB(); err != nil {
				return err
			}
			// 默认添加系统监视RTU，其下的固定特殊设备由 addRTU 同步创建
			monitorRTU := &RTU{BaseInfo: BaseInfo{ID: "800", Name: "系统监视"}, Type: RtuTypeMonit, RtuAddr: "800", ChannelGroupID: -1, ChannelID: -1}
			if err := m.addRTU(monitorRTU, false); err != nil {
				return err
			}
			return m.save()
		}
		return err
	} else if data == nil || len(data) == 0 {
		return initDB()
	}

	// 反序列化XML
	var db DB
	if err := xml.Unmarshal(data, &db); err != nil {
		return err
	}

	m.db = &db
	return nil
}

// save 将内存中的DB保存到文件
func (m *DbConfigXMLManager) save() error {
	// m.mu.Lock()
	// defer m.mu.Unlock()

	// 序列化XML（带缩进）
	data, err := xml.MarshalIndent(m.db, "", "    ")
	if err != nil {
		return err
	}

	// 添加XML声明
	fullData := append([]byte(xml.Header), data...)

	// 写入文件
	if err := os.WriteFile(m.filePath, fullData, 0644); err != nil {
		return err
	}

	return nil
}

func deleteSliceItemByIDs[T IDer](slicePtr *[]T, delIDs map[string]struct{}) {
	// 双指针：writeIdx 为写入位置，原地覆盖保留元素
	writeIdx := 0
	slice := *slicePtr
	for i := range slice {
		id := slice[i].GetID()
		if _, has := delIDs[id]; !has {
			slice[writeIdx] = slice[i]
			writeIdx++
		}
	}
	*slicePtr = slice[:writeIdx]
}

// -------------------------- Hierarchy RTU的CRUD --------------------------

// 从层次表中获取RTU信息
func (m *DbConfigXMLManager) getRTUFromHierarchy(rtuID string) (*RTU, error) {
	// 遍历层次表中的RTU
	for i := range m.db.Hierarchy.RTUs {
		if m.db.Hierarchy.RTUs[i].ID == rtuID {
			// 返回匹配到的RTU指针
			return m.db.Hierarchy.RTUs[i], nil
		}
	}
	// 未找到对应RTU，返回标准错误
	return nil, fmt.Errorf("RTU ID %s is not exist in Hierarchy", rtuID)
}

// 从单表中获取RTU信息
func (m *DbConfigXMLManager) getRTUFromSingle(rtuID string) (*RTU, error) {
	// 1. 获取Single中的rtu单表
	table, err := m.getSingleTable("rtu")
	if err != nil {
		return nil, fmt.Errorf("get rtu table failed: %w", err)
	}

	// 2. 遍历单表中的RTU，按ID匹配
	for i := range table.RTUs {
		if table.RTUs[i].ID == rtuID {
			// 返回匹配到的RTU指针
			return table.RTUs[i], nil
		}
	}

	// 3. 未找到对应RTU，返回标准错误
	return nil, fmt.Errorf("RTU ID %s is not exist in Single table", rtuID)
}

// 生成RTU ID
func (m *DbConfigXMLManager) generateRTUID(rtuType int) (string, error) {
	switch rtuType {
	case RtuTypeAcqu:
		for i := 0; i < 500; i++ {
			if _, has := m.rtuID[i]; !has {
				m.rtuID[i] = struct{}{}
				return strconv.Itoa(i), nil
			}
		}
		return "", fmt.Errorf("0-499 内所有RTU ID 已用完")
	case RtuTypeTrans:
		for i := 500; i < 699; i++ {
			if _, has := m.rtuID[i]; !has {
				m.rtuID[i] = struct{}{}
				return strconv.Itoa(i), nil
			}
		}
		return "", fmt.Errorf("500-699 内所有RTU ID 已用完")
	case RtuTypeAgc:
		if _, has := m.rtuID[700]; !has {
			m.rtuID[700] = struct{}{}
			return "700", nil
		}
		return "", fmt.Errorf("RTU ID 700 已存在")
	case RtuTypeMonit:
		if _, has := m.rtuID[800]; !has {
			m.rtuID[800] = struct{}{}
			return "800", nil
		}
		return "", fmt.Errorf("RTU ID 800 已存在")
	case RtuTypeFormula:
		if _, has := m.rtuID[900]; !has {
			m.rtuID[900] = struct{}{}
			return "900", nil
		}
		return "", fmt.Errorf("RTU ID 900 已存在")
	default:
		return "", fmt.Errorf("invalid rtu type: %d", rtuType)
	}
}

// GenerateRTUID 公开方法：预生成 RTU ID（加锁），用于创建通道前先确定 RTU ID
func (m *DbConfigXMLManager) GenerateRTUID(rtuType int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generateRTUID(rtuType)
}

// UseRTUID 将已预生成的 ID 注册到 rtuID map 中（ID 已在 GenerateRTUID 中注册，此处无需操作）
// 仅用于 addRTU 跳过重复生成

func (m *DbConfigXMLManager) addRTUWithPreAssignedID(rtu *RTU, autoSave bool) error {
	rtus, err := m.getSingleTable("rtu")
	if err != nil {
		return err
	}

	rtu.RtuAddr = rtu.ID
	// 层次表
	m.db.Hierarchy.RTUs = append(m.db.Hierarchy.RTUs, rtu)
	// 单边添加拷贝
	copyRtu := *rtu
	rtus.RTUs = append(rtus.RTUs, &copyRtu)

	if autoSave {
		return m.save()
	}
	return nil
}

func (m *DbConfigXMLManager) addRTU(rtu *RTU, autoSave bool) error {
	rtus, err := m.getSingleTable("rtu")
	if err != nil {
		return err
	}
	rtu.ID, err = m.generateRTUID(rtu.Type)
	if err != nil {
		return err
	}

	rtu.RtuAddr = rtu.ID
	// 层次表
	m.db.Hierarchy.RTUs = append(m.db.Hierarchy.RTUs, rtu)
	// 单边添加拷贝
	copyRtu := *rtu
	rtus.RTUs = append(rtus.RTUs, &copyRtu)

	// 系统监视RTU(800)同步创建固定特殊设备
	if rtu.ID == "800" {
		if err := m.addMonitorRTUDevices(rtu.ID); err != nil {
			return err
		}
	}

	if autoSave {
		return m.save()
	}
	return nil
}

// addMonitorRTUDevices 为系统监视RTU(800)添加固定特殊设备
func (m *DbConfigXMLManager) addMonitorRTUDevices(rtuID string) error {
	devices := []*Device{
		{BaseInfo: BaseInfo{ID: "60001", Name: "RTU状态"}, DeviceAddr: "1"},
		{BaseInfo: BaseInfo{ID: "60002", Name: "通道状态"}, DeviceAddr: "2"},
		{BaseInfo: BaseInfo{ID: "60003", Name: "程序状态"}, DeviceAddr: "3"},
	}
	for _, dev := range devices {
		if err := m.addDeviceToRTU(rtuID, dev); err != nil {
			return err
		}
	}
	return nil
}

// AddHierarchyRTU 添加Hierarchy中的RTU
// 参数：rtu 要添加的RTU
// 返回：错误信息（如ID重复）
func (m *DbConfigXMLManager) AddRTU(rtu *RTU) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 保存到文件
	return m.addRTU(rtu, true)
}

// setRTUChannelInternal 设置 RTU 绑定的通道组号和通道号（不 save，需在锁内调用）
func (m *DbConfigXMLManager) setRTUChannelInternal(rtuID string, channelGroupID, channelID int) error {
	rtu, err := m.getRTUFromHierarchy(rtuID)
	if err != nil {
		return err
	}
	rtu.ChannelGroupID = channelGroupID
	rtu.ChannelID = channelID

	// 同步更新单表
	sRtu, err := m.getRTUFromSingle(rtuID)
	if err != nil {
		return err
	}
	sRtu.ChannelGroupID = channelGroupID
	sRtu.ChannelID = channelID

	return nil
}

// AddRTUWithChannel 创建 RTU 并绑定通道，统一加锁，一次 save
// 要求 rtu.ID 已通过 GenerateRTUID 预分配
func (m *DbConfigXMLManager) AddRTUWithChannel(rtu *RTU, channelGroupID, channelID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 创建 RTU（使用预分配的 ID，不 save）
	if err := m.addRTUWithPreAssignedID(rtu, false); err != nil {
		return err
	}

	// 2. 设置绑定关系（不 save）
	if err := m.setRTUChannelInternal(rtu.ID, channelGroupID, channelID); err != nil {
		return err
	}

	// 3. 统一 save 一次
	return m.save()
}

// deleteRTU 内部无锁版本，autoSave=true 时立即写文件
func (m *DbConfigXMLManager) deleteRTU(rtuID string, autoSave bool) error {
	// 在层次结构和单表结构中查询rtu
	rtu, err := m.getRTUFromHierarchy(rtuID)
	if err != nil {
		return err
	}
	_, err = m.getRTUFromSingle(rtuID)
	if err != nil {
		return err
	}

	rtus, err := m.getSingleTable("rtu")
	if err != nil {
		return err
	}

	delID := make(map[string]struct{})
	delID[rtuID] = struct{}{}

	//先删除rtu下所有的device
	for i := range rtu.Devices {
		m.deleteDeviceFromRTU(rtuID, rtu.Devices[i].ID)
	}

	// 删除单表和层次表中的rtu
	deleteSliceItemByIDs(&m.db.Hierarchy.RTUs, delID)
	deleteSliceItemByIDs(&rtus.RTUs, delID)

	rtuIDInt, err := strconv.Atoi(rtuID)
	if err != nil {
		log.Errorf("RTU ID[%s]转换失败: %v", rtuID, err)
	} else {
		delete(m.rtuID, rtuIDInt)
	}

	if autoSave {
		return m.save()
	}
	return nil
}

// DeleteHierarchyRTU 删除Hierarchy中的RTU
// 参数：rtuID RTU的ID
// 返回：错误信息
func (m *DbConfigXMLManager) DeleteRTU(rtuID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.deleteRTU(rtuID, true)
}

// DeleteRTUWithChannel 删除 RTU 并同步删除通道组+通道，统一加锁，一次 save
func (m *DbConfigXMLManager) DeleteRTUWithChannel(rtuID string, deleteCGFunc func(rtuID string) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 删除通道组+通道（由 feConfig 管理，独立 save）
	if err := deleteCGFunc(rtuID); err != nil {
		log.Warnf("DeleteRTUWithChannel: delete channel group failed: %v", err)
	}

	// 2. 删除 RTU（不 save，最后统一 save）
	if err := m.deleteRTU(rtuID, false); err != nil {
		return err
	}

	// 3. 统一 save
	return m.save()
}

// GetHierarchyRTU 查询Hierarchy中的RTU
// 参数：rtuID RTU的ID
// 返回：RTU实例、错误信息
func (m *DbConfigXMLManager) GetHierarchyRTU(rtuID string) (*RTU, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.getRTUFromHierarchy(rtuID)
}

func updateRTU(oldRTU, newRTU *RTU) {
	oldRTU.Name = newRTU.Name
	oldRTU.InService = newRTU.InService
	oldRTU.MemOfCabinet = newRTU.MemOfCabinet
	oldRTU.IsConnected = newRTU.IsConnected
}

// UpdateHierarchyRTU 更新Hierarchy中的RTU
// 参数：rtu 要更新的RTU（根据ID匹配）
// 返回：错误信息
func (m *DbConfigXMLManager) UpdateHierarchyRTU(newRtu RTU, rtuID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 在层次结构和单表结构中查询rtu
	rtu, err := m.getRTUFromHierarchy(rtuID)
	if err != nil {
		return err
	}
	sRtu, err := m.getRTUFromSingle(rtuID)
	if err != nil {
		return err
	}

	updateRTU(rtu, &newRtu)
	updateRTU(sRtu, &newRtu)

	// 保存到文件
	return m.save()
}

func (m *DbConfigXMLManager) GetAllRTUs() []*RTU {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.db.Hierarchy.RTUs
}

// LoadRTUChannelRelation 启动时调用，遍历 RTU 填充 Channel 内存指针
func (m *DbConfigXMLManager) LoadRTUChannelRelation() error {
	fecfg, err := GetFeConfigXMLManager()
	if err != nil {
		return err
	}

	for _, rtu := range m.db.Hierarchy.RTUs {
		if rtu.ChannelGroupID < 0 {
			continue
		}
		cg, err := fecfg.GetChannelGroupByID(rtu.ChannelGroupID)
		if err != nil {
			log.Warnf("LoadRTUChannelRelation: RTU %s channel group %d not found: %v", rtu.ID, rtu.ChannelGroupID, err)
			continue
		}
		rtu.ChannelGroupName = cg.Name
		// 填充 FOR_TRANSFER
		for _, p := range cg.Model.Parameters {
			if p.Name == "FOR_TRANSFER" {
				rtu.ForTransfer = p.Value
				break
			}
		}
		if len(cg.Channels) == 0 {
			log.Warnf("LoadRTUChannelRelation: RTU %s channel group %d has no channel", rtu.ID, rtu.ChannelGroupID)
			continue
		}
		rtu.Channel = &cg.Channels[0]
	}
	return nil
}

// GetDB 返回DB指针（用于只读遍历层级结构）
func (m *DbConfigXMLManager) GetDB() *DB {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.db
}

// GetAllRTUsExceptPoints 获取所有rtu信息（不包含测点）
func (m *DbConfigXMLManager) GetAllRTUsExceptPoints() []RTU {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 浅拷贝创建新切片，仅保留设备基础信息，清空所有测点
	tmpRTUs := make([]RTU, len(m.db.Hierarchy.RTUs))
	for i, originalRTU := range m.db.Hierarchy.RTUs {
		// 1. 浅拷贝整个RTU（无深拷贝，Go结构体值复制默认浅拷贝）
		newRTU := *originalRTU

		// 2. 浅拷贝Device切片，创建独立的设备列表（不修改原始数据）
		newDevices := make([]*Device, len(newRTU.Devices))
		for j, originalDevice := range newRTU.Devices {
			// 浅拷贝单个Device
			newDevice := *originalDevice
			// 3. 清空测点（仅修改新对象，原始数据 untouched）
			newDevice.YC = nil
			newDevice.YX = nil
			newDevice.YM = nil
			newDevice.YK = nil
			newDevice.YT = nil
			newDevice.SX = nil
			newDevices[j] = &newDevice
		}

		// 替换为清空测点后的设备列表
		newRTU.Devices = newDevices
		tmpRTUs[i] = newRTU
	}

	// 最终返回 tmpRTUs 即可

	return tmpRTUs
}

// -------------------------- RTU下Device的CRUD --------------------------

// 从层次表中获取设备信息
func (m *DbConfigXMLManager) getDeviceFromRTU(rtuID, deviceID string) (*Device, error) {
	// 查找RTU
	for i := range m.db.Hierarchy.RTUs {
		if m.db.Hierarchy.RTUs[i].ID == rtuID {
			// 查找Device
			for j := range m.db.Hierarchy.RTUs[i].Devices {
				if m.db.Hierarchy.RTUs[i].Devices[j].ID == deviceID {
					return m.db.Hierarchy.RTUs[i].Devices[j], nil
				}
			}
			return nil, fmt.Errorf("RTU %s Device ID %s not exist", rtuID, deviceID)
		}
	}
	return nil, fmt.Errorf("RTU ID %s not exist", rtuID)
}

// 从单表中获取设备信息
func (m *DbConfigXMLManager) getDeviceFromSingle(deviceID string) (*Device, error) {
	// 1. 获取Single中的device单表
	table, err := m.getSingleTable("device")
	if err != nil {
		return nil, fmt.Errorf("get device table failed: %w", err)
	}

	// 2. 遍历单表中的设备，按ID匹配
	for i := range table.Devices {
		if table.Devices[i].ID == deviceID {
			// 返回匹配到的设备指针
			return table.Devices[i], nil
		}
	}

	// 3. 未找到对应设备，返回标准错误
	return nil, fmt.Errorf("Device ID %s not exist in Single table", deviceID)
}

// addDeviceToRTU 内部调用无锁版函数
func (m *DbConfigXMLManager) addDeviceToRTU(rtuID string, device *Device) error {
	// 在层次结构和单表结构中查询rtu
	rtu, err := m.getRTUFromHierarchy(rtuID)
	if err != nil {
		return err
	}

	// 校验设备Addr是否重复
	for i := range rtu.Devices {
		if rtu.Devices[i].DeviceAddr == device.DeviceAddr {
			return fmt.Errorf("设备地址:[%s]已存在", device.DeviceAddr)
		}
	}

	// agc rtu下只能添加一个设备
	if rtuID == "700" && len(rtu.Devices) >= 1 {
		return fmt.Errorf("agc类型的rtu下只能添加一个设备")
	}

	// 获取设备单表
	table, err := m.getSingleTable("device")
	if err != nil {
		return err
	}

	// 非系统监视RTU自动生成设备id
	if rtuID != "800" {
		m.deviceID++
		device.ID = strconv.Itoa(m.deviceID)
	}
	device.XMLName = xml.Name{Local: "Device"}
	device.MemOfRtu = rtuID
	// 单表中添加拷贝
	copyDevice := *device
	table.Devices = append(table.Devices, &copyDevice)

	initDev(device)

	// 添加Device
	rtu.Devices = append(rtu.Devices, device)

	// 保存到文件
	return m.save()
}

// AddDeviceToRTU 给指定RTU添加Device
// 参数：rtuID RTU的ID，device 要添加的Device
// 返回：错误信息
func (m *DbConfigXMLManager) AddDeviceToRTU(rtuID string, device *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.addDeviceToRTU(rtuID, device)
}

// deleteDeviceFromRTU 从指定RTU删除Device
// 参数：rtuID RTU的ID，deviceID Device的ID
// 返回：错误信息
func (m *DbConfigXMLManager) deleteDeviceFromRTU(rtuID, deviceID string) error {
	// 在层次结构和单表结构中查询rtu
	rtu, err := m.getRTUFromHierarchy(rtuID)
	if err != nil {
		return err
	}

	// 获取设备单表
	table, err := m.getSingleTable("device")
	if err != nil {
		return err
	}

	// 先删除设备下所有测点
	if err := m.deleteAllPointsFromDevice(rtuID, deviceID); err != nil {
		return err
	}

	// 待删除设备ID
	delID := make(map[string]struct{})
	delID[deviceID] = struct{}{}

	// 删除单表结构和层次结构中的设备信息
	deleteSliceItemByIDs(&rtu.Devices, delID)
	deleteSliceItemByIDs(&table.Devices, delID)

	// 保存到文件
	return m.save()
}

// DeleteDeviceFromRTU 从指定RTU删除Device, 外部调用，加锁
func (m *DbConfigXMLManager) DeleteDeviceFromRTU(rtuID, deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.deleteDeviceFromRTU(rtuID, deviceID)
}

// 公有方法：对外调用，加锁
func (m *DbConfigXMLManager) GetDeviceFromRTU(rtuID, deviceID string) (*Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 调用内部无锁方法
	return m.getDeviceFromRTU(rtuID, deviceID)
}

func updateDevice(oldDev, newDev *Device) {
	oldDev.Name = newDev.Name
	oldDev.InService = newDev.InService
	oldDev.MqttKey = newDev.MqttKey
}

// UpdateDeviceInRTU 更新指定RTU下的Device
// 参数：rtuID RTU的ID，device 要更新的Device（根据ID匹配）
// 返回：错误信息
func (m *DbConfigXMLManager) UpdateDeviceInRTU(rtuID string, device *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 在层次结构和单表中查询对应的设备
	dev, err := m.getDeviceFromRTU(rtuID, device.ID)
	if err != nil {
		return err
	}
	sDev, err := m.getDeviceFromSingle(dev.ID)
	if err != nil {
		return err
	}

	updateDevice(dev, device)
	updateDevice(sDev, device)

	// 保存到文件
	return m.save()
}

// UpdateDeviceInRTU 获取指定RTU下的全部Device
// 参数：rtuID RTU的ID
// 返回：全部Device信息，错误信息
func (m *DbConfigXMLManager) GetAllDevicesFromRTU(rtuID string) ([]*Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.db.Hierarchy.RTUs {
		r := m.db.Hierarchy.RTUs[i]
		if r.ID == rtuID {
			return r.Devices, nil
		}
	}
	return nil, fmt.Errorf("RTU ID %s not exist", rtuID)
}

// -------------------------- Device下测点的CRUD --------------------------

type RtuDevPoints struct {
	Analogs    []*Analog    `json:"analog"`    // 遥测
	Digitals   []*Digital   `json:"digital"`   // 遥信
	Pulses     []*Pulse     `json:"pulse"`     // 遥脉
	Controls   []*Control   `json:"control"`   // 遥控
	Regulates  []*Regulate  `json:"regulate"`  // 遥调
	Attributes []*Attribute `json:"attribute"` // 属性
}

func transTableName(rtuID, tableName string) string {
	id, _ := strconv.Atoi(rtuID)
	if id >= 500 && id < 699 {
		return "trans_" + tableName
	} else {
		return tableName
	}
}

// getPointIDs 通用提取测点ID列表
func getPointIDs[T IDer](points []T) map[string]struct{} {
	ids := make(map[string]struct{}, len(points))
	for i := range points {
		id := points[i].GetID()
		ids[id] = struct{}{}
	}
	return ids
}

// getDataSources 获取所有数据源
func getDataSources[T Pointer](points []T) map[string]struct{} {
	ids := make(map[string]struct{}, len(points))
	for i := range points {
		id := points[i].GetDataSource()
		ids[id] = struct{}{}
	}
	return ids
}

// getNameIDMap 获取测点名 -> 测点ID的映射关系
func getNameIDMap[T IDer](points []T) map[string]string {
	nameIDMap := make(map[string]string, len(points))
	for i := range points {
		name := points[i].GetName()
		id := points[i].GetID()
		nameIDMap[name] = id
	}
	return nameIDMap
}

// getIDNameMap 获取测ID -> 测点名的映射关系
func getIDNameMap[T IDer](points []T) map[string]string {
	idNameMap := make(map[string]string, len(points))
	for i := range points {
		name := points[i].GetName()
		id := points[i].GetID()
		idNameMap[id] = name
	}
	return idNameMap
}

// getMaxGin 获取最大点号
func getMaxGin[T Pointer](points []T) int {
	maxGin := 0
	for i := range points {
		ginStr := points[i].GetGin()
		gin, _ := strconv.Atoi(ginStr)
		maxGin = max(maxGin, gin)
	}
	return maxGin
}

// 生成测点ID，同时校验ID是否重复
func GenAndcheckPointsID[T Pointer](m *DbConfigXMLManager, rtuID, deviceID, tableName string, sourcePoints, targetPoints []T) (map[string]struct{}, error) {
	// 判断是否需要放到转发表
	tableName = transTableName(rtuID, tableName)

	// 获取对应单表
	table, err := m.getSingleTable(tableName)
	if err != nil {
		return nil, err
	}

	// 已存在ID集合
	existIDMap := getPointIDs(sourcePoints)
	// agc rtu 需要自动生成点号
	maxGin := getMaxGin(sourcePoints)

	// 生成ID并校验是否存在
	for i := range targetPoints {
		// 点号为空时自动生成点号
		if targetPoints[i].GetGin() == "" {
			maxGin++
			targetPoints[i].SetGin(strconv.Itoa(maxGin))
		}
		gin := targetPoints[i].GetGin()
		newID := fmt.Sprintf("%s-%s-%s-%s", rtuID, deviceID, table.TableNo, gin)
		if _, has := existIDMap[newID]; has {
			return nil, fmt.Errorf("%s第[%d]个测点ID重复: %s", tableName, i+1, newID)
		}
		existIDMap[newID] = struct{}{}
		targetPoints[i].SetID(newID)
	}
	return existIDMap, nil
}

// 生成相关遥测、相关遥信的ID字段
func genRelateID(rtuID, deviceID, tableNo, relateID string) string {
	// 判断相关遥测中是否包括"-"字符, 不包含按照统一逻辑生成
	if has := strings.Contains(relateID, "-"); has {
		return relateID
	} else {
		return fmt.Sprintf("%s-%s-%s-%s", rtuID, deviceID, tableNo, relateID)
	}
}

func mergeMaps[K comparable, V any](dst, src map[K]V) {
	for k, v := range src {
		dst[k] = v
	}
}

// 检查添加的测点ID与已存在的测点ID是否重复，添加的测点ID内部是否存在重复, 遥调相关遥测和遥控相关遥信是否存在
func (m *DbConfigXMLManager) checkAddPointsID(rtuID string, dev *Device, points *RtuDevPoints) error {
	// 校验五遥和属性数据ID是否重复
	analogIDs, err := GenAndcheckPointsID(m, rtuID, dev.ID, "analog", dev.YC.Analogs, points.Analogs) // 【遥测 Analog】
	if err != nil {
		return err
	}
	digitalIDs, err := GenAndcheckPointsID(m, rtuID, dev.ID, "digital", dev.YX.Digitals, points.Digitals) // 【遥信 Digital】
	if err != nil {
		return err
	}
	if _, err = GenAndcheckPointsID(m, rtuID, dev.ID, "pulse", dev.YM.Pulses, points.Pulses); err != nil { // 【遥脉 Pulses】
		return err
	}
	if _, err = GenAndcheckPointsID(m, rtuID, dev.ID, "regulate", dev.YT.Regulates, points.Regulates); err != nil { // 【遥调 Regulates】
		return err
	}
	if _, err = GenAndcheckPointsID(m, rtuID, dev.ID, "control", dev.YK.Controls, points.Controls); err != nil { // 【遥控 Controls】
		return err
	}
	if _, err = GenAndcheckPointsID(m, rtuID, dev.ID, "attribute", dev.SX.Attributes, points.Attributes); err != nil { // 【属性 Attributes】
		return err
	}

	// agc rtu需要自动生成相关遥测和相关遥信字段
	if rtuID == "700" {
		// 生成遥调相关遥测, 根据名称匹配，与遥调同名的遥测为相关遥测
		analogNameIDMap := getNameIDMap(dev.YC.Analogs)
		mergeMaps(analogNameIDMap, getNameIDMap(points.Analogs))
		for i := range points.Regulates {
			if relateAnalog, has := analogNameIDMap[points.Regulates[i].Name]; !has {
				return fmt.Errorf("遥调: [%s]相关遥测未添加(第%d个)", points.Regulates[i].Name, i+1)
			} else {
				points.Regulates[i].RelateAnalog = relateAnalog
			}
		}
		// 生成遥控相关遥调，逻辑与遥调相同
		digitalNameIDMap := getNameIDMap(dev.YX.Digitals)
		mergeMaps(digitalNameIDMap, getNameIDMap(points.Digitals))
		for i := range points.Controls {
			if relateDigital, has := digitalNameIDMap[points.Controls[i].Name]; !has {
				return fmt.Errorf("遥控: [%s]相关遥信未添加(第%d个)", points.Controls[i].Name, i+1)
			} else {
				points.Controls[i].RelateDigital = relateDigital
			}
		}
		return nil
	}

	// 校验遥调的相关遥测是否存在
	for i := range points.Regulates {
		relateAnalog := genRelateID(rtuID, dev.ID, "102", points.Regulates[i].RelateAnalog)
		// 检查相关遥测是否存在
		if _, has := analogIDs[relateAnalog]; !has {
			return fmt.Errorf("遥调: [%s] 相关遥测: [%s] 不存在(第%d个)", points.Regulates[i].Name, points.Regulates[i].RelateAnalog, i+1)
		} else {
			if dev.ID != "65536" {
				points.Regulates[i].RelateAnalog = relateAnalog
			}
		}
	}
	// 校验遥控的相关遥信是否存在
	for i := range points.Controls {
		relateDigital := genRelateID(rtuID, dev.ID, "101", points.Controls[i].RelateDigital)
		//检查相关遥信是否存在
		if _, has := digitalIDs[relateDigital]; !has {
			return fmt.Errorf("遥控: [%s] 相关遥信: [%s] 不存在(第%d个)", points.Controls[i].Name, points.Controls[i].RelateDigital, i+1)
		} else {
			if dev.ID != "65536" {
				points.Controls[i].RelateDigital = relateDigital
			}
		}
	}
	return nil
}

// handlePointSlice 使用反射实现的通用测点批量添加处理函数
// 功能：将测点追加到设备Hierarchy层级切片与Single单表切片，并更新对应类型测点计数
// 参数：
//
//	m         *DbConfigXMLManager  XML配置管理器实例
//	tableName string               单表名称(analog/digital/pulse/regulate/control)
//	sourceSlice Pointer            设备层级测点切片指针(如&dev.YC.Analogs/&dev.YX.Digitals)
//	targetSlice Pointer            待添加的测点切片(支持Analog/Digital/Pulse/Regulate/Control)
func handlePointSlice[T Pointer](m *DbConfigXMLManager, tableName string, sourcePoints, newPoints *[]T) {
	// 1. 获取对应单表
	table, _ := m.getSingleTable(tableName)

	// 2. 在层次表和单表中添加元素
	*sourcePoints = append(*sourcePoints, *newPoints...)
	table.addItemsToSingleTable(*newPoints)
}

// 导出到export.json文件中的测点信息
type ExportJsonPoint struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func collectPoints[T Pointer](points []T) []ExportJsonPoint {
	exportjsonPoint := make([]ExportJsonPoint, len(points))
	for i := range points {
		exportjsonPoint[i] = ExportJsonPoint{
			Name: points[i].GetPath(),
			ID:   points[i].GetID(),
		}
	}
	return exportjsonPoint
}

// 生成export.json文件
func writeExportJson(device *Device, filePath string) {
	export := map[string][]ExportJsonPoint{
		"digital": collectPoints(device.YX.Digitals),
		"analog":  collectPoints(device.YC.Analogs),
		"pluse":   collectPoints(device.YM.Pulses),
		"control": collectPoints(device.YK.Controls),
		"regu":    collectPoints(device.YT.Regulates),
		"para":    collectPoints(device.SX.Attributes),
	}

	data, _ := json.MarshalIndent(export, "", "  ")
	os.WriteFile(filePath, data, 0644)
}

// AddPointsToDevice 向设备中添加测点信息（先全量校验ID，校验通过再统一添加）
func (m *DbConfigXMLManager) addPointsToDevice(rtuID, deviceID string, points RtuDevPoints) error {
	// 1. 获取层次表中对应的设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return err
	}

	// 2. id校验
	if err := m.checkAddPointsID(rtuID, dev, &points); err != nil {
		return err
	}

	// ====================== 校验完成：所有ID无重复，开始统一	修改 ======================

	// 3. agc rtu的属性 值类型全部置为2 系数全部置为1
	if rtuID == "700" {
		for i := range points.Attributes {
			points.Attributes[i].ValueType = "2"
		}
		for i := range points.Analogs {
			points.Analogs[i].Coeff = "1"
		}
		for i := range points.Pulses {
			points.Pulses[i].Coeff = "1"
		}
	}

	// 4. 统一执行添加
	handlePointSlice(m, transTableName(rtuID, "analog"), &dev.YC.Analogs, &points.Analogs)          // 【遥测 Analog】
	handlePointSlice(m, transTableName(rtuID, "digital"), &dev.YX.Digitals, &points.Digitals)       // 【遥信 Digital】
	handlePointSlice(m, transTableName(rtuID, "pulse"), &dev.YM.Pulses, &points.Pulses)             // 【遥脉 Pulses】
	handlePointSlice(m, transTableName(rtuID, "regulate"), &dev.YT.Regulates, &points.Regulates)    // 【遥调 Regulates】
	handlePointSlice(m, transTableName(rtuID, "control"), &dev.YK.Controls, &points.Controls)       // 【遥控 Controls】
	handlePointSlice(m, transTableName(rtuID, "attribute"), &dev.SX.Attributes, &points.Attributes) // 【属性 Attributes】

	// 5. agc rtu需要导出export.json
	if rtuID == "700" {
		writeExportJson(dev, m.exportJsonPath)
	}

	// 6. 全部修改完成，保存XML
	return m.save()
}

// AddPointsToDevice 向设备中添加测点信息（先全量校验ID，校验通过再统一添加）
func (m *DbConfigXMLManager) AddPointsToDevice(rtuID, deviceID string, points RtuDevPoints) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addPointsToDevice(rtuID, deviceID, points)
}

func checkPointsIDExist[T IDer](tableName string, sourcePoints, targetPoints []T) error {
	existIDMap := getPointIDs(sourcePoints)

	for i := range targetPoints {
		id := targetPoints[i].GetID()
		if _, has := existIDMap[id]; !has {
			return fmt.Errorf("%s 测点ID不存在: %s", tableName, id)
		}
	}
	return nil
}

// checkPointsID 校验测点ID是否都存在于设备中
func checkPointsID(rtuID string, dev *Device, points *RtuDevPoints) error {
	if err := checkPointsIDExist("analog", dev.YC.Analogs, points.Analogs); err != nil {
		return err
	}
	if err := checkPointsIDExist("digital", dev.YX.Digitals, points.Digitals); err != nil {
		return err
	}
	if err := checkPointsIDExist("pulse", dev.YM.Pulses, points.Pulses); err != nil {
		return err
	}
	if err := checkPointsIDExist("regulate", dev.YT.Regulates, points.Regulates); err != nil {
		return err
	}
	if err := checkPointsIDExist("control", dev.YK.Controls, points.Controls); err != nil {
		return err
	}
	if err := checkPointsIDExist("attribute", dev.SX.Attributes, points.Attributes); err != nil {
		return err
	}
	return nil
}

// handleDeletePointSliceByID 按测点ID批量删除设备层级与Single表中的测点数据
// 功能：同步删除Hierarchy层级结构和Single单表中的指定ID测点，更新对应类型的测点计数
// 参数：
//
//	m         *DbConfigXMLManager  XML配置管理器实例
//	tableName string               单表名称(analog/digital/pulse/regulate/control)
//	dev       *Device              目标设备结构体指针
//	delIDs    map[string]struct{}  待删除的测点ID集合(键为ID，值为空结构体)
//	sourcePoints any               设备层级测点切片指针(如&dev.YC.Analogs)
func handleDeletePointSliceByID[T IDer](m *DbConfigXMLManager, tableName string, sourcePoints *[]T, delIDs map[string]struct{}) {
	if len(delIDs) == 0 {
		return
	}
	// 1. 层级结构：按ID删除
	deleteSliceItemByIDs(sourcePoints, delIDs)

	// 2. 单表同步按ID删除
	table, err := m.getSingleTable(tableName)
	if err != nil {
		return
	}
	table.deleteItemsFromSingleTable(delIDs)
}

// DeletePointsFromDevice 按ID批量删除测点（先校验存在，再精准删除，不清空）
func (m *DbConfigXMLManager) DeletePointsFromDevice(rtuID, deviceID string, points RtuDevPoints) error {
	// 1. 单次加锁，杜绝死锁
	m.mu.Lock()
	defer m.mu.Unlock()

	// 2. 获取层次结构和单表中对应的设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return err
	}

	// 3. 校验所有待删ID都存在（不存在直接报错）
	if err := checkPointsID(rtuID, dev, &points); err != nil {
		return err
	}

	// 4. 如果是计算rtu还需要删除所有关联的公式
	if rtuID == "900" {
		m.deleteDeviceFormulaAndFactorIDs(dev, points)
	}

	// 5. 统一执行按ID删除
	handleDeletePointSliceByID(m, transTableName(rtuID, "analog"), &dev.YC.Analogs, getPointIDs(points.Analogs))
	handleDeletePointSliceByID(m, transTableName(rtuID, "digital"), &dev.YX.Digitals, getPointIDs(points.Digitals))
	handleDeletePointSliceByID(m, transTableName(rtuID, "pulse"), &dev.YM.Pulses, getPointIDs(points.Pulses))
	handleDeletePointSliceByID(m, transTableName(rtuID, "regulate"), &dev.YT.Regulates, getPointIDs(points.Regulates))
	handleDeletePointSliceByID(m, transTableName(rtuID, "control"), &dev.YK.Controls, getPointIDs(points.Controls))
	handleDeletePointSliceByID(m, transTableName(rtuID, "attribute"), &dev.SX.Attributes, getPointIDs(points.Attributes))

	// 6. 保存XML
	return m.save()
}

func (m *DbConfigXMLManager) deleteAllPointsFromDevice(rtuID, deviceID string) error {
	// 1. 获取层次结构和单表中对应的设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return err
	}

	// 2. 如果计算rtu还需要删除所有关联的公式
	if rtuID == "900" {
		m.deleteDeviceFormulaAndFactorIDs(dev, RtuDevPoints{Analogs: dev.YC.Analogs, Digitals: dev.YX.Digitals, Pulses: dev.YM.Pulses})
	}

	// 3. 统一执行按ID删除
	handleDeletePointSliceByID(m, transTableName(rtuID, "analog"), &dev.YC.Analogs, getPointIDs(dev.YC.Analogs))
	handleDeletePointSliceByID(m, transTableName(rtuID, "digital"), &dev.YX.Digitals, getPointIDs(dev.YX.Digitals))
	handleDeletePointSliceByID(m, transTableName(rtuID, "pulse"), &dev.YM.Pulses, getPointIDs(dev.YM.Pulses))
	handleDeletePointSliceByID(m, transTableName(rtuID, "regulate"), &dev.YT.Regulates, getPointIDs(dev.YT.Regulates))
	handleDeletePointSliceByID(m, transTableName(rtuID, "control"), &dev.YK.Controls, getPointIDs(dev.YK.Controls))
	handleDeletePointSliceByID(m, transTableName(rtuID, "attributr"), &dev.SX.Attributes, getPointIDs(dev.SX.Attributes))

	// 4. 保存XML
	return m.save()
}

// DeletePointsFromDevice 删除Device下所有测点
func (m *DbConfigXMLManager) DeleteAllPointsFromDevice(rtuID, deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 调用内部无锁函数
	return m.deleteAllPointsFromDevice(rtuID, deviceID)
}

// updateSliceItemByID 按ID反射更新切片元素
// 功能：遍历切片，根据ID匹配并更新对应元素，无匹配则跳过
func updateSliceItemByID[T Pointer](oldPoints, newPoints []T) {
	// 构建原切片ID映射
	idMap := make(map[string]int, len(oldPoints))
	for i := range oldPoints {
		id := oldPoints[i].GetID()
		idMap[id] = i
	}

	// 遍历新数据，按ID更新
	for i := range newPoints {
		id := newPoints[i].GetID()
		if oldIdx, exist := idMap[id]; exist {
			oldPoints[oldIdx] = newPoints[i]
		}
	}
}

// handleUpdatePointSliceByID 按测点ID批量更新设备层级与Single表中的测点数据
// 功能：同步更新Hierarchy层级结构和Single单表中的指定ID测点
func handleUpdatePointSliceByID[T Pointer](m *DbConfigXMLManager, rtuID, tableName string, oldPoints, newPoints []T) {
	// 1. 更新层级结构
	updateSliceItemByID(oldPoints, newPoints)

	tableName = transTableName(rtuID, tableName)

	// 2. 同步更新Single表
	table, err := m.getSingleTable(tableName)
	if err != nil {
		return
	}
	table.updateItemsInSingleTable(newPoints)
}

// 替换点号生成新OID
func generateNewID(id, newGin string) string {
	if i := strings.LastIndex(id, "-"); i != -1 {
		return id[:i+1] + newGin
	}
	return newGin // 无分隔符时全替换
}

// checkGins 校验修改后的gin是否重复
func checkGins[T Pointer](oldPoints, updatePoints []T) (map[string]struct{}, error) {
	allPointsID := getPointIDs(oldPoints)
	// 删除所有待修改的测点id
	for i := range updatePoints {
		delete(allPointsID, updatePoints[i].GetID())
	}
	// 遍历生成新id
	for i := range updatePoints {
		newID := generateNewID(updatePoints[i].GetID(), updatePoints[i].GetGin())
		if _, has := allPointsID[newID]; has {
			return allPointsID, fmt.Errorf("oid为: [%s] 的点修改后的点号重复", updatePoints[i].GetID())
		}
		allPointsID[newID] = struct{}{}
	}
	return allPointsID, nil
}

// checkUpdatePoints 校验修改后的点号、相关遥测、相关遥信是否正确
func checkUpdatePoints(dev *Device, points RtuDevPoints) error {
	newAnalogIDs, err := checkGins(dev.YC.Analogs, points.Analogs)
	if err != nil {
		return err
	}
	newDigitalIDs, err := checkGins(dev.YX.Digitals, points.Digitals)
	if err != nil {
		return err
	}
	if _, err := checkGins(dev.YM.Pulses, points.Pulses); err != nil {
		return err
	}
	if _, err := checkGins(dev.YT.Regulates, points.Regulates); err != nil {
		return err
	}
	if _, err := checkGins(dev.YK.Controls, points.Controls); err != nil {
		return err
	}
	if _, err := checkGins(dev.SX.Attributes, points.Attributes); err != nil {
		return err
	}

	// 校验遥调的相关遥测是否存在
	for i := range points.Regulates {
		relateAnalog := genRelateID(dev.MemOfRtu, dev.ID, "102", points.Regulates[i].RelateAnalog)
		// 检查相关遥测是否存在
		if _, has := newAnalogIDs[relateAnalog]; !has {
			return fmt.Errorf("遥调: [%s] 相关遥测: [%s] 不存在(第%d个)", points.Regulates[i].Name, points.Regulates[i].RelateAnalog, i+1)
		} else {
			points.Regulates[i].RelateAnalog = relateAnalog
		}
	}
	// 校验遥控的相关遥信是否存在
	for i := range points.Controls {
		relateDigital := genRelateID(dev.MemOfRtu, dev.ID, "101", points.Controls[i].RelateDigital)
		//检查相关遥信是否存在
		if _, has := newDigitalIDs[relateDigital]; !has {
			return fmt.Errorf("遥控: [%s] 相关遥信: [%s] 不存在(第%d个)", points.Controls[i].Name, points.Controls[i].RelateDigital, i+1)
		} else {
			points.Controls[i].RelateDigital = relateDigital
		}
	}
	return nil
}

// UpdatePointsInDevice 按ID批量更新Device下所有测点内容
// 功能：先校验测点ID存在性，再同步更新Hierarchy层级与Single单表测点数据
// 参数：
//
//	rtuID     string      所属RTU ID
//	deviceID  string      设备ID
//	points    RtuDevPoints 待更新的五遥测点数据（按ID匹配更新）
//
// 返回：更新错误信息
func (m *DbConfigXMLManager) UpdatePointsInDevice(rtuID, deviceID string, points RtuDevPoints) error {
	// 1. 单次加锁，杜绝死锁
	m.mu.Lock()
	defer m.mu.Unlock()

	// 2. 内部无锁获取设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return err
	}

	// 3. 校验所有待更新ID都存在（不存在直接报错，保证更新安全）
	if err := checkPointsID(rtuID, dev, &points); err != nil {
		return err
	}

	// 4. 校验点号的修改是否合法
	if err := checkUpdatePoints(dev, points); err != nil {
		return err
	}

	// 4. 统一执行按ID更新（层级+单表同步）
	handleUpdatePointSliceByID(m, rtuID, "analog", dev.YC.Analogs, points.Analogs)
	handleUpdatePointSliceByID(m, rtuID, "digital", dev.YX.Digitals, points.Digitals)
	handleUpdatePointSliceByID(m, rtuID, "pulse", dev.YM.Pulses, points.Pulses)
	handleUpdatePointSliceByID(m, rtuID, "regulate", dev.YT.Regulates, points.Regulates)
	handleUpdatePointSliceByID(m, rtuID, "control", dev.YK.Controls, points.Controls)
	handleUpdatePointSliceByID(m, rtuID, "attribute", dev.SX.Attributes, points.Attributes)

	// 5. 保存XML
	return m.save()
}

// GetPointsFromDevice 获取指定设备下的所有五遥测点数据，返回统一结构体RtuDevPoints
// 参数：
//
//	rtuID    RTU的ID
//	deviceID 设备ID
//
// 返回：
//
//	RtuDevPoints 包含遥测、遥信、遥脉、遥调、遥控所有测点的统一结构体
//	error        获取失败的错误信息
func (m *DbConfigXMLManager) GetPointsFromDevice(rtuID, deviceID string) (RtuDevPoints, error) {
	// 加锁保证并发安全
	m.mu.Lock()
	defer m.mu.Unlock()

	// 调用内部无锁方法获取设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return RtuDevPoints{}, err
	}

	// 组装并返回所有测点数据
	return RtuDevPoints{
		Analogs:    dev.YC.Analogs,
		Digitals:   dev.YX.Digitals,
		Pulses:     dev.YM.Pulses,
		Regulates:  dev.YT.Regulates,
		Controls:   dev.YK.Controls,
		Attributes: dev.SX.Attributes,
	}, nil
}

// GetTransPoints 获取某类测点信息，可根据exceptIDs进行排除
func GetTransPoints[T Pointer](points []T, exceptIDs map[string]struct{}) []T {
	transPoints := make([]T, 0, len(points))
	for i := range points {
		// exceptIDs不为空，为挑选转发RTU下的点，需要排除已添加的点
		if exceptIDs != nil {
			if points[i].GetTrans() == "1" {
				if _, has := exceptIDs[points[i].GetID()]; !has {
					transPoints = append(transPoints, points[i])
				}
			}
		} else {
			transPoints = append(transPoints, points[i])
		}
	}
	return transPoints
}

// GetPointsFromDeviceByType 获取指定设备下指定类型的测点数据，返回统一结构体RtuDevPoints
func (m *DbConfigXMLManager) GetPointsFromDeviceByType(rtuID, deviceID, dataType string) (RtuDevPoints, error) {
	// 加锁保证并发安全
	m.mu.Lock()
	defer m.mu.Unlock()

	// 调用内部无锁方法获取设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return RtuDevPoints{}, err
	}

	var points RtuDevPoints
	switch dataType {
	case "analog":
		points.Analogs = GetTransPoints(dev.YC.Analogs, nil)
	case "digital":
		points.Digitals = GetTransPoints(dev.YX.Digitals, nil)
	case "pulse":
		points.Pulses = GetTransPoints(dev.YM.Pulses, nil)
	case "regulate":
		points.Regulates = GetTransPoints(dev.YT.Regulates, nil)
	case "control":
		points.Controls = GetTransPoints(dev.YK.Controls, nil)
	case "attribute":
		points.Attributes = GetTransPoints(dev.SX.Attributes, nil)
	default:
		return points, fmt.Errorf("invalid dataType : %s", dataType)
	}
	// 组装并返回所有测点数据
	return points, nil
}

// GetPointsIDFromDeviceByType 获取指定设备指定类型下所有的测点id
func (m *DbConfigXMLManager) GetPointsIDFromDeviceByType(rtuID, deviceID, dataType string) (map[string]string, error) {
	// 加锁保证并发安全
	m.mu.Lock()
	defer m.mu.Unlock()

	// 调用内部无锁方法获取设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return nil, err
	}

	switch dataType {
	case "analog":
		return getIDNameMap(dev.YC.Analogs), nil
	case "digital":
		return getIDNameMap(dev.YX.Digitals), nil
	case "pulse":
		return getIDNameMap(dev.YM.Pulses), nil
	case "regulate":
		return getIDNameMap(dev.YT.Regulates), nil
	case "control":
		return getIDNameMap(dev.YK.Controls), nil
	case "attribute":
		return getIDNameMap(dev.SX.Attributes), nil
	default:
		return nil, fmt.Errorf("invalid dataType : %s", dataType)
	}
	// 组装并返回所有测点数据
}

type AgcBindPoint struct {
	PointID string `json:"pointID"`
	Name    string `json:"name"`
	MqttKey string `json:"mqttkey"`
}

// GetAllPointsByType 按测点类型全局获取所有测点（统一转为 AgcBindPoint）
// 参数：ptype 测点类型缩写(yc=遥测/yx=遥信/ym=遥脉/yt=遥调/yk=遥控)
// 返回：AgcBindPoint列表、错误信息
func (m *DbConfigXMLManager) GetAllPointsByType(ptype string) ([]AgcBindPoint, error) {
	// 加锁保证并发安全
	m.mu.Lock()
	defer m.mu.Unlock()

	// 类型映射：缩写 → 单表名
	typeMap := map[string]string{
		"yc": "analog",
		"yx": "digital",
		"ym": "pulse",
		"yt": "regulate",
		"yk": "control",
		"sx": "attribute",
	}

	// 校验类型合法性
	tableName, ok := typeMap[ptype]
	if !ok {
		return nil, fmt.Errorf("不支持的测点类型: %s", ptype)
	}

	// 获取对应单表
	table, err := m.getSingleTable(tableName)
	if err != nil {
		return nil, fmt.Errorf("获取单表失败: %w", err)
	}

	// 定义结果列表
	result := make([]AgcBindPoint, 0)

	// 按类型遍历测点，转为 AgcBindPoint
	switch tableName {
	case "analog":
		for i := range table.Analogs {
			result = append(result, AgcBindPoint{PointID: table.Analogs[i].ID, Name: table.Analogs[i].Name, MqttKey: table.Analogs[i].MqttKey})
		}
	case "digital":
		for i := range table.Digitals {
			result = append(result, AgcBindPoint{PointID: table.Digitals[i].ID, Name: table.Digitals[i].Name, MqttKey: table.Digitals[i].MqttKey})
		}
	case "pulse":
		for i := range table.Pulses {
			result = append(result, AgcBindPoint{PointID: table.Pulses[i].ID, Name: table.Pulses[i].Name, MqttKey: table.Pulses[i].MqttKey})
		}
	case "regulate":
		for i := range table.Regulates {
			result = append(result, AgcBindPoint{PointID: table.Regulates[i].ID, Name: table.Regulates[i].Name, MqttKey: table.Regulates[i].MqttKey})
		}
	case "control":
		for i := range table.Controls {
			result = append(result, AgcBindPoint{PointID: table.Controls[i].ID, Name: table.Controls[i].Name, MqttKey: table.Controls[i].MqttKey})
		}
	case "attribute":
		for i := range table.Attributes {
			result = append(result, AgcBindPoint{PointID: table.Attributes[i].ID, Name: table.Attributes[i].Name, MqttKey: table.Attributes[i].MqttKey})
		}
	}

	return result, nil
}

// GetPointFormulaFromDevice 获取指定测点的公式信息
func (m *DbConfigXMLManager) GetPointFormulaFromDevice(rtuID, deviceID, pointID string) (string, error) {
	// 加锁保证并发安全
	m.mu.Lock()
	defer m.mu.Unlock()

	// 调用内部无锁方法获取设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return "", err
	}

	// 分别在遥测、遥信、遥脉下查找关联的公式
	for i := range dev.YC.Analogs {
		if dev.YC.Analogs[i].ID == pointID {
			return dev.YC.Analogs[i].Formula, nil
		}
	}

	for i := range dev.YX.Digitals {
		if dev.YX.Digitals[i].ID == pointID {
			return dev.YX.Digitals[i].Formula, nil
		}
	}

	for i := range dev.YM.Pulses {
		if dev.YM.Pulses[i].ID == pointID {
			return dev.YM.Pulses[i].Formula, nil
		}
	}

	return "", fmt.Errorf("rtu:[%s], device:[%s], point:[%s] not found", rtuID, deviceID, pointID)
}

// -------------------------- 测点下公式的CRUD --------------------------

// deleteDeviceFormulaAndFactorIDs 删除指定Device下关联公式的ID + 关联的公式因子ID（仅返回ID）
// 参数：rtuID RTUID，deviceID 设备ID
// 返回：仅包含ID的集合、错误信息
func (m *DbConfigXMLManager) deleteDeviceFormulaAndFactorIDs(dev *Device, points RtuDevPoints) error {
	// 1. 获取公式表、公式因子表
	formulaTable, err := m.getSingleTable("formula")
	if err != nil {
		return err
	}
	formulaFTable, err := m.getSingleTable("formula_factor")
	if err != nil {
		return err
	}

	// 2. 收集设备下所有测点的非空FormulaID
	formulaIDSet := make(map[string]struct{})
	for i := range points.Analogs { // 遥测
		if points.Analogs[i].Formula != "" {
			formulaIDSet[points.Analogs[i].Formula] = struct{}{}
		}
	}
	for i := range points.Digitals { // 遥信
		if points.Digitals[i].Formula != "" {
			formulaIDSet[points.Digitals[i].Formula] = struct{}{}
		}
	}
	for i := range points.Pulses { // 遥脉
		if points.Pulses[i].Formula != "" {
			formulaIDSet[points.Pulses[i].Formula] = struct{}{}
		}
	}
	// 删除单表中关联的公式
	deleteSliceItemByIDs(&formulaTable.Formulas, formulaIDSet)

	// 收集所有关联的公式因子ID
	factorIDSet := make(map[string]struct{})
	for i := range formulaFTable.FormulaFactors {
		// 只保留属于当前设备绑定公式的因子ID
		if _, exists := formulaIDSet[formulaFTable.FormulaFactors[i].MemofFormula]; exists {
			factorIDSet[formulaFTable.FormulaFactors[i].ID] = struct{}{}
		}
	}
	// 删除单表中关联的公式因子
	deleteSliceItemByIDs(&formulaFTable.FormulaFactors, factorIDSet)

	return nil
}

// findFormulaByID 根据公式ID在公式表中精准查找公式
func findFormulaByID(formulaTable *Table, formulaID string) (*Formula, bool) {
	// 用索引遍历，直接取切片原元素的地址
	for i := range formulaTable.Formulas {
		item := formulaTable.Formulas[i]
		if item.ID == formulaID {
			return item, true
		}
	}
	return nil, false
}

// GetPointFormula 根据公式ID获取公式信息及关联的公式因子
// 参数：
//
//	formulaID  公式唯一ID
//
// 返回：
//
//	Formula      匹配到的公式信息
//	[]FormulaFactor 该公式关联的所有公式因子
//	error        获取失败的错误信息（公式不存在/表获取失败）
func (m *DbConfigXMLManager) GetPointFormula(formulaID string) (Formula, []FormulaFactor, error) {
	// 加锁保证并发安全
	m.mu.Lock()
	defer m.mu.Unlock()

	// 空ID直接返回空结果，不报错
	if formulaID == "" {
		log.Warn("GetPointFormula: formulaID is empty, return empty result")
		return Formula{}, []FormulaFactor{}, nil
	}

	// 1. 获取公式表并查找对应公式
	formulaTable, err := m.getSingleTable("formula")
	if err != nil {
		return Formula{}, nil, fmt.Errorf("获取公式表失败: %w", err)
	}

	targetFormulaPtr, found := findFormulaByID(formulaTable, formulaID)
	if !found {
		return Formula{}, []FormulaFactor{}, fmt.Errorf("公式ID[%s]不存在", formulaID)
	}

	// 2. 获取公式因子表，筛选关联当前公式的因子
	factorTable, err := m.getSingleTable("formula_factor")
	if err != nil {
		return Formula{}, nil, fmt.Errorf("获取公式因子表失败: %w", err)
	}

	var targetFactors []FormulaFactor
	for i := range factorTable.FormulaFactors {
		if factorTable.FormulaFactors[i].MemofFormula == formulaID {
			targetFactors = append(targetFactors, *factorTable.FormulaFactors[i])
		}
	}

	// 3. 返回结果（找到公式，因子可为空）
	log.Infof("GetPointFormula: get formula[%s] success, factor count: %d", formulaID, len(targetFactors))
	return *targetFormulaPtr, targetFactors, nil
}

// setFormulaByID 按pointID匹配切片元素，并设置Formula字段
func setFormulaByID[T FormulaPointer](points []T, pointID string, formulaID string) bool {
	for i := range points {
		id := points[i].GetID()
		if id == pointID {
			points[i].SetFormula(formulaID)
			return true
		}
	}
	return false
}

// updatePointFormulaByID 按pointID同步更新Hierarchy和Single表中测点的Formula字段
func updatePointFormulaByID(m *DbConfigXMLManager, dev *Device, pointID, formulaID string) {
	if setFormulaByID(dev.YC.Analogs, pointID, formulaID) == true {
		table, _ := m.getSingleTable("analog")
		setFormulaByID(table.Analogs, pointID, formulaID)
	} else if setFormulaByID(dev.YX.Digitals, pointID, formulaID) == true {
		table, _ := m.getSingleTable("digital")
		setFormulaByID(table.Digitals, pointID, formulaID)
	} else if setFormulaByID(dev.YM.Pulses, pointID, formulaID) == true {
		table, _ := m.getSingleTable("pulse")
		setFormulaByID(table.Pulses, pointID, formulaID)
	}
}

// UpdatePointFormula 更新测点绑定的公式（新增/更新公式 + 同步公式因子 + 双表更新测点Formula字段）
func (m *DbConfigXMLManager) UpdatePointFormula(rtuID, deviceID, pointID string, formula Formula, formulaFactors []FormulaFactor) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 获取公式表、公式因子表
	formulaTable, err := m.getSingleTable("formula")
	if err != nil {
		return err
	}
	formulaFTable, err := m.getSingleTable("formula_factor")
	if err != nil {
		return err
	}

	// 2. 内部无锁获取设备
	dev, err := m.getDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		return err
	}

	if formula.ID == "" { // ====================== 【新增公式】 ======================
		log.Info("UpdatePointFormula add new formula")
		// 生成新公式ID并添加到公式表
		m.formulaID++
		formula.ID = fmt.Sprintf("%s-%s-%s-%d", rtuID, "0", formulaTable.TableNo, m.formulaID)
		formulaTable.Formulas = append(formulaTable.Formulas, &formula)

		// 生成新公式因子ID并添加到公式因子表
		for idx := range formulaFactors {
			m.formulaFactorID++
			formulaFactors[idx].MemofFormula = formula.ID
			formulaFactors[idx].ID = fmt.Sprintf("%s-%s-%s-%d", rtuID, "0", formulaFTable.TableNo, m.formulaFactorID)
			formulaFTable.FormulaFactors = append(formulaFTable.FormulaFactors, &formulaFactors[idx])
		}

		// 同步更新【Hierarchy + Single】双表中对应pointID的Formula字段为新公式ID
		updatePointFormulaByID(m, dev, pointID, formula.ID)

	} else { // ====================== 【更新公式】 ======================
		log.Info("UpdatePointFormula update formula:", formula.ID)

		// 查找并更新公式
		targetFormulaPtr, found := findFormulaByID(formulaTable, formula.ID)
		if !found {
			return fmt.Errorf("公式ID[%s]不存在", formula.ID)
		}
		*targetFormulaPtr = formula

		// 删除旧公式因子
		keepIdx := 0
		for i := range formulaFTable.FormulaFactors {
			if formulaFTable.FormulaFactors[i].MemofFormula != formula.ID {
				formulaFTable.FormulaFactors[keepIdx] = formulaFTable.FormulaFactors[i]
				keepIdx++
			}
		}
		formulaFTable.FormulaFactors = formulaFTable.FormulaFactors[:keepIdx]

		// 添加新公式因子
		for i := range formulaFactors {
			if formulaFactors[i].ID == "" {
				formulaFactors[i].MemofFormula = formula.ID
				m.formulaFactorID++
				formulaFactors[i].ID = fmt.Sprintf("%s-%s-%s-%d", rtuID, "0", formulaFTable.TableNo, m.formulaFactorID)
			}
			formulaFTable.FormulaFactors = append(formulaFTable.FormulaFactors, &formulaFactors[i])
			log.Info("UpdatePointFormula append formulaFactors:", formulaFactors[i])
		}
	}

	// 3. 保存XML
	return m.save()
}

// -------------------------- 转发RTU的CRUD --------------------------

// handleTransPointSlice 过滤原始设备中所有Trans字段为1的测点赋值给新的转发设备
func handleTransPointSlice[T Pointer](m *DbConfigXMLManager, rtuID, deviceID string, tableName string, sourcePoints *[]T, newPoints []T) {
	// 1. 获取对应单表
	table, err := m.getSingleTable(tableName)
	if err != nil {
		return
	}

	// 2. 创建空切片，用于储存Trans为1的测点信息
	var transPoints []T
	for i := range newPoints {
		trans := newPoints[i].GetTrans()
		if trans == "1" {
			clonePoint := newPoints[i].Clone().(T)
			newID := fmt.Sprintf("%s-%s-%s-%s", rtuID, deviceID, table.TableNo, clonePoint.GetGin())
			clonePoint.SetID(newID)
			clonePoint.SetDataSource(newPoints[i].GetID())
			transPoints = append(transPoints, clonePoint)
		}
	}

	if len(transPoints) == 0 {
		return
	}

	*sourcePoints = append(*sourcePoints, transPoints...)
	table.addItemsToSingleTable(transPoints)
}

// 添加转发设备，srcRtuID：待转发设备所属的rtuID，dstRtuID：待添加设备的RTUID，deviceID：待转发的设备ID
func (m *DbConfigXMLManager) AddTransDeviceToRTU(sourceRtuID, sourceDeviceID string, device *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 不需要绑定转发设备
	if sourceRtuID == "-1" && sourceDeviceID == "-1" {
		rtuID := device.MemOfRtu
		return m.addDeviceToRTU(rtuID, device)
	}

	// 从层次结构中获取待转发device的信息
	device.SourceDevice = sourceDeviceID
	sourceDevice, err := m.getDeviceFromRTU(sourceRtuID, sourceDeviceID)
	if err != nil {
		return err
	}

	// 添加转发设备
	rtuID := device.MemOfRtu
	if err = m.addDeviceToRTU(rtuID, device); err != nil {
		return err
	}

	handleTransPointSlice(m, rtuID, device.ID, transTableName(rtuID, "analog"), &device.YC.Analogs, sourceDevice.YC.Analogs)          // 【遥测 Analog】
	handleTransPointSlice(m, rtuID, device.ID, transTableName(rtuID, "digital"), &device.YX.Digitals, sourceDevice.YX.Digitals)       // 【遥信 Digital】
	handleTransPointSlice(m, rtuID, device.ID, transTableName(rtuID, "pulse"), &device.YM.Pulses, sourceDevice.YM.Pulses)             // 【遥脉 Pulses】
	handleTransPointSlice(m, rtuID, device.ID, transTableName(rtuID, "regulate"), &device.YT.Regulates, sourceDevice.YT.Regulates)    // 【遥调 Regulates】
	handleTransPointSlice(m, rtuID, device.ID, transTableName(rtuID, "control"), &device.YK.Controls, sourceDevice.YK.Controls)       // 【遥控 Controls】
	handleTransPointSlice(m, rtuID, device.ID, transTableName(rtuID, "attribute"), &device.SX.Attributes, sourceDevice.SX.Attributes) // 【属性 Attributes】

	// 保存xml
	return m.save()
}

// getPointPaths 通用提取测点Path列表
func getPointPaths[T Pointer](points []T) map[string]struct{} {
	ids := make(map[string]struct{}, len(points))
	for i := range points {
		id := points[i].GetPath()
		ids[id] = struct{}{}
	}
	return ids
}

// GetAgcRtuPointsName 获取所有测点的路径信息
func (m *DbConfigXMLManager) GetAgcRtuPointsName(pointType string) (map[string]struct{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rtu, err := m.getRTUFromHierarchy("700")
	if err != nil {
		return nil, err
	}
	if len(rtu.Devices) != 1 {
		return nil, fmt.Errorf("agc rtu下的设备数量不为1")
	}
	device := rtu.Devices[0]

	allPoints := make(map[string]struct{})
	if pointType == "all" || pointType == "ANALOG" {
		mergeMaps(allPoints, getPointPaths(device.YC.Analogs))
	}
	if pointType == "all" || pointType == "DIGITAL" {
		mergeMaps(allPoints, getPointPaths(device.YX.Digitals))
	}
	if pointType == "all" || pointType == "PLUSE" {
		mergeMaps(allPoints, getPointPaths(device.YM.Pulses))
	}
	if pointType == "all" || pointType == "CONTROL" {
		mergeMaps(allPoints, getPointPaths(device.YK.Controls))
	}
	if pointType == "all" || pointType == "REGULATE" {
		mergeMaps(allPoints, getPointPaths(device.YT.Regulates))
	}
	if pointType == "all" || pointType == "ATTRIBUTE" {
		mergeMaps(allPoints, getPointPaths(device.SX.Attributes))
	}

	return allPoints, nil
}

type GetTransRtuPointsReq struct {
	SourceRtuID string `json:"source_rtu_id"`
	SourceDevID string `json:"source_dev_id"`
	RtuID       string `json:"rtu_id"`
	DevID       string `json:"dev_id"`
	DataType    string `json:"data_type"`
}

// GetTransPointsByType 获取指定设备下所有可以添加的转发点
func (m *DbConfigXMLManager) GetTransPointsByType(req GetTransRtuPointsReq) (RtuDevPoints, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var points RtuDevPoints
	sourceDev, err := m.getDeviceFromRTU(req.SourceRtuID, req.SourceDevID)
	if err != nil {
		return points, err
	}
	dev, err := m.getDeviceFromRTU(req.RtuID, req.DevID)
	if err != nil {
		return points, err
	}

	switch req.DataType {
	case "analog":
		exceptIDs := getDataSources(dev.YC.Analogs)
		points.Analogs = GetTransPoints(sourceDev.YC.Analogs, exceptIDs)
	case "digital":
		exceptIDs := getDataSources(dev.YX.Digitals)
		points.Digitals = GetTransPoints(sourceDev.YX.Digitals, exceptIDs)
	case "pulse":
		exceptIDs := getDataSources(dev.YM.Pulses)
		points.Pulses = GetTransPoints(sourceDev.YM.Pulses, exceptIDs)
	case "regulate":
		exceptIDs := getDataSources(dev.YT.Regulates)
		points.Regulates = GetTransPoints(sourceDev.YT.Regulates, exceptIDs)
	case "control":
		exceptIDs := getDataSources(dev.YK.Controls)
		points.Controls = GetTransPoints(sourceDev.YK.Controls, exceptIDs)
	case "attribute":
		exceptIDs := getDataSources(dev.SX.Attributes)
		points.Attributes = GetTransPoints(sourceDev.SX.Attributes, exceptIDs)
	default:
		return points, fmt.Errorf("invalid dataType : %s", req.DataType)
	}
	return points, nil
}
