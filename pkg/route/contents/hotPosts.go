package contents

import (
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/spf13/viper"
	"log"
	"sync"
	"treehollow-v3-backend/pkg/base"
)

type HotPostsRW struct {
	mu       sync.RWMutex
	hotPosts []base.Post
}

var HotPosts HotPostsRW

func (hotPostRW *HotPostsRW) Get() []base.Post {
	hotPostRW.mu.RLock()
	rtn := hotPostRW.hotPosts
	hotPostRW.mu.RUnlock()
	return rtn
}

func (hotPostRW *HotPostsRW) Set(item []base.Post) {
	hotPostRW.mu.Lock()
	hotPostRW.hotPosts = item
	hotPostRW.mu.Unlock()
}

func RefreshHotPosts() {
	avg, err := load.Avg()
	if err == nil {
		if avg.Load1 <= viper.GetFloat64("sys_load_threshold") {
			hotPosts, err2 := base.GetHotPosts()
			if err2 == nil {
				HotPosts.Set(hotPosts)
			} else {
				log.Printf("db.GetHotPosts() failed: err=%s\n", err2)
			}
		}
	} else {
		log.Printf("load.Avg() failed: err=%s\n", err)
	}
}

func InitHotPostsRefreshCron() {
	c := cron.New()
	_, _ = c.AddFunc("*/1 * * * *", func() {
		RefreshHotPosts()
	})
	c.Start()
}
