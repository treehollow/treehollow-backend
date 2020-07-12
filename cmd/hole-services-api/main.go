package main

import (
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"log"
	"thuhole-go-backend/pkg/config"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/logger"
	"thuhole-go-backend/pkg/route"
	"time"
)

func main() {
	logger.InitLog(consts.ServicesApiLogFile)
	config.InitConfigFile()

	db.InitDb()

	log.Println("start time: ", time.Now().Format("01-02 15:04:05"))
	if false == viper.GetBool("is_debug") {
		gin.SetMode(gin.ReleaseMode)
	}

	route.HotPosts, _ = db.GetHotPosts()
	c := cron.New()
	_, _ = c.AddFunc("*/1 * * * *", func() {
		route.HotPosts, _ = db.GetHotPosts()
		//log.Println("refreshed hotPosts ,err=", err)
	})
	c.Start()

	route.ServicesApiListenHttp()
}
