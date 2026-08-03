package emsxmldb

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"webserver-arm/log"
	"webserver-arm/models"
)

/*
<?xml version="1.0" encoding="UTF-8"?>
<Users>
    <User>
        <Id>user_001</Id>
        <Name>超级管理员小明</Name>
        <Password>e10adc3949ba59abbe56e057f20f883e</Password> <!-- 示例 MD5 加密后密码，实际建议用 bcrypt -->
        <Type>1</Type> <!-- 角色类型：1=SA, 2=Config, 3=Ops, 4=Guest -->
        <Role>sa</Role> <!-- 角色英文名称（冗余字段，提升可读性） -->
    </User>
    <User>
        <Id>user_002</Id>
        <Name>配置员小李</Name>
        <Password>e10adc3949ba59abbe56e057f20f883e</Password>
        <Type>2</Type>
        <Role>config</Role>
    </User>
</Users>
*/

const (
	UserTypeAdmin  = "admin"  // 超级管理员
	UserTypeConfig = "config" // 配置人员
	UserTypeOper   = "ops"    // 运维人员
	UserTypeMonit  = "guest"  // 访客
)

var (
	xmlFileMutex sync.Mutex
	userXmlPath  = filepath.Join(getConfigBasePath(), "/webdata/user.xml") //"data/user.xml"
)

var typeToRole = map[int]string{
	1: UserTypeAdmin,  // 超级管理员
	2: UserTypeConfig, // 配置人员
	3: UserTypeOper,   // 运维人员
	4: UserTypeMonit,  // 访客
}

// User 单个用户节点结构体
type User struct {
	Id       string `xml:"Id"`       // 用户ID
	Name     string `xml:"Name"`     // 用户名
	Password string `xml:"Password"` // 密码
	Type     int    `xml:"Type"`     // 角色类型（Type 字段，1=sa/2=config/3=ops/4=guest）
	Role     string `xml:"Role"`     // 角色英文名称（冗余字段，提升 XML 可读性）
}

type Users struct {
	XMLName xml.Name `xml:"Users"`        // 根节点名称
	Version string   `xml:"version,attr"` // 可选：XML 配置版本（便于后续兼容升级）
	User    []User   `xml:"User"`         // 多个用户节点
}

// type UserInfo struct {
// 	Id   string `xml:"Users"`
// 	Type int    `xml:"column:type;type:int(4)"`
// 	Name string `xml:"column:name;type:char(64)"`
// 	Pwd  string `xml:"column:password;type:char(64)"`
// }

// type UserType struct {
// 	UserTypeName string `gorm:"column:角色名;type:char(24)"`
// 	UserTypeId   int    `gorm:"column:操作员类型;type:int(4)"`
// }

// login
type LoginRet struct {
	Result   int // 1 is ok
	ErrInfo  string
	UserType string
}

func ReadUsersFromXML() (Users, error) {
	xmlFileMutex.Lock()
	defer xmlFileMutex.Unlock()
	// 读取文件内容
	data, err := os.ReadFile(userXmlPath)
	if err != nil {
		// 如果文件不存在，返回空用户列表（而非错误，便于首次使用）
		if errors.Is(err, os.ErrNotExist) {
			return Users{Version: "1.0"}, nil
		}
		return Users{}, fmt.Errorf("read xml failed:%w", err)
	}

	// 反序列化 XML 到结构体
	var users Users
	err = xml.Unmarshal(data, &users)
	if err != nil {
		return Users{}, fmt.Errorf("XML Unmarshal failed:%w", err)
	}

	// 校验反序列化后的角色信息合法性
	for _, user := range users.User {
		if _, ok := typeToRole[user.Type]; !ok {
			return Users{}, fmt.Errorf("User %s(ID%s) :Type=%d is invalid, valid roles: %v", user.Name, user.Id, user.Type, typeToRole)
		}
	}

	return users, nil
}

