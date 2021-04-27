package utils

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"log"
	"sync"
)

type GeoDbRW struct {
	mu    sync.RWMutex
	geoDb *geoip2.Reader
}

var GeoDb GeoDbRW

func (GeoDbRW *GeoDbRW) Get() *geoip2.Reader {
	GeoDbRW.mu.RLock()
	rtn := GeoDbRW.geoDb
	GeoDbRW.mu.RUnlock()
	return rtn
}

func (GeoDbRW *GeoDbRW) Set(item *geoip2.Reader) {
	GeoDbRW.mu.Lock()
	GeoDbRW.geoDb = item
	GeoDbRW.mu.Unlock()
}

func RefreshGeoDb() {
	geoDb, err := geoip2.Open(viper.GetString("mmdb_path"))
	if err != nil {
		log.Println("geoip2 db load failed. No IP location restrictions would be available.")
	} else {
		GeoDb.Set(geoDb)
		log.Println("geoip2 db loaded.")
	}
}

func InitGeoDbRefreshCron() {
	c := cron.New()
	_, _ = c.AddFunc("00 05 * * *", func() {
		RefreshGeoDb()
	})
	c.Start()
}
