package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"webserver-arm/cgolibparse"
	"webserver-arm/emsxmldb"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
)

// @Summary		获取所有通道组
// @Description	获取所有通道组
// @ID				get-channelgroups
// @Tags			fecfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	[]emsxmldb.ChannelGroup	 "获取成功 "
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\":
// @Router			/api/v1/log/fecfg/cgroups [get]
func GetChannelGroups(c *gin.Context) {
	log.Info("GetChannelGroups in ...")

	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("GetChannelGroups GetFeConfigXMLManager error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "GetChannelGroups failed"})
		return
	}

	chGroups := fecfg.ListAllChannelGroups()
	log.Info("GetChannelGroups chGroups:", chGroups)
	c.JSON(http.StatusOK, chGroups)
}

// @Summary		新增通道组
// @Description	新增通道组
// @ID				add-channelgroup
// @Tags			fecfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			data			body		emsxmldb.ChannelGroupParam	true	"通道组参数"
// @Success		200				{object}	models.CommonResp	 "添加成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/cgroup [post]
func AddChannelGroup(c *gin.Context) {
	log.Info("AddChannelGroup in ...")
	chGroup := emsxmldb.ChannelGroupParam{}
	if err := c.ShouldBindJSON(&chGroup); err != nil {
		log.Error("AddChannelGroup c.ShouldBindJSON error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "data not match ChannelGroupParam json"})
		return
	}

	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("AddChannelGroup GetFeConfigXMLManager error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "AddChannelGroup failed"})
		return
	}
	err = fecfg.CreateChannelGroup(chGroup)
	if err != nil {
		log.Error("AddChannelGroup error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "AddChannelGroup failed"})
		return
	}
	log.Info("AddChannelGroup chGroup success:", chGroup)
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "AddChannelGroup success"})
}

