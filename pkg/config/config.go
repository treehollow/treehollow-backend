package config

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
	"thuhole-go-backend/pkg/utils"
)

func InitConfigFile() {
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.SetConfigFile("config.json")
	err := viper.ReadInConfig() // Find and read the config file
	utils.FatalErrorHandle(&err, "error while reading config file")

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file changed:", e.Name)
	})
}
