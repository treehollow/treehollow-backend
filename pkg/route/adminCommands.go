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
				info += "`statistics`: 查看树洞统计信息\n"
			}
			if permissions.CanViewDeletedPost(user) {
				info += "`deleted`: 搜索所有被删的树洞和回复\n"
				info += "`//setflag NOT_SHOW_DELETED=on`(注意大小写): 在除了`deleted`搜索界面外的其他界面隐藏被删除的树洞\n"
			}
			if permissions.CanViewAllSystemMessages(user) {
				info += "`messages`: 查看所有用户收到的系统消息\n"
			}
			if permissions.CanViewReports(user) {
				info += "`reports`: 查看所有用户的删除举报(树洞or回复)\n"
			}
			if permissions.CanViewActions(user) {
				info += "`folds`: 查看所有【用户举报折叠】的操作日志\n"
				info += "`set_tags`: 查看所有【管理员打Tag】的操作日志\n"
				info += "`deletes`: 查看所有的【撤回】或【管理员删除】\n"
				info += "`undelete_unbans`: 查看所有【撤销删除并解禁】的操作日志\n"
				info += "`delete_bans`: 查看所有【删帖禁言】的操作日志\n"
				info += "`unbans`: 查看所有用户【解禁】的操作日志\n"
				info += "`actions`: 查看所有举报、删帖、打tag的操作日志\n"
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
		if permissions.CanViewStatistics(user) && keywords == "statistics" {
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
		if permissions.CanViewReports(user) && keywords == "reports" {
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

func adminActionsCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(structs.User)
		keywords := c.Query("keywords")
		if permissions.CanViewActions(user) {
			if _, ok := utils.ContainsString([]string{"actions", "reports", "folds", "set_tags", "deletes",
				"undelete_unbans", "delete_bans", "unbans"}, keywords); ok {

				page := c.MustGet("page").(int)
				pageSize := c.MustGet("page_size").(int)
				offset := (page - 1) * pageSize
				limit := pageSize
				var reports []structs.Report

				var err error
				if keywords == "actions" {
					err = db.GetDb(false).Order("id desc").
						Limit(limit).Offset(offset).Find(&reports).Error
				} else {
					typ := getReportType(keywords[:len(keywords)-1])
					err = db.GetDb(false).Order("id desc").Where("type = ?", typ).
						Limit(limit).Offset(offset).Find(&reports).Error
				}
				if err != nil {
					log.Printf("search actions failed. err=%s\n", err)
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
		if permissions.CanViewAllSystemMessages(user) && keywords == "messages" {
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
