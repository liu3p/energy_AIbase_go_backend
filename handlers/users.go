package handlers

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
	"webserver-arm/emsclient"
	"webserver-arm/emsxmldb"
	"webserver-arm/log"
	"webserver-arm/middleware"
	"webserver-arm/models"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func init() {
	//OnlineUsers = make(map[string]OnlineUser)
}

const (
	UserTypeAdmin  = "admin"  // 超级管理员
	UserTypeConfig = "config" // 配置人员
	UserTypeOper   = "ops"    // 运维人员
	UserTypeMonit  = "guest"  // 访客
)

var ClientMgr *emsclient.Manager

//nolint:govet
type OnlineUser struct {
	Uname           string
	Utype           string
	State           int // 0 uninit, 1 login, 2 websocket connected(normal)
	Pwd             string
	AccessToken     string
	AccessTokenExp  int64
	RefreshToken    string
	RefreshTokenExp int64
	PushConn        *websocket.Conn
	LastUpdateCache int64
}

var OnlineUsers sync.Map //map[string]OnlineUser
// var OnlineTypes sync.Map

type LoginReq struct {
	Uname string `json:"username"`
	Pwd   string `json:"password"`
}

//nolint:govet
type LoginResp struct {
	// Uname        string `json:"username"`
	// Utype        string `json:"usertype"`
	AccessToken  string `json:"access_token"`
	Expires      int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func removeUserInfo(uName string) {
	if data, ok := OnlineUsers.LoadAndDelete(uName); ok {
		onlines, match := data.(*OnlineUser)
		if !match {
			return
		}
		middleware.InvalidateToken(onlines.AccessToken, onlines.AccessTokenExp)
		middleware.InvalidateToken(onlines.RefreshToken, onlines.RefreshTokenExp)
		KickOutWsByUname(uName)
	} else {
		log.Warnf("removeUserInfo uName:%s not exist", uName)
	}
	log.Info("removeUserInfo uName:", uName)
}

// @Summary		用户登录
// @Description	用户使用提供的用户名和密码进行登录
// @ID				user-login
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			loginReq	body		LoginReq			true	"登录请求参数"
// @Success		200			{object}	LoginResp			"登录成功 {\"access_token\": \"\", \"expires_in\": 0, \"refresh_token\":\"\", \"token_type\":\"\"}"
// @Failure		400			{object}	models.CommonResp	"用户名密码错误  {\"code\": 400, \"msg\": \"\"}"
// @Failure		409			{object}	models.CommonResp	"用户已登录 {\"code\": 409, \"msg\": \"conflict: already login\"}"
// @Failure		500			{object}	models.CommonResp	"内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Failure		503			{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/user/login [post]
func LoginUser(c *gin.Context) {
	loginreq := &LoginReq{}
	err := c.ShouldBindJSON(loginreq)
	if err != nil {
		log.Error("LoginUser ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}
	uName := loginreq.Uname
	pwd := loginreq.Pwd

	log.Info("LoginUser in... uname:", uName, ",pwd:", pwd, ",c:", c.Request.Form, ",userinfo:", loginreq)

	logResp := &emsxmldb.LoginRet{}
	if uName != UserTypeAdmin {
		logResp = emsxmldb.Login(uName, pwd)
	} else {
		if pwd == "superman" {
			logResp.Result = 1
			logResp.UserType = UserTypeAdmin
		} else {
			logResp.ErrInfo = "invalid admin pwd"
		}
	}

	if logResp.Result == 1 {
		// check conflict
		if data, ok := OnlineUsers.Load(uName); ok {
			onlines, match := data.(*OnlineUser)
			if !match {
				log.Info("data not OnlineUser.")
				c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
					Message: "Internal Service Error"})
				return
			}
			if onlines.State == 2 { // 在线
				log.Info("LoginUser CONFLICT. uname:", uName)
				loginResp := &models.CommonResp{Code: http.StatusConflict, Message: "conflict: already login"}
				c.JSON(http.StatusConflict, loginResp)
				return
			}
			// 此时ws不在线，前用户被踢
			log.Info("LoginUser CONFLICT but not online, kick previous. uname:", uName)
			removeUserInfo(uName)
		}

		userType := logResp.UserType
		token, expTime, refreshToken, refreshExpTime, err := middleware.GetToken(
			&models.UserInfo{UserName: uName, UserType: userType})
		if err != nil {
			log.Error("get token faild:", err.Error())
			c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
				Message: "Internal Service Error"})
			return
		}

		log.Debug("store userName:", uName, "userType:", userType)
		OnlineUsers.Store(uName, &OnlineUser{Uname: uName,
			Utype:           userType,
			State:           1,
			Pwd:             pwd,
			AccessToken:     token,
			AccessTokenExp:  expTime,
			RefreshToken:    refreshToken,
			RefreshTokenExp: refreshExpTime})

		loginResp := &LoginResp{AccessToken: token, Expires: expTime, RefreshToken: refreshToken, TokenType: "Bearer"}
		c.JSON(http.StatusOK, loginResp)
		recordLastLogin(uName)
		return
	}
	log.Error("login failed:", logResp.ErrInfo, logResp.Result)
	c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: "invalid user or password"})
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token"`
}

