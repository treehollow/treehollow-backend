package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
)

func sendCode(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,Date,Content-Length")
	code := genCode()
	user := c.Query("user")
	if !(strings.HasSuffix(user, "pku.edu.cn") || strings.HasSuffix(user, "mails.tsinghua.edu.cn")) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "很抱歉，您的邮箱无法注册T大树洞",
		})
		return
	}

	err := saveCode(user, code)
	if err != nil {
		log.Printf("save code failed: %s\n", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "数据库写入失败，请联系管理员",
		})
		return
	}

	_, err = sendMail(code, user)
	if err != nil {
		log.Printf("send mail failed: %s\n", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "验证码邮件发送失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "验证码发送成功，请记得查看垃圾邮件。由于未知bug，验证码可能需要较长时间才能收到。",
	})
}

func login(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,Date,Content-Length")
	user := c.Query("user")
	code := c.Query("valid_code")
	hashedUser := hashEmail(user)
	success, err := checkCode(hashedUser, code)
	if err != nil {
		log.Printf("check code failed: %s\n", err)
	}
	if !success {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "验证码无效或过期",
		})
		return
	}
	token := genToken()
	err = saveToken(token, hashedUser)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "数据库写入失败，请联系管理员",
		})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":       0,
			"msg":        "登录成功！",
			"user_token": token,
		})
		return
	}
}

func systemMsg(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,Date,Content-Length")
	//TODO: implement this
	token := c.Query("user_token")
	_, _, err := getInfoByToken(token)
	if err == nil {
		c.String(http.StatusOK, `{"error":null,"result":[{"content":"test","timestamp":0,"title":""}]}`)
	} else {
		log.Printf("check token failed: %s\n", err)
		c.String(http.StatusOK, `{"error":null,"result":[]}`)
	}
}

func optionsDebug(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,Date,Content-Length")
	c.String(http.StatusOK, `{"test": "test"}`)
}

func listenHttp() {
	r := gin.Default()
	//if viper.GetBool("is_debug") {
	r.OPTIONS("/api_xmcp/login/send_code", optionsDebug) // OPTIONS method for bypassing CORS
	r.OPTIONS("/api_xmcp/login/login", optionsDebug)
	r.OPTIONS("/services/thuhole/api.php", optionsDebug)
	//}
	r.POST("/api_xmcp/login/send_code", sendCode)
	r.POST("/api_xmcp/login/login", login)
	r.GET("/api_xmcp/hole/system_msg", systemMsg)
	r.GET("/services/thuhole/api.php", apiGet)
	r.POST("/services/thuhole/api.php", apiPost)
	_ = r.Run(listenAddress)
}
