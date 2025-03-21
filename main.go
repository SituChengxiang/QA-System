package main

import (
	global "QA-System/internal/global/config"
	"QA-System/internal/middleware"
	"QA-System/internal/pkg/database/mongodb"
	"QA-System/internal/pkg/database/mysql"
	"QA-System/internal/pkg/log"
	"QA-System/internal/pkg/session"
	"QA-System/internal/pkg/utils"
	"QA-System/internal/router"
	"QA-System/internal/service"
	"QA-System/pkg/extension"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 如果配置文件中开启了调试模式
	if !global.Config.GetBool("server.debug") {
		gin.SetMode(gin.ReleaseMode)
	}
	// 初始化日志系统
	log.ZapInit()
	// 初始化数据库
	db := mysql.Init()
	mdb := mongodb.Init()
	// 初始化dao
	service.Init(db, mdb)
	if err := utils.Init(); err != nil {
		zap.L().Fatal(err.Error())
	}

	// 初始化插件管理器并加载插件
	// pm := extension.NewPluginManager(zap.NewExample())
	pm := extension.GetDefaultManager()
	_, err := pm.LoadPlugins()
	if err != nil {
		zap.L().Error("Error loading plugins", zap.Error(err))
	}
	err = pm.ExecutePlugins()
	if err != nil {
		zap.L().Error("Error executing plugins", zap.Error(err))
	}

	// 初始化gin
	r := gin.Default()
	r.Use(middleware.ErrHandler())
	r.NoMethod(middleware.HandleNotFound)
	r.NoRoute(middleware.HandleNotFound)
	r.Static("public/static", "./public/static")
	r.Static("public/xlsx", "./public/xlsx")
	session.Init(r)
	router.Init(r)
	err = r.Run(":" + global.Config.GetString("server.port"))
	if err != nil {
		zap.L().Fatal("Failed to start the server:" + err.Error())
	}
}
