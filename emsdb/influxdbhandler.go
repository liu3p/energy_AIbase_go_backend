package emsdb

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

const (
	org    = "sfere"
	bucket = "history_data"
	token  = "pX_zsLVHUaRFbKKN6jcgz9PE7UW00QO-7pD7d-UUBhn1THj-eiMXW0ZoweFiSo7oq1chTZbDg-m-LN02Bdl3OA=="
	url    = "http://127.0.0.1:8086"
)

var (
	once         sync.Once
	influxClient influxdb2.Client
	queryAPI     api.QueryAPI
	initErr      error
)

func initInfluxDB() {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	options := influxdb2.DefaultOptions().SetHTTPClient(httpClient)
	influxClient = influxdb2.NewClientWithOptions(url, token, options)
	if influxClient == nil {
		initErr = fmt.Errorf("failed to create influxdb client")
		return
	}
	queryAPI = influxClient.QueryAPI(org)
}

func GetQueryAPI() (api.QueryAPI, error) {
	once.Do(initInfluxDB)
	if initErr != nil {
		return nil, initErr
	}
	return queryAPI, nil
}

func Close() {
	if influxClient != nil {
		influxClient.Close()
	}
}

// buildFluxQuery 构建 Flux 查询语句
func buildFluxQuery(ids []string, start, stop time.Time) string {
	quotedIds := make([]string, len(ids))
	for i, id := range ids {
		quotedIds[i] = fmt.Sprintf("%q", id)
	}
	idListStr := "[" + strings.Join(quotedIds, ", ") + "]"
	startStr := start.Format(time.RFC3339)
	stopStr := stop.Format(time.RFC3339)

	return fmt.Sprintf(`
        from(bucket: "history_data")
            |> range(start: %s, stop: %s)
            |> filter(fn: (r) => r._measurement == "analog")
            |> filter(fn: (r) => r._field == "value")
            |> filter(fn: (r) => contains(value: r.id, set: %s))
    `, startStr, stopStr, idListStr)
}

// queryAndCollectData 执行查询，收集数据到中间结构
// 返回：时间点->(ID->值)映射，ID集合，时间集合
func queryAndCollectData(qAPI api.QueryAPI, fluxQuery string) (map[time.Time]map[string]float64, map[string]struct{}, map[time.Time]struct{}, error) {
	result, err := qAPI.Query(context.Background(), fluxQuery)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer result.Close()

	timeValueMap := make(map[time.Time]map[string]float64)
	idSet := make(map[string]struct{})
	timeSet := make(map[time.Time]struct{})

	for result.Next() {
		record := result.Record()
		idVal := record.ValueByKey("id")
		idStr, ok := idVal.(string)
		if !ok {
			continue
		}
		val := record.Value()
		floatVal, ok := val.(float64)
		if !ok {
			continue
		}
		t := record.Time()

		idSet[idStr] = struct{}{}
		timeSet[t] = struct{}{}

		if _, exists := timeValueMap[t]; !exists {
			timeValueMap[t] = make(map[string]float64)
		}
		timeValueMap[t][idStr] = floatVal
	}
	if result.Err() != nil {
		return nil, nil, nil, fmt.Errorf("查询结果遍历出错: %v", result.Err())
	}
	return timeValueMap, idSet, timeSet, nil
}

// writeCSV 将收集的数据写入 CSV 文件
func writeCSV(filename string, timeValueMap map[time.Time]map[string]float64, idSet map[string]struct{}, timeSet map[time.Time]struct{}) error {
	if len(timeValueMap) == 0 {
		return fmt.Errorf("没有数据可导出")
	}

	// 排序 ID
	allIDs := make([]string, 0, len(idSet))
	for id := range idSet {
		allIDs = append(allIDs, id)
	}
	sort.Strings(allIDs)

	// 排序时间
	allTimes := make([]time.Time, 0, len(timeSet))
	for t := range timeSet {
		allTimes = append(allTimes, t)
	}
	sort.Slice(allTimes, func(i, j int) bool { return allTimes[i].Before(allTimes[j]) })

	// 创建文件
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()
	// 写入 UTF-8 BOM，解决 Excel 中文乱码
	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("写入 BOM 失败: %v", err)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	header := append([]string{"时间"}, allIDs...)
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("写入表头失败: %v", err)
	}

	// 写入数据行
	for _, t := range allTimes {
		row := []string{t.Format(time.RFC3339)}
		inner := timeValueMap[t]
		for _, id := range allIDs {
			if val, ok := inner[id]; ok {
				row = append(row, fmt.Sprintf("%f", val))
			} else {
				row = append(row, "")
			}
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("写入数据行失败: %v", err)
		}
	}
	return nil
}

// ExportAnalogDataToCSV 导出历史数据到 CSV 文件
func ExportHistoryDataToCSV(ids []string, start, stop time.Time, filename string) error {
	// 构建influxDB查询语句
	fluxQuery := buildFluxQuery(ids, start, stop)

	// 调用api查询数据
	qAPI, err := GetQueryAPI()
	if err != nil {
		return err
	}
	timeValueMap, idSet, timeSet, err := queryAndCollectData(qAPI, fluxQuery)
	if err != nil {
		return err
	}

	// 写csv文件
	return writeCSV(filename, timeValueMap, idSet, timeSet)
}

