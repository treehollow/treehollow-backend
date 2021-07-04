package contents

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

//TODO: (middle priority) better result for `reports`
func adminDecryptionCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !viper.GetBool("allow_admin_commands") {
			c.Next()
			return
		}
		user := c.MustGet("user").(base.User)
		keywords := c.Query("keywords")
		if base.CanViewDecryptionMessages(&user) {
			info := ""
			var uid int32 = -1
			reg := regexp.MustCompile("decrypt pid=([0-9]+)")
			if reg.MatchString(keywords) {
				pidStr := reg.FindStringSubmatch(keywords)[1]
				pid, _ := strconv.Atoi(pidStr)
				var post base.Post
				err3 := base.GetDb(true).First(&post, int32(pid)).Error
				if err3 != nil {
					if errors.Is(err3, gorm.ErrRecordNotFound) {
						base.HttpReturnWithErrAndAbort(c, -101, logger.NewSimpleError("DecryptPostNoPid", "找不到这条树洞", logger.WARN))
					} else {
						base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err3, "GetSavedPostFailed", consts.DatabaseReadFailedString))

					}
					return
				}
				uid = post.UserID
				info += fmt.Sprintf("Decryption information for post #%d:", post.ID)
			}

			reg = regexp.MustCompile("decrypt cid=([0-9]+)")
			if reg.MatchString(keywords) {
				cidStr := reg.FindStringSubmatch(keywords)[1]
				cid, _ := strconv.Atoi(cidStr)
				var comment base.Comment
				err3 := base.GetDb(true).First(&comment, int32(cid)).Error
				if err3 != nil {
					if errors.Is(err3, gorm.ErrRecordNotFound) {
						base.HttpReturnWithErrAndAbort(c, -101, logger.NewSimpleError("DecryptCommentNoPid", "找不到这条树洞评论", logger.WARN))
					} else {
						base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err3, "GetSavedCommentFailed", consts.DatabaseReadFailedString))

					}
					return
				}
				uid = comment.UserID
				info += fmt.Sprintf("Decryption information for comment #%d-%d:", comment.PostID, comment.ID)
			}

			if uid > 0 {
				var toBeDecryptedUser base.User
				err3 := base.GetDb(true).First(&toBeDecryptedUser, uid).Error
				if err3 != nil {
					if errors.Is(err3, gorm.ErrRecordNotFound) {
						base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("DecryptNoUser", "找不到发帖用户", logger.WARN))
					} else {
						base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err3, "GetDecryptionUserFailed", consts.DatabaseReadFailedString))

					}
					return
				}

				var decryptionMsgs []base.DecryptionKeyShares
				err := base.GetDb(false).Where("email_encrypted = ?", toBeDecryptedUser.EmailEncrypted).
					Find(&decryptionMsgs).Error
				if err != nil {
					base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetDecryptionMsgsFailed", consts.DatabaseReadFailedString))
					return
				}
				info += "\nEncrypted email = " + toBeDecryptedUser.EmailEncrypted + "\n"
				for _, msg := range decryptionMsgs {
					info += "\n***\nKeykeeper email:" + msg.PGPEmail + "\nPGP encrypted message:\n```\n" +
						msg.PGPMessage + "\n```"
				}

				httpReturnInfo(c, info)
				return
			}
		}
		c.Next()
	}
}

func adminHelpCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !viper.GetBool("allow_admin_commands") {
			c.Next()
			return
		}
		user := c.MustGet("user").(base.User)
		keywords := c.Query("keywords")
		if base.CanShowHelp(&user) && keywords == "help" {
			info := ""
			if base.CanViewStatistics(&user) {
				info += "`stats`: 查看树洞统计信息\n"
			}
			if base.CanViewDecryptionMessages(&user) {
				info += "`decrypt pid=123`, `decrypt cid=1234`: 查看树洞发帖人个人信息的待解密消息\n"
			}
			if base.CanViewDeletedPost(&user) {
				info += "`dels`: 搜索所有被管理员删除的树洞和回复(包括删除后恢复的)\n"
				info += "`//setflag NOT_SHOW_DELETED=on`(注意大小写): 在除了`deleted`搜索界面外的其他界面隐藏被删除的树洞\n"
			}
			if base.CanViewAllSystemMessages(&user) {
				info += "`msgs`: 查看所有用户收到的系统消息\n"
			}
			if base.CanViewReports(&user) {
				info += "`rep_dels`: 查看所有用户的【删除举报】(树洞or回复)\n"
			}
			if base.CanViewLogs(&user) {
				info += "`rep_recalls`: 查看所有用户的【撤回】(树洞or回复)\n"
				info += "`rep_folds`: 查看所有用户的【折叠举报】(树洞or回复)\n"
				info += "`log_tags`: 查看所有【管理员打Tag】的操作日志\n"
				info += "`log_dels`: 查看所有的【管理员删除】\n"
				info += "`log_unbans`: 查看所有【撤销删除】、【解禁】的操作日志\n"
				info += "`logs`: 查看所有举报、删帖、打tag的操作日志\n"
			}
			if base.CanShutdown(&user) {
				info += "`shutdown`: 关闭树洞, 请谨慎使用此命令\n"
			}

			if base.GetDeletePostRateLimitIn24h(user.Role) > 0 {
				uidStr := strconv.Itoa(int(user.ID))
				ctx, err := deleteBanLimiter.Peek(c, uidStr)
				if err != nil {
					base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "DeleteBanLimiterFailed", consts.DatabaseReadFailedString))
					return
				}
				limit := base.GetDeletePostRateLimitIn24h(user.Role)

				info += "\n---\n"
				info += fmt.Sprintf("您的【删帖禁言】操作次数额度剩余（24h内）：%d/%d\n",
					limit+ctx.Remaining-ctx.Limit,
					limit)
			}

			httpReturnInfo(c, info)
			return
		}
		c.Next()
	}
}

func adminStatisticsCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !viper.GetBool("allow_admin_commands") {
			c.Next()
			return
		}
		user := c.MustGet("user").(base.User)
		keywords := c.Query("keywords")
		if base.CanViewStatistics(&user) && keywords == "stats" {
			var count int64
			var count2 int64

			info := ""
			err := base.GetDb(true).Model(&base.User{}).Where("email_encrypted != \"\"").Count(&count).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetTotalUserFailed", consts.DatabaseReadFailedString))
				return
			}
			info += "总注册人数（包含已注销账户）：" + strconv.Itoa(int(count)) + "\n"

			err = base.GetDb(true).Model(&base.Email{}).Count(&count).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetTotalRegisteredUserFailed", consts.DatabaseReadFailedString))
				return
			}
			info += "总注册人数（不包含已注销账户）：" + strconv.Itoa(int(count)) + "\n"

			err = base.GetDb(false).Model(&base.User{}).Where("email_encrypted != \"\"").Count(&count2).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetTotalRegisteredUser2Failed", consts.DatabaseReadFailedString))
				return
			}
			if count != count2 {
				info += "警告：数据库邮箱验证系统自洽性检验失败，请修复！（" + strconv.Itoa(int(count2)) + "）\n"
			}

			err = base.GetDb(true).Model(&base.User{}).Count(&count).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetTotalNewOldUserFailed", consts.DatabaseReadFailedString))
				return
			}
			info += "总注册人数（包含老版本树洞账户）：" + strconv.Itoa(int(count)) + "\n"

			err = base.GetDb(true).Model(&base.Post{}).
				Where("created_at > ?", time.Now().AddDate(0, 0, -1)).
				Count(&count).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetPostPerDayFailed", consts.DatabaseReadFailedString))
				return
			}
			info += "24h内发帖数：" + strconv.Itoa(int(count)) + "\n"

			err = base.GetDb(true).Model(&base.Post{}).
				Where("deleted_at is not null and created_at > ?", time.Now().AddDate(0, 0, -1)).
				Count(&count).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetDeletedPostStatsFailed", consts.DatabaseReadFailedString))
				return
			}
			info += "24h内树洞删帖数：" + strconv.Itoa(int(count)) + "\n"

			err = base.GetDb(true).Model(&base.Comment{}).
				Where("deleted_at is not null and created_at > ?", time.Now().AddDate(0, 0, -1)).
				Count(&count).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetDeletedCommentStatsFailed", consts.DatabaseReadFailedString))
				return
			}
			info += "24h内评论删帖数：" + strconv.Itoa(int(count)) + "\n"

			httpReturnInfo(c, info)
			return
		}
		c.Next()
	}
}

func adminReportsCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !viper.GetBool("allow_admin_commands") {
			c.Next()
			return
		}
		user := c.MustGet("user").(base.User)
		keywords := c.Query("keywords")
		if base.CanViewReports(&user) && keywords == "rep_dels" {
			page := c.MustGet("page").(int)
			offset := (page - 1) * consts.SearchPageSize
			limit := consts.SearchPageSize
			var reports []base.Report

			err := base.GetDb(false).Order("id desc").Where("type = ?", base.UserReport).
				Where("created_at > ?", time.Now().AddDate(0, 0, -1)).
				Limit(limit).Offset(offset).Find(&reports).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetReportedPostsFailed", consts.DatabaseReadFailedString))
				return
			}
			var data []gin.H
			for _, report := range reports {
				data = append(data, textToFrontendJson(report.ID, report.CreatedAt.Unix(), report.ToString()))
			}

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": utils.IfThenElse(data != nil, data, []string{}),
				//"timestamp": utils.GetTimeStamp(),
				"count":    utils.IfThenElse(data != nil, len(data), 0),
				"comments": map[string]string{},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func adminLogsCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !viper.GetBool("allow_admin_commands") {
			c.Next()
			return
		}
		user := c.MustGet("user").(base.User)
		keywords := c.Query("keywords")
		if base.CanViewLogs(&user) {
			if _, ok := utils.ContainsString([]string{"logs", "rep_dels", "rep_folds", "log_tags", "log_dels",
				"rep_recalls", "log_unbans"}, keywords); ok {

				page := c.MustGet("page").(int)
				offset := (page - 1) * consts.SearchPageSize
				limit := consts.SearchPageSize
				var reports []base.Report

				var err error
				if keywords == "logs" {
					err = base.GetDb(false).Order("id desc").
						Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "log_dels" {
					err = base.GetDb(false).Order("id desc").Where(base.GetDb(false).
						Where("type = ?", base.UserDelete).
						Where("user_id != reported_user_id")).
						Or("type = ?", base.AdminDeleteAndBan).Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "rep_recalls" {
					err = base.GetDb(false).Order("id desc").Where("type = ?", base.UserDelete).
						Where("user_id = reported_user_id").Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "rep_dels" {
					err = base.GetDb(false).Order("id desc").Where("type = ?", base.UserReport).
						Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "rep_folds" {
					err = base.GetDb(false).Order("id desc").Where("type = ?", base.UserReportFold).
						Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "log_tags" {
					err = base.GetDb(false).Order("id desc").Where("type = ?", base.AdminTag).
						Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "log_unbans" {
					err = base.GetDb(false).Order("id desc").Where("type in (?)",
						[]base.ReportType{base.AdminUnban, base.AdminUndelete}).
						Limit(limit).Offset(offset).Find(&reports).Error
				}
				if err != nil {
					base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "LogsCommandFailed", consts.DatabaseReadFailedString))
					return
				}
				var data []gin.H
				for _, report := range reports {
					data = append(data, textToFrontendJson(report.ID, report.CreatedAt.Unix(), report.ToDetailedString()))
				}

				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"data": utils.IfThenElse(data != nil, data, []string{}),
					//"timestamp": utils.GetTimeStamp(),
					"count":    utils.IfThenElse(data != nil, len(data), 0),
					"comments": map[string]string{},
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func adminSysMsgsCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !viper.GetBool("allow_admin_commands") {
			c.Next()
			return
		}
		user := c.MustGet("user").(base.User)
		keywords := c.Query("keywords")
		if base.CanViewAllSystemMessages(&user) && keywords == "msgs" {
			page := c.MustGet("page").(int)
			offset := (page - 1) * consts.SearchPageSize
			limit := consts.SearchPageSize
			var msgs []base.SystemMessage

			err := base.GetDb(false).Where("title != ?", "新的登录").Order("id desc").Limit(limit).Offset(offset).Find(&msgs).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "SysMsgCommandFailed", consts.DatabaseReadFailedString))
				return
			}
			var data []gin.H
			for _, msg := range msgs {
				data = append(data, textToFrontendJson(msg.ID, msg.CreatedAt.Unix(), msg.ToString()))
			}

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": utils.IfThenElse(data != nil, data, []string{}),
				//"timestamp": utils.GetTimeStamp(),
				"count":    utils.IfThenElse(data != nil, len(data), 0),
				"comments": map[string]string{},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

var shutdownCountDown int

func adminShutdownCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !viper.GetBool("allow_admin_commands") {
			c.Next()
			return
		}
		user := c.MustGet("user").(base.User)
		keywords := c.Query("keywords")
		if base.CanShutdown(&user) && keywords == "shutdown" {
			uidStr := strconv.Itoa(int(user.ID))
			log.Printf("Super user " + uidStr + " shutdown. shutdownCountDown=" + strconv.Itoa(shutdownCountDown))
			if shutdownCountDown > 0 {
				httpReturnInfo(c, strconv.Itoa(shutdownCountDown)+" more times to fully shutdown.")
				shutdownCountDown -= 1
				c.Abort()
			} else {
				os.Exit(0)
			}
			return
		}
		c.Next()
	}
}
