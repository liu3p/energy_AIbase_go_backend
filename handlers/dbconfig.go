package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"webserver-arm/cgolibparse"
	"webserver-arm/emsxmldb"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
)

var validAlphaNumUnderscoreRegex = regexp.MustCompile(`^[0-9a-zA-Z_]+$`) // 仅允许 字母/数字/下划线
var validOnlyDigitsRegex = regexp.MustCompile(`^[0-9]+$`)                // 仅允许 纯数字

func validateAlphaNumUnderscore(input string) bool {
	if input == "" {
		return false
	}
	return validAlphaNumUnderscoreRegex.MatchString(input)
}

func validateOnlyDigits(input string) bool {
	if input == "" {
		return false
	}
	return validOnlyDigitsRegex.MatchString(input)
}

// @Summary		查询全部rtu信息
// @Description	查询全部rtu信息
// @ID				query-rtus
// @Tags			dbcfg
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			include_channel	query		bool				false	"是否返回channel字段"	default(true)
// @Success		200				{object}	[]emsxmldb.RTU	"查询成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtus [get]
func GetRTUs(c *gin.Context) {
	log.Info("GetRTUs in ...")
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetRTUs: GetDbConfigManager failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	rtus := dbcfg.GetAllRTUs()

	// 默认返回 channel，前端传 include_channel=false 可省略
	includeChannel := true
	if v := c.Query("include_channel"); v == "false" || v == "0" {
		includeChannel = false
	}

	if !includeChannel {
		// 创建不含 Channel 的副本，避免修改内存数据，保留 ChannelGroupName
		result := make([]emsxmldb.RTU, len(rtus))
		for i, rtu := range rtus {
			result[i] = *rtu
			result[i].Channel = nil
		}
		c.JSON(http.StatusOK, result)
		return
	}

	c.JSON(http.StatusOK, rtus)
}

// 获取所有RTU和Device的名字
func GetRTUsExceptPoints(c *gin.Context) {
	log.Info("GetRTUs in ...")
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetRTUs: GetDbConfigManager failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	rtus := dbcfg.GetAllRTUsExceptPoints()

	includeChannel := true
	if v := c.Query("include_channel"); v == "false" || v == "0" {
		includeChannel = false
	}

	if !includeChannel {
		// 创建不含 Channel 的副本，避免修改内存数据，保留 ChannelGroupName
		result := make([]emsxmldb.RTU, len(rtus))
		for i := range rtus {
			result[i] = rtus[i]
			result[i].Channel = nil
		}
		c.JSON(http.StatusOK, result)
		return
	}

	c.JSON(http.StatusOK, rtus)
}

