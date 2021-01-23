package route

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/ulule/limiter/v3"
	"log"
	"net/http"
	"strconv"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/permissions"
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
	"time"
	"unicode/utf8"
)

var emailLimiter *limiter.Limiter
var postLimiter *limiter.Limiter
var postLimiter2 *limiter.Limiter
var commentLimiter *limiter.Limiter
var commentLimiter2 *limiter.Limiter
var detailPostLimiter *limiter.Limiter
var doAttentionLimiter *limiter.Limiter
var searchLimiter *limiter.Limiter
var searchShortTimeLimiter *limiter.Limiter
var deleteBanLimiter *limiter.Limiter

func initLimiters() {
	postLimiter = db.InitLimiter(limiter.Rate{
		Period: 20 * time.Second,
		Limit:  1,
	}, "postLimiter")
	postLimiter2 = db.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  100,
	}, "postLimiter2")
	commentLimiter = db.InitLimiter(limiter.Rate{
		Period: 10 * time.Second,
		Limit:  1,
	}, "commentLimiter")
	commentLimiter2 = db.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  500,
	}, "commentLimiter2")
	detailPostLimiter = db.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  8000,
	}, "detailPostLimiter")
	searchShortTimeLimiter = db.InitLimiter(limiter.Rate{
		Period: 2 * time.Second,
		Limit:  1,
	}, "searchShortTimeLimiter")
	searchLimiter = db.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  1000,
	}, "searchLimiter")
	doAttentionLimiter = db.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  2000,
	}, "doAttentionLimiter")
	deleteBanLimiter = db.InitLimiter(limiter.Rate{
		Period: 24 * time.Hour,
		Limit:  permissions.GetDeletePostRateLimitIn24h(structs.SuperUserRole),
	}, "deleteBanLimiter")
}

func limiterMiddleware(limiter *limiter.Limiter, msg string, doLog bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(structs.User)
		uidStr := strconv.Itoa(int(user.ID))

		if permissions.NeedLimiter(&user) {
			context, err6 := limiter.Get(c, uidStr)
			if err6 != nil {
				c.AbortWithStatus(500)
				return
			}
			if context.Reached {
				if doLog {
					log.Printf("limiter reached: " + msg)
				}
				utils.HttpReturnWithCodeOneAndAbort(c, msg)
				return
			}
		}
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("user_token")
		var user structs.User
		err := db.GetDb(false).Where("token = ?", token).First(&user).Error
		if err != nil {
			fmt.Println(err.Error())
			if err.Error() != "record not found" {
				log.Printf("auth failed. err=%s\n", err)
				utils.HttpReturnWithCodeOneAndAbort(c, "数据库读取失败，请联系管理员。")
				return
			}
			if !viper.GetBool("allow_unregistered_access") && !utils.IsInAllowedSubnet(c.ClientIP()) {
				utils.HttpReturnWithCodeOneAndAbort(c, "登录凭据过期，请使用邮箱重新登录。")
				return
			} else {
				c.Set("user", structs.User{ID: -1, Role: structs.UnregisteredRole, EmailHash: ""})
				c.Next()
			}
		} else {
			if user.Role == structs.BannedUserRole {
				c.JSON(http.StatusOK, gin.H{
					"msg": "您的账户已被冻结。如果需要解冻，请联系" + viper.GetString("contact_email") + "。",
				})
				c.Abort()
				return
			}
			c.Set("user", user)
			c.Next()
		}
	}
}

func disallowUnregisteredUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(structs.User)
		if user.Role == structs.UnregisteredRole {
			utils.HttpReturnWithCodeOneAndAbort(c, "登录凭据过期，请使用邮箱重新登录。")
			return
		}
		c.Next()
	}
}

func textToFrontendJson(id int32, timestamp int64, text string) gin.H {
	return gin.H{
		"pid":         id,
		"text":        text,
		"type":        "text",
		"timestamp":   timestamp,
		"reply":       0,
		"likenum":     0,
		"attention":   false,
		"permissions": []string{},
		"url":         "",
		"tag":         nil,
	}
}

