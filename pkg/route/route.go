package route

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"thuhole-go-backend/pkg/consts"
)

func ServicesApiListenHttp() {
	r := gin.Default()
	r.Use(cors.Default())

	initLimiters()
	shutdownCountDown = 2
	c := cron.New()
	_, _ = c.AddFunc("0 0 * * *", func() {
		shutdownCountDown = 2
	})

	r.Use(authMiddleware())
	r.GET("/contents/system_msg",
		disallowUnregisteredUsers(),
		systemMsg)
	r.GET("/contents/post/list",
		checkParameterPage(consts.MaxPage),
		listPost)
	r.GET("/contents/post/detail",
		limiterMiddleware(detailPostLimiter, "你今天刷了太多树洞了，明天再来吧", true),
		detailPost)
	r.GET("/contents/search",
		limiterMiddleware(searchLimiter, "你今天搜索太多树洞了，明天再来吧", true),
		checkParameterPage(consts.SearchMaxPage),
		checkParameterPageSize(),
		searchHotPosts(),
		adminHelpCommand(),
		adminActionsCommand(),
		adminReportsCommand(),
		adminStatisticsCommand(),
		adminSysMsgsCommand(),
		adminShutdownCommand(),
		searchPost)
	r.GET("/contents/post/attentions",
		disallowUnregisteredUsers(),
		checkParameterPage(consts.MaxPage),
		attentionPosts)
	r.GET("/contents/search/attentions",
		disallowUnregisteredUsers(),
		checkParameterPage(consts.SearchMaxPage),
		limiterMiddleware(searchLimiter, "你今天搜索太多树洞了，明天再来吧", true),
		checkParameterPageSize(),
		searchAttentionPost)
	r.POST("/send/post",
		disallowUnregisteredUsers(),
		limiterMiddleware(postLimiter, "请不要短时间内连续发送树洞", false),
		limiterMiddleware(postLimiter2, "你24小时内已经发送太多树洞了", true),
		disallowBannedPostUsers(),
		checkParameterTextAndImage(),
		sendPost)
	r.POST("/send/comment",
		disallowUnregisteredUsers(),
		limiterMiddleware(commentLimiter, "请不要短时间内连续发送树洞回复", false),
		limiterMiddleware(commentLimiter2, "你24小时内已经发送太多树洞回复了", true),
		disallowBannedPostUsers(),
		checkParameterTextAndImage(),
		sendComment)
	r.POST("/edit/attention",
		disallowUnregisteredUsers(),
		limiterMiddleware(doAttentionLimiter, "你今天关注太多树洞了，明天再来吧", true),
		editAttention)
	r.POST("/edit/report/post",
		disallowUnregisteredUsers(),
		checkReportParams(true),
		preprocessReportPost,
		handleReport)
	r.POST("/edit/report/comment",
		disallowUnregisteredUsers(),
		checkReportParams(false),
		preprocessReportComment,
		handleReport)
	_ = r.Run(viper.GetString("services_api_listen_address"))
}
