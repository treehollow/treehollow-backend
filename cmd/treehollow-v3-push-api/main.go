package main

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/config"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/push"
)

func main() {
	logger.InitLog(consts.PushApiLogFile)
	config.InitConfigFile()

	base.InitDb()
	base.AutoMigrateDb()

	log.Println("start time: ", time.Now().Format("01-02 15:04:05"))
	if false == viper.GetBool("is_debug") {
		gin.SetMode(gin.ReleaseMode)
	}

	push.ApiListenHttp()
}
