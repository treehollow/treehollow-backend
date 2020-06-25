package main

import (
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
	"log"
	"thuhole-go-backend/pkg/config"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/logger"
	"thuhole-go-backend/pkg/route"
	"thuhole-go-backend/pkg/utils"
)

func main() {
	logger.InitLog()
	config.InitConfigFile()
	db.InitDb()
	log.Println("start timestamp: ", utils.GetTimeStamp())
	if false == viper.GetBool("is_debug") {
		gin.SetMode(gin.ReleaseMode)
	}

	var err error
	route.HotPosts, _ = db.DbGetHotPosts()
	c := cron.New()
	_, _ = c.AddFunc("*/1 * * * *", func() {
		route.HotPosts, err = db.DbGetHotPosts()
		//log.Println("refreshed hotPosts ,err=", err)
	})
	c.Start()

	route.ListenHttp()

}
