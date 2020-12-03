package route

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"strconv"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/permissions"
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
	"time"
)

func adminHelpCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(structs.User)
		keywords := c.Query("keywords")
		if permissions.CanShowHelp(user) && keywords == "help" {
			info := ""
			if permissions.CanViewStatistics(user) {
				info += "`stats`: 查看树洞统计信息\n"
			}
			if permissions.CanViewDeletedPost(user) {
				info += "`dels`: 搜索所有被管理员删除的树洞和回复(包括删除后恢复的)\n"
				info += "`//setflag NOT_SHOW_DELETED=on`(注意大小写): 在除了`deleted`搜索界面外的其他界面隐藏被删除的树洞\n"
			}
			if permissions.CanViewAllSystemMessages(user) {
				info += "`msgs`: 查看所有用户收到的系统消息\n"
			}
			if permissions.CanViewReports(user) {
				info += "`rep_dels`: 查看所有用户的【删除举报】(树洞or回复)\n"
			}
			if permissions.CanViewLogs(user) {
				info += "`rep_recalls`: 查看所有用户的【撤回】(树洞or回复)\n"
				info += "`rep_folds`: 查看所有用户的【折叠举报】(树洞or回复)\n"
				info += "`log_tags`: 查看所有【管理员打Tag】的操作日志\n"
				info += "`log_dels`: 查看所有的【管理员删除】\n"
				info += "`log_unbans`: 查看所有【撤销删除】、【解禁】的操作日志\n"
				info += "`logs`: 查看所有举报、删帖、打tag的操作日志\n"
			}
			if permissions.CanShutdown(user) {
				info += "`shutdown`: 关闭树洞, 请谨慎使用此命令\n"
			}
			httpReturnInfo(c, info)
			return
		}
		c.Next()
	}
}

func adminStatisticsCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(structs.User)
		keywords := c.Query("keywords")
		if permissions.CanViewStatistics(user) && keywords == "stats" {
			var count int64
			err := db.GetDb(true).Model(&structs.User{}).Count(&count).Error
			info := ""
			if err != nil {
				log.Printf("search user count failed. err=%s\n", err)
				utils.HttpReturnWithCodeOneAndAbort(c, "数据库读取失败，请联系管理员")
				return
			}
			info += "总注册人数：" + strconv.Itoa(int(count)) + "\n"

			err = db.GetDb(true).Model(&structs.Post{}).
				Where("created_at > ?", time.Unix(utils.GetTimeStamp()-86400, 0)).
				Count(&count).Error
			if err != nil {
				log.Printf("search 24h posts count failed. err=%s\n", err)
				utils.HttpReturnWithCodeOneAndAbort(c, "数据库读取失败，请联系管理员")
				return
			}
			info += "24h内发帖数：" + strconv.Itoa(int(count)) + "\n"

			err = db.GetDb(true).Model(&structs.Post{}).
				Where("deleted_at is not null and created_at > ?", time.Unix(utils.GetTimeStamp()-86400, 0)).
				Count(&count).Error
			if err != nil {
				log.Printf("search 24h deleted posts count failed. err=%s\n", err)
				utils.HttpReturnWithCodeOneAndAbort(c, "数据库读取失败，请联系管理员")
				return
			}
			info += "24h内删帖数：" + strconv.Itoa(int(count)) + "\n"

			httpReturnInfo(c, info)
			return
		}
		c.Next()
	}
}

func adminReportsCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(structs.User)
		keywords := c.Query("keywords")
		if permissions.CanViewReports(user) && keywords == "rep_dels" {
			page := c.MustGet("page").(int)
			pageSize := c.MustGet("page_size").(int)
			offset := (page - 1) * pageSize
			limit := pageSize
			var reports []structs.Report

			err := db.GetDb(false).Order("id desc").Where("type = ?", structs.UserReport).
				Where("created_at > ?", time.Now().AddDate(0, 0, -1)).
				Limit(limit).Offset(offset).Find(&reports).Error
			if err != nil {
				log.Printf("search reports failed. err=%s\n", err)
				utils.HttpReturnWithCodeOneAndAbort(c, "数据库读取失败，请联系管理员")
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
				"count": utils.IfThenElse(data != nil, len(data), 0),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func adminLogsCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(structs.User)
		keywords := c.Query("keywords")
		if permissions.CanViewLogs(user) {
			if _, ok := utils.ContainsString([]string{"logs", "rep_dels", "rep_folds", "log_tags", "log_dels",
				"rep_recalls", "log_unbans"}, keywords); ok {

				page := c.MustGet("page").(int)
				pageSize := c.MustGet("page_size").(int)
				offset := (page - 1) * pageSize
				limit := pageSize
				var reports []structs.Report

				var err error
				if keywords == "logs" {
					err = db.GetDb(false).Order("id desc").
						Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "log_dels" {
					err = db.GetDb(false).Order("id desc").Where(db.GetDb(false).
						Where("type = ?", structs.UserDelete).
						Where("user_id != reported_user_id")).
						Or("type = ?", structs.AdminDeleteAndBan).Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "rep_recalls" {
					err = db.GetDb(false).Order("id desc").Where("type = ?", structs.UserDelete).
						Where("user_id = reported_user_id").Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "rep_dels" {
					err = db.GetDb(false).Order("id desc").Where("type = ?", structs.UserReport).
						Where("user_id = reported_user_id").Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "rep_folds" {
					err = db.GetDb(false).Order("id desc").Where("type = ?", structs.UserReportFold).
						Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "log_tags" {
					err = db.GetDb(false).Order("id desc").Where("type = ?", structs.AdminTag).
						Limit(limit).Offset(offset).Find(&reports).Error
				} else if keywords == "log_unbans" {
					err = db.GetDb(false).Order("id desc").Where("type in (?)",
						[]structs.ReportType{structs.AdminUnban, structs.AdminUndelete}).
						Limit(limit).Offset(offset).Find(&reports).Error
				}
				if err != nil {
					log.Printf("search logs failed. err=%s\n", err)
					utils.HttpReturnWithCodeOneAndAbort(c, "数据库读取失败，请联系管理员")
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
					"count": utils.IfThenElse(data != nil, len(data), 0),
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
		user := c.MustGet("user").(structs.User)
		keywords := c.Query("keywords")
		if permissions.CanViewAllSystemMessages(user) && keywords == "msgs" {
			page := c.MustGet("page").(int)
			pageSize := c.MustGet("page_size").(int)
			offset := (page - 1) * pageSize
			limit := pageSize
			var msgs []structs.SystemMessage

			err := db.GetDb(false).Order("id desc").Limit(limit).Offset(offset).Find(&msgs).Error
			if err != nil {
				log.Printf("search system msgs failed. err=%s\n", err)
				utils.HttpReturnWithCodeOneAndAbort(c, "数据库读取失败，请联系管理员")
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
				"count": utils.IfThenElse(data != nil, len(data), 0),
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
		user := c.MustGet("user").(structs.User)
		keywords := c.Query("keywords")
		if permissions.CanShutdown(user) && keywords == "shutdown" {
			log.Printf("Super user " + user.EmailHash + " shutdown. shutdownCountDown=" + strconv.Itoa(shutdownCountDown))
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
