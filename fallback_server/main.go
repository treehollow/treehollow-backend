package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
)

func apiFallBack(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"msg": "系统正在维护升级，请稍后重试...",
	})
	return
}

func serviceFallBack(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  "系统正在维护升级，请稍后重试...",
	})
	return
}

func main() {
	r := gin.Default()
	r.Use(cors.Default())
	r.POST("/api_xmcp/login/send_code", apiFallBack)
	r.POST("/api_xmcp/login/login", apiFallBack)
	r.GET("/api_xmcp/hole/system_msg", serviceFallBack)
	r.GET("/services/thuhole/api.php", serviceFallBack)
	r.POST("/services/thuhole/api.php", serviceFallBack)
	_ = r.Run("127.0.0.1:3002")
}
