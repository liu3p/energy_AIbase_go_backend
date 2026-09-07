package main

import (
	"embed"
	"html/template"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	//"localweb/db"
	"webserver-arm/emsclient"
	"webserver-arm/emsxmldb"
	"webserver-arm/handlers"
	"webserver-arm/homepage"
	"webserver-arm/log"
	"webserver-arm/routes"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

func loadI18NData() bool {
	fileI18n, err := os.Open("data/clientlan.txt")
	if err != nil {
		log.Error("open data/clientlan.txt failed. err:", err.Error())
		return false
	}
	defer fileI18n.Close()

	dataI18n, err := io.ReadAll(fileI18n)
	if err != nil {
		log.Error("read data/clientlan.txt failed. err:", err.Error())
		return false
	}

	err = json.Unmarshal(dataI18n, handlers.I18nData)
	if err != nil {
		log.Error("read data/clientlan.txt failed. err:", err.Error())
		return false
	}

	return true
}

func loadUnitData() bool {
	fileUnit, err := os.Open("data/dataunit.txt")
	if err != nil {
		log.Error("open data/dataunit.txt failed. err:", err.Error())
		return false
	}
	defer fileUnit.Close()

	dataUnit, err := io.ReadAll(fileUnit)
	if err != nil {
		log.Error("read data/dataunit.txt failed. err:", err.Error())
		return false
	}

	err = json.Unmarshal(dataUnit, handlers.DataUnit)
	if err != nil {
		log.Error("read data/dataunit.txt failed. err:", err.Error())
		return false
	}

	err = json.Unmarshal(dataUnit, handlers.DataUnitEx)
	if err != nil {
		log.Error("read data/dataunit.txt failed. err:", err.Error())
		return false
	}
	return true
}

func loadI18NDataMb() bool {
	fileI18n, err := os.Open("data/clientlan_mb.txt")
	if err != nil {
		log.Error("open data/clientlan_mb.txt failed. err:", err.Error())
		return false
	}
	defer fileI18n.Close()

	dataI18n, err := io.ReadAll(fileI18n)
	if err != nil {
		log.Error("read data/clientlan_mb.txt failed. err:", err.Error())
		return false
	}

	err = json.Unmarshal(dataI18n, &handlers.I18nDataModbus)
	if err != nil {
		log.Error("read data/clientlan_mb.txt failed. err:", err.Error())
		return false
	}

	return true
}

func loadUnitDataMb() bool {
	fileUnit, err := os.Open("data/dataunit_mb.txt")
	if err != nil {
		log.Error("open data/dataunit_mb.txt failed. err:", err.Error())
		return false
	}
	defer fileUnit.Close()

	dataUnit, err := io.ReadAll(fileUnit)
	if err != nil {
		log.Error("read data/dataunit_mb.txt failed. err:", err.Error())
		return false
	}

	err = json.Unmarshal(dataUnit, &handlers.DataUnitModbus)
	if err != nil {
		log.Error("read data/dataunit_mb.txt failed. err:", err.Error())
		return false
	}

	return true
}

func loadLoginRd() bool {
	fileUnit, err := os.Open("data/loginrecord.txt")
	if err != nil {
		log.Warn("open data/loginrecord.txt failed. err:", err.Error())
		return false
	}
	defer fileUnit.Close()

	dataUnit, err := io.ReadAll(fileUnit)
	if err != nil {
		log.Error("read data/loginrecord.txt failed. err:", err.Error())
		return false
	}

	err = json.Unmarshal(dataUnit, &handlers.LoginRecordData)
	if err != nil {
		log.Error("read data/loginrecord.txt failed. err:", err.Error())
		return false
	}
	return true
}

func loadOverViewTemplate() bool {
	ovModel, err := os.Open("data/ovmodel.txt")
	if err != nil {
		log.Error("open data/ovmodel.txt failed. err:", err.Error())
		return false
	}
	defer ovModel.Close()

	ovModelData, err := io.ReadAll(ovModel)
	if err != nil {
		log.Error("read data/ovmodel.txt failed. err:", err.Error())
		return false
	}

	err = json.Unmarshal(ovModelData, &handlers.AllModelInfo)
	if err != nil {
		log.Error("read data/ovmodel.txt failed. err:", err.Error())
		return false
	}
	return true
}

func loadData() {
	loadI18NData()
	loadI18NDataMb()
	loadUnitData()
	loadUnitDataMb()
	loadLoginRd()
	loadOverViewTemplate()
}

// CGO_LDFLAGS="-L${ARM_SF}/lib/scada -L${ARM_SF}/lib" go build -o arm-web-backend

var APP_VERSION string

func main() {
	logger := log.InitLog()
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Info("logger sync failed")
		}
	}()
	log.Info("start ...")

	loadData()

	var err error
	handlers.ClientMgr, err = emsclient.NewManager( /*[]string{ *serverEndpoint }*/ )
	if err != nil {
		panic(err)
	}

	// 初始化首页推送管理器，注入广播回调
	homepage.GetHomepageManager(func(topic string, data interface{}) {
		handlers.ClientMgr.PushMsg(&emsclient.PushResp{Topic: topic, Content: data})
	})

	r := gin.Default()
	//pprof.Register(r)
	loadResources(r)

	store := cookie.NewStore([]byte("localwebserver"))
	store.Options(sessions.Options{
		HttpOnly: true,
		MaxAge:   3600, // keep 15min
		Path:     "/",
	})
	r.Use(sessions.Sessions("mySession", store))
	routes.LoadRoutes(r) //加载api接口

	log.Info("local web version is :", APP_VERSION)

	err = r.Run(":38080")
	if err != nil {
		log.Error("run server failed:", err.Error())
	}
}

//go:embed dist/*
var webFS embed.FS

//go:embed dist/assets/*
var staticFS embed.FS

// 加载前端资源
func loadResources(r *gin.Engine) {
	r.SetHTMLTemplate(template.Must(template.New("").ParseFS(webFS, "dist/*.html")))
	r.Use(func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		if !strings.HasPrefix(urlPath, "/api") && !strings.HasPrefix(urlPath, "/swagger") {
			if urlPath == "/config.js" {
				file, err := webFS.ReadFile("dist/config.js")
				if err != nil {
					c.AbortWithStatus(http.StatusNotFound)
					return
				}
				c.Header("Content-Type", mime.TypeByExtension(strings.ToLower(path.Ext("config.js"))))
				c.Data(http.StatusOK, "", file)
			} else if strings.HasPrefix(urlPath, "/assets") {
				file, err := staticFS.ReadFile("dist" + urlPath)
				if err != nil {
					c.AbortWithStatus(http.StatusNotFound)
					return
				}
				c.Header("Content-Type", mime.TypeByExtension(strings.ToLower(path.Ext(urlPath))))
				c.Data(http.StatusOK, "", file)
			} else if urlPath == "/template.xlsx" {
				file, err := webFS.ReadFile("dist/template.xlsx")
				if err != nil {
					c.AbortWithStatus(http.StatusNotFound)
					return
				}
				c.Header("Content-Type", mime.TypeByExtension(strings.ToLower(path.Ext("template.xlsx"))))
				c.Data(http.StatusOK, "", file)
			} else {
				c.HTML(http.StatusOK, "index.html", gin.H{})
			}
			return
		}
		c.Next()
	})
}
