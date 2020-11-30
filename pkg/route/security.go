package route

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/ulule/limiter/v3"
	"gopkg.in/ezzarghili/recaptcha-go.v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"net"
	"net/http"
	"regexp"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/mail"
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
	"time"
)

func sendCode(c *gin.Context) {
	code := utils.GenCode()
	user := c.Query("user")
	recaptchaVersion := c.Query("recaptcha_version")
	recaptchaToken := c.Query("recaptcha_token")
	if recaptchaToken == "" || recaptchaToken == "undefined" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "recaptcha校验失败，请稍等片刻或刷新重试。如果注册持续失败，可邮件联系" + viper.GetString("contact_email") + "人工注册。",
		})
		return
	}

	emailCheck, err := regexp.Compile(viper.GetString("email_check_regex"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "服务器配置错误，请联系管理员。",
		})
		return
	}
	if !emailCheck.MatchString(user) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "很抱歉，您的邮箱无法注册" + viper.GetString("name"),
		})
		return
	}

	hashedUser := utils.HashEmail(user)
	if _, b := utils.ContainsString(viper.GetStringSlice("banned_email_hashes"), hashedUser); b {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "您的账户已被冻结。如果需要解冻，请联系" + viper.GetString("contact_email") + "。",
		})
		return
	}
	now := utils.GetTimeStamp()
	_, timeStamp, _, err := db.GetVerificationCode(hashedUser)
	//if err != nil {
	//	log.Printf("dbGetCode failed when sendCode: %s\n", err)
	//}
	if now-timeStamp < 300 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "请不要短时间内重复发送邮件。",
		})
		return
	}

	context, err2 := emailLimiter.Get(c, c.ClientIP())
	if err2 != nil {
		log.Printf("send mail to %s failed, limiter fatal error. IP=%s,err=%s\n", user, c.ClientIP(), err2)
		c.AbortWithStatus(500)
		return
	}

	if context.Reached {
		log.Printf("send mail to %s failed, too many requests. IP=%s,err=%s\n", user, c.ClientIP(), err)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"msg":     "您今天已经发送了过多验证码，请24小时之后重试。",
		})
		return
	}

	if utils.GeoDb != nil && len(viper.GetStringSlice("allowed_register_countries")) != 0 {
		ip := net.ParseIP(c.ClientIP())
		record, err5 := utils.GeoDb.Country(ip)
		if err5 == nil {
			country := record.Country.Names["zh-CN"]
			if _, ok := utils.ContainsString(viper.GetStringSlice("allowed_register_countries"), country); !ok {
				log.Println("register not allowed:", c.ClientIP(), country, user)
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"msg":     "您所在的国家暂未开放注册。",
				})
				return
			}
		}
	}

	var captcha recaptcha.ReCAPTCHA
	if recaptchaVersion == "v2" {
		captcha, _ = recaptcha.NewReCAPTCHA(viper.GetString("recaptcha_v2_private_key"), recaptcha.V2, 10*time.Second)
	} else {
		captcha, _ = recaptcha.NewReCAPTCHA(viper.GetString("recaptcha_private_key"), recaptcha.V3, 10*time.Second)
	}
	captcha.ReCAPTCHALink = "https://www.recaptcha.net/recaptcha/api/siteverify"
	err = captcha.VerifyWithOptions(recaptchaToken, recaptcha.VerifyOption{
		RemoteIP:  c.ClientIP(),
		Threshold: float32(viper.GetFloat64("recaptcha_threshold")),
	})
	if err != nil {
		log.Println("recaptcha server error", err, c.ClientIP(), user)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "recaptcha风控系统校验失败，请检查网络环境并刷新重试。如果注册持续失败，可邮件联系" + viper.GetString("contact_email") + "人工注册。",
		})
		return
	}

	err = mail.SendMail(code, user)
	if err != nil {
		log.Printf("send mail to %s failed: %s\n", user, err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "验证码邮件发送失败。",
		})
		return
	}

	err = db.GetDb(false).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&structs.VerificationCode{Code: code, EmailHash: hashedUser, FailedTimes: 0}).Error
	if err != nil {
		log.Printf("save verification code failed: %s\n", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "数据库写入失败，请联系管理员",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "验证码发送成功。如果要在多客户端登录请不要使用邮件登录而是Token登录。5分钟内无法重复发送验证码，请记得查看垃圾邮件。",
	})
}

func login(c *gin.Context) {
	user := c.Query("user")
	code := c.Query("valid_code")
	hashedUser := utils.HashEmail(user)
	//TODO: use sql instead of config file to get banned email hashes
	if _, b := utils.ContainsString(viper.GetStringSlice("banned_email_hashes"), hashedUser); b {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "您的账户已被冻结。如果需要解冻，请联系" + viper.GetString("contact_email") + "。",
		})
		return
	}
	now := utils.GetTimeStamp()

	emailCheck, err := regexp.Compile(viper.GetString("email_check_regex"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "服务器配置错误，请联系管理员。",
		})
		return
	}
	if !emailCheck.MatchString(user) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     "很抱歉，您的邮箱无法注册" + viper.GetString("name"),
		})
		return
	}

	correctCode, timeStamp, failedTimes, err := db.GetVerificationCode(hashedUser)
	if err != nil {
		log.Printf("check code failed: %s\n", err)
	}
	if failedTimes >= 10 && now-timeStamp <= 43200 {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "验证码错误尝试次数过多，请重新发送验证码",
		})
		return
	}
	if correctCode != code || now-timeStamp > 43200 {
		log.Printf("验证码无效或过期: %s, %s\n", user, code)
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "验证码无效或过期",
		})
		_ = db.GetDb(false).Model(&structs.VerificationCode{}).Where("email_hash = ?", hashedUser).
			Update("failed_times", gorm.Expr("failed_times + 1")).Error
		return
	}
	token := utils.GenToken()
	err = db.GetDb(false).Create(&structs.User{EmailHash: hashedUser, Token: token, Role: structs.NormalUserRole}).Error
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

func LoginApiListenHttp() {
	r := gin.Default()
	r.Use(cors.Default())

	emailLimiter = utils.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  viper.GetInt64("max_email_per_ip_per_day"),
	}, "emailLimiter")

	r.POST("/security/login/send_code", sendCode)
	r.POST("/security/login/login", login)
	_ = r.Run(viper.GetString("login_api_listen_address"))
}
