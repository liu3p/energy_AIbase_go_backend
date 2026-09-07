package handlers

import (
	"net/http"
	"webserver-arm/homepage"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-gonic/gin"
)

// GetHomepageConfig 查询首页配置
// @Summary     查询首页配置
// @Description 返回完整首页配置，有 OID 的条目已补充 XML 中的 name 和 unit
// @Tags        homepage
// @Produce     json
// @Param       Authorization header string true "JWT Token"
// @Success     200 {object} homepage.HomepageConfig
// @Router      /api/v1/log/homepage/config [get]
func GetHomepageConfig(c *gin.Context) {
	log.Info("GetHomepageConfig in...")
	mgr, err := homepage.GetConfigManager()
	if err != nil {
		log.Error("GetHomepageConfig failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{
			Code:    http.StatusInternalServerError,
			Message: "homepage config manager init failed",
		})
		return
	}

	config := mgr.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": config,
	})
}

// UpdateHomepageTopology 修改拓扑配置
// @Summary     修改拓扑配置
// @Description 接收完整的 topology 数组，no 去重后保存
// @Tags        homepage
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "JWT Token"
// @Param       items body []homepage.ConfigItem true "topology 配置数组"
// @Success     200 {object} models.CommonResp
// @Router      /api/v1/log/homepage/config/topology [post]
func UpdateHomepageTopology(c *gin.Context) {
	updateHomepageSection(c, "topology")
}

// UpdateHomepageRealtime 修改实时数据配置
// @Summary     修改实时数据配置
// @Tags        homepage
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "JWT Token"
// @Param       items body []homepage.ConfigItem true "realtime 配置数组"
// @Success     200 {object} models.CommonResp
// @Router      /api/v1/log/homepage/config/realtime [post]
func UpdateHomepageRealtime(c *gin.Context) {
	updateHomepageSection(c, "realtime")
}

// UpdateHomepageBasicInfo 修改基础信息配置
// @Summary     修改基础信息配置
// @Tags        homepage
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "JWT Token"
// @Param       items body []homepage.ConfigItem true "basic_info 配置数组"
// @Success     200 {object} models.CommonResp
// @Router      /api/v1/log/homepage/config/basic_info [post]
func UpdateHomepageBasicInfo(c *gin.Context) {
	updateHomepageSection(c, "basic_info")
}

// UpdateHomepagePowerLevel 修改电力等级配置
// @Summary     修改电力等级配置
// @Tags        homepage
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "JWT Token"
// @Param       items body []homepage.ConfigItem true "power_level 配置数组"
// @Success     200 {object} models.CommonResp
// @Router      /api/v1/log/homepage/config/power_level [post]
func UpdateHomepagePowerLevel(c *gin.Context) {
	updateHomepageSection(c, "power_level")
}

// updateHomepageSection 通用的 section 更新逻辑
func updateHomepageSection(c *gin.Context, section string) {
	log.Info("UpdateHomepageSection", section, "in...")

	var items []homepage.ConfigItem
	if err := c.ShouldBindJSON(&items); err != nil {
		log.Error("parse request body failed:", err.Error())
		c.JSON(http.StatusBadRequest, models.CommonResp{
			Code:    http.StatusBadRequest,
			Message: "invalid request body",
		})
		return
	}

	mgr, err := homepage.GetConfigManager()
	if err != nil {
		log.Error("UpdateHomepageSection failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{
			Code:    http.StatusInternalServerError,
			Message: "homepage config manager init failed",
		})
		return
	}

	if err := mgr.UpdateSection(section, items); err != nil {
		log.Error("update section failed:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{
			Code:    http.StatusInternalServerError,
			Message: "update section failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.CommonResp{Code: 200, Message: "success"})
}
