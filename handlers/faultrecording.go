package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"webserver-arm/emsdb"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	FaultRecordingDataPath   = "/home/eco_ems/eco_ems/dat/wave"
	FaultRecordingConfigPath = "/home/eco_ems/eco_ems/etc/waveform/wave.json"
	FaultRecordingStatusPath = "/home/eco_ems/eco_ems/etc/waveform/config_save_flag.ini"
	RcdPointTypeYX           = 0
	RcdPointTypeYC           = 1

	saveFlagConfigModify = 1
	saveFlagConfigNoitfy = 2
)

var (
	recordingFileData *RecordingFileData = nil
)

type CurveFileInfo struct {
	Filename    string `json:"filename"`
	Signal      string `json:"signal"`
	StartTime   string `json:"starttime"`
	TriggerTime string `json:"triggertime"`
	EndTime     string `json:"endtime"`
	Period      int    `json:"period"`
}

type YCStatsInfo struct {
	Max  float32 `json:"max"`
	Min  float32 `json:"min"`
	Diff float32 `json:"diff"`
	Avg  float32 `json:"avg"`
}

type ChangeEntry map[string]string
type YXChangeInfo []ChangeEntry // [{"2025-10-10 10:12:56.236": "0->1"}, ...]

type RecordingFileData struct {
	*CurveFileInfo
	Keys       []string       `json:"keys"`
	Type       []int          `json:"type"`
	YCStats    []YCStatsInfo  `json:"ycinfo"`
	YXStats    []YXChangeInfo `json:"yxinfo"`
	Timestamps []string       `json:"timestamps"`
	Values     [][]float32    `json:"values"`
}

type RecordingYCInfo struct {
	Key string `json:"key"`
	YCStatsInfo
}

type RecordingYXInfo struct {
	Key    string       `json:"key"`
	Change YXChangeInfo `json:"change"`
}

type RecordingWaveInfo struct {
	Keys       []string    `json:"keys"`
	Type       []int       `json:"type"`
	Timestamps []string    `json:"timestamps"`
	Values     [][]float32 `json:"values"`
}

func parseRecordingFile(filePath string, allInfo bool) (*RecordingFileData, error) {
	fmt.Println("parseRecordingFile... filePath:", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("parseRecordingFile can not open : %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	lineNum := 0
	var keys []string
	var ycyxtypes []int
	triggerSignal := ""
	startTimeStr := ""
	triggerTimeStr := ""
	endTimeStr := ""
	intervalMs := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if lineNum == 1 {
			// line1, trigger signal
			lineStr := gbkStr2utf8Str(line)
			fmt.Printf("触发信号: %s\n", lineStr)
			triggerSignal = lineStr
		} else if lineNum == 2 {
			// line 2, keys
			lineStr := gbkStr2utf8Str(line)
			keys = strings.Split(lineStr, ",")
			fmt.Printf("检测到 %d 个数据项\n", len(keys))
		} else if lineNum == 3 {
			// line 3, types. 0 or 1
			ycyxtypes = make([]int, len(keys))
			typesline := strings.Split(line, ",")
			for i, t := range typesline {
				val, err := strconv.Atoi(t)
				if err != nil || (val != RcdPointTypeYX && val != RcdPointTypeYC) {
					return nil, fmt.Errorf("type value '%s' invalid, must be 0(yx) or 1(yc)", t)
				}
				ycyxtypes[i] = val
			}

		} else if lineNum == 4 {
			// line 4, start time, trigger time, end time, period
			// e.g. 2023-07-05 09:09:09.000,2023-07-05 09:09:09.000,2023-07-05 09:09:09.000,1000
			waveInfo := strings.Split(line, ",")
			startTimeStr = waveInfo[0]
			triggerTimeStr = waveInfo[1]
			endTimeStr = waveInfo[2]
			intervalMs, err = strconv.Atoi(waveInfo[3])
			if err != nil {
				return nil, fmt.Errorf("parseRecordingFile wave period invalid: %v", err)
			}

			break
		}
	}

	fileInfo := &CurveFileInfo{
		Filename:    gbkStr2utf8Str(filepath.Base(filePath)),
		Signal:      triggerSignal,
		StartTime:   startTimeStr,
		TriggerTime: triggerTimeStr,
		EndTime:     endTimeStr,
		Period:      intervalMs,
	}

	log.Info("parseRecordingFile parse CurveFileInfo done. FileName:", fileInfo.Filename,
		",Signal:", fileInfo.Signal, ",StartTime:", fileInfo.StartTime, ",TriggerTime:", fileInfo.TriggerTime,
		",EndTime:", fileInfo.EndTime, ",Period:", fileInfo.Period)
	if !allInfo {
		return &RecordingFileData{fileInfo, nil, nil, nil, nil, nil, nil}, nil
	}

	// start to read all data

	timestamps := make([]string, 0)
	ycinfo := make([]YCStatsInfo, len(keys))
	yxinfo := make([]YXChangeInfo, len(keys))
	values := make([][]float32, len(keys))
	for i := range values {
		values[i] = make([]float32, 0)
	}

	lineNum = 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		data := strings.Split(line, ",")
		timestamp := data[0] // first column is timestamp
		timestamps = append(timestamps, timestamp)

		for i := 1; i <= len(keys); i++ { // data start from second column
			valStr := data[i]

			val, err := strconv.ParseFloat(valStr, 32)
			if err != nil {
				log.Error("parseRecordingFile failed, file:", file.Name(),
					" line:", lineNum, " column:", i, " value:", valStr, ", error:", err.Error())
				val = 0
			}
			values[i-1] = append(values[i-1], float32(val))
		}
	}

	// analyze data
	for i := range keys {
		if ycyxtypes[i] == RcdPointTypeYX {
			yxinfo[i], err = analyzeYXChanges(timestamps, values[i])
			if err != nil {
				return nil, fmt.Errorf("analyzeYXChanges ERROR : %v", err)
			}
		} else { // yc
			ycinfo[i] = analyzeYCStats(values[i])
		}

	}

	result := &RecordingFileData{
		CurveFileInfo: fileInfo,
		Keys:          keys,
		Type:          ycyxtypes,
		YCStats:       ycinfo,
		YXStats:       yxinfo,
		Timestamps:    timestamps,
		Values:        values,
	}

	log.Info("parseRecordingFile parse all data done. FileName:", result.Filename,
		",Keys:", result.Keys, ",Type:", result.Type, ",YCStats:", result.YCStats, ",YXStats:", result.YXStats,
		",Timestamps:", result.Timestamps, ",Values:", result.Values)
	return result, nil
}

