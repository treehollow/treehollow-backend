package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
)

func initConfigFile() {
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.SetConfigFile("config.json")
	err := viper.ReadInConfig() // Find and read the config file
	fatalErrorHandle(&err, "error while reading config file")

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Config file changed:", e.Name)
	})
}
