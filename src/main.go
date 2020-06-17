package main

import (
	"github.com/gin-gonic/gin"
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
	listenHttp()

}