func roundToThreeDecimals(value float32) float32 {
	// 保留3位小数
	return float32(math.Round(float64(value)*1000)) / 1000
}
func analyzeYCStats(values []float32) YCStatsInfo {
	var stat YCStatsInfo
	if len(values) == 0 {
		return stat
	}

	// // 提取所有数值类型的值
	// var nums []float64
	// for _, v := range values {
	// 	nums = append(nums, val)
	// }

	// if len(nums) == 0 {
	// 	return stat
	// }

	// 计算最大值、最小值
	stat.Max = values[0]
	stat.Min = values[0]
	sum := float32(0.0)

	for _, num := range values {
		if num > stat.Max {
			stat.Max = num
		}
		if num < stat.Min {
			stat.Min = num
		}
		sum += num
	}

	// 计算差值和平均值
	stat.Diff = stat.Max - stat.Min
	stat.Avg = sum / float32(len(values))

	return stat
}

func analyzeYXChanges(timestamps []string, values []float32) (YXChangeInfo, error) {
	var result YXChangeInfo

	if len(values) == 0 {
		return nil, fmt.Errorf("values slice can not be empty")
	}

	if len(timestamps) != len(values) { // 时间戳数量与值数量不匹配
		return nil, fmt.Errorf("timestampes and values slices must have the same length")
	}

	firstVal := int(values[0])
	prevVal := firstVal
	for i := 1; i < len(values); i++ {
		currentVal := int(values[i])
		if currentVal != prevVal {
			changeStr := fmt.Sprintf("%d->%d", prevVal, currentVal)
			result = append(result, ChangeEntry{timestamps[i]: changeStr})
			prevVal = currentVal
		}
	}
	if len(result) == 0 { // no changes
		result = append(result, ChangeEntry{"-": fmt.Sprintf("%d", firstVal)})
	}

	return result, nil

	/*
		// 遍历每个点（第一层切片）
		for pointIdx, pointValues := range values {
			// 检查当前点的值数量是否与时间戳数量匹配
			if len(pointValues) != len(timestamps) {
				return nil, fmt.Errorf("点 %d 的值数量与时间戳数量不匹配", pointIdx)
			}

			//var changes []ChangeEntry
			var changes YXChangeInfo
			allSame := true
			firstVal := int(pointValues[0]) // 转换为int，因为值只能是0或1

			// 处理只有一个时间点的情况
			// if len(pointValues) == 1 {
			// 	changes = append(changes, ChangeEntry{"-": fmt.Sprintf("%d", firstVal)})
			// 	//results = append(results, ChangeResult{Change: changes})

			// 	result = append(result, changes)
			// 	continue
			// }

			// 记录前一个值，用于检测变化
			prevVal := firstVal

			// 从第二个时间点开始检查变化
			for i := 1; i < len(pointValues); i++ {
				currentVal := int(pointValues[i])

				// 检查值是否发生变化
				if currentVal != prevVal {
					// 记录变化信息
					changeStr := fmt.Sprintf("%d->%d", prevVal, currentVal)
					changes = append(changes, ChangeEntry{timestamps[i]: changeStr})

					prevVal = currentVal
					allSame = false
				}
			}

			if allSame {
				changes = append(changes, ChangeEntry{"-": fmt.Sprintf("%d", firstVal)})
			}

			//results = append(results, ChangeResult{Change: changes})
			result = append(result, changes)
			result[pointIdx] = changes
		}

		return result, nil //results, nil
	*/
}

func getCurveFilePath(filename string) (path string, err error) {
	// e.g.  BMS1h_2025-10-10_10_12_56.736.curve
	// e.g.  BMS1h_2025-10-10_10:12:56.736.curve
	parts := strings.Split(filename, "_")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid filename format: %s", filename)
	}
	dateStr := parts[1]
	return filepath.Join(FaultRecordingDataPath, dateStr, filename), nil
}

func refreshRecordingFileData(fileName string) error {
	if recordingFileData == nil || recordingFileData.Filename != fileName {
		recordingFilePath, err := getCurveFilePath(fileName)
		if err != nil {
			log.Error("refreshRecordingFileData getCurveFilePath failed:", err.Error())
			return err
		}
		gbkPath := utf8Str2gbkStr(recordingFilePath)
		recordingFileData, err = parseRecordingFile(gbkPath, true)
		if err != nil {
			log.Error("refreshRecordingFileData parseRecordingFile failed:", err.Error())
			return err
		}
	}
	return nil
}

// @Summary		查找录波文件
// @Description	查找已经生成的录波文件
// @ID				faultrecording_list
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			startdate		path		string				true	"开始日期"
// @Param			enddate			path		string				true	"结束日期"
// @Success		200				{object}	[]CurveFileInfo	"查询成功 [{\"filename\":\"BMS1h_2025-10-10_10_12_56.736.curve\", \"time\":\"2025-10-10 10:12:56.736\", \"signal\": \"BMS1h\"}]"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/list/{startdate}/{enddate} [get]
func FaultRecordingList(c *gin.Context) {
	startDateStr := c.Param("startdate")
	endDateStr := c.Param("enddate")
	log.Info("FaultRecordingList in... startdate:", startDateStr, " enddate:", endDateStr)

	// 解析日期参数
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		log.Error("FaultRecordingList startdate invalid:", startDateStr, "error", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest,
			Message: "invalid startdate"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		log.Error("FaultRecordingList enddate invalid:", endDateStr, "error", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest,
			Message: "invalid enddate"})
		return
	}

	// 确保起始日期不晚于结束日期
	if startDate.After(endDate) {
		log.Error("FaultRecordingList data invalid: startdate after enddate")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest,
			Message: "enddate must after startdate"})
		return
	}

	curvafile, err := findCurveFilesInDateRange(FaultRecordingDataPath, startDate, endDate)
	if err != nil {
		log.Error("FaultRecordingList findCurveFilesInDateRange failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "internal server error"})
		return
	}

	log.Info("FaultRecordingList result:", curvafile)
	c.JSON(http.StatusOK, curvafile)
}

func parseCurveFilename(filename string) (signal string, timeStr string, err error) {
	// e.g.  BMS1h_2025-10-10_10_12_56.736.curve
	parts := strings.Split(filename, "_")
	if len(parts) < 5 {
		return "", "", fmt.Errorf("invalid filename format: %s", filename)
	}
	signal = parts[0]
	timeStr = fmt.Sprintf("%s %s:%s:%s", parts[1], parts[2], parts[3], parts[4])
	return signal, timeStr, nil
}

