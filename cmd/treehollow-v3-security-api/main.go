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
	"treehollow-v3-backend/pkg/route/security"
	"treehollow-v3-backend/pkg/utils"
)

func main() {
	logger.InitLog(consts.SecurityApiLogFile)
	config.InitConfigFile()

	//if false == viper.GetBool("is_debug") {
	//	fmt.Print("Read salt from stdin: ")
	//	_, _ = fmt.Scanln(&utils.Salt)
	//	if utils.SHA256(utils.Salt) != viper.GetString("salt_hashed") {
	//		panic("salt verification failed!")
	//	}
	//}
	utils.Salt = viper.GetString("salt")

	base.InitDb()
	base.AutoMigrateDb()

	utils.InitGeoDbRefreshCron()

	log.Println("start time: ", time.Now().Format("01-02 15:04:05"))
	if false == viper.GetBool("is_debug") {
		gin.SetMode(gin.ReleaseMode)
	}

	security.ApiListenHttp()
}
