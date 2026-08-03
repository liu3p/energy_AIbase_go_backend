package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var loginSecret string

var tokenExpireTime = 60       // token有效期, min
var refreshExpireTime = 60 * 7 // refresh token有效期, min

var OnlineUsers sync.Map

// 用于存储失效令牌的黑名单
var blacklistedTokens sync.Map

const (
	UNameKey = "username"
	UTypeKey = "usertype"
)

func init() {
	// 生成长度为 32 的随机字符串作为 loginSecret
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic("Error generating login secret: " + err.Error())
	}
	loginSecret = base64.URLEncoding.EncodeToString(randomBytes)

	// TODO: consider to remove it
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   3600,
		Path:     "/",
	})

	// 启动一个携程清理无用的stringToken
	go func() {
		for {
			time.Sleep(time.Hour)
			blacklistedTokens.Range(func(key, value interface{}) bool {
				expTime, ok := value.(int64)
				if !ok {
					log.Error("blacklistedTokens value not int")
					return true
				}

				if time.Now().Unix() > expTime {
					stringToken, ok := key.(string)
					if !ok {
						log.Error("blacklistedTokens key not string")
						return true
					}
					blacklistedTokens.Delete(stringToken)
					log.Info("token removed from blacklistedTokens, stringToken:", stringToken, ",expTime:", expTime)
				}

				return true
			})
		}
	}()
}

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 鉴权逻辑，例如检查Token

		// failed  c.Abort()
		stringToken := c.GetHeader("Authorization")
		if stringToken == "" {
			log.Debug("get stringToken from url params, 1111.")
			stringToken = c.Query("authorization")
			log.Debug("get stringToken from url params, authorization:", stringToken)
		}
		stringToken = strings.TrimPrefix(stringToken, "Bearer ")

		if _, ok := blacklistedTokens.Load(stringToken); ok {
			c.Abort()
			log.Info("stringToken in blacklistedTokens, stringToken:", stringToken)
			c.JSON(http.StatusUnauthorized, models.CommonResp{Code: http.StatusUnauthorized, Message: "Unauthorized"})
			return
		}

		if valid, claims := VerifyToken(stringToken); !valid {
			c.Abort()
			log.Info("VerifyToken err")
			c.JSON(http.StatusUnauthorized, models.CommonResp{Code: http.StatusUnauthorized, Message: "Unauthorized"})
			return
		} else {
			c.Set(UNameKey, claims[UNameKey])
			c.Set(UTypeKey, claims[UTypeKey])
		}

		c.Next()
	}
}

func GetToken(user *models.UserInfo) (string, int64, string, int64, error) {
	// claims := jwt.MapClaims{
	// 	UNameKey: user.UserName,
	// 	//"exp":     time.Now().Unix() + int64(maxInActiveTime), // time.Now().Add(time.Microsecond * 10).Unix(), //
	// 	"exp": float64(time.Now().Add(time.Hour).Unix()),
	// 	"nbf": time.Now().Unix(),
	// 	"iat": time.Now().Unix(),
	// }

	// token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// expTime, err := claims.GetExpirationTime()
	// if err != nil {
	// 	return "", nil, err
	// }
	// tokenString, err := token.SignedString([]byte(loginSecret))
	// if err != nil {
	// 	return "", nil, err
	// }

	// return tokenString, expTime, nil

	data := map[string]any{UNameKey: user.UserName, UTypeKey: user.UserType}

	tokenString, expTime, err := generateTokens(data, time.Duration(tokenExpireTime)*time.Minute)
	if err != nil {
		return "", 0, "", 0, err
	}

	refreshTokenString, refreshTokenExpTime, err := generateTokens(data, time.Duration(refreshExpireTime)*time.Minute)
	if err != nil {
		return "", 0, "", 0, err
	}

	return tokenString, expTime, refreshTokenString, refreshTokenExpTime, nil
}

func generateTokens(data map[string]any, exp time.Duration) (string, int64, error) {
	// 生成 Token
	tokenClaims := jwt.MapClaims{
		"exp": float64(time.Now().Add(exp).Unix()),
		"nbf": time.Now().Unix(),
		"iat": time.Now().Unix(),
	}
	for k, v := range data {
		tokenClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	tokenString, err := token.SignedString([]byte(loginSecret))
	if err != nil {
		return "", 0, err
	}
	expTime, err := tokenClaims.GetExpirationTime()
	if err != nil {
		return "", 0, err
	}

	return tokenString, expTime.UnixMilli(), nil
}

func VerifyToken(tokenString string) (bool, jwt.MapClaims) {
	if tokenString == "" {
		return false, nil
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Errorf("VerifyToken unexpected signing method: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(loginSecret), nil
	})
	if err != nil {
		log.Error("VerifyToken parse err:", err.Error())
		return false, nil
	}

	//同一个用户登录两次，没有覆盖？
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Error("VerifyToken Claims not MapClaims!")
		return false, nil
	}
	if !token.Valid {
		log.Errorf("VerifyToken uName:", claims[UNameKey], " token is invalid. exp:", claims["exp"], ", token:", token.Valid)
		return false, nil
	}
	return true, claims
}

func RefreshToken(tokenString string) (string, int64, error) {
	// token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
	// 	// Don't forget to validate the alg is what you expect:
	// 	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
	// 		return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
	// 	}
	// 	return []byte(loginSecret), nil
	// })
	// if err != nil {
	// 	log.Error("verify Parse err:", err.Error())
	// 	return "", 0, err
	// }

	// if !token.Valid {
	// 	return "", 0, errors.New("token invalid")
	// }

	// claims, clmok := token.Claims.(jwt.MapClaims)
	// if !clmok {
	// 	return "", 0, errors.New("claims invalid")
	// }

	if _, ok := blacklistedTokens.Load(tokenString); ok {
		log.Info("tokenString in blacklistedTokens, tokenString:", tokenString)
		return "", 0, errors.New("token invalid")
	}

	ok, claims := VerifyToken(tokenString)
	if !ok {
		return "", 0, errors.New("token invalid")
	}
	uName, unameok := claims[UNameKey].(string)
	if !unameok {
		return "", 0, errors.New("uName invalid")
	}

	uType, unameok := claims[UTypeKey].(string)
	if !unameok {
		return "", 0, errors.New("uType invalid")
	}
	data := map[string]any{UNameKey: uName, UTypeKey: uType}
	tokenString, expTime, err := generateTokens(data, time.Duration(tokenExpireTime)*time.Minute)
	if err != nil {
		return "", 0, err
	}

	return tokenString, expTime, nil
}

func InvalidateToken(tokenString string, expTime int64) {
	blacklistedTokens.Store(tokenString, expTime+1000)
}

func GetSession(c *gin.Context) sessions.Session {
	session := sessions.Default(c)
	// no need to Options because we have done store.Options to make a base configuration.
	// session.Options(sessions.Options{
	// 	HttpOnly: true,
	// 	MaxAge:   900,
	// 	Path:     "/",
	// })
	return session
}
