package contents

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/iancoleman/orderedmap"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/spf13/viper"
	"github.com/ulule/limiter/v3"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
	"unicode/utf8"
)

var EmailLimiter *limiter.Limiter
var postLimiter *limiter.Limiter
var postLimiter2 *limiter.Limiter
var commentLimiter *limiter.Limiter
var commentLimiter2 *limiter.Limiter
var detailPostLimiter *limiter.Limiter
var randomListLimiter *limiter.Limiter
var doAttentionLimiter *limiter.Limiter
var searchLimiter *limiter.Limiter
var searchShortTimeLimiter *limiter.Limiter
var deleteBanLimiter *limiter.Limiter

func initLimiters() {
	randomListLimiter = base.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  200,
	}, "randomListLimiter")
	postLimiter = base.InitLimiter(limiter.Rate{
		Period: 6 * time.Second,
		Limit:  1,
	}, "postLimiter")
	postLimiter2 = base.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  100,
	}, "postLimiter2")
	commentLimiter = base.InitLimiter(limiter.Rate{
		Period: 3 * time.Second,
		Limit:  1,
	}, "commentLimiter")
	commentLimiter2 = base.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  500,
	}, "commentLimiter2")
	detailPostLimiter = base.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  8000,
	}, "detailPostLimiter")
	searchShortTimeLimiter = base.InitLimiter(limiter.Rate{
		Period: 2 * time.Second,
		Limit:  1,
	}, "searchShortTimeLimiter")
	searchLimiter = base.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  1000,
	}, "searchLimiter")
	doAttentionLimiter = base.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  2000,
	}, "doAttentionLimiter")
	deleteBanLimiter = base.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  base.GetDeletePostRateLimitIn24h(base.SuperUserRole),
	}, "deleteBanLimiter")
}

func limiterMiddleware(limiter *limiter.Limiter, msg string, level logger.LogLevel) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(base.User)
		uidStr := strconv.Itoa(int(user.ID))

		if base.NeedLimiter(&user) {
			context, err6 := limiter.Get(c, uidStr)
			if err6 != nil {
				c.AbortWithStatus(500)
				return
			}
			if context.Reached {
				logMsg := "limiter reached: " + msg
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError(logMsg, msg, level))
				return
			}
		}
		c.Next()
	}
}

func sysLoadWarningMiddleware(threshold float64, msg string) gin.HandlerFunc {
	return func(c *gin.Context) {
		avg, err := load.Avg()
		if err == nil {
			if avg.Load1 > threshold {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError(msg, msg, logger.WARN))
				return
			}
		}
		c.Next()
	}
}

func textToFrontendJson(id int32, timestamp int64, text string) gin.H {
	return gin.H{
		"pid":            id,
		"text":           text,
		"type":           "text",
		"timestamp":      timestamp,
		"updated_at":     timestamp,
		"reply":          0,
		"likenum":        0,
		"attention":      false,
		"permissions":    []string{},
		"url":            "",
		"tag":            nil,
		"deleted":        false,
		"image_metadata": gin.H{},
		"vote":           gin.H{},
	}
}

func httpReturnInfo(c *gin.Context, text string) {
	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"data":     []map[string]interface{}{textToFrontendJson(0, 2147483647, text)},
		"comments": map[string]string{},
		//"timestamp": utils.GetTimeStamp(),
		"count": 1,
	})
	c.Abort()
}

func disallowBannedPostUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(base.User)
		if !base.CanOverrideBan(&user) {
			timestamp := utils.GetTimeStamp()
			bannedTimes, err := base.GetBannedTime(base.GetDb(false), user.ID, timestamp)
			if bannedTimes > 0 && err == nil {
				var ban base.Ban
				err2 := base.GetDb(false).Model(&base.Ban{}).Where("user_id = ? and expire_at > ?", user.ID, timestamp).
					Order("expire_at desc").First(&ban).Error
				if err2 == nil {
					base.HttpReturnWithCodeMinusOneAndAbort(c,
						logger.NewSimpleError("DisallowBan", "很抱歉，您当前处于禁言状态，在"+
							utils.TimestampToString(ban.ExpireAt)+"之前您将无法发布树洞。", logger.WARN))
					return
				}
			}
		}
		c.Next()
	}
}