func findCurveFilesInDateRange(rootPath string, startDate, endDate time.Time) ([]*CurveFileInfo, error) {
	var result []*CurveFileInfo

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, fmt.Errorf("can not read FaultRecordingPath: %s, err: %v", rootPath, err)
	}

	dateDirRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		if !dateDirRegex.MatchString(dirName) {
			log.Warn("findCurveFilesInDateRange ignore not data-format dir:", dirName)
			continue
		}

		dirDate, err := time.Parse("2006-01-02", dirName)
		if err != nil {
			log.Warn("findCurveFilesInDateRange ignore parse failed dir:", dirName)
			continue
		}

		if dirDate.Before(startDate) || dirDate.After(endDate) {
			continue
		}

		dirPath := filepath.Join(rootPath, dirName)
		files, err := os.ReadDir(dirPath)
		if err != nil {
			log.Warn("findCurveFilesInDateRange can not read dir:", dirPath, ", err:", err.Error())
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				log.Warn("findCurveFilesInDateRange ignore directory:", file.Name(), " in date dir:", dirName)
				continue
			}
			fileName := gbkStr2utf8Str(file.Name())
			if !strings.HasSuffix(fileName, ".curve") {
				log.Warn("findCurveFilesInDateRange ignore not curve file. gbk:", file.Name(), ", utf8:", fileName, " in date dir:", dirName)
				continue
			}
			// signalName, timeStr, err := parseCurveFilename(file.Name())
			// if err != nil {
			// 	log.Warn("findCurveFilesInDateRange ignore parse failed file:", file.Name(), " in date dir:", dirName)
			// 	continue
			// }

			filePath := filepath.Join(dirPath, file.Name())
			fileInfo, err := parseRecordingFile(filePath, false)
			if err != nil {
				log.Warn("findCurveFilesInDateRange parseRecordingFile failed:", err.Error(), ", filePath:", filePath)
				continue
			}
			// result = append(result, CurveFileInfo{
			// 	Filename: file.Name(),
			// 	Time:     timeStr,
			// 	Signal:   signalName,
			// })

			result = append(result, fileInfo.CurveFileInfo)
			log.Info("findCurveFilesInDateRange find file:", file.Name(), " in date dir:", dirName)
		}
	}

	return result, nil
}

