package route

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"strings"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/mail"
	"thuhole-go-backend/pkg/utils"
)

func sendCode(c *gin.Context) {
	code := utils.GenCode()
	user := c.Query("user")
	if !(strings.HasSuffix(user, "@mails.tsinghua.edu.cn")) || !utils.CheckEmail(user) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "很抱歉，您的邮箱无法注册T大树洞。目前只有@mails.tsinghua.edu.cn的邮箱开放注册。",
		})
		return
	}

	hashedUser := utils.HashEmail(user)
	if strings.Contains(viper.GetString("bannedEmailHashed"), hashedUser) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "很抱歉，您已被永久封禁。如对封禁有异议，请联系thuhole@protonmail.com。",
		})
		return
	}
	now := utils.GetTimeStamp()
	_, timeStamp, err := db.GetCode(hashedUser)
	//if err != nil {
	//	log.Printf("dbGetCode failed when sendCode: %s\n", err)
	//}
	if now-timeStamp < 600 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "请不要短时间内重复发送邮件。",
		})
		return
	}

	_, err = mail.SendMail(code, user)
	if err != nil {
		log.Printf("send mail to %s failed: %s\n", user, err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "验证码邮件发送失败。现阶段暂时每小时开放注册100个账户。",
		})
		return
	}

	err = db.SaveCode(user, code)
	if err != nil {
		log.Printf("save code failed: %s\n", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "数据库写入失败，请联系管理员",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "验证码发送成功，有效期【12小时】。由于清华邮箱的审查系统，验证码可能需要较长时间才能收到，不要多次发送验证码。请记得查看垃圾邮件。",
	})
}

func login(c *gin.Context) {
	user := c.Query("user")
	code := c.Query("valid_code")
	hashedUser := utils.HashEmail(user)
	if strings.Contains(viper.GetString("bannedEmailHashed"), hashedUser) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "很抱歉，您已被永久封禁。如对封禁有异议，请联系thuhole@protonmail.com。",
		})
		return
	}
	now := utils.GetTimeStamp()

	if !(strings.HasSuffix(user, "@mails.tsinghua.edu.cn")) || !utils.CheckEmail(user) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "邮箱格式不正确",
		})
		return
	}

	correctCode, timeStamp, err := db.GetCode(hashedUser)
	if err != nil {
		log.Printf("check code failed: %s\n", err)
	}
	if correctCode != code || now-timeStamp > 43200 {
		log.Printf("验证码无效或过期: %s, %s\n", user, code)
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "验证码无效或过期",
		})
		return
	}
	token := utils.GenToken()
	err = db.SaveToken(token, hashedUser)
	if err != nil {
		log.Printf("failed dbSaveToken while login, %s\n", err)
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
	token := c.Query("user_token")
	emailHash, err := db.GetInfoByToken(token)
	if err == nil {
		data, err2 := db.GetBannedMsgs(emailHash)
		if err2 != nil {
			log.Printf("dbGetBannedMsgs failed while systemMsg: %s\n", err2)
			utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"error":  nil,
				"result": data,
			})
		}
	} else {
		log.Printf("check token failed: %s\n", err)
		c.String(http.StatusOK, `{"error":null,"result":[]}`)
	}
}

//func optionsDebug(c *gin.Context) {
//	c.Header("Content-Type", "application/json; charset=utf-8")
//	c.Header("Access-Control-Allow-Origin", "*")
//	c.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,Date,Content-Length")
//	c.String(http.StatusOK, `{"test": "test"}`)
//}

func ListenHttp() {
	r := gin.Default()
	r.Use(cors.Default())
	////if viper.GetBool("is_debug") {
	//r.OPTIONS("/api_xmcp/login/send_code", optionsDebug) // OPTIONS method for bypassing CORS
	//r.OPTIONS("/api_xmcp/login/login", optionsDebug)
	//r.OPTIONS("/services/thuhole/api.php", optionsDebug)
	////}
	r.POST("/api_xmcp/login/send_code", sendCode)
	r.POST("/api_xmcp/login/login", login)
	r.GET("/api_xmcp/hole/system_msg", systemMsg)
	r.GET("/services/thuhole/api.php", apiGet)
	r.POST("/services/thuhole/api.php", apiPost)
	_ = r.Run(consts.ListenAddress)
}