type ExportHistoryReq struct {
	IDs       []string  `json:"ids"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// IntervalAll 特殊的 interval 取值：不做时间插值，返回时间区间内的全部原始数据
const IntervalAll = -1

// QueryHistoryReq 查询历史数据请求参数
type QueryHistoryReq struct {
	Type      string    `json:"type"`       // 测点类型：digital遥信、analog遥测、pulse遥脉
	IDs       []string  `json:"ids"`        // 测点ID数组
	StartTime time.Time `json:"start_time"` // 开始时间
	EndTime   time.Time `json:"end_time"`   // 结束时间
	Interval  int       `json:"interval"`   // 时间间隔（秒），最小300秒；传 -1(IntervalAll) 表示不插值，返回区间内全部原始数据
}

// parsePointType 将字符串类型映射为整数类型
func parsePointType(typeStr string) (int, error) {
	switch typeStr {
	case "digital":
		return 101, nil
	case "analog":
		return 102, nil
	case "pulse":
		return 105, nil
	default:
		return 0, fmt.Errorf("参数 Type 无效，仅支持 digital(遥信)、analog(遥测)、pulse(遥脉)")
	}
}

// QueryHistoryDataItem 单个时间点的数据
type QueryHistoryDataItem struct {
	Time time.Time              `json:"time"`
	Data map[string]interface{} `json:"data"`
}

// QueryHistoryResp 查询历史数据响应
type QueryHistoryResp []QueryHistoryDataItem

// validatePointIDs 验证测点ID格式和类型匹配
// 测点ID格式：xxx-xxxxx-xxx-xxx，第三段数字需与type匹配
func validatePointIDs(ids []string, pointType int) error {
	typeStr := fmt.Sprintf("%d", pointType)
	for _, id := range ids {
		parts := strings.Split(id, "-")
		if len(parts) < 4 {
			return fmt.Errorf("测点ID格式错误: %s，正确格式应为 xxx-xxxxx-xxx-xxx", id)
		}
		if parts[2] != typeStr {
			return fmt.Errorf("测点ID %s 的类型 %s 与请求的测点类型 %d 不匹配", id, parts[2], pointType)
		}
	}
	return nil
}

// buildIDFilter 构建 ID 过滤条件，使用 == 拼接
func buildIDFilter(ids []string) string {
	conditions := make([]string, len(ids))
	for i, id := range ids {
		conditions[i] = fmt.Sprintf(`r.id == "%s"`, id)
	}
	if len(conditions) == 1 {
		return conditions[0]
	}
	return "(" + strings.Join(conditions, " or ") + ")"
}

// buildFluxQueryWithInterpolation 构建带时间插值的Flux查询语句
func buildFluxQueryWithInterpolation(ids []string, measurement string, start, stop time.Time, interval int) string {
	startStr := start.Format(time.RFC3339)
	stopStr := stop.Format(time.RFC3339)
	intervalStr := fmt.Sprintf("%ds", interval)
	idFilter := buildIDFilter(ids)

	// 遥信使用 last（取窗口最后状态），遥测/遥脉使用 mean（取平均值）
	aggFn := "mean"
	if measurement == "digital" {
		aggFn = "last"
	}

	return fmt.Sprintf(`
		from(bucket: "history_data")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "%s")
			|> filter(fn: (r) => r._field == "value")
			|> filter(fn: (r) => %s)
			|> aggregateWindow(every: %s, fn: %s, createEmpty: false)
			|> fill(usePrevious: true)
			|> yield(name: "result")
	`, startStr, stopStr, measurement, idFilter, intervalStr, aggFn)
}

// QueryHistoryData 查询历史数据，支持时间插值
func QueryHistoryData(req *QueryHistoryReq) (QueryHistoryResp, error) {
	// 校验必填参数
	pointType, err := parsePointType(req.Type)
	if err != nil {
		return nil, err
	}
	if len(req.IDs) == 0 {
		return nil, fmt.Errorf("参数 IDs 缺失或为空")
	}
	if req.StartTime.IsZero() {
		return nil, fmt.Errorf("参数 StartTime 缺失或无效")
	}
	if req.EndTime.IsZero() {
		return nil, fmt.Errorf("参数 EndTime 缺失或无效")
	}
	if !req.StartTime.Before(req.EndTime) {
		return nil, fmt.Errorf("参数 StartTime 必须早于 EndTime")
	}

	// 参数校验：-1 表示"全部原始数据"，不参与最小 300 秒的归一
	if req.Interval != IntervalAll && req.Interval < 300 {
		req.Interval = 300 // 默认5分钟
	}

	// 验证测点ID格式
	if err := validatePointIDs(req.IDs, pointType); err != nil {
		return nil, err
	}

	qAPI, err := GetQueryAPI()
	if err != nil {
		return nil, err
	}

	var fluxQuery string
	if req.Interval == IntervalAll {
		// 不插值：直接返回区间内的原始记录（无窗口聚合、无补点）
		fluxQuery = buildFluxQueryRaw(req.IDs, req.Type, req.StartTime, req.EndTime)
	} else {
		fluxQuery = buildFluxQueryWithInterpolation(req.IDs, req.Type, req.StartTime, req.EndTime, req.Interval)
	}
	result, err := qAPI.Query(context.Background(), fluxQuery)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer result.Close()

	timeDataMap := make(map[time.Time]map[string]interface{})
	for result.Next() {
		record := result.Record()
		idVal := record.ValueByKey("id")
		idStr, ok := idVal.(string)
		if !ok {
			continue
		}

		t := record.Time()
		val := record.Value()

		if _, exists := timeDataMap[t]; !exists {
			timeDataMap[t] = make(map[string]interface{})
		}

		// 遥信结果取整，遥测/遥脉保留三位小数
		if pointType == 101 {
			if floatVal, ok := val.(float64); ok {
				timeDataMap[t][idStr] = int64(math.Round(floatVal))
			} else {
				timeDataMap[t][idStr] = val
			}
		} else if pointType == 102 || pointType == 105 {
			if floatVal, ok := val.(float64); ok {
				timeDataMap[t][idStr] = math.Round(floatVal*1000) / 1000
			} else {
				timeDataMap[t][idStr] = val
			}
		} else {
			timeDataMap[t][idStr] = val
		}
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("查询结果遍历出错: %v", result.Err())
	}

	// 转换为有序数组
	resp := make(QueryHistoryResp, 0, len(timeDataMap))
	times := make([]time.Time, 0, len(timeDataMap))
	for t := range timeDataMap {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

	for _, t := range times {
		resp = append(resp, QueryHistoryDataItem{
			Time: t,
			Data: timeDataMap[t],
		})
	}

	return resp, nil
}

// RawHistoryItem 一条原始历史数据记录
type RawHistoryItem struct {
	Time  time.Time
	ID    string
	Value interface{}
}

// buildFluxQueryRaw 构建不插值的 Flux 查询语句：直接返回原始记录，不做窗口聚合与补点
// 与 buildFluxQueryWithInterpolation 的唯一区别是省掉了 aggregateWindow / fill / yield
// （省掉显式 yield 是安全的：Flux 会补一个隐式 yield，客户端按顺序读表、不依赖名字）
// start/stop 原样格式化，不做时区转换（与 /his/query 一致）
func buildFluxQueryRaw(ids []string, measurement string, start, stop time.Time) string {
	return fmt.Sprintf(`
		from(bucket: "history_data")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "%s")
			|> filter(fn: (r) => r._field == "value")
			|> filter(fn: (r) => %s)
	`, start.Format(time.RFC3339), stop.Format(time.RFC3339), measurement, buildIDFilter(ids))
}

// QueryRawHistoryData 查询单个测点某段时间的原始数据，不做插值、不做取整
func QueryRawHistoryData(measurement, id string, start, stop time.Time) ([]RawHistoryItem, error) {
	qAPI, err := GetQueryAPI()
	if err != nil {
		return nil, err
	}

	fluxQuery := buildFluxQueryRaw([]string{id}, measurement, start, stop)
	result, err := qAPI.Query(context.Background(), fluxQuery)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer result.Close()

	items := make([]RawHistoryItem, 0, 1024)
	for result.Next() {
		record := result.Record()
		items = append(items, RawHistoryItem{
			Time:  record.Time(),
			ID:    id,
			Value: record.Value(),
		})
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("查询结果遍历出错: %v", result.Err())
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Time.Before(items[j].Time) })
	return items, nil
}

// WriteRawHistoryCSV 将原始数据以长表格式(time,id,value)写入 w
// 时间列原样输出 UTC（record.Time() 即 UTC），不做时区转换
func WriteRawHistoryCSV(w io.Writer, items []RawHistoryItem) error {
	// 写入 UTF-8 BOM，解决 Excel 中文乱码
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("写入 BOM 失败: %v", err)
	}

	writer := csv.NewWriter(w)

	if err := writer.Write([]string{"时间", "测点ID", "值"}); err != nil {
		return fmt.Errorf("写入表头失败: %v", err)
	}

	for _, it := range items {
		// 用 RFC3339Nano 保留亚秒精度，不用 RFC3339（那会截断到秒）；
		// it.Time 已是 UTC，直接格式化即 Z 形式
		row := []string{it.Time.Format(time.RFC3339Nano), it.ID, formatRawValue(it.Value)}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("写入数据行失败: %v", err)
		}
	}

	// 显式 Flush 并检查错误：csv.Writer 是带缓冲的，
	// 若用 defer writer.Flush()，底层 writer 的写失败会被吞掉，函数照样 return nil
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("刷新 CSV 缓冲失败: %v", err)
	}
	return nil
}

// formatRawValue 原样输出值：float64 用最短精确表示，避免 %f 的补零/截断
func formatRawValue(v interface{}) string {
	if f, ok := v.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return fmt.Sprintf("%v", v)
}