// @Summary		查询遥测表格记录
// @Description	查询遥测表格记录
// @ID				faultrecording_yc
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			filename		path		string				true	"录波文件名"
// @Success		200				{object}	[]RecordingYCInfo	"查询成功 [{\"key\":\"A相电压\", \"max\": 200,\"min\":150, \"diff\": 50, \"avg\":170}]"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/recording/{filename}/yc [get]
func FaultRecordingFileYc(c *gin.Context) {
	fileName := c.Param("filename")
	log.Info("FaultRecordingFileYc in... filename:", fileName)

	if err := refreshRecordingFileData(fileName); err != nil {
		log.Error("FaultRecordingFileYc refreshRecordingFileData failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "internal server error"})
		return
	}

	result := make([]RecordingYCInfo, 0, len(recordingFileData.Keys))
	for i, key := range recordingFileData.Keys {
		if recordingFileData.Type[i] == RcdPointTypeYC {
			result = append(result, RecordingYCInfo{Key: key, YCStatsInfo: recordingFileData.YCStats[i]})
		}
	}

	log.Info("FaultRecordingFileYc result size:", len(result))
	c.JSON(http.StatusOK, result)
}

// @Summary		查询遥信表格记录
// @Description	查询遥信表格记录
// @ID				faultrecording_yx
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			filename		path		string				true	"录波文件名"
// @Success		200				{object}	[]RecordingYXInfo	"查询成功 [{"key":"开机状态","change":[{"2025-10-10 10:12:56.236":"0->1"}]}]"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/recording/{filename}/yx [get]
func FaultRecordingFileYx(c *gin.Context) {
	fileName := c.Param("filename")
	log.Info("FaultRecordingFileYx in... filename:", fileName)

	if err := refreshRecordingFileData(fileName); err != nil {
		log.Error("FaultRecordingFileYx refreshRecordingFileData failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "internal server error"})
		return
	}

	result := make([]RecordingYXInfo, 0, len(recordingFileData.Keys))
	for i, key := range recordingFileData.Keys {
		if recordingFileData.Type[i] == RcdPointTypeYX {
			result = append(result, RecordingYXInfo{Key: key, Change: recordingFileData.YXStats[i]})
		}
	}

	log.Info("FaultRecordingFileYx result size:", len(result))
	c.JSON(http.StatusOK, result)
}

/*
// parseDateFromFilename 从文件名解析时间
func parseDateFromFilename(filename string) (time.Time, error) {
	// 文件名格式: BMS1h_2025-10-10_10_12_56.736.curve
	// 去掉扩展名
	baseName := strings.TrimSuffix(filename, ".curve")

	// 分割各部分
	parts := strings.Split(baseName, "_")
	if len(parts) < 5 {
		return time.Time{}, fmt.Errorf("invalid filename format: %s", filename)
	}

	// 组合时间字符串: 2025-10-10 10:12:56.736
	timeStr := fmt.Sprintf("%s %s:%s:%s", parts[1], parts[2], parts[3], parts[4])

	// 解析时间
	return time.Parse("2006-01-02 15:04:05.000", timeStr)
}

// getSignalFromFilename 从文件名提取信号名
func getSignalFromFilename(filename string) string {
	// 第一个下划线之前的部分就是信号名
	parts := strings.Split(filename, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func findCurveFilesInDateRange2(rootPath string, startDate, endDate time.Time) ([]CurveFileInfo, error) {
	var result []CurveFileInfo

	// 读取根目录下的所有子目录
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取目录 %s: %v", rootPath, err)
	}

	// 正则表达式用于匹配日期格式的目录名 (YYYY-MM-DD)
	dateDirRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

	// 遍历每个子目录
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // 只处理目录
		}

		dirName := entry.Name()
		// 检查目录名是否为日期格式
		if !dateDirRegex.MatchString(dirName) {
			continue
		}

		// 解析目录日期
		dirDate, err := time.Parse("2006-01-02", dirName)
		if err != nil {
			continue // 日期格式不正确，跳过
		}

		// 检查是否在日期范围内（包含起止日期）
		if dirDate.Before(startDate) || dirDate.After(endDate) {
			continue
		}

		// 处理该日期目录下的所有curve文件
		dirPath := filepath.Join(rootPath, dirName)
		files, err := os.ReadDir(dirPath)
		if err != nil {
			fmt.Printf("警告: 无法读取目录 %s: %v\n", dirPath, err)
			continue
		}

		// 正则表达式用于解析curve文件名
		// 格式: [信号名]_[日期]_[小时]_[分钟]_[秒].[毫秒].curve
		fileRegex := regexp.MustCompile(`^(.+)_(\d{4}-\d{2}-\d{2})_(\d{2})_(\d{2})_(\d{2})\.(\d{3})\.curve$`)

		for _, file := range files {
			if file.IsDir() {
				continue // 只处理文件
			}

			filename := file.Name()
			// 检查文件是否匹配curve文件格式
			matches := fileRegex.FindStringSubmatch(filename)
			if len(matches) != 7 {
				continue
			}

			// 提取文件名各部分
			signal := matches[1]
			datePart := matches[2]
			hour := matches[3]
			minute := matches[4]
			second := matches[5]
			millisecond := matches[6]

			// 构建时间字符串
			timeStr := fmt.Sprintf("%s %s:%s:%s.%s", datePart, hour, minute, second, millisecond)

			// 添加到结果列表
			result = append(result, CurveFileInfo{
				Filename: filename,
				Time:     timeStr,
				Signal:   signal,
			})
		}
	}

	return result, nil
}

func findCurveFiles(rootPath, startDate, endDate string) ([]CurveFile, error) {
	var results []CurveFile

	// 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %v", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %v", err)
	}

	// 结束日期应该包含当天的所有文件，所以设置为当天的23:59:59
	end = end.Add(24*time.Hour - time.Second)

	// 遍历根目录
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %v", rootPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 检查目录名是否为日期格式
		dirName := entry.Name()
		dirDate, err := time.Parse("2006-01-02", dirName)
		if err != nil {
			// 如果不是日期格式的目录，跳过
			continue
		}

		// 检查目录日期是否在指定范围内
		if (dirDate.After(start) || dirDate.Equal(start)) &&
			(dirDate.Before(end) || dirDate.Equal(end)) {

			// 遍历该日期目录下的文件
			dateDirPath := filepath.Join(rootPath, dirName)
			files, err := os.ReadDir(dateDirPath)
			if err != nil {
				// 如果无法读取子目录，跳过但继续处理其他目录
				fmt.Printf("Warning: cannot read directory %s: %v\n", dateDirPath, err)
				continue
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				filename := file.Name()
				// 检查是否为.curve文件
				if strings.HasSuffix(filename, ".curve") {
					// 解析文件时间
					fileTime, err := parseDateFromFilename(filename)
					if err != nil {
						fmt.Printf("Warning: cannot parse time from filename %s: %v\n", filename, err)
						continue
					}

					// 检查文件时间是否在日期范围内
					if (fileTime.After(start) || fileTime.Equal(start)) &&
						(fileTime.Before(end) || fileTime.Equal(end)) {

						signal := getSignalFromFilename(filename)
						results = append(results, CurveFile{
							Filename: filename,
							Time:     fileTime.Format("2006-01-02 15:04:05.000"),
							Signal:   signal,
						})
					}
				}
			}
		}
	}

	return results, nil
}
*/

// @Summary		查询录波数据记录
// @Description	查询录波数据记录
// @ID				faultrecording_wave
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			filename		path		string				true	"录波文件名"
// @Success		200				{object}	[]RecordingWaveInfo	"查询成功 {\"keys\":[\"开机状态\",\"有功功率设定值\",\"A相电压\"],\"type\":[0,1,1],\"timestamps\":[\"2025-09-28 17:25:14.779\"],\"values\":[[0,1,1,0]]}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/recording/{filename}/wave [post]
func FaultRecordingWave(c *gin.Context) {
	fileName := c.Param("filename")
	log.Info("FaultRecordingWave in... filename:", fileName)

	keyinfo := make([]string, 0)
	err := c.ShouldBindJSON(&keyinfo)
	if err != nil {
		log.Error("FaultRecordingWave ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest,
			Message: "invalid keys info"})
		return
	}
	keysmap := make(map[string]string, 0)
	for _, key := range keyinfo {
		keysmap[key] = ""
	}

	if err := refreshRecordingFileData(fileName); err != nil {
		log.Error("FaultRecordingWave refreshRecordingFileData failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "internal server error"})
		return
	}

	result := &RecordingWaveInfo{
		Timestamps: recordingFileData.Timestamps,
	}

	for i, key := range recordingFileData.Keys {
		if _, ok := keysmap[key]; ok {
			result.Keys = append(result.Keys, key)
			result.Type = append(result.Type, recordingFileData.Type[i])
			result.Values = append(result.Values, recordingFileData.Values[i])
		}
	}

	c.JSON(http.StatusOK, result)
}

////////////////////config///////////////////////

var (
	mapMux sync.Mutex
)

// 主配置结构体，对应完整的JSON文件
type Config struct {
	SamplingPeriod      int             `json:"samplingPeriod"`      // 采样周期，默认100ms
	BeforeTriggerPoints int             `json:"beforeTriggerPoints"` // 触发前采集点数，默认2000
	AfterTriggerPoints  int             `json:"afterTriggerPoints"`  // 触发后采集点数，默认2000
	RedundancyPoints    int             `json:"redundancyPoints"`    // 冗余点号按照最少上述之和来计算，循环链表的总个数 为，三个点号数量之和。默认200
	CurveFileType       int             `json:"curveFileType"`       // 文件格式 0 都生成, 1 json,2 txt,3 comtrade. 默认0
	TriggerType         int             `json:"triggerType"`         // 触发方式 0 变分触发，1 变合触发，2 变位触发，默认1
	TriggerSignals      []TriggerSignal `json:"triggerSignals"`
}

// 触发信号结构体
type TriggerSignal struct {
	Desc             string         `json:"desc"`
	ID               string         `json:"id"`
	Table            string         `json:"table"`
	Type             int            `json:"type"`
	TerminalID       string         `json:"terminalid"`
	ScdDevID         string         `json:"scddevid"`
	RelatedPoints    []RelatedPoint `json:"relatedPoints"`
	RelatedTerminals []RelatedPoint `json:"relatedTerminals"`
}

// 关联点结构体
type RelatedPoint struct {
	Desc       string `json:"desc"`
	ID         string `json:"id"`
	Table      string `json:"table"`
	Type       int    `json:"type"`
	TerminalID string `json:"terminalid"`
	ScdDevID   string `json:"scddevid"`
}

// 基础配置查询响应
type BasicConfigResponse struct {
	TriggerType  int `json:"triggertype"`
	Period       int `json:"period"`
	BeforePoints int `json:"brforepoints"`
	AfterPoints  int `json:"afterpoints"`
}

// 基础配置保存请求
type BasicConfigRequest struct {
	TriggerType  int `json:"triggertype"`
	Period       int `json:"period"`
	BeforePoints int `json:"brforepoints"`
	AfterPoints  int `json:"afterpoints"`
}

// 终端设备信息
type Terminal struct {
	TerminalID   string   `json:"terminalid"`
	TerminalDesc string   `json:"terminaldesc"`
	ScdDevs      []string `json:"scddevs"`
}

// SCD设备信息
type ScdDevice struct {
	ScdDevID   string `json:"scddevid"`
	ScdDevDesc string `json:"scddevdesc"`
}

// 遥测点
type YcPoint struct {
	Code     string `json:"code"`
	Desc     string `json:"desc"`
	Table    string `json:"table"`
	Terminal string `json:"terminal"`
	ScdDev   string `json:"scddev"`
}

// 遥信点
type YxPoint struct {
	Code     string `json:"code"`
	Desc     string `json:"desc"`
	Table    string `json:"table"`
	Terminal string `json:"terminal"`
	ScdDev   string `json:"scddev"`
}

// 全部遥测遥信点查询响应
type AllPointsResponse struct {
	Terminal     []Terminal  `json:"terminal"`
	Scd          []ScdDevice `json:"scd"`
	YcPoint      []YcPoint   `json:"ycpoint"`
	YxPoint      []YxPoint   `json:"yxpoint"`
	TriggerPoint []string    `json:"triggerpoint"`
}

// 触发参数信息
type TriggerParam struct {
	Desc         string `json:"desc"`
	ID           string `json:"id"`
	Table        string `json:"table"`
	Type         int    `json:"type"`
	TerminalID   string `json:"terminalid"`
	TerminalDesc string `json:"terminaldesc"`
	ScdDevID     string `json:"scddevid"`
	ScdDevDesc   string `json:"scddevdesc"`
}

// 添加触发参数请求
type AddTriggerParamRequest struct {
	Desc       string `json:"desc"`
	ID         string `json:"id"`
	Table      string `json:"table"`
	Type       int    `json:"type"`
	TerminalID string `json:"terminalid"`
	ScdDevID   string `json:"scddevid"`
}

func loadConfig() (*Config, error) {
	mapMux.Lock()
	defer mapMux.Unlock()
	// check file exists
	if _, err := os.Stat(FaultRecordingConfigPath); os.IsNotExist(err) {
		// file not exists, return default config
		return &Config{
			SamplingPeriod:      100,
			BeforeTriggerPoints: 2000,
			AfterTriggerPoints:  2000,
			RedundancyPoints:    200,
			CurveFileType:       0,
			TriggerType:         1,
			TriggerSignals:      []TriggerSignal{},
		}, nil
	}

	data, err := os.ReadFile(FaultRecordingConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read config file failed: %v", err)
	}

	//utf8Data := gbk2utf8(data)

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("parse config file failed: %v", err)
	}

	return &config, nil
}

func saveConfig(config *Config) error {
	mapMux.Lock()
	defer mapMux.Unlock()
	// 确保目录存在
	dir := filepath.Dir(FaultRecordingConfigPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("create directory failed: %v", err)
	}

	// 序列化JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config failed: %v", err)
	}
	//gbkData := utf82gbk(data)
	// 写入文件
	err = os.WriteFile(FaultRecordingConfigPath, data, 0644)
	if err != nil {
		return fmt.Errorf("write config file failed: %v", err)
	}

	return nil
}

