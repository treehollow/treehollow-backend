package security

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/ulule/limiter/v3"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/route/contents"
	"treehollow-v3-backend/pkg/utils"
)

func ApiListenHttp() {
	r := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "TOKEN")
	r.Use(cors.New(corsConfig))

	contents.EmailLimiter = base.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  viper.GetInt64("max_email_per_ip_per_day"),
	}, "emailLimiter")

	r.POST("/v3/security/login/check_email",
		checkEmailParamsCheckMiddleware,
		checkEmailRegexMiddleware,
		checkEmailIsRegisteredUserMiddleware,
		checkEmailIsOldTreeholeUserMiddleware,
		checkEmailRateLimitVerificationCode,
		checkEmailReCaptchaValidationMiddleware,
		checkEmail)
	r.POST("/v3/security/login/check_email_unregister",
		checkEmailParamsCheckMiddleware,
		checkAccountIsRegistered,
		checkEmailRateLimitVerificationCode,
		checkEmailReCaptchaValidationMiddleware,
		unregisterEmail)
	r.POST("/v3/security/login/create_account",
		loginParamsCheckMiddleware,
		checkAccountNotRegistered,
		loginCheckIOSToken,
		createAccount)
	r.POST("/v3/security/login/login",
		loginParamsCheckMiddleware,
		checkAccountIsRegistered,
		loginGetUserMiddleware,
		loginCheckMaxDevices,
		loginCheckIOSToken,
		login)
	r.POST("/v3/security/login/change_password",
		checkAccountIsRegistered,
		changePassword)
	r.POST("/v3/security/login/unregister",
		checkAccountIsRegistered,
		deleteAccount)
	r.GET("/v3/security/devices/list", listDevices)
	r.POST("/v3/security/devices/terminate", terminateDevice)
	r.POST("/v3/security/logout", logout)
	r.POST("/v3/security/update_ios_token", updateIOSToken)

	listenAddr := viper.GetString("security_api_listen_address")
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
