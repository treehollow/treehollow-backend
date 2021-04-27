package main

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"treehollow-v3-backend/pkg/config"
	"treehollow-v3-backend/pkg/utils"
)

//func apiFallBack(c *gin.Context) {
//	c.JSON(http.StatusOK, gin.H{
//		"msg": viper.GetString("fallback_announcement"),
//	})
//	return
//}

func upgradePrompt(c *gin.Context) {
	action := c.Query("action")
	if action == "getlist" {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"config": gin.H{
				"img_base_url":         viper.GetString("img_base_url"),
				"img_base_url_bak":     viper.GetString("img_base_url_bak"),
				"fold_tags":            viper.GetStringSlice("fold_tags"),
				"web_frontend_version": "v2.0.0",
				"announcement":         "发现树洞新版本，正在更新...",
			},
			"data": []gin.H{{
				"pid":       0,
				"text":      "请更新到最新版本树洞。（点击界面右上角“账户”( i )，点击“强制检查更新”）",
				"type":      "text",
				"timestamp": 2147483647,
				"reply":     0,
				"likenum":   0,
				"url":       "",
				"tag":       nil,
			}},
			"count": 1,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"msg": "请更新到最新版本树洞。（点击界面右上角“账户”( i )，点击“强制检查更新”）",
		})
	}
}

func upgradePromptV2(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"config": gin.H{
			"img_base_url":         viper.GetString("img_base_url"),
			"img_base_url_bak":     viper.GetString("img_base_url_bak"),
			"fold_tags":            viper.GetStringSlice("fold_tags"),
			"web_frontend_version": "v3.0.0",
			"announcement":         "发现树洞新版本，正在更新...",
		},
		"data": []gin.H{{
			"pid":         0,
			"text":        "请更新到最新版本树洞。（点击界面右上角“账户”( i )，点击“强制检查更新”）",
			"type":        "text",
			"timestamp":   2147483647,
			"reply":       0,
			"likenum":     0,
			"url":         "",
			"tag":         nil,
			"updated_at":  2147483647,
			"attention":   false,
			"permissions": []string{},
			"deleted":     false,
		}},
		"count": 1,
	})
}

func serviceFallBack(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": -1,
		"msg":  viper.GetString("fallback_announcement"),
	})
	return
}

func main() {
	config.InitConfigFile()
	viper.SetDefault("fallback_announcement", "系统正在维护升级，请稍后重试...")
	viper.SetDefault("fallback_listen_address", "127.0.0.1:3002")

	r := gin.Default()
	//r.Use(cors.Default())

	//Old v1, compatibility fallback
	r.POST("/api_xmcp/login/send_code", upgradePrompt)
	r.POST("/api_xmcp/login/login", upgradePrompt)
	r.GET("/api_xmcp/hole/system_msg", upgradePrompt)
	r.GET("/services/thuhole/api.php", upgradePrompt)
	r.POST("/services/thuhole/api.php", upgradePrompt)

	//Old v2, compatibility fallback
	r.GET("/contents/post/list", upgradePromptV2)
	r.GET("/contents/search", upgradePromptV2)
	r.GET("/contents/system_msg", upgradePrompt)
	r.GET("/contents/post/detail", upgradePrompt)
	r.GET("/contents/post/attentions", upgradePrompt)
	r.GET("/contents/search/attentions", upgradePrompt)
	r.POST("/send/post", upgradePrompt)
	r.POST("/send/comment", upgradePrompt)
	r.POST("/edit/attention", upgradePrompt)
	r.POST("/edit/report/post", upgradePrompt)
	r.POST("/edit/report/comment", upgradePrompt)
	r.POST("/security/login/send_code", upgradePrompt)
	r.POST("/security/login/login", upgradePrompt)

	//v3 fallback
	r.POST("/v3/config/set_push", serviceFallBack)
	r.GET("/v3/config/get_push", serviceFallBack)
	r.GET("/v3/contents/system_msg", serviceFallBack)
	r.GET("/v3/contents/post/list", serviceFallBack)
	r.GET("/v3/contents/post/randomlist", serviceFallBack)
	r.GET("/v3/contents/post/detail", serviceFallBack)
	r.GET("/v3/contents/search", serviceFallBack)
	r.GET("/v3/contents/post/attentions", serviceFallBack)
	r.GET("/v3/contents/my_msgs", serviceFallBack)
	r.GET("/v3/contents/search/attentions", serviceFallBack)
	r.POST("/v3/send/post", serviceFallBack)
	r.POST("/v3/send/vote", serviceFallBack)
	r.POST("/v3/send/comment", serviceFallBack)
	r.POST("/v3/edit/attention", serviceFallBack)
	r.POST("/v3/edit/report/post", serviceFallBack)
	r.POST("/v3/edit/report/comment", serviceFallBack)
	r.POST("/v3/security/login/check_email", serviceFallBack)
	r.POST("/v3/security/login/create_account", serviceFallBack)
	r.POST("/v3/security/login/login", serviceFallBack)
	r.POST("/v3/security/login/change_password", serviceFallBack)
	r.GET("/v3/security/devices/list", serviceFallBack)
	r.POST("/v3/security/devices/terminate", serviceFallBack)
	r.POST("/v3/security/logout", serviceFallBack)

	listenAddr := viper.GetString("fallback_listen_address")
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
