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
	Type       string    `json:"type"`        // 测点类型：digital遥信、analog遥测、pulse遥脉
	IDs        []string  `json:"ids"`         // 测点ID数组
	StartTime  time.Time `json:"start_time"`  // 开始时间
	EndTime    time.Time `json:"end_time"`    // 结束时间
	Interval   int       `json:"interval"`    // 时间间隔（秒），最小300秒；传 -1(IntervalAll) 表示不插值，返回区间内全部原始数据
	IncludeEnd *bool     `json:"include_end"` // 默认 true（未传等价于 true）；仅插值模式生效：若原始末点未体现在插值序列上（含末点的末尾窗口被 cutoff 剪掉），把原始末点补在响应末尾；显式 false 不追加；interval=-1 时无效
	Diff       bool      `json:"diff"`        // 是否做差：true 时每个返回点额外给出 diff（差值 = 本点值 − 该测点在更早时间点最近一次的已知值；该测点首次出现记 0）。插值/不插值两种模式均生效
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
	// Diff 为差值（仅请求 diff=true 时出现，key 集合与 Data 一致）：
	// diff[id] = 本点 data[id] − 该测点在更早时间点最近一次的已知值；该测点首次出现时为 0。
	// 空 map 经 omitempty 省略，因此 diff=false 时该字段不出现。
	Diff map[string]interface{} `json:"diff,omitempty"`
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
// 取值语义为"步进末值"而非窗口均值，且网格锚定到 start_time（g_k = start + k*interval）：
//   - 锚定 offset：offset ≡ start+1ns (mod every) 时窗口边界落在 start+k*interval+1ns，
//     Go 侧统一 -1ns 即归一为网格点 start+k*interval；不再对齐 UTC epoch；
//   - offset 里的 +1ns 使窗口右闭：网格 g 的窗口实际覆盖 [g-Δ+1ns, g+1ns)（恰好含 g），
//     恰在网格点 g 上的上报会进入自身窗口，值立即显示在 g 网格（不再顺延下一格）；
//   - fn: last 统一取窗口内最后一条真实上报（遥信/遥测/遥脉一致）；
//   - createEmpty: false → 窗口内无上报则不产出行（空窗口跳过、不复用前值填充，保留真实缺口）；
//   - 时间归一（stop g+1ns → 精确网格 g）在 Go 侧收集循环里用 record.Time().Add(-1ns) 完成：
//     所有窗口 stop 统一是 g+1ns，整体回退 1ns 安全；不在 Flux 里做（map 内 time/duration
//     时间运算会被静态类型检查器报错，见 task 方案注意事项 8）。
//   - range 起点即 start（不再回看）：start_time 网格的值改由 queryFirstRawPoints 单独补入
//     （取区间内第一条上报），因此这里不需要、也不应把 range 提前到 start-interval。
//   - 时间用 RFC3339Nano：RFC3339 会截断亚秒，导致 range 起点与 offset 锚点不在同一瞬时。
func buildFluxQueryWithInterpolation(ids []string, measurement string, start, stop time.Time, interval int) string {
	// 锚定 offset：把窗口边界平移到 start 的相位上，+1ns 保持右闭
	everyNs := int64(interval) * int64(time.Second)
	offsetNs := start.UnixNano() % everyNs
	if offsetNs < 0 { // 防御 <1970 的负时间戳（Go % 为截断取余）
		offsetNs += everyNs
	}
	offsetNs = (offsetNs + 1) % everyNs

	idFilter := buildIDFilter(ids)
	return fmt.Sprintf(`
		from(bucket: "history_data")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "%s")
			|> filter(fn: (r) => r._field == "value")
			|> filter(fn: (r) => %s)
			|> aggregateWindow(every: %ds, offset: %dns, fn: last, createEmpty: false)
			|> yield(name: "result")
	`, start.Format(time.RFC3339Nano), stop.Format(time.RFC3339Nano), measurement, idFilter, interval, offsetNs)
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

	// include_end 尾点取数（仅插值模式且未显式关闭时执行）：查询每个测点在区间内的最后一条原始记录，
	// 供末尾追加复用（避免对主结果尾部再做一次步进求值）。在主查询之前执行；
	// 失败时静默降级（tl 保持零值 → 不追加），不因尾点缺失使主流程查询报错。
	var (
		tails []RawHistoryItem
		tl    time.Time // 零值表示"未取到尾点"：include_end 静默跳过
	)
	if req.IncludeEnd == nil || *req.IncludeEnd {
		if req.Interval != IntervalAll {
			var terr error
			tails, terr = queryLastRawPoints(req.Type, req.IDs, req.StartTime, req.EndTime)
			if terr == nil && len(tails) > 0 {
				// TL = 区间内所有测点原始记录的最大时间戳
				tl = tails[0].Time
				for _, it := range tails[1:] {
					if it.Time.After(tl) {
						tl = it.Time
					}
				}
			}
		}
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

	// 越界点阈值：请求区间末尾 与 当前时刻 中较早者。
	// aggregateWindow 的输出取窗口 _stop（边界锚定 start_time + k×interval），末尾残余窗口的
	// _stop 可能晚于请求的 end_time；当 end_time 接近或晚于当前时刻时，聚合点还可能落在"未来"。
	// 取两者较早值作为上界，晚于它的记录一律丢弃（阈值只算一次，保证同一响应内一致）。
	cutoff := req.EndTime
	if now := time.Now(); now.Before(cutoff) {
		cutoff = now
	}

	// 网格点时间上界再收紧到 InfluxDB 里最后一条真实原始记录的时刻 tl。
	// 网格点是窗口右沿，末尾窗口的右沿可能晚于该窗口内最后一条真实上报（例：末条上报 11:30、
	// 窗口右沿 11:32），该网格点承载的仍是"11:30 的旧值"而非新数据；留着它会让序列止于一个
	// 晚于真实末点的时刻。收紧后这些网格点被丢弃，末尾由下方 include_end 用真实末点（时间+值）收尾。
	// tl 仅在 include_end 生效且取到尾点时才非零；-1 模式下恒为零，此处不生效。
	gridUpper := cutoff
	if !tl.IsZero() && tl.Before(gridUpper) {
		gridUpper = tl
	}

	timeDataMap := make(map[time.Time]map[string]interface{})
	for result.Next() {
		record := result.Record()
		idVal := record.ValueByKey("id")
		idStr, ok := idVal.(string)
		if !ok {
			continue
		}

		// 统一用 UTC 作为 map key：time.Time 按 (wall, ext, loc) 结构相等比较，
		// 需与下方补入的 req.StartTime.UTC() 同 loc 才能正确合并。
		t := record.Time().UTC()
		// 插值模式下窗口 stop 统一为 g+1ns（offset = 锚定相位+1ns 的右闭窗口），在 Go 侧归一回到精确网格 g
		// （所有窗口右沿一致，整体回退 1ns 安全）。不能对 -1 模式做同样平移：原始记录是真实
		// 上报时刻、非窗口 stop。归一放在上界（gridUpper）过滤之前，保证网格恰等于上界时不误删。
		if req.Interval != IntervalAll {
			t = t.Add(-time.Nanosecond)
			// 网格已锚定 start_time，正常不会出现早于 start 的网格；此处仅作防御，
			// 保证响应起点不早于 start_time。
			if t.Before(req.StartTime) {
				continue
			}
		}
		// 晚于 gridUpper 的记录丢弃：在 cutoff（end_time 与 now 的较早者）基础上，插值模式还把上界
		// 收紧到末条真实上报时刻 tl（见上方说明），避免序列止于晚于真实末点的网格右沿。
		// 注意：不插值模式(range stop 为开区间)的原始记录天然满足 t < end_time，此处几乎不触发。
		if t.After(gridUpper) {
			continue
		}
		val := record.Value()
		// createEmpty:false 下窗口有数据才产出行，一般不会出现 null；保留防御性跳过
		if val == nil {
			continue
		}

		if _, exists := timeDataMap[t]; !exists {
			timeDataMap[t] = make(map[string]interface{})
		}
		timeDataMap[t][idStr] = roundHistoryValue(pointType, val)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("查询结果遍历出错: %v", result.Err())
	}

	// 首点补入（仅插值模式）：网格锚定 start_time，但 start_time 处未必有上报，
	// 用 queryFirstRawPoints 取区间内每个测点的第一条上报，落在网格 start_time 上，
	// 保证响应首点恒为 start_time、取值为"[start_time, end_time] 内第一条上报"。
	// 失败静默降级（首点缺失不阻断主流程）；merge 而非覆盖，避免丢掉聚合已放在该时刻的测点。
	// 整个区间落在未来时（start_time > cutoff）不补，交由下方 queryNearestBefore 处理。
	if req.Interval != IntervalAll && !req.StartTime.After(cutoff) {
		if firsts, ferr := queryFirstRawPoints(req.Type, req.IDs, req.StartTime, req.EndTime); ferr == nil && len(firsts) > 0 {
			key := req.StartTime.UTC()
			if timeDataMap[key] == nil {
				timeDataMap[key] = make(map[string]interface{}, len(firsts))
			}
			for _, it := range firsts {
				timeDataMap[key][it.ID] = roundHistoryValue(pointType, it.Value)
			}
		}
	}

	// 未来时间区间补点：插值模式下，当整个查询时段都落在未来（start_time > now）且无任何数据时，
	// 查询 start_time 之前最近的一条记录并补入响应，保证前端在"未来时间轴"上仍有参考点。
	if req.Interval != IntervalAll && len(timeDataMap) == 0 && req.StartTime.After(time.Now()) {
		nearestItems, nerr := queryNearestBefore(req.Type, req.IDs, req.StartTime)
		if nerr == nil && len(nearestItems) > 0 {
			// 取所有测点中时间最近的一条作为补点时刻
			nearestTime := nearestItems[0].Time
			for _, it := range nearestItems[1:] {
				if it.Time.After(nearestTime) {
					nearestTime = it.Time
				}
			}
			nearestData := make(map[string]interface{}, len(nearestItems))
			for _, it := range nearestItems {
				nearestData[it.ID] = roundHistoryValue(pointType, it.Value)
			}
			timeDataMap[nearestTime] = nearestData
		}
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

	// include_end：插值模式下用 InfluxDB 里最后一条真实原始记录（时间 tl + 各测点值）收尾，
	// 保证响应最后一个点是"库中存储的最后一个点"的时间与值，而不是晚于它的网格右沿。
	// 上方已把网格上界收紧到 tl（晚于 tl 的网格点被丢弃），故此处追加的 tl 必不早于最后一个
	// 网格点；恰等于时不重复追加（tl.After 为假）。
	// tl 只在 [（IncludeEnd==nil||true） 且 interval!=-1] 时非零，此处不再重判。
	// resp 可能为空（全部数据都落在被剪掉的末尾窗口）——此时同样追加（退化为仅 TL 一点）。
	// !tl.After(cutoff) 为防御性约束：正常数据 TL 天然 <= now 且 < end_time；仅当库中存在未来
	// 时间戳记录（设备时钟偏斜等）时 TL 才可能越过 cutoff，此时静默不追加。
	if !tl.IsZero() && !tl.After(cutoff) && (len(resp) == 0 || tl.After(resp[len(resp)-1].Time)) {
		data := make(map[string]interface{}, len(tails))
		for _, it := range tails {
			data[it.ID] = roundHistoryValue(pointType, it.Value)
		}
		resp = append(resp, QueryHistoryDataItem{Time: tl, Data: data})
	}

	// 做差（diff=true）：在最终有序序列上按测点计算，插值/不插值两种模式一致
	if req.Diff {
		applyDiff(resp, pointType)
	}

	// 输出时区与请求的 start_time 保持一致：用 start_time 的固定偏移重新序列化，JSON 里即带同样的
	// 偏移（start_time 给 +08:00 则返回 2026-09-30T23:55:00+08:00，而不是 UTC 的 ...Z）。
	// 刻意用 FixedZone(偏移) 而不是 start_time 解析出来的 Location：JSON 的 RFC3339 只允许数值偏移，
	// 而 Go 的 Parse 在偏移与服务器本地时区相同时会返回 Local，跨夏令时会让输出偏移与请求漂移；
	// 固定成请求当时的偏移可保证逐字一致。偏移为 0（请求给 Z）时不动，仍输出 Z，与旧行为一致。
	// 上面的过滤/排序/补点/做差全是瞬时比较，不受 Location 影响，这里只改序列化出来的时区表示。
	if _, offset := req.StartTime.Zone(); offset != 0 {
		loc := time.FixedZone("", offset)
		for i := range resp {
			resp[i].Time = resp[i].Time.In(loc)
		}
	}

	return resp, nil
}

// roundHistoryValue 按测点类型处理响应值精度：遥信取整，遥测/遥脉保留三位小数
// 供主流程收集循环与 include_end 尾点组装共用（行为等价于原内联分支）。
func roundHistoryValue(pointType int, val interface{}) interface{} {
	f, ok := val.(float64)
	if !ok {
		return val
	}
	if pointType == 101 {
		return int64(math.Round(f))
	}
	return math.Round(f*1000) / 1000 // 102 / 105
}

// applyDiff 就地计算差值：对最终有序序列 resp，逐点给出
// diff[id] = 本点 data[id] − 该测点在更早时间点最近一次的已知值
// （跨缺口沿用"最近的上一个已知值"，不因中间缺格而回退为 0）；该测点首次出现时记 0。
// 只对当前点 data 中出现的测点产出 diff，保持 diff 的 key 集合与 data 一致。
// 与 roundHistoryValue 同一套精度规则，插值/不插值两种模式共用。
func applyDiff(resp QueryHistoryResp, pointType int) {
	lastKnown := make(map[string]float64)
	for i := range resp {
		if len(resp[i].Data) == 0 {
			continue
		}
		diffs := make(map[string]interface{}, len(resp[i].Data))
		for id, v := range resp[i].Data {
			cur, ok := asFloat64(v)
			if !ok {
				continue
			}
			if prev, seen := lastKnown[id]; seen {
				diffs[id] = roundHistoryValue(pointType, cur-prev)
			} else {
				diffs[id] = roundHistoryValue(pointType, float64(0))
			}
			lastKnown[id] = cur
		}
		resp[i].Diff = diffs
	}
}

// asFloat64 把响应值（遥信为 int64、遥测/遥脉为 float64）统一转成 float64，供差值计算
func asFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// RawHistoryItem 一条原始历史数据记录
type RawHistoryItem struct {
	Time  time.Time
	ID    string
	Value interface{}
}

// buildFluxQueryRaw 构建不插值的 Flux 查询语句：直接返回原始记录，不做窗口聚合与补点
// 与 buildFluxQueryWithInterpolation 的区别是省掉了 aggregateWindow / map / yield
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

// queryLastRawPoints 查询每个测点在区间内的最后一条原始记录（用于 include_end 尾点）。
// 复用 RawHistoryItem（Time/ID/Value）。返回 ≤ len(ids) 行，按时间升序排序。
// range 开区间天然不会取到 stop 时刻本身；组键含 id，last 每个测点至多一行。
// 必须先 sort(_time) 再 last：Flux 的 last() 取的是"输入表内最后一行"，from() 的输出顺序
// 没有语言级的时间保证，直接 last 可能取到 _time 并非最大的那条（表现为末点时间早于库里真正的末条）。
func queryLastRawPoints(measurement string, ids []string, start, stop time.Time) ([]RawHistoryItem, error) {
	qAPI, err := GetQueryAPI()
	if err != nil {
		return nil, err
	}
	flux := fmt.Sprintf(`
		from(bucket: "history_data")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "%s")
			|> filter(fn: (r) => r._field == "value")
			|> filter(fn: (r) => %s)
			|> sort(columns: ["_time"])
			|> last(column: "_time")
	`, start.Format(time.RFC3339), stop.Format(time.RFC3339), measurement, buildIDFilter(ids))

	result, err := qAPI.Query(context.Background(), flux)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer result.Close()

	items := make([]RawHistoryItem, 0, len(ids))
	for result.Next() {
		record := result.Record()
		idStr, ok := record.ValueByKey("id").(string)
		if !ok {
			continue
		}
		items = append(items, RawHistoryItem{
			Time:  record.Time(),
			ID:    idStr,
			Value: record.Value(),
		})
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("查询结果遍历出错: %v", result.Err())
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Time.Before(items[j].Time) })
	return items, nil
}

// queryFirstRawPoints 查询每个测点在区间内的第一条原始记录（用于插值模式锚定在 start_time 的首点）。
// 先 sort(_time) 再 first：确保取到 _time 最小的那条（理由同 queryLastRawPoints）。
// 与 queryLastRawPoints 对称：last → first，组键含 id，每个测点至多一行，按时间升序排序。
// range 起点为 start 且取到 start 本身；早于 start 的记录不返回。
func queryFirstRawPoints(measurement string, ids []string, start, stop time.Time) ([]RawHistoryItem, error) {
	qAPI, err := GetQueryAPI()
	if err != nil {
		return nil, err
	}
	flux := fmt.Sprintf(`
		from(bucket: "history_data")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "%s")
			|> filter(fn: (r) => r._field == "value")
			|> filter(fn: (r) => %s)
			|> sort(columns: ["_time"])
			|> first(column: "_time")
	`, start.Format(time.RFC3339Nano), stop.Format(time.RFC3339Nano), measurement, buildIDFilter(ids))

	result, err := qAPI.Query(context.Background(), flux)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer result.Close()

	items := make([]RawHistoryItem, 0, len(ids))
	for result.Next() {
		record := result.Record()
		idStr, ok := record.ValueByKey("id").(string)
		if !ok {
			continue
		}
		items = append(items, RawHistoryItem{
			Time:  record.Time(),
			ID:    idStr,
			Value: record.Value(),
		})
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("查询结果遍历出错: %v", result.Err())
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Time.Before(items[j].Time) })
	return items, nil
}

