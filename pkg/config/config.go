package config

import (
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/oschwald/geoip2-golang"
	"github.com/spf13/viper"
	"log"
	"net"
	"thuhole-go-backend/pkg/utils"
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

func refreshGeoIpDb() {
	var err error
	utils.GeoDb, err = geoip2.Open(viper.GetString("mmdb_path"))
	if err != nil {
		utils.GeoDb = nil
		log.Println("geoip2 db load failed. No IP location restrictions would be available.")
	} else {
		log.Println("geoip2 db loaded.")
	}
}

func refreshConfig() {
	refreshAllowedSubnets()
	refreshGeoIpDb()
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
	// TODO: swap img_base_url if not in China
	return gin.H{
		"img_base_url":         viper.GetString("img_base_url"),
		"img_base_url_bak":     viper.GetString("img_base_url_bak"),
		"fold_tags":            viper.GetStringSlice("fold_tags"),
		"web_frontend_version": viper.GetString("web_frontend_version"),
		"announcement":         viper.GetString("announcement"),
	}
}