func httpReturnInfo(c *gin.Context, text string) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": []map[string]interface{}{textToFrontendJson(0, 2147483647, text)},
		//"timestamp": utils.GetTimeStamp(),
		"count": 1,
	})
	c.Abort()
}

func disallowBannedPostUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(structs.User)
		if !permissions.CanOverrideBan(&user) {
			timestamp := utils.GetTimeStamp()
			bannedTimes, err := db.GetBannedTime(user.ID, timestamp)
			if bannedTimes > 0 && err != nil {
				var ban structs.Ban
				err2 := db.GetDb(false).Model(&structs.Ban{}).Where("user_id = ? and expire_at > ?", user.ID, timestamp).
					Order("expire_at desc").First(&ban).Error
				if err2 == nil {
					utils.HttpReturnWithCodeOneAndAbort(c, "很抱歉，您当前处于禁言状态，在"+
						utils.TimestampToString(ban.ExpireAt)+"之前您将无法发布树洞。")
					return
				}
			}
		}
		c.Next()
	}
}

func checkParameterPageSize() gin.HandlerFunc {
	return func(c *gin.Context) {
		pageSize, err := strconv.Atoi(c.Query("pagesize"))
		if err != nil || pageSize > consts.SearchMaxPageSize || pageSize <= 0 {
			utils.HttpReturnWithCodeOneAndAbort(c, "获取失败，参数pagesize不合法")
			return
		}

		c.Set("page_size", pageSize)
		c.Next()
	}
}

func checkReportParams(isPost bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		reason := c.PostForm("reason")
		if len(reason) > consts.ReportMaxLength {
			utils.HttpReturnWithCodeOneAndAbort(c, "字数过长！字数限制为"+strconv.Itoa(consts.ReportMaxLength)+"字节。")
			return
		}
		id, err := strconv.Atoi(c.PostForm("id"))
		if err != nil {
			utils.HttpReturnWithCodeOneAndAbort(c, "操作失败，id不合法")
			return
		}
		if isPost {
			typ := c.PostForm("type")
			if _, ok := utils.ContainsInt(viper.GetIntSlice("disallow_report_pids"), id); ok && typ == "report" {
				utils.HttpReturnWithCodeOneAndAbort(c, "这个树洞无法举报哦")
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
			utils.HttpReturnWithCodeOneAndAbort(c, "字数过长！字数限制为"+strconv.Itoa(consts.PostMaxLength)+"字。")
			return
		} else if len(text) == 0 && typ == "text" {
			utils.HttpReturnWithCodeOneAndAbort(c, "请输入内容")
			return
		} else if typ != "text" && typ != "image" {
			utils.HttpReturnWithCodeOneAndAbort(c, "未知类型的树洞")
			return
		} else if int(float64(len(img))/consts.Base64Rate) > consts.ImgMaxLength {
			utils.HttpReturnWithCodeOneAndAbort(c, "图片大小超出限制！")
			return
		}
		c.Next()
	}
}

func safeSubSlice(slice []structs.Post, low int, high int) []structs.Post {
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
		pageSize := c.MustGet("page_size").(int)
		keywords := c.Query("keywords")

		if keywords == "热榜" {
			user := c.MustGet("user").(structs.User)
			posts := safeSubSlice(HotPosts, (page-1)*pageSize, page*pageSize)
			attentionPids, err3 := getAttentionPidsInPosts(user, posts)
			if err3 != nil {
				log.Printf("dbGetAttentionPids failed while search posts: %s\n", err3)
				utils.HttpReturnWithCodeOneAndAbort(c, "数据库读取失败，请联系管理员")
				return
			}
			rtn := postsToJson(posts, &user, attentionPids)
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": utils.IfThenElse(rtn != nil, rtn, []string{}),
				//"timestamp": utils.GetTimeStamp(),
				"count": utils.IfThenElse(rtn != nil, len(rtn), 0),
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
			utils.HttpReturnWithCodeOneAndAbort(c, "获取失败，参数page不合法")
			return
		}

		if page > maxPage || page <= 0 {
			utils.HttpReturnWithCodeOneAndAbort(c, "获取失败，参数page超出范围")
			return
		}
		c.Set("page", page)
		c.Next()
	}
}

