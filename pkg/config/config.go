package config

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
	"net"
	"thuhole-go-backend/pkg/utils"
)

func refreshAllowedSubnets() {
	utils.AllowedSubnets = make([]*net.IPNet, 0)
	subnets := viper.GetStringSlice("allow_unregistered_subnets")
	for _, subnet := range subnets {
		_, tmp, _ := net.ParseCIDR(subnet)
		utils.AllowedSubnets = append(utils.AllowedSubnets, tmp)
	}
	log.Println("subnets: ", subnets)
}

func InitConfigFile() {
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.SetConfigFile("config.json")
	err := viper.ReadInConfig() // Find and read the config file
	utils.FatalErrorHandle(&err, "error while reading config file")

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		refreshAllowedSubnets()
		log.Println("Config file changed:", e.Name)
	})
	refreshAllowedSubnets()
}