func WriteUsersToXML(users Users) error {
	for _, user := range users.User {
		// check role valid
		expectedRole, typeValid := typeToRole[user.Type]
		if !typeValid || expectedRole != user.Role {
			return fmt.Errorf("User %s(ID:%s):Type=%d, Role=%s is invalid. valid roles: %v", user.Name, user.Id, user.Type, user.Role, typeToRole)
		}
	}

	xmlData, err := xml.MarshalIndent(users, "", "    ")
	if err != nil {
		return fmt.Errorf("XML MarshalIndent failed: %w", err)
	}

	// 手动添加 XML 声明（xml.MarshalIndent 不会自动生成）
	fullXML := append([]byte(xml.Header), xmlData...)

	xmlFileMutex.Lock()
	defer xmlFileMutex.Unlock()
	err = os.WriteFile(userXmlPath, fullXML, 0644) // 权限：所有者读写，其他只读
	if err != nil {
		return fmt.Errorf("WriteUsersToXML WriteFile faield: %w", err)
	}

	return nil
}

func Login(username, userpwd string) *LoginRet {
	ret := &LoginRet{}

	users, err := ReadUsersFromXML()
	if err != nil {
		log.Error("Login ReadUsersFromXML err:", err)
		ret.ErrInfo = "get users info err: " + err.Error()
		return ret
	}

	hash := md5.Sum([]byte(userpwd))
	hashPwd := strings.ToUpper(hex.EncodeToString(hash[:]))

	for _, user := range users.User {
		if user.Name == username {
			if user.Password != hashPwd {
				ret.ErrInfo = "username or password error"
				log.Error("username:", username, " password error")
				return ret
			}
			ret.Result = 1
			ret.UserType = typeToRole[user.Type]
			return ret
		}
	}

	ret.ErrInfo = "username or password error"
	log.Error("username:", username, " not exist in xml.")
	return ret
}

// UpdatePwd
type UpdatePwdRet struct {
	Result  int // 0 is ok
	ErrInfo string
}

func updatePwd(username, userNewPwd string) UpdatePwdRet {
	ret := UpdatePwdRet{}

	users, err := ReadUsersFromXML()
	if err != nil {
		log.Error("updatePwd ReadUsersFromXML err:", err)
		ret.ErrInfo = "UpdatePwd get users info err: " + err.Error()
		ret.Result = 1
		return ret
	}

	hash := md5.Sum([]byte(userNewPwd))
	hashPwd := strings.ToUpper(hex.EncodeToString(hash[:]))

	for idx, user := range users.User {
		if user.Name == username {
			users.User[idx].Password = hashPwd
			break
		}
	}
	err = WriteUsersToXML(users)
	if err != nil {
		log.Error("updatePwd WriteUsersToXML err:", err)
		ret.ErrInfo = "UpdatePwd WriteUsersToXML err: " + err.Error()
		ret.Result = 1
		return ret
	}
	log.Info("updatePwd ok:", username)
	ret.Result = 0
	return ret
}

func UpdatePwd(username, userOldPwd, userNewPwd string) UpdatePwdRet {
	loginRet := Login(username, userOldPwd)
	if loginRet.Result != 1 {
		log.Error("UpdatePwd login err:", loginRet.ErrInfo)
		return UpdatePwdRet{ErrInfo: loginRet.ErrInfo, Result: 1}
	}

	return updatePwd(username, userNewPwd)
}

func ResetPwd(username string) UpdatePwdRet {
	return updatePwd(username, "sfereems")
}

func GetAllUsers() []models.UserInfo {
	users, err := ReadUsersFromXML()
	if err != nil {
		log.Error("GetAllUsers ReadUsersFromXML err:", err)
		return nil
	}

	userInfo := make([]models.UserInfo, 0)
	for _, user := range users.User {
		if user.Type == 1 {
			continue
		}
		userInfo = append(userInfo, models.UserInfo{
			UserName:  user.Name,
			UserType:  typeToRole[user.Type],
			LastLogin: "-",
			Status:    0,
		})
	}
	return userInfo
}