// @Summary		获取录波基础配置
// @Description	获取录波基础配置
// @ID				faultrecording_getconfigbase
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Success		200				{object}	BasicConfigResponse	"查询成功 {"triggertype":1,"period":100,"brforepoints":1000,"afterpoints":1000}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/base [get]
func GetConfigBase(c *gin.Context) {
	log.Info("GetConfigBase in ...")

	config, err := loadConfig()
	if err != nil {
		log.Info("GetConfigBase loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "load config error"})
		return
	}

	result := &BasicConfigResponse{
		TriggerType:  config.TriggerType,
		Period:       config.SamplingPeriod,
		BeforePoints: config.BeforeTriggerPoints,
		AfterPoints:  config.AfterTriggerPoints,
	}
	log.Info("GetConfigBase result:", result)
	c.JSON(http.StatusOK, result)
	return
}

// @Summary		提交录波基础配置
// @Description	提交录波基础配置
// @ID				faultrecording_setconfigbase
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param		BasicConfigRequest		body		BasicConfigRequest			true	"录波基础配置"
// @Success		200				{object}	models.CommonResp	"提交成功 {\"code\": 200, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/base [post]
func PostConfigBase(c *gin.Context) {
	log.Info("PostConfigBase in ...")
	var req BasicConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Info("PostConfigBase ShouldBindJSON err:", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "bind json error"})
		return
	}
	log.Info("PostConfigBase req:", req)
	if req.Period < 100 {
		log.Info("PostConfigBase err: Period ", req.Period, " must be greater than or equal to 100ms")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Period must be greater than or equal to 100ms"})
		return
	}

	// check trigger type
	if req.TriggerType < 0 || req.TriggerType > 2 {
		log.Info("PostConfigBase err: TriggerType ", req.TriggerType, " is invalid, must be 0, 1 or 2")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "TriggerType is invalid"})
		return
	}

	config, err := loadConfig()
	if err != nil {
		log.Info("PostConfigBase loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "load config error"})
		return
	}

	config.TriggerType = req.TriggerType
	config.SamplingPeriod = req.Period
	config.BeforeTriggerPoints = req.BeforePoints
	config.AfterTriggerPoints = req.AfterPoints

	err = saveConfig(config)
	if err != nil {
		log.Info("PostConfigBase saveConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "save config error"})
		return
	}

	err = setRecordProgramStatus(saveFlagConfigModify)
	if err != nil {
		log.Error("SetFaultRecordSaveFlag setRecordProgramStatus err:", err)
		// c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "set save flag error"})
		// return
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "success"})
	return
}

