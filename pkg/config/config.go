package config

import (
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/utils"
)

func refreshAllowedSubnets() {
	utils.AllowedSubnets = make([]*net.IPNet, 0)
	subnets := viper.GetStringSlice("subnets_whitelist")
	for _, subnet := range subnets {
		_, tmp, _ := net.ParseCIDR(subnet)
		utils.AllowedSubnets = append(utils.AllowedSubnets, tmp)
	}
	log.Println("subnets: ", subnets)
}

func refreshConfig() {
	refreshAllowedSubnets()
	utils.RefreshGeoDb()
	viper.SetDefault("sys_load_threshold", consts.SystemLoadThreshold)
	viper.SetDefault("ws_ping_period_sec", 90)
	viper.SetDefault("ws_pong_timeout_sec", 10)
	viper.SetDefault("push_internal_api_listen_address", "127.0.0.1:3009")
}

func InitConfigFile() {
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetConfigFile("config.yml")
	err := viper.ReadInConfig() // Find and read the config file
	utils.FatalErrorHandle(&err, "error while reading config file")

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file changed:", e.Name)
		refreshConfig()
	})
	refreshConfig()
}

func GetFrontendConfigInfo() gin.H {
	return gin.H{
		"web_frontend_version": viper.GetString("web_frontend_version"),
		"announcement":         viper.GetString("announcement"),
	}
}