// queryNearestBefore 查询每个测点在 stop 之前的最后一条记录（用于未来时间区间补点）。
// 先 sort(_time) 再 last：确保取到 _time 最大的那条（理由同 queryLastRawPoints）。
// 查询窗口 [stop - 30天, stop)，返回每个测点至多一行，按时间升序。
func queryNearestBefore(measurement string, ids []string, stop time.Time) ([]RawHistoryItem, error) {
	qAPI, err := GetQueryAPI()
	if err != nil {
		return nil, err
	}
	flux := fmt.Sprintf(`
		from(bucket: "history_data")
			|> range(start: %s, stop: %s)
			|> filter(fn: (r) => r._measurement == "%s")
			|> filter(fn: (r) => r._field == "value")
			|> filter(fn: (r) => %s)
			|> sort(columns: ["_time"])
			|> last(column: "_time")
	`, stop.Add(-30*24*time.Hour).Format(time.RFC3339), stop.Format(time.RFC3339), measurement, buildIDFilter(ids))

	result, err := qAPI.Query(context.Background(), flux)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer result.Close()

	items := make([]RawHistoryItem, 0, len(ids))
	for result.Next() {
		record := result.Record()
		idStr, ok := record.ValueByKey("id").(string)
		if !ok {
			continue
		}
		items = append(items, RawHistoryItem{
			Time:  record.Time(),
			ID:    idStr,
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
// loc 决定时间列呈现的时区（nil 表示保持 UTC）；只改序列化出来的偏移，不改时间瞬时。
// 时间列用 RFC3339 精确到秒：loc 为空输出 Z 形式，否则输出该时区的数值偏移（如 +08:00）
func WriteRawHistoryCSV(w io.Writer, items []RawHistoryItem, loc *time.Location) error {
	// 写入 UTF-8 BOM，解决 Excel 中文乱码
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("写入 BOM 失败: %v", err)
	}

	writer := csv.NewWriter(w)

	if err := writer.Write([]string{"时间", "测点ID", "值"}); err != nil {
		return fmt.Errorf("写入表头失败: %v", err)
	}

	for _, it := range items {
		// it.Time 来自 InfluxDB，恒为 UTC；In(loc) 只换表示时区，不移动瞬时。
		t := it.Time
		if loc != nil {
			t = t.In(loc)
		}
		row := []string{t.Format(time.RFC3339), it.ID, formatRawValue(it.Value)}
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