// @Summary		修改通道组
// @Description	修改通道组
// @ID				update-channelgroup
// @Tags			fecfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			cgid			path		string				true	"通道组ID"
// @Param			data			body		emsxmldb.ChannelGroupParam	true	"通道组参数"
// @Success		200				{object}	models.CommonResp	 "修改成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/cgroup/{cgid} [put]
func UpdateChannelGroup(c *gin.Context) {
	cgID := c.Param("cgid")
	log.Info("UpdateChannelGroup in ... cgID:", cgID)
	chGroup := emsxmldb.ChannelGroupParam{}
	if err := c.ShouldBindJSON(&chGroup); err != nil {
		log.Error("c.ShouldBindJSON error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "data not match ChannelGroupParam"})
		return
	}
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("UpdateChannelGroup GetFeConfigXMLManager error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "UpdateChannelGroup failed"})
		return
	}
	chGroupID, err := strconv.Atoi(cgID)
	if err != nil {
		log.Error("UpdateChannelGroup strconv.Atoi error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "channel group ID not int"})
		return
	}

	err = fecfg.UpdateChannelGroup(chGroupID, chGroup)
	if err != nil {
		log.Error("UpdateChannelGroup error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "UpdateChannelGroup failed"})
		return
	}
	log.Info("UpdateChannelGroup chGroup:", chGroup)
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "UpdateChannelGroup success"})
}

// @Summary		删除通道组
// @Description	删除通道组
// @ID				delete-channelgroup
// @Tags			fecfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			cgid			path		string				true	"通道组ID"
// @Success		200				{object}	models.CommonResp	 "删除成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/cgroup/{cgid} [delete]
func DeleteChannelGroup(c *gin.Context) {
	cgID := c.Param("cgid")
	log.Info("DeleteChannelGroup in ... cgID:", cgID)
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("DeleteChannelGroup GetFeConfigXMLManager error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "DeleteChannelGroup failed"})
		return
	}
	chGroupID, err := strconv.Atoi(cgID)
	if err != nil {
		log.Error("DeleteChannelGroup strconv.Atoi error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "channel group ID not int"})
		return
	}
	err = fecfg.DeleteChannelGroup(chGroupID)
	if err != nil {
		log.Error("DeleteChannelGroup error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "DeleteChannelGroup failed"})
		return
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "DeleteChannelGroup success"})
}

// @Summary		获取通道组下的通道
// @Description	获取通道组下的通道
// @ID				get-cg-channels
// @Tags			fecfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			cgid			path		string				true	"通道组ID"
// @Success		200				{object}	models.CommonResp	 "获取成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/cgroup/{cgid}/channels [get]
func GetCGChannels(c *gin.Context) {
	cgID := c.Param("cgid")
	log.Info("GetCGChannels in ... cgID:", cgID)
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("GetCGChannels GetFeConfigXMLManager error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "GetCGChannels failed"})
		return
	}

	chGroupID, err := strconv.Atoi(cgID)
	if err != nil {
		log.Error("GetCGChannels strconv.Atoi error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "channel group ID not int"})
		return
	}
	channels, err := fecfg.ListChannelsByCGID(chGroupID)
	if err != nil {
		log.Error("GetCGChannels ListChannelsByCGID error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "GetCGChannels failed"})
		return
	}
	log.Info("GetCGChannels channels:", channels)
	c.JSON(http.StatusOK, channels)
}

// @Summary		添加通道组下的通道
// @Description	添加通道组下的通道
// @ID				add-cg-channel
// @Tags			fecfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			cgid			path		string				true	"通道组ID"
// @Param			data			body		emsxmldb.ChannelParam	true	"通道参数"
// @Success		200				{object}	models.CommonResp	 "添加成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/cgroup/{cgid}/channel [post]
func AddCGChannel(c *gin.Context) {
	cgID := c.Param("cgid")
	log.Info("AddCGChannel in ... cgID:", cgID)
	channel := emsxmldb.ChannelParam{}
	if err := c.ShouldBindJSON(&channel); err != nil {
		log.Error("c.ShouldBindJSON error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "data not match ChannelParam json"})
		return
	}
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("AddCGChannel GetFeConfigXMLManager error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "AddChannelGroup failed"})
		return
	}
	chGroupID, err := strconv.Atoi(cgID)
	if err != nil {
		log.Error("AddCGChannel strconv.Atoi error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "channel group ID not int"})
		return
	}
	err = fecfg.CreateChannel(chGroupID, channel)
	if err != nil {
		if errors.Is(err, emsxmldb.ErrPluginLayer) {
			log.Error("AddCGChannel plugin layer invalid: %v", err)
			c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: err.Error()})
			return
		}
		log.Error("AddCGChannel error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "AddChannelGroup failed"})
		return
	}
	log.Info("AddCGChannel channel:", channel)
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "AddChannelGroup success"})
}

// @Summary		更新通道组下的通道
// @Description	更新通道组下的通道
// @ID				update-cg-channel
// @Tags			fecfg
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			cgid			path		string				true	"通道组ID"
// @Param			cid				path		string				true	"通道ID"
// @Param			data			body		emsxmldb.ChannelParam	true	"通道参数"
// @Success		200				{object}	models.CommonResp	 "更新成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/cgroup/{cgid}/channel/{cid} [put]
func UpdateCGChannel(c *gin.Context) {
	cgID := c.Param("cgid")
	channelID := c.Param("cid")
	log.Info("UpdateCGChannel in ... cgID:", cgID, ", channelID:", channelID)
	channel := emsxmldb.ChannelParam{}
	if err := c.ShouldBindJSON(&channel); err != nil {
		log.Error("UpdateCGChannel ShouldBindJSON error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "data not match ChannelParam json"})
		return
	}
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("UpdateCGChannel GetFeConfigXMLManager error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "UpdateChannelGroup failed"})
		return
	}

	chGroupID, err := strconv.Atoi(cgID)
	if err != nil {
		log.Error("GetCGChannels strconv.Atoi error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "channel group ID not int"})
		return
	}

	err = fecfg.UpdateChannel(chGroupID, channel)
	if err != nil {
		if errors.Is(err, emsxmldb.ErrPluginLayer) {
			log.Error("UpdateCGChannel plugin layer invalid: %v", err)
			c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: err.Error()})
			return
		}
		log.Error("UpdateCGChannel error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "UpdateChannelGroup failed"})
		return
	}
	log.Info("UpdateCGChannel channel:", channel)
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "UpdateChannelGroup success"})
}

// @Summary		删除通道组下的通道
// @Description	删除通道组下的通道
// @ID				delete-cg-channel
// @Tags			fecfg
// @Param			Authorization	header		string	true	"JWT Token"	default(<token>)
// @Param			cgid			path		string	true	"通道组ID"
// @Param			cid				path		string	true	"通道ID"
// @Success		200				{object}	models.CommonResp	 "删除成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/cgroup/{cgid}/channel/{cid} [delete]
func DeleteCGChannel(c *gin.Context) {
	log.Info("DeleteCGChannel in ...")
	cgID := c.Param("cgid")
	cid := c.Param("cid")
	log.Info("DeleteCGChannel in ... cgID:", cgID, ", channelID:", cid)
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Error("DeleteCGChannel GetFeConfigXMLManager error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "DeleteChannelGroup failed"})
		return
	}
	chGroupID, err := strconv.Atoi(cgID)
	if err != nil {
		log.Error("DeleteCGChannel strconv.Atoi error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "channel group ID not int"})
		return
	}
	channelID, err := strconv.Atoi(cid)
	if err != nil {
		log.Error("DeleteCGChannel strconv.Atoi error: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "channel ID not int"})
		return
	}
	err = fecfg.DeleteChannel(chGroupID, channelID)
	if err != nil {
		log.Error("DeleteCGChannel error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "DeleteChannelGroup failed"})
		return
	}
	log.Info("DeleteCGChannel success chGroupID:", chGroupID, ", channelID:", channelID)
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "DeleteChannelGroup success"})
}

// @Summary		获取通道协议插件（按层分组）
// @Description	获取通道协议插件（按层分组）
// @ID				get-channel-plugins
// @Tags			fecfg
// @Param			Authorization	header		string	true	"JWT Token"	default(<token>)
// @Success		200				{object}	[]emsxmldb.PluginLayer	 "获取成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/plugins [get]
func GetChannelPlugins(c *gin.Context) {
	log.Info("GetChannelPlugins in ...")
	layers, err := emsxmldb.GetChannelPlugins()
	if err != nil {
		log.Error("GetChannelPlugins error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "GetChannelPlugins failed"})
		return
	}
	// 每层的协议按 parameters 排序（拷贝，不修改原始数据）
	for i := range layers {
		layers[i].Plugins = sortChannelPlugins(layers[i].Plugins)
	}
	log.Info("GetChannelPlugins success:", layers)
	c.JSON(http.StatusOK, layers)
}

// @Summary 通知前置重新加载配置
// @Description 通知前置重新加载配置
// @ID				get-channel-reload
// @Tags			fecfg
// @Param			Authorization	header		string	true	"JWT Token"	default(<token>)
// @Success		200				{object}	models.CommonResp	 "通知成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/notifyreload [post]
func NotifyFeConfigReload(c *gin.Context) {
	log.Info("NotifyFeConfigReload in ...")
	err := cgolibparse.NotifyFeServerReload()
	if err != nil {
		log.Errorf("NotifyFeConfigReload error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "NotifyFeServerReload failed"})
		return
	}
	log.Info("NotifyFeConfigReload success")
	c.JSON(http.StatusOK, models.CommonResp{Code: http.StatusOK, Message: "NotifyFeServerReload success"})
}

// ChannelRestartItem 通道组维度的重启请求/结果
type ChannelRestartItem struct {
	ChannelGroupID int   `json:"channel_group_id"` // 通道组 ID
	ChannelIDs     []int `json:"channel_ids"`      // 该通道组下的通道 ID 列表
}

// RestartChannelsResponse 重启响应
type RestartChannelsResponse struct {
	Success []ChannelRestartItem `json:"success"`
	Fail    []ChannelRestartItem `json:"fail"`
}

// execCommandOutput 执行命令并返回输出
func execCommandOutput(cmd string) (string, error) {
	command := exec.Command("sh", "-c", cmd)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

// findFeChannelPID 通过 server_id 查找 fe_channel 进程的 PID
// 进程名格式: fe_channel <fedb_id> <server_id>
func findFeChannelPID(serverID string) (string, error) {
	cmd := fmt.Sprintf("ps aux | grep 'fe_channel' | grep ' %s$' | grep -v grep | awk '{print $2}'", serverID)
	output, err := execCommandOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("ps command failed: %w", err)
	}
	pid := strings.TrimSpace(output)
	if pid == "" {
		return "", fmt.Errorf("fe_channel process with server_id %s not found", serverID)
	}
	return pid, nil
}

// killProcess 强制杀死指定 PID 的进程
func killProcess(pid string) error {
	return execCommand(fmt.Sprintf("kill -9 %s", pid))
}

// @Summary		重启指定通道（通过杀死 fe_channel 进程）
// @Description	重启指定通道，杀死 fe_channel 进程后由看门狗自动重启
// @ID				restart-channels
// @Tags			fecfg
// @Param			Authorization	header		string					true	"JWT Token"	default(<token>)
// @Param			data			body		[]ChannelRestartItem		true	"重启请求"
// @Success		200				{object}	RestartChannelsResponse	"重启完成"
// @Failure		400				{object}	models.CommonResp		"请求体格式错误 {\"code\": 400, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp		"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp		"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/channels/restart [post]
func RestartChannels(c *gin.Context) {
	var req []ChannelRestartItem
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("RestartChannels: ShouldBindJSON failed: %v", err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	if len(req) == 0 {
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "request body is required"})
		return
	}

	// 校验每个通道组的参数
	for i, item := range req {
		if item.ChannelGroupID < 0 {
			log.Errorf("RestartChannels: item[%d].channel_group_id is invalid: %d", i, item.ChannelGroupID)
			c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: fmt.Sprintf("item[%d].channel_group_id must be >= 0", i)})
			return
		}
		if len(item.ChannelIDs) == 0 {
			log.Errorf("RestartChannels: item[%d].channel_ids is empty", i)
			c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: fmt.Sprintf("item[%d].channel_ids is required and cannot be empty", i)})
			return
		}
		for j, chID := range item.ChannelIDs {
			if chID < 0 {
				log.Errorf("RestartChannels: item[%d].channel_ids[%d] is invalid: %d", i, j, chID)
				c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: fmt.Sprintf("item[%d].channel_ids[%d] must be >= 0, got %d", i, j, chID)})
				return
			}
		}
	}

	// 获取 feconfig，用于根据 channel_group_id + channel_id 查找 SERVER_GROUP_ID
	fecfg, err := emsxmldb.GetFeConfigXMLManager()
	if err != nil {
		log.Errorf("RestartChannels: GetFeConfigXMLManager failed: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	// 按通道组维度收集成功和失败的通道
	resp := RestartChannelsResponse{
		Success: make([]ChannelRestartItem, 0),
		Fail:    make([]ChannelRestartItem, 0),
	}
	successMap := make(map[int][]int)
	failMap := make(map[int][]int)

	for _, item := range req {
		for _, chID := range item.ChannelIDs {
			// 通过 channel_group_id + channel_id 查找通道的 SERVER_GROUP_ID（即 server_id）
			ch, err := fecfg.GetChannelByID(item.ChannelGroupID, chID)
			if err != nil {
				log.Errorf("RestartChannels: GetChannelByID(%d, %d) failed: %v", item.ChannelGroupID, chID, err)
				failMap[item.ChannelGroupID] = append(failMap[item.ChannelGroupID], chID)
				continue
			}
			serverID := strconv.Itoa(ch.ServerGroupID)

			// 查找 fe_channel 进程 PID
			pid, err := findFeChannelPID(serverID)
			if err != nil {
				log.Errorf("RestartChannels: findFeChannelPID(%s) failed: %v", serverID, err)
				failMap[item.ChannelGroupID] = append(failMap[item.ChannelGroupID], chID)
				continue
			}

			// kill -9 强制杀死进程，c_fe_channel_manage 看门狗会自动重启
			log.Infof("RestartChannels killing fe_channel process: pid=%s, server_id=%s", pid, serverID)
			if err := killProcess(pid); err != nil {
				log.Errorf("RestartChannels kill process %s failed: %v", pid, err)
				failMap[item.ChannelGroupID] = append(failMap[item.ChannelGroupID], chID)
				continue
			}

			successMap[item.ChannelGroupID] = append(successMap[item.ChannelGroupID], chID)
			log.Infof("RestartChannels group %d channel %d killed (server %s, pid %s), watchdog will restart", item.ChannelGroupID, chID, serverID, pid)
		}
	}

	// 将 map 转换为数组
	for cgID, chIDs := range successMap {
		resp.Success = append(resp.Success, ChannelRestartItem{ChannelGroupID: cgID, ChannelIDs: chIDs})
	}
	for cgID, chIDs := range failMap {
		resp.Fail = append(resp.Fail, ChannelRestartItem{ChannelGroupID: cgID, ChannelIDs: chIDs})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "restart channels completed",
		"data":    resp,
	})
}
