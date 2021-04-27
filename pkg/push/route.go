package push

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/route/auth"
	"treehollow-v3-backend/pkg/utils"
)

func ApiListenHttp() {
	r := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "TOKEN")
	r.Use(cors.New(corsConfig))

	Api = New(time.Duration(viper.GetInt64("ws_ping_period_sec"))*time.Second,
		time.Duration(viper.GetInt64("ws_pong_timeout_sec"))*time.Second)

	go func() {
		r2 := gin.Default()
		r2.POST("/send_messages", func(c *gin.Context) {
			var messages []base.PushMessage
			//data, err := ioutil.ReadAll(c.Request.Body)
			c.String(http.StatusOK, "")
			err := c.BindJSON(&messages)
			if err != nil {
				//base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "error reading request body", "error reading request body"))
				log.Printf("push service read request body error: %s\n", err)
				return
			}
			SendMessages(messages, Api, false)
		})
		r2.POST("/delete_messages", func(c *gin.Context) {
			var commendID int32
			//data, err := ioutil.ReadAll(c.Request.Body)
			c.String(http.StatusOK, "")
			err := c.BindJSON(&commendID)
			if err != nil {
				//base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "error reading request body", "error reading request body"))
				log.Printf("push deletion service read request body error: %s\n", err)
				return
			}

			var msgs []base.PushMessage
			err = base.GetDb(false).Model(&base.PushMessage{}).
				Where("comment_id = ? and do_push = 1", commendID).
				Find(&msgs).Error
			if err != nil {
				log.Printf("push deletion service read push messages error: %s\n", err)
				return
			}

			err = base.GetDb(false).Where("comment_id = ?", commendID).Delete(&base.PushMessage{}).Error
			if err != nil {
				log.Printf("push deletion service delete push messages error: %s\n", err)
				return
			}

			SendMessages(msgs, Api, true)
		})
		_ = r2.Run(viper.GetString("push_internal_api_listen_address"))
	}()

	r.Use(auth.AuthMiddleware())
	r.GET("/v3/stream",
		auth.DisallowUnregisteredUsers(),
		Api.Handle)

	listenAddr := viper.GetString("push_api_listen_address")
	if strings.Contains(listenAddr, ":") {
		_ = r.Run(listenAddr)
	} else {
		_ = os.MkdirAll(filepath.Dir(listenAddr), os.ModePerm)
		_ = os.Remove(listenAddr)

		listener, err := net.Listen("unix", listenAddr)
		utils.FatalErrorHandle(&err, "bind failed")
		log.Printf("Listening and serving HTTP on unix: %s.\n"+
			"Note: 0777 is not a safe permission for the unix socket file. "+
			"It would be better if the user manually set the permission after startup\n",
			listenAddr)
		_ = os.Chmod(listenAddr, 0777)
		err = http.Serve(listener, r)
	}

}
