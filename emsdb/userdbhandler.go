package emsdb

type UserInfo struct {
	Id   string `gorm:"column:id;type:char(64)"`
	Type int    `gorm:"column:type;type:int(4)"`
	Name string `gorm:"column:name;type:char(64)"`
	Pwd  string `gorm:"column:password;type:char(64)"`
}

func (UserInfo) TableName() string {
	return "userinfo"
}

type UserType struct {
	UserTypeName string `gorm:"column:角色名;type:char(24)"`
	UserTypeId   int    `gorm:"column:操作员类型;type:int(4)"`
}

func (UserType) TableName() string {
	return "角色权限参数表"
}

// login
type LoginRet struct {
	Result   int // 1 is ok
	ErrInfo  string
	UserType string
}

// func Login(username, userpwd string) *LoginRet {
// 	ret := &LoginRet{}

// 	userdb, err := db.NewMsDB()
// 	if err != nil {
// 		log.Info("Login open db err:", err)
// 		ret.ErrInfo = "Login open db err: " + err.Error()
// 		return ret
// 	}

// 	var userinfo []UserInfo
// 	tx := userdb.Where("name = ?", username).Find(&userinfo)
// 	if tx.Error != nil {
// 		log.Error("Login Find userinfo err:", tx.Error)
// 		ret.ErrInfo = "Login Find userinfo err: " + tx.Error.Error()
// 		return ret
// 	}
// 	if len(userinfo) == 0 {
// 		log.Error("username:", username, " not exist in db.")
// 		ret.ErrInfo = "username: " + username + " not exist in db."
// 		return ret
// 	}
// 	hash := md5.Sum([]byte(userpwd))
// 	hashString := strings.ToUpper(hex.EncodeToString(hash[:]))

// 	if hashString != userinfo[0].Pwd {
// 		ret.ErrInfo = "password error"
// 		return ret
// 	}

// 	var usertype []UserType
// 	tx = userdb.Where("操作员类型 = ?", userinfo[0].Type).Find(&usertype)
// 	if tx.Error != nil {
// 		log.Error("Login Find usertype err:", tx.Error)
// 		ret.ErrInfo = "Login Find usertype err: " + tx.Error.Error()
// 		return ret
// 	}
// 	if len(usertype) == 0 {
// 		log.Error("usertype:", userinfo[0].Type, " not exist in db.")
// 		ret.ErrInfo = fmt.Sprintf("usertype id: %d not exist in db.", userinfo[0].Type)
// 		return ret
// 	}

// 	ret.Result = 1
// 	ret.UserType = usertype[0].UserTypeName
// 	return ret
// }

// // UpdatePwd
// type UpdatePwdRet struct {
// 	Result  int // 0 is ok
// 	ErrInfo string
// }

// func updatePwd(username, userNewPwd string) UpdatePwdRet {
// 	ret := UpdatePwdRet{}
// 	userdb, err := db.NewMsDB()
// 	if err != nil {
// 		log.Info("UpdatePwd open db err:", err)
// 		ret.ErrInfo = "UpdatePwd open db err: " + err.Error()
// 		ret.Result = 1
// 		return ret
// 	}

// 	hash := md5.Sum([]byte(userNewPwd))
// 	hashString := strings.ToUpper(hex.EncodeToString(hash[:]))
// 	data := make(map[string]interface{})
// 	data["password"] = hashString
// 	tx, err := userdb.Updates(&UserInfo{}, data, "name = ?", username)
// 	if err != nil {
// 		log.Error("UpdatePwd Update userinfo err:", err)
// 		ret.ErrInfo = "UpdatePwd Update userinfo err: " + err.Error()
// 		ret.Result = 1
// 		return ret
// 	}

// 	log.Info("UpdatePwd Update userinfo ok:", tx.RowsAffected, ", username:", username)
// 	ret.Result = 0
// 	return ret
// }

// func UpdatePwd(username, userOldPwd, userNewPwd string) UpdatePwdRet {
// 	loginRet := Login(username, userOldPwd)
// 	if loginRet.Result != 1 {
// 		log.Error("UpdatePwd login err:", loginRet.ErrInfo)
// 		return UpdatePwdRet{ErrInfo: loginRet.ErrInfo, Result: 1}
// 	}

// 	return updatePwd(username, userNewPwd)
// }

// func ResetPwd(username string) UpdatePwdRet {
// 	return updatePwd(username, "sfereems")
// }
