package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"net/http"
	"thuhole-go-backend/pkg/config"
)

func apiFallBack(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"msg": viper.GetString("fallback_announcement"),
	})
	return
}

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

func serviceFallBack(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  viper.GetString("fallback_announcement"),
	})
	return
}

func main() {
	config.InitConfigFile()
	viper.SetDefault("fallback_announcement", "系统正在维护升级，请稍后重试...")
	viper.SetDefault("fallback_listen_address", "127.0.0.1:3002")

	r := gin.Default()
	r.Use(cors.Default())

	//Old, compatibility fallback
	r.POST("/api_xmcp/login/send_code", upgradePrompt)
	r.POST("/api_xmcp/login/login", upgradePrompt)
	r.GET("/api_xmcp/hole/system_msg", upgradePrompt)
	r.GET("/services/thuhole/api.php", upgradePrompt)
	r.POST("/services/thuhole/api.php", upgradePrompt)

	r.GET("/contents/system_msg", serviceFallBack)
	r.GET("/contents/post/list", serviceFallBack)
	r.GET("/contents/post/detail", serviceFallBack)
	r.GET("/contents/search", serviceFallBack)
	r.GET("/contents/post/attentions", serviceFallBack)
	r.GET("/contents/search/attentions", serviceFallBack)
	r.POST("/send/post", serviceFallBack)
	r.POST("/send/comment", serviceFallBack)
	r.POST("/edit/attention", serviceFallBack)
	r.POST("/edit/report/post", serviceFallBack)
	r.POST("/edit/report/comment", serviceFallBack)

	r.POST("/security/login/send_code", apiFallBack)
	r.POST("/security/login/login", apiFallBack)
	_ = r.Run(viper.GetString("fallback_listen_address"))
}