//nolint:govet
type RefreshTokenResp struct {
	AccessToken  string `json:"access_token"`
	Expires      int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// @Summary		更新token
// @Description	用户更新过期的token
// @ID				refresh-token
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			RefreshTokenReq	body		RefreshTokenReq		true	"token"	default(<token>)
// @Success		200				{object}	RefreshTokenResp	"更新成功 {\"access_token\": \"\", \"expires_in\": 0, \"refresh_token\":\"\", \"token_type\":\"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Router			/api/v1/user/refresh_token [post]
func RefreshToken(c *gin.Context) {
	tokenInfo := &RefreshTokenReq{}
	err := c.ShouldBindJSON(tokenInfo)
	if err != nil {
		log.Error("RefreshToken ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}
	refreshToken := strings.TrimPrefix(tokenInfo.RefreshToken, "Bearer ")
	log.Debug("RefreshToken in... tokenInfo:", tokenInfo)

	newToken, expTime, err := middleware.RefreshToken(refreshToken)
	if err != nil {
		log.Error("RefreshToken failed:", err.Error())
		c.JSON(http.StatusUnauthorized, models.CommonResp{Code: http.StatusUnauthorized, Message: "Unauthorized"})
		return
	}

	// 更新用户token
	uName := c.GetString(middleware.UNameKey)
	if data, ok := OnlineUsers.Load(uName); ok {
		onlines, match := data.(*OnlineUser)
		if match {
			onlines.AccessToken = newToken
			onlines.AccessTokenExp = expTime
			OnlineUsers.Store(uName, onlines)
		}
	}
	c.JSON(http.StatusOK, RefreshTokenResp{AccessToken: newToken,
		Expires:      expTime,
		RefreshToken: refreshToken,
		TokenType:    "Bearer"})
}

// @Summary		用户登出
// @Description	用户退出登录
// @ID				user-logout
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	models.CommonResp	"登出成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Router			/api/v1/log/user/logout [post]
func LogoutUser(c *gin.Context) {
	// session := middleware.GetSession(c)
	// session.Clear()
	// session.Save()
	uName := c.GetString(middleware.UNameKey)

	c.JSON(http.StatusOK, gin.H{})

	removeUserInfo(uName)
	log.Info("LogoutUser over. uName:", uName)
}

// @Summary		用户信息
// @Description	获取用户信息
// @ID				user-info
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	models.UserInfo		"获取成功 {\"username\": \"config\", \"usertype\": \"configurer\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Router			/api/v1/log/user/info [get]
func UserInfo(c *gin.Context) {
	uName := c.GetString(middleware.UNameKey)
	uType := c.GetString(middleware.UTypeKey)
	log.Info("UserInfo in ...", "uName:", uName, ",uType:", uType)
	c.JSON(http.StatusOK, models.UserInfo{UserName: uName, UserType: uType})
}

type UpdatePwdReq struct {
	OldPwd string `json:"oldpwd"`
	NewPwd string `json:"newpwd"`
}

// 检查密码是否包含至少一个指定类型的字符
func containsType(password string, charTypes ...func(rune) bool) bool {
	for _, char := range password {
		for _, charType := range charTypes {
			if charType(char) {
				return true
			}
		}
	}
	return false
}

// 校验密码函数
func validatePassword(password string) bool {
	// 密码长度校验
	if len(password) < 8 || len(password) > 24 {
		return false
	}

	isLetter := func(r rune) bool { return unicode.IsLetter(r) }
	isDigit := func(r rune) bool { return unicode.IsDigit(r) }
	isSymbol := func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsMark(r)
	}

	// 检查是否包含至少两种类型的字符
	typesCount := 0
	if containsType(password, isLetter) {
		typesCount++
	}
	if containsType(password, isDigit) {
		typesCount++
	}
	if containsType(password, isSymbol) {
		typesCount++
	}

	return typesCount >= 2
}

// @Summary		修改密码
// @Description	修改用户自己的密码
// @ID				update-pwd
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			UpdatePwdReq	body		UpdatePwdReq		true	"登录请求参数"
// @Success		200				{object}	models.CommonResp	"修改密码成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		400				{object}	models.CommonResp	"参数错误 {\"code\": 400, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		412				{object}	models.CommonResp	"旧密码不正确 {\"code\": 412, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/user/updatepwd [post]
func UpdatePwd(c *gin.Context) {
	log.Info("UpdatePwd in...")
	//updatePwdInfo := make(map[string]string)
	updatePwdInfo := &UpdatePwdReq{}
	err := c.ShouldBindJSON(&updatePwdInfo)
	if err != nil {
		log.Error("UpdatePwd ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}
	oldPwd := updatePwdInfo.OldPwd
	newPwd := updatePwdInfo.NewPwd
	uName := c.GetString(middleware.UNameKey)
	log.Info("oldPwd:", oldPwd, ",newpwd:", newPwd)
	// check new pwd
	if !validatePassword(newPwd) {
		log.Error("not validate pwd:", newPwd)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest,
			Message: "The password must be 8 to 24 characters, including at least two types of letters, numbers, and symbols"})
		return
	}

	ret := emsxmldb.UpdatePwd(uName, oldPwd, newPwd)
	if ret.Result != 0 {
		log.Error("UpdatePwd for user:", uName, " result:", ret.Result, ", err:", ret.ErrInfo)
		c.JSON(http.StatusPreconditionFailed, models.CommonResp{Code: http.StatusPreconditionFailed,
			Message: "UpdatePwd failed:" + ret.ErrInfo})
		return
	}

	log.Info("UpdatePwd success for user:", uName)

	c.JSON(http.StatusOK, models.CommonResp{Code: 0, Message: "UpdatePwd success"})

	removeUserInfo(uName)
}

type ResetPwdReq struct {
	Uname string `json:"username"`
}

// @Summary		重置密码
// @Description	超管重置用户密码
// @ID				reset-pwd
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			ResetPwdReq		body		ResetPwdReq			true	"踢下线用户名"
// @Success		200				{object}	models.CommonResp	"重置密码成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/user/resetpwd [post]
func ResetPwd(c *gin.Context) {
	log.Info("ResetPwd in...")

	userType := c.GetString(middleware.UTypeKey)
	if userType != UserTypeAdmin {
		log.Error("ResetPwd unaccessable for user:", c.GetString(middleware.UNameKey))
		c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "Admin access required"})
		return
	}

	resetPwdInfo := &ResetPwdReq{}
	err := c.ShouldBindJSON(&resetPwdInfo)
	if err != nil {
		log.Error("ResetPwd ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}
	uName := resetPwdInfo.Uname
	ret := emsxmldb.ResetPwd(uName)
	if ret.Result != 0 {
		log.Error("ResetPwd for user:", uName, " failed:", ret.ErrInfo)
		c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden,
			Message: "ResetPwd failed:" + ret.ErrInfo})

		return
	}
	log.Info("ResetPwd success for user:", uName)
	c.JSON(http.StatusOK, models.CommonResp{Code: 0, Message: ""})
}

