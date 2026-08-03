package handlers

import (
	"net/http"
	"strconv"
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

type ChannelPlugin struct {
	AppPlugins  []emsxmldb.Plugin `json:"appplugin"`
	LinkPlugins []emsxmldb.Plugin `json:"linkplugin"`
}

// @Summary		获取通道组下的通道插件
// @Description	获取通道组下的通道插件
// @ID				get-channel-plugins
// @Tags			fecfg
// @Param			Authorization	header		string	true	"JWT Token"	default(<token>)
// @Success		200				{object}	ChannelPlugin	 "获取成功"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401
// @Failure		500				{object}	models.CommonResp	"服务内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/fecfg/plugins [get]
func GetChannelPlugins(c *gin.Context) {
	log.Info("GetChannelPlugins in ...")
	appPlugins, linkPlugins, err := emsxmldb.GetChannelPlugins()
	if err != nil {
		log.Error("GetChannelPlugins error: %v", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: "GetChannelPlugins failed"})
		return
	}
	log.Info("GetChannelPlugins success:", appPlugins, linkPlugins)
	result := ChannelPlugin{AppPlugins: appPlugins, LinkPlugins: linkPlugins}
	c.JSON(http.StatusOK, result)
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