// @Summary		获取设备可用遥测遥信点
// @Description	获取设备可用遥测遥信点
// @ID				faultrecording_getallpoints
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string			true	"JWT Token"	default(<token>)
// @Success		200				{object}	AllPointsResponse	"查询成功 {\"terminal\":[{\"terminalid\":\"xxxx1\",\"terminaldesc\":\"yyyy1\"},{\"terminalid\":\"xxxx2\",\"terminaldesc\":\"yyyy2\"}],\"scd\":[{\"scddevid\":\"xxxx3\",\"scddevdesc\":\"yyyy3\"},{\"scddevid\":\"xxxx4\",\"scddevdesc\":\"yyyy4\"}],\"ycpoint\":[{\"code\":\"a123\",\"desc\":\"xdx\",\"table\":\"analog\",\"terminal\":\"xxxx\",\"scddev\":\"xxxx3\"}],\"yxpoint\":[{\"code\":\"a124\",\"desc\":\"xdx\",\"table\":\"break\",\"terminal\":\"xxxx\",\"scddev\":\"xxxx3\"}],\"triggerpoint\":[\"aaa\",\"bbb\"]}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/points [get]
func GetConfigPoints(c *gin.Context) {
	log.Info("GetConfigPoints in ...")
	dbScdDevices, err := emsdb.GetAllDevices()
	if err != nil {
		log.Info("GetConfigPoints GetAllDevices err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "load config error"})
		return
	}
	termialIDmap := make(map[string]string)
	terminalScddevMap := make(map[string][]string, 0)
	var scdDevices []ScdDevice
	for _, scdDeviceParam := range dbScdDevices {
		scdDevices = append(scdDevices, ScdDevice{
			ScdDevID:   scdDeviceParam.Code,
			ScdDevDesc: scdDeviceParam.Name,
		})
		termialIDmap[scdDeviceParam.Code] = scdDeviceParam.StationCode
		terminalScddevMap[scdDeviceParam.StationCode] = append(terminalScddevMap[scdDeviceParam.StationCode], scdDeviceParam.Code)
	}

	dbTerminals, err := emsdb.GetAllStationParam()
	if err != nil {
		log.Info("GetConfigPoints GetAllStationParam err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "load config error"})
		return
	}
	var terminals []Terminal
	for _, terminalParam := range dbTerminals {
		terminals = append(terminals, Terminal{
			TerminalID:   terminalParam.Code,
			TerminalDesc: terminalParam.Name,
			ScdDevs:      terminalScddevMap[terminalParam.Code],
		})
	}

	dbYcPoints, err := emsdb.GetAllYcParams()
	if err != nil {
		log.Info("GetConfigPoints GetAllYcParams err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "load config error"})
		return
	}
	var ycPoints []YcPoint
	for _, ycParam := range dbYcPoints {
		ycPoints = append(ycPoints, YcPoint{
			Code:     ycParam.Code,
			Desc:     ycParam.Name,
			Table:    "analog",
			Terminal: termialIDmap[ycParam.ScdCode],
			ScdDev:   ycParam.ScdCode,
		})
	}

	dbYxPoints, err := emsdb.GetAllYxParams()
	if err != nil {
		log.Info("GetConfigPoints GetAllYxParams err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "load config error"})
		return
	}
	var yxPoints []YxPoint
	for _, yxParam := range dbYxPoints {
		yxPoints = append(yxPoints, YxPoint{
			Code: yxParam.Code,
			Desc: yxParam.Name,
			Table: func(yxtype int) string {
				if yxtype == 300 {
					return "break"
				}
				return "prot" // 600 is Bh
			}(yxParam.Type),
			Terminal: termialIDmap[yxParam.ScdCode],
			ScdDev:   yxParam.ScdCode,
		})
	}

	// get existed trigger point
	config, err := loadConfig()
	if err != nil {
		log.Info("GetConfigPoints loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "load config error"})
		return
	}

	triggerPoints := make([]string, len(config.TriggerSignals))
	for i, signal := range config.TriggerSignals {
		triggerPoints[i] = signal.Desc
	}

	result := AllPointsResponse{
		Terminal:     terminals,
		Scd:          scdDevices,
		YcPoint:      ycPoints,
		YxPoint:      yxPoints,
		TriggerPoint: triggerPoints,
	}
	log.Info("GetConfigPoints result, terminal:", result.Terminal, "yc len:", len(result.YcPoint), "yx len:", len(result.YxPoint), "trigger:", result.TriggerPoint)
	c.JSON(http.StatusOK, result)
	return
}

// @Summary		获取录波触发点配置
// @Description	获取录波触发点配置
// @ID				faultrecording_gettriggerpoints
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Success		200				{object}	[]TriggerParam	"查询成功 [{\"desc\":\"\",\"id\":\"\",\"table\":\"hh\",\"type\":0,\"terminalid\":\"xxxx\",\"scddevid\":\"mmmm\"}]"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/triggerpoints [get]
func GetTriggerPoints(c *gin.Context) {
	log.Info("GetTriggerPoints in ...")
	config, err := loadConfig()
	if err != nil {
		log.Info("GetTriggerPoints loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "load config error"})
		return
	}

	terminals, err := emsdb.GetAllStationParam()
	if err != nil {
		log.Info("GetTriggerPoints GetAllStationParam err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "GetAllStationParam error"})
		return
	}
	scdDevices, err := emsdb.GetAllDevices()
	if err != nil {
		log.Info("GetTriggerPoints GetAllDevices err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: 500, Message: "GetAllDevices error"})
		return
	}

	result := make([]TriggerParam, len(config.TriggerSignals))

	for i, signal := range config.TriggerSignals {
		terminalDesc := ""
		for _, term := range terminals {
			if term.Code == signal.TerminalID {
				terminalDesc = term.Name
				break
			}
		}

		scdDesc := ""
		for _, scd := range scdDevices {
			if scd.Code == signal.ScdDevID {
				scdDesc = scd.Name
				break
			}
		}

		result[i] = TriggerParam{
			Desc:         signal.Desc,
			ID:           signal.ID,
			Table:        signal.Table,
			Type:         signal.Type,
			TerminalID:   signal.TerminalID,
			TerminalDesc: terminalDesc,
			ScdDevID:     signal.ScdDevID,
			ScdDevDesc:   scdDesc,
		}
	}

	log.Info("GetTriggerPoints result:", result)
	c.JSON(http.StatusOK, result)
	return
}

func utf82gbk(utf8Data []byte) []byte {
	// 创建一个Reader来读取UTF-8字符串，并通过编码器转换为GBK
	utf8Reader := transform.NewReader(bytes.NewReader(utf8Data), simplifiedchinese.GBK.NewEncoder())
	gbkData, err := io.ReadAll(utf8Reader)
	if err != nil {
		log.Error("utf82gbk failed:", err.Error(), ", ret utf8Str:", utf8Data)
		return utf8Data
	}
	return gbkData
}

func utf8Str2gbkStr(utf8Str string) string {
	// 创建一个Reader来读取UTF-8字符串，并通过编码器转换为GBK
	return string(utf82gbk([]byte(utf8Str)))
}

func gbk2utf8(gbkData []byte) []byte {
	gbkReader := transform.NewReader(bytes.NewReader(gbkData), simplifiedchinese.GBK.NewDecoder())
	utf8Data, err := io.ReadAll(gbkReader)
	if err != nil {
		log.Error("gbk2utf8 failed:", err.Error(), ", ret gbkStr:", gbkData)
		return gbkData
	}
	return utf8Data
}

func gbkStr2utf8Str(gdkStr string) string {
	// 创建一个Reader来读取UTF-8字符串，并通过编码器转换为GBK
	return string(gbk2utf8([]byte(gdkStr)))
}

// func utf82gbk(utf8Str string) string {
// 	// 创建一个Reader来读取UTF-8字符串，并通过编码器转换为GBK
// 	utf8Reader := transform.NewReader(bytes.NewReader([]byte(utf8Str)), simplifiedchinese.GBK.NewEncoder())
// 	gbkData, err := io.ReadAll(utf8Reader)
// 	if err != nil {
// 		log.Error("utf82gbk failed:", err.Error(), ", ret utf8Str:", utf8Str)
// 		return utf8Str
// 	}
// 	return string(gbkData)
// }

// func gbk2utf8(gbkStr string) string {
// 	gbkReader := transform.NewReader(bytes.NewReader([]byte(gbkStr)), simplifiedchinese.GBK.NewDecoder())
// 	utf8Data, err := io.ReadAll(gbkReader)
// 	if err != nil {
// 		log.Error("gbk2utf8 failed:", err.Error(), ", ret gbkStr:", gbkStr)
// 		return gbkStr
// 	}
// 	return string(utf8Data)
// }

// @Summary		提交录波触发点配置
// @Description	提交录波触发点配置
// @ID			faultrecording_settriggerpoints
// @Tags		faultrecording
// @Accept		json
// @Produce		json
// @Param		Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param		AddTriggerParamRequest		body		AddTriggerParamRequest			true	"触发点配置"
// @Success		200				{object}	models.CommonResp	"操作成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/triggerpoints [post]
func PostTriggerPoints(c *gin.Context) {
	req := AddTriggerParamRequest{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Info("PostTriggerPoints ShouldBindJSON err:", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}
	log.Info("PostTriggerPoints in ...req:", req)

	config, err := loadConfig()
	if err != nil {
		log.Info("PostTriggerPoints loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "load config error"})
		return
	}

	// check if signalID exists
	for _, signal := range config.TriggerSignals {
		if signal.ID == req.ID {
			log.Info("PostTriggerPoints signalID:", signal.ID, " already exists")
			c.JSON(http.StatusBadRequest, models.CommonResp{Code: 400, Message: "signalID already exists"})
			return
		}
	}

	newSignal := TriggerSignal{
		Desc:             req.Desc,
		ID:               req.ID,
		Table:            req.Table,
		Type:             req.Type,
		TerminalID:       req.TerminalID,
		ScdDevID:         req.ScdDevID,
		RelatedPoints:    []RelatedPoint{},
		RelatedTerminals: []RelatedPoint{},
	}

	config.TriggerSignals = append(config.TriggerSignals, newSignal)

	log.Info("PostTriggerPoints save config:", config)
	err = saveConfig(config)
	if err != nil {
		log.Info("PostTriggerPoints saveConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "save config error"})
		return
	}

	err = setRecordProgramStatus(saveFlagConfigModify)
	if err != nil {
		log.Error("SetFaultRecordSaveFlag setRecordProgramStatus err:", err)
		// c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "set save flag error"})
		// return
	}

	log.Info("PostTriggerPoints OK")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "success"})
	return
}

// @Summary		删除录波触发点配置
// @Description	删除录波触发点配置
// @ID				faultrecording_deltriggerpoints
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			[]string		body		[]string			true	"删除的触发点id"
// @Success		200				{object}	models.CommonResp	"删除成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/triggerpoints [delete]
func DeleteTriggerPoints(c *gin.Context) {
	log.Info("DeleteTriggerPoints in ...")
	req := make([]string, 0)
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Info("DeleteTriggerPoints ShouldBindJSON err:", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}

	deleteSignalMap := make(map[string]bool)
	for _, id := range req {
		deleteSignalMap[id] = true
	}

	config, err := loadConfig()
	if err != nil {
		log.Info("DeleteTriggerPoints loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "load config error"})
		return
	}

	found := false
	newSignals := make([]TriggerSignal, 0, len(config.TriggerSignals)-1)

	for _, signal := range config.TriggerSignals {
		if _, ok := deleteSignalMap[signal.ID]; ok {
			found = true
			continue
		}
		newSignals = append(newSignals, signal)
	}

	if !found {
		log.Warn("DeleteTriggerPoints not found signalID:", req)
	}

	config.TriggerSignals = newSignals
	err = saveConfig(config)
	if err != nil {
		log.Info("DeleteTriggerPoints saveConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "save config error"})
		return
	}

	err = setRecordProgramStatus(saveFlagConfigModify)
	if err != nil {
		log.Error("SetFaultRecordSaveFlag setRecordProgramStatus err:", err)
		// c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "set save flag error"})
		// return
	}

	log.Info("DeleteTriggerPoints OK")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "success"})
	return
}

// @Summary		获取录波关联点配置
// @Description	获取录波关联点配置
// @ID				faultrecording_getrelatedpoints
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Success		200				{object}	[]TriggerParam	"查询成功 [{\"desc\":\"\",\"id\":\"\",\"table\":\"hh\",\"type\":0,\"terminalid\":\"xxxx\",\"scddevid\":\"mmmm\"}]"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/{triggerpoint}/relatedpoints [get]
func GetRelatedPoints(c *gin.Context) {
	triggerID := c.Param("triggerpoint")
	log.Info("GetRelatedPoints in ...triggerID:", triggerID)
	config, err := loadConfig()
	if err != nil {
		log.Info("GetRelatedPoints loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "load config error"})
		return
	}

	var triggerSignal *TriggerSignal
	for i, signal := range config.TriggerSignals {
		if signal.ID == triggerID {
			triggerSignal = &config.TriggerSignals[i]
			break
		}
	}

	if triggerSignal == nil {
		log.Error("GetRelatedPoints triggerID:", triggerID, " not found")
		c.JSON(http.StatusNotFound, models.CommonResp{Code: http.StatusNotFound, Message: "trigger point not found"})
		return
	}

	// 获取所有终端和SCD设备信息用于补充描述
	terminals, _ := emsdb.GetAllStationParam()
	terminalsMap := make(map[string]string)
	for _, term := range terminals {
		terminalsMap[term.Code] = term.Name
	}

	scdDevices, _ := emsdb.GetAllDevices()
	scdDevicesMap := make(map[string]string)
	for _, scd := range scdDevices {
		scdDevicesMap[scd.Code] = scd.Name
	}

	// 合并相关点和相关终端的点
	allRelated := triggerSignal.RelatedPoints

	result := make([]TriggerParam, len(allRelated))
	for i, point := range allRelated {
		result[i] = TriggerParam{
			Desc:         point.Desc,
			ID:           point.ID,
			Table:        point.Table,
			Type:         point.Type,
			TerminalID:   point.TerminalID,
			TerminalDesc: terminalsMap[point.TerminalID],
			ScdDevID:     point.ScdDevID,
			ScdDevDesc:   scdDevicesMap[point.ScdDevID],
		}
	}
	log.Info("GetRelatedPoints result:", result)
	c.JSON(http.StatusOK, result)
	return
}

// @Summary		提交录波关联点配置
// @Description	提交录波关联点配置
// @ID				faultrecording_setrelatedpoints
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			triggerpoint				path		string							true	"触发点的id"
// @Param			[]AddTriggerParamRequest		body		[]AddTriggerParamRequest			true	"关联点配置"
// @Success		200				{object}	models.CommonResp	"操作成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/{triggerpoint}/relatedpoints [post]
func PostRelatedPoints(c *gin.Context) {
	triggerID := c.Param("triggerpoint")
	var req []AddTriggerParamRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Error("PostRelatedPoints ShouldBindJSON err:", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}
	log.Info("PostRelatedPoints in ... triggerpoint:", triggerID, " req:", req)

	config, err := loadConfig()
	if err != nil {
		log.Error("PostRelatedPoints loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "load config error"})
		return
	}

	// 查找触发参数
	var triggerSignal *TriggerSignal
	for i := range config.TriggerSignals {
		if config.TriggerSignals[i].ID == triggerID {
			triggerSignal = &config.TriggerSignals[i]
			break
		}
	}

	if triggerSignal == nil {
		log.Error("PostRelatedPoints triggerID:", triggerID, " not exists")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: 400, Message: "triggerID not exists"})
		return
	}

	for _, relatedPoint := range req {
		newRelatedPoint := RelatedPoint{
			Desc:       relatedPoint.Desc,
			ID:         relatedPoint.ID,
			Table:      relatedPoint.Table,
			Type:       relatedPoint.Type,
			TerminalID: relatedPoint.TerminalID,
			ScdDevID:   relatedPoint.ScdDevID,
		}
		triggerSignal.RelatedPoints = append(triggerSignal.RelatedPoints, newRelatedPoint)
	}

	err = saveConfig(config)
	if err != nil {
		log.Info("PostRelatedPoints saveConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "save config error"})
		return
	}

	err = setRecordProgramStatus(saveFlagConfigModify)
	if err != nil {
		log.Error("SetFaultRecordSaveFlag setRecordProgramStatus err:", err)
		// c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "set save flag error"})
		// return
	}
	log.Info("PostRelatedPoints OK")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "success"})
	return
}

// @Summary		删除录波关联点配置
// @Description	删除录波关联点配置
// @ID				faultrecording_delrelatedpoints
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Param			triggerpoint				path		string							true	"触发点的id"
// @Param			[]string		body		[]string			true	"删除的关联点id"
// @Success		200				{object}	models.CommonResp	"删除成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/config/{triggerpoint}/relatedpoints [delete]
func DeleteRelatedPoints(c *gin.Context) {
	triggerID := c.Param("triggerpoint")
	req := make([]string, 0)
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Info("DeleteRelatedPoints ShouldBindJSON err:", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "Bad Request"})
		return
	}
	log.Info("DeleteRelatedPoints in ...triggerID:", triggerID, " req:", req)

	deletePointMap := make(map[string]bool)
	for _, id := range req {
		deletePointMap[id] = true
	}

	config, err := loadConfig()
	if err != nil {
		log.Info("DeleteRelatedPoints loadConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "load config error"})
		return
	}

	var triggerSignal *TriggerSignal
	for i := range config.TriggerSignals {
		if config.TriggerSignals[i].ID == triggerID {
			triggerSignal = &config.TriggerSignals[i]
			break
		}
	}

	if triggerSignal == nil {
		log.Warn("DeleteRelatedPoints not found signalID:", triggerID)
		c.JSON(http.StatusNotFound, models.CommonResp{Code: http.StatusNotFound, Message: "trigger point not found"})
		return
	}

	newRelatedPoints := make([]RelatedPoint, 0)
	for _, point := range triggerSignal.RelatedPoints {
		// filter out deleted points
		if _, ok := deletePointMap[point.ID]; !ok {
			newRelatedPoints = append(newRelatedPoints, point)
		}
	}

	triggerSignal.RelatedPoints = newRelatedPoints
	err = saveConfig(config)
	if err != nil {
		log.Info("DeleteRelatedPoints saveConfig err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "save config error"})
		return
	}

	err = setRecordProgramStatus(saveFlagConfigModify)
	if err != nil {
		log.Error("SetFaultRecordSaveFlag setRecordProgramStatus err:", err)
		// c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "set save flag error"})
		// return
	}

	log.Info("DeleteRelatedPoints OK")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "success"})
	return
}

/*
	file format:

[common]
saveFlag=0
*/
func getRecordProgramStatus() (int, error) {
	cfg, err := ini.Load(FaultRecordingStatusPath)
	if err != nil {
		return 0, fmt.Errorf("getRecordProgramStatus load '%s' err: %v", FaultRecordingStatusPath, err)
	}

	commonSection, err := cfg.GetSection("common")
	if err != nil {
		return 0, fmt.Errorf("getRecordProgramStatus GetSection 'common' err: %v", err)
	}

	saveFlagKey, err := commonSection.GetKey("saveFlag")
	if err != nil {
		return 0, fmt.Errorf("getRecordProgramStatus GetKey 'saveFlag' in 'common' err: %v", err)
	}

	saveFlagVal, err := saveFlagKey.Int()
	if err != nil {
		return 0, fmt.Errorf("getRecordProgramStatus saveFlagKey.Int() err: %v", err)
	}

	return saveFlagVal, nil
}

func setRecordProgramStatus(saveFlag int) error {
	cfg, err := ini.Load(FaultRecordingStatusPath)
	if err != nil {
		// if os.IsNotExist(err) {
		// 	log.Warnf("setRecordProgramStatus FaultRecordingStatusPath '%s' not exists, create a new one", FaultRecordingStatusPath)
		// 	cfg = ini.Empty() // 初始化空INI对象，后续创建section和key
		// } else {

		// }
		// we can not create a new one for localweb create file is root permission, others can not use
		return fmt.Errorf("setRecordProgramStatus load '%s' err: %v", FaultRecordingStatusPath, err)
	}

	commonSection, err := cfg.GetSection("common")
	if err != nil {
		log.Warn("setRecordProgramStatus section 'common' not exist, create a new one")
		commonSection, err = cfg.NewSection("common")
		if err != nil {
			return fmt.Errorf("setRecordProgramStatus NewSection 'common' err: %v", err)
		}
	}

	saveFlagKey, err := commonSection.GetKey("saveFlag")
	if err != nil {
		log.Warn("setRecordProgramStatus key 'saveFlag' in section 'common' not exist, create a new one")
		saveFlagKey, err = commonSection.NewKey("saveFlag", "1")
		if err != nil {
			return fmt.Errorf("setRecordProgramStatus NewKey 'saveFlag' err: %v", err)
		}
	}
	saveFlagStr := "1"
	switch saveFlag {
	case 1:
		saveFlagStr = "1" // if any config modified, set 1. show buttun
	case 2:
		saveFlagStr = "2" // manual set 2, to noitify faultrecording-program to load config
	default:
		return fmt.Errorf("setRecordProgramStatus invalid saveFlag: %d", saveFlag)
	}
	saveFlagKey.SetValue(saveFlagStr)
	log.Info("setRecordProgramStatus saveFlag:", saveFlag, " saveFlagStr:", saveFlagStr)

	return cfg.SaveTo(FaultRecordingStatusPath)
}

// @Summary		获取配置生效信息
// @Description	获取配置生效信息
// @ID				faultrecording_getsaveflag
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Success		200				{object}	map[string]int	"查询成功 {\"saveflag\":1}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/saveflag [get]
func GetFaultRecordSaveFlag(c *gin.Context) {
	log.Info("GetFaultRecordSaveFlag in ...")

	saveFlag, err := getRecordProgramStatus()
	// if err != nil {
	// 	log.Error("GetFaultRecordSaveFlag getRecordProgramStatus err:", err)
	// 	 c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "get save flag error"})
	// 	 return
	// } if err, saveFlag is 0, just return 0 to client

	log.Info("GetFaultRecordSaveFlag saveFlag:", saveFlag, ", getRecordProgramStatus err:", err)
	result := map[string]int{"saveflag": saveFlag}
	c.JSON(http.StatusOK, result)
}

// @Summary		提交配置生效信息
// @Description	提交配置生效信息
// @ID				faultrecording_setsaveflag
// @Tags			faultrecording
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string						true	"JWT Token"	default(<token>)
// @Success		200				{object}	map[string]int	"提交成功 {\"saveflag\":1}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务器错误 {\"code\": 500, \"msg\": \"\"}"
// @Router		/api/v1/log/faultrecording/saveflag [post]
func SetFaultRecordSaveFlag(c *gin.Context) {
	log.Info("SetFaultRecordSaveFlag in ...")
	err := setRecordProgramStatus(saveFlagConfigNoitfy)
	if err != nil {
		log.Error("SetFaultRecordSaveFlag setRecordProgramStatus err:", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "set save flag error"})
		return
	}
	log.Info("SetFaultRecordSaveFlag OK")
	result := map[string]int{"saveflag": saveFlagConfigNoitfy}
	c.JSON(http.StatusOK, result)
}
