package contents

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"treehollow-v3-backend/pkg/bot"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/logger/ginLogger"
	"treehollow-v3-backend/pkg/route/auth"
	"treehollow-v3-backend/pkg/utils"
)

func ServicesApiListenHttp() {
	r := gin.New()

	bot.InitBot()
	initLimiters()
	shutdownCountDown = 2
	c := cron.New()
	_, _ = c.AddFunc("0 0 * * *", func() {
		shutdownCountDown = 2
	})

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "TOKEN")
	if viper.GetBool("debug_log") {
		logFile, err := os.OpenFile(consts.DetailLogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			panic(err)
		}

		r.Use(cors.New(corsConfig), auth.AuthMiddleware(), ginLogger.LoggerWithConfig(ginLogger.LoggerConfig{
			Output: logFile,
		}), gin.Recovery())
	} else {
		r.Use(gin.Logger(), gin.Recovery(), cors.New(corsConfig), auth.AuthMiddleware())
	}
	r.POST("/v3/config/set_push",
		auth.DisallowUnregisteredUsers(),
		setPush)
	r.GET("/v3/config/get_push",
		auth.DisallowUnregisteredUsers(),
		getPush)
	r.GET("/v3/contents/system_msg",
		auth.DisallowUnregisteredUsers(),
		systemMsg)
	r.GET("/v3/contents/post/list",
		checkParameterPage(consts.MaxPage),
		listPost)
	r.GET("/v3/contents/post/randomlist",
		limiterMiddleware(randomListLimiter, "你今天刷了太多树洞了，明天再来吧", logger.WARN),
		wanderListPost)
	r.GET("/v3/contents/post/detail",
		limiterMiddleware(detailPostLimiter, "你今天刷了太多树洞了，明天再来吧", logger.WARN),
		detailPost)
	r.GET("/v3/contents/search",
		checkParameterPage(consts.SearchMaxPage),
		limiterMiddleware(searchShortTimeLimiter, "请不要短时间内连续搜索树洞", logger.INFO),
		limiterMiddleware(searchLimiter, "你今天搜索太多树洞了，明天再来吧", logger.WARN),
		searchHotPosts(),
		adminHelpCommand(),
		adminDecryptionCommand(),
		adminLogsCommand(),
		adminReportsCommand(),
		adminStatisticsCommand(),
		adminSysMsgsCommand(),
		adminShutdownCommand(),
		sysLoadWarningMiddleware(viper.GetFloat64("sys_load_threshold"), "目前树洞服务器负载较高，搜索功能已被暂时停用"),
		searchPost)
	r.GET("/v3/contents/post/attentions",
		auth.DisallowUnregisteredUsers(),
		checkParameterPage(consts.MaxPage),
		attentionPosts)
	r.GET("/v3/contents/my_msgs",
		auth.DisallowUnregisteredUsers(),
		checkParameterPage(consts.MaxPage),
		myMsgs)
	r.GET("/v3/contents/search/attentions",
		auth.DisallowUnregisteredUsers(),
		checkParameterPage(consts.SearchMaxPage),
		limiterMiddleware(searchShortTimeLimiter, "请不要短时间内连续搜索树洞", logger.INFO),
		limiterMiddleware(searchLimiter, "你今天搜索太多树洞了，明天再来吧", logger.WARN),
		searchAttentionPost)
	r.POST("/v3/send/post",
		auth.DisallowUnregisteredUsers(),
		limiterMiddleware(postLimiter, "请不要短时间内连续发送树洞", logger.INFO),
		limiterMiddleware(postLimiter2, "你24小时内已经发送太多树洞了", logger.WARN),
		disallowBannedPostUsers(),
		checkParameterTextAndImage(),
		checkParameterVoteOptions,
		sendPost)
	r.POST("/v3/send/vote",
		auth.DisallowUnregisteredUsers(),
		disallowBannedPostUsers(),
		sendVote)
	r.POST("/v3/send/comment",
		auth.DisallowUnregisteredUsers(),
		limiterMiddleware(commentLimiter, "请不要短时间内连续发送树洞回复", logger.INFO),
		limiterMiddleware(commentLimiter2, "你24小时内已经发送太多树洞回复了", logger.WARN),
		disallowBannedPostUsers(),
		checkParameterTextAndImage(),
		sendComment)
	r.POST("/v3/edit/attention",
		auth.DisallowUnregisteredUsers(),
		limiterMiddleware(doAttentionLimiter, "你今天关注太多树洞了，明天再来吧", logger.WARN),
		editAttention)
	r.POST("/v3/edit/report/post",
		auth.DisallowUnregisteredUsers(),
		checkReportParams(true),
		handleReport(false))
	r.POST("/v3/edit/report/comment",
		auth.DisallowUnregisteredUsers(),
		checkReportParams(false),
		handleReport(true))

	listenAddr := viper.GetString("services_api_listen_address")
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
		return
	}
}