type KickOutReq struct {
	Uname string `json:"username"`
}

// @Summary		踢在线用户下线
// @Description	超管踢在线用户下线
// @ID				kickout-user
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Param			KickOutReq		body		KickOutReq			true	"踢下线用户名"
// @Success		200				{object}	models.CommonResp	"重置密码成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问，仅允许超级管理员调用 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/user/kickout [post]
func KickOutUser(c *gin.Context) {
	log.Info("KickOutUser in...")

	userType := c.GetString(middleware.UTypeKey)
	if userType != UserTypeAdmin { // TODO  judge 超管
		log.Error("KickOutUser unaccessable for user:", c.GetString(middleware.UNameKey))
		c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "Admin access required"})
		return
	}

	kickOutInfo := &KickOutReq{}
	err := c.ShouldBindJSON(kickOutInfo)
	if err != nil {
		log.Error("KickOutUser ShouldBindJSON err:", err.Error())
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError,
			Message: "Internal Service Error"})
		return
	}
	uName := kickOutInfo.Uname
	removeUserInfo(uName)

	log.Info("KickOutUser kickout user:", uName)

	c.JSON(http.StatusOK, models.CommonResp{Code: 0, Message: ""})
}

// @Summary		查询在线用户信息
// @Description	超管查询在线用户信息
// @ID				onlines-user
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	[]models.UserInfo	"查询成功 [{\"username\":\"oper-2\", \"usertype\":\"操作\"}]
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问，仅允许超级管理员调用 {\"code\": 403, \"msg\": \"\"}"
// @Failure		503				{object}	models.CommonResp	"服务不可访问 {\"code\": 503, \"msg\": \"\"}"
// @Router			/api/v1/log/user/onlines [get]
func OnlinesUser(c *gin.Context) {
	log.Info("OnlinesUser in...")

	userType := c.GetString(middleware.UTypeKey)
	if userType != UserTypeAdmin {
		log.Error("OnlinesUser unaccessable for user:", c.GetString(middleware.UNameKey))
		c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "Admin access required"})
		return
	}

	onlines := make([]models.UserInfo, 0)
	OnlineUsers.Range(func(key, value interface{}) bool {
		uname, ok := key.(string)
		if !ok {
			return true
		}
		onlineData, ok := value.(*OnlineUser)
		if !ok {
			return true
		}
		onlines = append(onlines, models.UserInfo{UserName: uname, UserType: onlineData.Utype})
		log.Debug("Key: %v, Value: %v\n", key, value)

		// 返回 true 继续遍历，返回 false 停止遍历
		return true
	})

	c.JSON(http.StatusOK, onlines)
}

