package main

import (
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
	"log"
)

func main() {
	initLog()
	initConfigFile()
	initDb()
	log.Println("start timestamp: ", getTimeStamp())
	if false == viper.GetBool("is_debug") {
		gin.SetMode(gin.ReleaseMode)
	}

	var err error
	hotPosts, _ = dbGetHotPosts()
	c := cron.New()
	_, _ = c.AddFunc("*/1 * * * *", func() {
		hotPosts, err = dbGetHotPosts()
		//log.Println("refreshed hotPosts ,err=", err)
	})
	c.Start()

	listenHttp()

}
