package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"thuhole-go-backend/pkg/config"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/logger"
	"thuhole-go-backend/pkg/route"
	"thuhole-go-backend/pkg/utils"
	"time"
)

func main() {
	logger.InitLog(consts.LoginApiLogFile)
	config.InitConfigFile()

	fmt.Print("Read salt from stdin: ")
	_, _ = fmt.Scanln(&utils.Salt)
	if utils.Hash1(utils.Salt) != viper.GetString("salt_hashed") {
		panic("salt verification failed!")
	}

	db.InitDb()

	log.Println("start time: ", time.Now().Format("01-02 15:04:05"))
	if false == viper.GetBool("is_debug") {
		gin.SetMode(gin.ReleaseMode)
	}

	route.LoginApiListenHttp()
}
