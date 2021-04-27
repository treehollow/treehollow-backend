package base

import (
	libredis "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"github.com/ulule/limiter/v3"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
	"treehollow-v3-backend/pkg/utils"
)

var redisClient *libredis.Client

func initRedis() error {
	option, err := libredis.ParseURL(viper.GetString("redis_source"))
	if err != nil {
		utils.FatalErrorHandle(&err, "failed init redis url")
		return err
	}
	redisClient = libredis.NewClient(option)
	return nil
}

func InitLimiter(rate limiter.Rate, prefix string) *limiter.Limiter {
	client := GetRedisClient()
	store, err2 := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix: prefix,
	})
	if err2 != nil {
		utils.FatalErrorHandle(&err2, "failed init redis store")
		return nil
	}
	return limiter.New(store, rate)
}

func GetRedisClient() *libredis.Client {
	return redisClient
}