func recordLastLogin(uName string) error {
	LoginRecordData[uName] = time.Now().Format("2006-01-02 15:04:05")
	filePath := "data/loginrecord.txt"

	// 要写入的内容
	content, err := json.Marshal(LoginRecordData)
	if err != nil {
		log.Error("recordLastLogin Marshal failed:", err.Error())
		return err
	}

	// 使用os.WriteFile直接写入内容
	err = os.WriteFile(filePath, content, 0644) // 0644 是文件权限设置
	if err != nil {
		log.Error("recordLastLogin WriteFile failed:", err.Error())
		return err
	}
	return nil
}

var LoginRecordData = make(map[string]string) // username , date

// @Summary		获取全部用户信息
// @Description	获取全部用户信息
// @ID				all-user-info
// @Tags			user
// @Accept			json
// @Produce		json
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	[]models.UserInfo	"获取成功 [{\"username\": \"config\", \"usertype\": \"configurer\"}, {\"username\": \"oper\", \"usertype\": \"operator\"}]"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		403				{object}	models.CommonResp	"拒绝访问，仅允许超级管理员调用 {\"code\": 403, \"msg\": \"\"}"
// @Router			/api/v1/log/user/all [get]
func AllUsers(c *gin.Context) {
	log.Info("AllUsers in ...")

	userType := c.GetString(middleware.UTypeKey)
	if userType != UserTypeAdmin {
		log.Error("AllUsers unaccessable for user:", c.GetString(middleware.UNameKey))
		c.JSON(http.StatusForbidden, models.CommonResp{Code: http.StatusForbidden, Message: "Admin access required"})
		return
	}

	users := emsxmldb.GetAllUsers()

	for _, user := range users {
		user.LastLogin = LoginRecordData[user.UserName]
	}

	OnlineUsers.Range(func(key, value interface{}) bool {
		uname, ok := key.(string)
		if !ok {
			return true
		}

		for idx, user := range users {
			if user.UserName == uname {
				users[idx].Status = 1
			}
		}

		// 返回 true 继续遍历，返回 false 停止遍历
		return true
	})

	c.JSON(http.StatusOK, users)
}