func preprocessReportPost(c *gin.Context) {
	user := c.MustGet("user").(structs.User)
	canViewDelete := permissions.CanViewDeletedPost(&user)
	reason := c.PostForm("reason")
	typ := c.PostForm("type")
	id := c.MustGet("id").(int)
	var post structs.Post
	err3 := db.GetDb(canViewDelete).First(&post, int32(id)).Error
	if err3 != nil {
		utils.HttpReturnWithCodeOneAndAbort(c, "找不到这条树洞")
		return
	}
	c.Set("post", post)

	userPermissions := permissions.GetPermissionsByPost(&user, &post)
	if _, ok := utils.ContainsString(userPermissions, typ); !ok {
		utils.HttpReturnWithCodeOneAndAbort(c, "操作失败，权限不足")
		return
	}
	if typ == "fold" {
		if _, ok := utils.ContainsString(viper.GetStringSlice("fold_tags"), reason); !ok {
			utils.HttpReturnWithCodeOneAndAbort(c, "操作失败，不存在这个tag")
			return
		}
	}
	reportType := getReportType(typ)
	if typ == "report" || typ == "fold" {
		var reported int64
		db.GetDb(canViewDelete).Model(&structs.Report{}).
			Where("post_id = ? and user_id = ? and is_comment = ? and type = ?",
				post.ID, user.ID, false, reportType).Count(&reported)
		if reported == 1 {
			utils.HttpReturnWithCodeOneAndAbort(c, "已经举报过了，举报失败。")
			return
		}
	}
	report := structs.Report{
		UserID:         user.ID,
		ReportedUserID: post.UserID,
		PostID:         post.ID,
		CommentID:      0,
		Reason:         reason,
		Type:           reportType,
		IsComment:      false,
		Weight:         permissions.GetReportWeight(&user),
	}
	c.Set("report", report)
	c.Set("user", user)
	c.Next()
}

func preprocessReportComment(c *gin.Context) {
	user := c.MustGet("user").(structs.User)
	canViewDelete := permissions.CanViewDeletedPost(&user)
	reason := c.PostForm("reason")
	typ := c.PostForm("type")
	id := c.MustGet("id").(int)
	var comment structs.Comment
	err3 := db.GetDb(canViewDelete).First(&comment, int32(id)).Error
	if err3 != nil {
		utils.HttpReturnWithCodeOneAndAbort(c, "找不到这条树洞评论")
		return
	}
	c.Set("comment", comment)

	userPermissions := permissions.GetPermissionsByComment(&user, &comment)
	if _, ok := utils.ContainsString(userPermissions, typ); !ok {
		utils.HttpReturnWithCodeOneAndAbort(c, "操作失败，权限不足")
		return
	}
	if typ == "fold" {
		if _, ok := utils.ContainsString(viper.GetStringSlice("fold_tags"), reason); !ok {
			utils.HttpReturnWithCodeOneAndAbort(c, "操作失败，不存在这个tag")
			return
		}
	}
	reportType := getReportType(typ)
	if typ == "report" || typ == "fold" {
		var reported int64
		db.GetDb(canViewDelete).Model(&structs.Report{}).
			Where("comment_id = ? and user_id = ? and is_comment = ? and type = ?",
				comment.ID, user.ID, true, reportType).Count(&reported)
		if reported == 1 {
			utils.HttpReturnWithCodeOneAndAbort(c, "已经举报过了，举报失败。")
			return
		}
	}
	report := structs.Report{
		UserID:         user.ID,
		ReportedUserID: comment.UserID,
		PostID:         comment.PostID,
		CommentID:      comment.ID,
		Reason:         reason,
		Type:           reportType,
		IsComment:      true,
		Weight:         permissions.GetReportWeight(&user),
	}
	c.Set("report", report)
	c.Set("user", user)
	c.Next()
}
