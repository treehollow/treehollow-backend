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
	"treehollow-v3-backend/pkg/route/contents"
)

func main() {
	logger.InitLog(consts.ServicesApiLogFile)
	config.InitConfigFile()

	base.InitDb()
	base.AutoMigrateDb()

	log.Println("start time: ", time.Now().Format("01-02 15:04:05"))
	if false == viper.GetBool("is_debug") {
		gin.SetMode(gin.ReleaseMode)
	}

	//utils.InitGeoDbRefreshCron()
	contents.RefreshHotPosts()
	contents.InitHotPostsRefreshCron()

	contents.ServicesApiListenHttp()
}