func checkReportParams(isPost bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		reason := c.PostForm("reason")
		if len(reason) > consts.ReportMaxLength {
			base.HttpReturnWithCodeMinusOneAndAbort(c,
				logger.NewSimpleError("TooLongReport",
					"字数过长！字数限制为"+strconv.Itoa(consts.ReportMaxLength)+"字节。", logger.INFO))
			return
		}
		id, err := strconv.Atoi(c.PostForm("id"))
		if err != nil {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "InvalidIdReport", "操作失败，id不合法"))
			return
		}
		if isPost {
			typ := c.PostForm("type")
			if _, ok := utils.ContainsInt(viper.GetIntSlice("disallow_report_pids"), id); ok && typ == "report" {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("DisallowReport", "这个树洞无法举报哦", logger.WARN))
				return
			}
		}
		c.Set("id", id)
		c.Next()
	}
}

func checkParameterTextAndImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		text := c.PostForm("text")
		typ := c.PostForm("type")
		img := c.PostForm("data")
		if utf8.RuneCountInString(text) > consts.PostMaxLength {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("TooLongText", "字数过长！字数限制为"+strconv.Itoa(consts.PostMaxLength)+"字。", logger.INFO))
			return
		} else if len(text) == 0 && typ == "text" {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("NoContent", "请输入内容", logger.INFO))
			return
		} else if typ != "text" && typ != "image" {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("UnknownType", "未知类型的树洞", logger.WARN))
			return
		} else if int(float64(len(img))/consts.Base64Rate) > consts.ImgMaxLength {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("TooLargeImage", "图片大小超出限制！", logger.WARN))
			return
		}
		c.Next()
	}
}

func safeSubSlice(slice []base.Post, low int, high int) []base.Post {
	if high > len(slice) {
		high = len(slice)
	}
	if 0 <= low && low <= high {
		return slice[low:high]
	}
	return nil
}

func searchHotPosts() gin.HandlerFunc {
	return func(c *gin.Context) {
		page := c.MustGet("page").(int)
		pageSize := consts.SearchPageSize
		keywords := c.Query("keywords")

		if keywords == "热榜" {
			user := c.MustGet("user").(base.User)
			posts := safeSubSlice(HotPosts.Get(), (page-1)*pageSize, page*pageSize)
			rtn, err := appendPostDetail(base.GetDb(false), posts, &user)
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, err)
				return
			}

			comments, err4 := getCommentsByPosts(posts, &user)
			if err4 != nil {
				base.HttpReturnWithCodeMinusOne(c, err4)
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": utils.IfThenElse(rtn != nil, rtn, []string{}),
				//"timestamp": utils.GetTimeStamp(),
				"count":    utils.IfThenElse(rtn != nil, len(rtn), 0),
				"comments": comments,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func checkParameterPage(maxPage int) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, err := strconv.Atoi(c.Query("page"))
		if err != nil {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "PageConversionFailed", "获取失败，参数page不合法"))
			return
		}

		if page > maxPage || page <= 0 {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("PageOutOfBounds", "获取失败，参数page超出范围", logger.WARN))
			return
		}
		c.Set("page", page)
		c.Next()
	}
}

func checkParameterVoteOptions(c *gin.Context) {
	//voteOptions := c.PostForm("vote_options")
	//var optionsList []string
	//err := json.Unmarshal([]byte(voteOptions), &optionsList)
	//if err != nil {
	//	c.Set("vote_data", "{}")
	//	c.Next()
	//	return
	//}
	optionsList := c.PostFormArray("vote_options[]")
	for _, option := range optionsList {
		if utf8.RuneCountInString(option) > consts.VoteOptionMaxCharacters {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("TooLongVoteOption", "发送失败，选项长度最大是"+
				strconv.Itoa(consts.VoteOptionMaxCharacters)+"个字符", logger.INFO))
			return
		}
	}
	voteData := orderedmap.New()
	for _, option := range optionsList {
		striped := strings.TrimSpace(option)
		if len(striped) > 0 {
			voteData.Set(striped, 0)
		}
	}
	if len(voteData.Keys()) > consts.VoteMaxOptions {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("TooManyVoteOptions", "发送失败，最多4个投票选项", logger.WARN))
		return
	}
	if len(voteData.Keys()) == 1 {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("TooFewVoteOptions", "发送失败，至少两个选项", logger.WARN))
		return
	}
	_voteData, err := json.Marshal(voteData)
	if err != nil {
		log.Printf("error json marshal voteData! %s\n", err)
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "VoteDataMarshalFailed", "投票参数解析失败，请联系管理员"))
		return
	}
	strVoteData := string(_voteData)

	c.Set("vote_data", strVoteData)
	c.Next()
}