// @Summary		添加rtu信息
// @Description	添加rtu信息，有通道的RTU类型自动创建通道组+通道
// @ID				add-rtu
// @Tags			dbcfg
// @Accept		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			data	body	emsxmldb.RTU	true	"RTU信息"
// @Success		200				{object}	models.CommonResp	"添加成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu [post]
func AddRTU(c *gin.Context) {
	rtu := emsxmldb.RTU{}
	err := c.ShouldBindJSON(&rtu)
	if err != nil {
		log.Errorf("AddRTU: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	if rtu.Name == "" {
		log.Error("AddRTU failed: rtu.Name is empty")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "rtu.Name is empty"})
		return
	}

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("AddRTU: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 无通道的 RTU 类型：直接创建，ChannelGroupID 和 ChannelID 设为 -1
	if !rtuHasChannel(rtu.Type) {
		rtu.ChannelGroupID = -1
		rtu.ChannelID = -1
		err = dbcfg.AddRTU(&rtu)
		if err != nil {
			log.Errorf("AddRTU: AddRTU failed: %w", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "AddRTU success"})
		return
	}

	// 有通道的 RTU 类型：先预生成 ID，再创建通道组+通道，最后创建 RTU 并绑定
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Errorf("AddRTU: GetFeConfigXMLManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 1. 预生成 RTU ID（加锁，确保 ID 唯一）
	rtuID, err := dbcfg.GenerateRTUID(rtu.Type)
	if err != nil {
		log.Errorf("AddRTU: GenerateRTUID failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	rtu.ID = rtuID

	// 2. 创建通道组+通道（STATION_BASE_ID 和 PossibleOwner.ID 使用已知的 rtuID）
	cgName := channelGroupName(rtu)
	chName := rtu.Name + "的通道"
	rtuIDInt := rtuIDToInt(rtuID)

	forTransfer := "0"
	if rtuIDInt >= 500 && rtuIDInt < 700 {
		forTransfer = "1"
	}

	cgParam := emsxmldb.ChannelGroupParam{
		Name:          cgName,
		ForTransfer:   forTransfer,
		StationBaseID: rtuID,
	}
	chParam := emsxmldb.ChannelParam{
		Name:            chName,
		PossibleOwnerID: rtuIDInt,
	}

	cgID, chID, err := fecfg.CreateChannelGroupWithChannel(cgParam, chParam)
	if err != nil {
		log.Errorf("AddRTU: CreateChannelGroupWithChannel failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 填充内存字段，使新增后立即查询能拿到值
	cg, err := fecfg.GetChannelGroupByID(cgID)
	if err != nil {
		log.Errorf("AddRTU: GetChannelGroupByID failed: %w", err)
	} else {
		rtu.ChannelGroupName = cg.Name
		if len(cg.Channels) > 0 {
			rtu.Channel = &cg.Channels[0]
		}
		for _, p := range cg.Model.Parameters {
			if p.Name == "FOR_TRANSFER" {
				rtu.ForTransfer = p.Value
				break
			}
		}
	}

	// 3. 创建 RTU 并绑定通道，统一加锁，一次 save
	err = dbcfg.AddRTUWithChannel(&rtu, cgID, chID)
	if err != nil {
		// 回滚：删除刚创建的通道组
		fecfg.DeleteChannelGroup(cgID)
		log.Errorf("AddRTU: AddRTUWithChannel failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "AddRTU success",
		"data": gin.H{
			"rtuid":          rtu.ID,
			"channelgroupid": cgID,
			"channelid":      chID,
		},
	})
}

// rtuHasChannel 判断 RTU 类型是否有通道
func rtuHasChannel(rtuType int) bool {
	return rtuType == emsxmldb.RtuTypeAcqu ||
		rtuType == emsxmldb.RtuTypeTrans ||
		rtuType == emsxmldb.RtuTypeAgc
}

// channelGroupName 根据 RTU 类型生成通道组名称
func channelGroupName(rtu emsxmldb.RTU) string {
	if rtu.Type == emsxmldb.RtuTypeTrans {
		return rtu.Name + "的转发通道"
	}
	return rtu.Name + "的通道组"
}

// rtuIDToInt 将 RTU ID 字符串转为 int，失败返回 0
func rtuIDToInt(rtuID string) int {
	id, err := strconv.Atoi(rtuID)
	if err != nil {
		return 0
	}
	return id
}

// @Summary		修改rtu信息
// @Description	修改rtu信息
// @ID				update-rtu
// @Tags			dbcfg
// @Accept		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			data	body	emsxmldb.RTU	true	"RTU信息"
// @Success		200				{object}	models.CommonResp	"修改成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid} [put]
func UpdateRTU(c *gin.Context) {
	rtuID := c.Param("rid")
	log.Info("UpdateRTU in ... rid:", rtuID)
	rtu := emsxmldb.RTU{}
	err := c.ShouldBindJSON(&rtu)
	if err != nil {
		log.Errorf("UpdateRTU: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	if rtu.Name == "" {
		log.Error("UpdateRTU failed: rtu.Name is empty")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "rtu.Name is empty"})
		return
	}
	if !validateOnlyDigits(rtu.RtuAddr) {
		log.Error("UpdateRTU failed: rtu.RtuAddr is invalid")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "rtu.RtuAddr is invalid"})
		return
	}

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("UpdateRTU: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = dbcfg.UpdateHierarchyRTU(rtu, rtuID)
	if err != nil {
		log.Errorf("UpdateRTU: UpdateHierarchyRTU failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 更新通道组+通道（仅有通道的 RTU 类型且请求中携带 channel 时执行）
	if rtuHasChannel(rtu.Type) && rtu.Channel != nil {
		fecfg, err := emsxmldb.GetFeConfigXMLManager()
		if err != nil {
			log.Errorf("UpdateRTU: GetFeConfigXMLManager failed: %w", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
			return
		}

		// 从请求的 Channel.Plugins 中拆分 AppPlugin
		var appPlugin *emsxmldb.Plugin
		for _, p := range rtu.Channel.Plugins {
			if p.Type == "APP" {
				appPlugin = &p
				break
			}
		}

		// 更新通道组名称 + FOR_TRANSFER
		cgParam := emsxmldb.ChannelGroupParam{
			Name:          rtu.ChannelGroupName,
			ForTransfer:   rtu.ForTransfer,
			StationBaseID: rtuID,
		}
		if err := fecfg.UpdateChannelGroup(rtu.ChannelGroupID, cgParam); err != nil {
			log.Errorf("UpdateRTU: UpdateChannelGroup failed: %w", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
			return
		}

		// 全量替换通道
		chParam := emsxmldb.ChannelParam{
			ChannelID:       rtu.ChannelID,
			Name:            rtu.Channel.Name,
			Comment:         rtu.Channel.Comment,
			PossibleOwnerID: rtu.Channel.PossibleOwner.ID,
			AppPlugin:       appPlugin,
		}
		if err := fecfg.UpdateChannel(rtu.ChannelGroupID, chParam); err != nil {
			log.Errorf("UpdateRTU: UpdateChannel failed: %w", err)
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "UpdateRTU success"})
	return
}

// @Summary		删除rtu信息
// @Description	删除rtu信息，同步删除关联的通道组+通道
// @ID				delete-rtu
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Success		200				{object}	models.CommonResp	"删除成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid} [delete]
func DeleteRTU(c *gin.Context) {
	rtuID := c.Param("rid")
	log.Info("DeleteRTU in ... rid:", rtuID)

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("DeleteRTU: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Errorf("DeleteRTU: GetFeConfigXMLManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 删除 RTU + 通道组，统一加锁
	err = dbcfg.DeleteRTUWithChannel(rtuID, func(id string) error {
		return fecfg.DeleteChannelGroupByStationID(id)
	})
	if err != nil {
		log.Errorf("DeleteRTU: DeleteRTUWithChannel failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "DeleteRTU success"})
}

// @Summary		查询指定rtu下的所有设备信息
// @Description	查询指定rtu下的所有设备信息
// @ID				query-rtudevices
// @Tags			dbcfg
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Success		200				{object}	[]emsxmldb.Device	"查询成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"未找到 {\"code\": 404, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid}/devices [get]
func GetRtuDevices(c *gin.Context) {
	rtuID := c.Param("rid")
	log.Info("GetRtuDevices in ... rid:", rtuID)
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetRtuDevices: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	devices, err := dbcfg.GetAllDevicesFromRTU(rtuID)
	if err != nil {
		log.Errorf("GetRtuDevices: GetAllDevicesFromRTU failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("GetRtuDevices success:", devices)
	c.JSON(http.StatusOK, devices)
}

// @Summary		查询指定rtu下的指定设备信息
// @Description	查询指定rtu下的指定设备信息
// @ID				query-rtudevice
// @Tags			dbcfg
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			did	path	string	true	"Device ID"
// @Success		200				{object}	emsxmldb.Device	"查询成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		404				{object}	models.CommonResp	"未找到 {\"code\": 404, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/:rid/device/:did [get]
func GetRtuDevice(c *gin.Context) {
	rtuID := c.Param("rid")
	deviceID := c.Param("did")
	log.Info("GetRtuDevice in ... rid:", rtuID, " did:", deviceID)
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetRtuDevices: GetAllDevicesFromRTU failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	device, err := dbcfg.GetDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		log.Errorf("GetRtuDevices: GetDeviceFromRTU failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("GetRtuDevice success:", device)
	c.JSON(http.StatusOK, device)
}

// @Summary		添加指定rtu下的设备信息
// @Description	添加指定rtu下的设备信息
// @ID				add-rtudevice
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			device	body	emsxmldb.Device	true	"设备信息"
// @Success		200				{object}	models.CommonResp	"添加成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/:rid/device [post]
func AddRtuDevice(c *gin.Context) {
	log.Info("AddRtuDevice in ...")
	rtuID := c.Param("rid")
	dev := emsxmldb.Device{}
	err := c.ShouldBindJSON(&dev)
	if err != nil {
		log.Errorf("AddRtuDevice: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	if dev.Name == "" {
		log.Error("AddRtuDevice device name is empty")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "device name is empty"})
		return
	}

	if !validateOnlyDigits(dev.DeviceAddr) {
		log.Error("AddRtuDevice failed: dev.DeviceAddr is invalid")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "dev.DeviceAddr is invalid"})
		return
	}

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("AddRtuDevice: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = dbcfg.AddDeviceToRTU(rtuID, &dev)
	if err != nil {
		log.Errorf("AddRtuDevice: AddDeviceToRTU failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "AddRtuDevice success"})
}

func AddTransRtuDevice(c *gin.Context) {
	log.Info("AddTransRtuDevice in ...")
	rtuID := c.Param("rid")
	didId := c.Param("did")
	dev := emsxmldb.Device{}
	err := c.ShouldBindJSON(&dev)
	if err != nil {
		log.Errorf("AddTransRtuDevice: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	if dev.Name == "" {
		log.Error("AddTransRtuDevice device name is empty")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "device name is empty"})
		return
	}

	if !validateOnlyDigits(dev.DeviceAddr) {
		log.Error("AddTransRtuDevice failed: dev.DeviceAddr is invalid")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "dev.DeviceAddr is invalid"})
		return
	}

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("AddTransRtuDevice: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = dbcfg.AddTransDeviceToRTU(rtuID, didId, &dev)
	if err != nil {
		log.Errorf("AddTransRtuDevice: AddDeviceToRTU failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "AddTransRtuDevice success"})
}

// @Summary		更新指定rtu下的设备信息
// @Description	更新指定rtu下的设备信息
// @ID				update-rtudevice
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			device	body	emsxmldb.Device	true	"设备信息"
// @Success		200				{object}	models.CommonResp	"更新成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid}/device/{did} [put]
func UpdateRtuDevice(c *gin.Context) {
	rtuID := c.Param("rid")
	log.Info("UpdateRtuDevice in ... rtuID:", rtuID)
	dev := emsxmldb.Device{}
	err := c.ShouldBindJSON(&dev)
	if err != nil {
		log.Errorf("UpdateRtuDevice: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	if dev.Name == "" {
		log.Error("UpdateRtuDevice device name is empty")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "device name is empty"})
		return
	}

	if !validateOnlyDigits(dev.DeviceAddr) {
		log.Error("UpdateRtuDevice failed: dev.DeviceAddr is invalid")
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "dev.DeviceAddr is invalid"})
		return
	}

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("UpdateRtuDevice: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = dbcfg.UpdateDeviceInRTU(rtuID, &dev)
	if err != nil {
		log.Errorf("UpdateRtuDevice: UpdateDeviceInRTU failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "UpdateRtuDevice success"})
}

// @Summary		删除指定rtu下的设备信息
// @Description	删除指定rtu下的设备信息
// @ID				delete-rtudevice
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			did	path	string	true	"Device ID"
// @Success		200				{object}	models.CommonResp	"删除成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid}/device/{did} [delete]
func DeleteRtuDevice(c *gin.Context) {
	rtuID := c.Param("rid")
	deviceID := c.Param("did")
	log.Info("DeleteRtuDevice in ... rid:", rtuID, "did:", deviceID)
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("DeleteRtuDevice: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// 1. remove all points
	err = dbcfg.DeleteAllPointsFromDevice(rtuID, deviceID)
	if err != nil {
		log.Errorf("DeleteRtuDevice: DeleteAllPointsFromDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("DeleteRtuDevice DeleteAllPointsFromDevice success")

	err = dbcfg.DeleteDeviceFromRTU(rtuID, deviceID)
	if err != nil {
		log.Errorf("DeleteRtuDevice: DeleteDeviceFromRTU failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("DeleteRtuDevice success")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "DeleteRtuDevice success"})
}

// @Summary		导出指定rtu下的设备下的所有点信息
// @Description	导出指定rtu下的设备下的所有点信息
// @ID				get-rtudevexcel
// @Tags			dbcfg
// @Param			Authorization	header		string		true	"JWT Token"
// @Param			rid	path	string	true	"RTU ID"
// @Param			did	path	string	true	"Device ID"
// @Success		200	{file}	file	"excel文件"
// @Failure		401	{object}	models.CommonResp
// @Failure		500	{object}	models.CommonResp
// @Router			/api/v1/log/dbcfg/rtu/{rid}/device/{did}/excel [get]
func GetRtuDeviceExcel(c *gin.Context) {
	rtuID := c.Param("rid")
	deviceID := c.Param("did")

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetRtuDeviceExcel: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 1. 生成 Excel 到内存（不落地文件）
	buf, err := dbcfg.ExportAllPointsToBuffer(rtuID, deviceID)
	if err != nil {
		log.Errorf("GetRtuDeviceExcel: ExportAllPointsToBuffer failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 2. 设置下载响应头
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	fileName := "rtu" + rtuID + "_" + "device" + deviceID + ".xlsx"
	c.Header("Content-Disposition", `attachment; filename="`+fileName+`"`)
	c.Header("Content-Length", strconv.Itoa(buf.Len()))

	// 3. 直接把 Excel 二进制流返回前端
	_, err = c.Writer.Write(buf.Bytes())
	if err != nil {
		log.Error("写入响应失败:", err)
		return
	}
}

// @Summary		添加指定rtu下的设备下的点信息
// @Description	添加指定rtu下的设备下的点信息
// @ID				add-rtudevpoints
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			did	path	string	true	"Device ID"
// @Param			points	body	emsxmldb.RtuDevPoints	true	"设备下的点信息"
// @Success		200				{object}	models.CommonResp	"添加成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/:rid/device/:did/points [post]
func AddRtuDevicePoints(c *gin.Context) {
	rtuID := c.Param("rid")
	devID := c.Param("did")
	log.Info("AddRtuDevicePoints in ... rid:", rtuID, ",did:", devID)
	points := emsxmldb.RtuDevPoints{}
	err := c.ShouldBindJSON(&points)
	if err != nil {
		log.Errorf("AddRtuDevicePoints: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("AddRtuDevicePoints: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = dbcfg.AddPointsToDevice(rtuID, devID, points)
	if err != nil {
		log.Errorf("AddRtuDevicePoints: AddPointsToDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("AddRtuDevicePoints AddPointsToDevice success")

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "AddRtuDevicePoints success"})
}

// @Summary		删除指定rtu下的设备下的点信息
// @Description	删除指定rtu下的设备下的点信息
// @ID				delete-rtudevpoints
// @Tags			dbcfg
// @Param			Authorization	header		string		 		true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			did	path	string	true	"Device ID"
// @Success		200				{object}	models.CommonResp	"删除成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid}/device/{did}/points [delete]
func DeleteRtuDevicePoints(c *gin.Context) {
	rtuID := c.Param("rid")
	devID := c.Param("did")
	log.Info("DeleteRtuDevicePoints in ... rid:", rtuID, ",did:", devID)
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("DeleteRtuDevicePoints: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = dbcfg.DeleteAllPointsFromDevice(rtuID, devID)
	if err != nil {
		log.Errorf("DeleteRtuDevicePoints: DeletePointsFromDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	log.Info("DeleteRtuDevicePoints success")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "DeleteRtuDevicePoints success"})
}

func DeleteRtuDevicePointByID(c *gin.Context) {
	rtuID := c.Param("rid")
	devID := c.Param("did")
	log.Info("DeleteRtuDevicePointByID in ... rid:", rtuID, ",did:", devID)

	points := emsxmldb.RtuDevPoints{}
	err := c.ShouldBindJSON(&points)
	if err != nil {
		log.Errorf("DeleteRtuDevicePointByID: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("DeleteRtuDevicePointByID: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	err = dbcfg.DeletePointsFromDevice(rtuID, devID, points)
	if err != nil {
		log.Errorf("DeleteRtuDevicePointByID: DeletePointsFromDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	log.Info("DeleteRtuDevicePointByID success")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "DeleteRtuDevicePointByID success"})
}

func UpdateDevicePointsByID(c *gin.Context) {
	rtuID := c.Param("rid")
	deviceID := c.Param("did")
	log.Info("UpdateDevicePointsByID in ... rid:", rtuID, "did:", deviceID)

	points := emsxmldb.RtuDevPoints{}
	err := c.ShouldBindJSON(&points)
	if err != nil {
		log.Errorf("UpdateDevicePointsByID: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("UpdateDevicePointsByID: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// 1. remove all points
	err = dbcfg.UpdatePointsInDevice(rtuID, deviceID, points)
	if err != nil {
		log.Errorf("UpdateDevicePointsByID: DeleteAllPointsFromDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("UpdateDevicePointsByID DeleteAllPointsFromDevice success")

	log.Info("UpdateDevicePointsByID success")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "UpdateDevicePointsByID success"})
}

// @Summary		获取指定rtu下的设备下的所有点信息
// @Description	获取指定rtu下的设备下的所有点信息
// @ID				get-allpoints
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			did	path	string	true	"Device ID"
// @Success		200				{object}	emsxmldb.RtuDevPoints	"获取成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid}/device/{did}/points [get]
func GetPointsFromDevice(c *gin.Context) {
	rtuID := c.Param("rid")
	devID := c.Param("did")
	log.Info("GetAllPoints in ... rtuID:", rtuID, " devID:", devID)
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetAllPoints: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	allPoints, err := dbcfg.GetPointsFromDevice(rtuID, devID)
	if err != nil {
		log.Errorf("GetAllPoints: GetPointsFromDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, allPoints)
	log.Info("GetAllPoints success")
}

func GetPointsFromDeviceByType(c *gin.Context) {
	rtuID := c.Param("rid")
	devID := c.Param("did")
	dataType := c.Param("type")
	log.Info("GetPointsFromDeviceByType in ... rtuID:", rtuID, " devID:", devID, "dataType:", dataType)
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetAllPoints: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	allPoints, err := dbcfg.GetPointsFromDeviceByType(rtuID, devID, dataType)
	if err != nil {
		log.Errorf("GetAllPoints: GetPointsFromDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, allPoints)
	log.Info("GetAllPoints success")
}

func GetTransPointsByType(c *gin.Context) {
	log.Info("GetTransPointsByType in ")
	reqParam := emsxmldb.GetTransRtuPointsReq{}
	err := c.ShouldBindJSON(&reqParam)
	if err != nil {
		log.Errorf("GetTransPointsByType: ShouldBindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetAllPoints: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	allPoints, err := dbcfg.GetTransPointsByType(reqParam)
	if err != nil {
		log.Errorf("GetAllPoints: GetPointsFromDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, allPoints)
	log.Info("GetAllPoints success")
}

type PointFormula struct {
	Formula emsxmldb.Formula         `json:"formula"`
	Factors []emsxmldb.FormulaFactor `json:"factors"`
}

// @Summary		获取指定rtu下的设备下的指定点信息
// @Description	获取指定rtu下的设备下的指定点信息
// @ID				get-pointformula
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			did	path	string	true	"Device ID"
// @Param			pid	path	string	true	"Point ID"
// @Success		200				{object}	PointFormula	"获取成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid}/device/{did}/point/{pid}/formula [get]
func GetPointFormula(c *gin.Context) {
	rtuID := c.Param("rid")
	devID := c.Param("did")
	pointID := c.Param("pid")
	log.Info("GetPointFormula in ... rtuID:", rtuID, " devID:", devID, " pointID:", pointID)
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetPointFormula: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	formulaID, err := dbcfg.GetPointFormulaFromDevice(rtuID, devID, pointID)
	if err != nil {
		log.Errorf("GetPointFormula: GetPointFormulaFromDevice failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	formula, formulaFactors, err := dbcfg.GetPointFormula(formulaID)
	if err != nil {
		log.Errorf("GetPointFormula: GetPointFormula failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	result := PointFormula{Formula: formula, Factors: formulaFactors}

	log.Info("GetPointFormula success:", result)
	c.JSON(http.StatusOK, result)
}

// @Summary		更新指定指定点的公式信息
// @Description	更新指定指定点的公式信息
// @ID				update-pointformula
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			rid	path	string	true	"RTU ID"
// @Param			did	path	string	true	"Device ID"
// @Param			pid	path	string	true	"Point ID"
// @Param			pointformula	body	PointFormula	true	"Point Formula"
// @Success		200				{object}	models.CommonResp	"更新成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/rtu/{rid}/device/{did}/point/{pid}/formula [post]
func UpdatePointFormula(c *gin.Context) {
	rtuID := c.Param("rid")
	devID := c.Param("did")
	pointID := c.Param("pid")
	log.Info("UpdatePointFormula in ... rtuID:", rtuID, " devID:", devID, " pointID:", pointID)

	var pointFormula PointFormula
	err := c.BindJSON(&pointFormula)
	if err != nil {
		log.Errorf("UpdatePointFormula: BindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("UpdatePointFormula: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	err = dbcfg.UpdatePointFormula(rtuID, devID, pointID, pointFormula.Formula, pointFormula.Factors)
	if err != nil {
		log.Errorf("UpdatePointFormula: UpdatePointFormula failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "UpdatePointFormula success"})
	log.Info("UpdatePointFormula success")
}

type CheckFormulaReq struct {
	Formula string             `json:"formula"`
	Params  map[string]float64 `json:"params"`
}

type CheckFormulaResp struct {
	Result float64 `json:"result"`
	Err    string  `json:"err"`
}

// @Summary		检查公式
// @Description	检查公式
// @ID				check-formula
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			checkformula	body	CheckFormulaReq	true	"Check Formula"
// @Success		200				{object}	CheckFormulaResp	"检查成功 如果err为空，result值为计算结果；如果err不为空，则表示公式错误，内容为错误信息。"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/formula/check [post]
func CheckFormula(c *gin.Context) {
	var req CheckFormulaReq
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("CheckFormula: BindJSON failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	var varNames []string
	var varValues []float64
	for k, v := range req.Params {
		varNames = append(varNames, k)
		varValues = append(varValues, v)
	}

	resultValue, err := cgolibparse.EqEval(req.Formula, varNames, varValues)
	if err != nil {
		log.Errorf("CheckFormula: EqEval failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, &CheckFormulaResp{Result: resultValue})
	log.Info("CheckFormula success")
}

type AgcBindPoint struct {
	PointID string `json:"pointID"`
	Name    string `json:"name"`
	MqttKey string `json:"mqttkey"`
}

// @Summary		根据点类型获取所有点信息
// @Description	根据点类型获取所有点信息
// @ID				get-points-by-type
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			ptype	path	string	true	"测点类型，遥信yx 遥测yc 遥脉ym 遥调yt 遥控yk"
// @Success		200				{object}	[]emsxmldb.AgcBindPoint	"获取成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/points/{ptype} [get]
func GetPointsByType(c *gin.Context) {
	ptype := c.Param("ptype")
	log.Info("GetPointsByType in ... ptype:", ptype)
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("GetPointsByType: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	allPoints, err := dbcfg.GetAllPointsByType(ptype)
	if err != nil {
		log.Errorf("GetPointsByType: GetAllPointsByType failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	log.Info("GetPointsByType success")

	c.JSON(http.StatusOK, allPoints)
}

// @Summary		根据点类型获取所有点信息
// @Description	根据点类型获取所有点信息
// @ID				get-points-by-type
// @Tags			dbcfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			ptype	path	string	true	"测点类型，遥信yx 遥测yc 遥脉ym 遥调yt 遥控yk"
// @Success		200				{object}	[]emsxmldb.AgcBindPoint	"获取成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/dbcfg/points/{ptype} [get]
func ImportPointsFromExcel(c *gin.Context) {
	rtuID := c.Param("rid")
	deviceID := c.Param("did")

	log.Info("ImportPointsFromExcel in")
	dbcfg, err := emsxmldb.GetDbConfigManager()
	if err != nil {
		log.Errorf("ImportPointsFromExcel: GetDbConfigManager failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	if err := dbcfg.ImportPointsFromExcel(c.Request.Body, rtuID, deviceID); err != nil {
		log.Errorf("ImportPointsFromExcel: ImportPointsFromExcel failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "ImportPointsFromExcel success"})
}
