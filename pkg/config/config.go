package config

import (
	"github.com/fsnotify/fsnotify"
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
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.SetConfigFile("config.json")
	err := viper.ReadInConfig() // Find and read the config file
	utils.FatalErrorHandle(&err, "error while reading config file")

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file changed:", e.Name)
		refreshConfig()
	})
	refreshConfig()
}
