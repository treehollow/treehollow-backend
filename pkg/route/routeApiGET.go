package route

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/utils"
)

func getOne(c *gin.Context) {
	pid, err := strconv.Atoi(c.Query("pid"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "获取失败，pid不合法")
		return
	}

	token := c.Query("user_token")
	if !viper.GetBool("allow_unregistered_access") && !utils.IsInAllowedSubnet(c.ClientIP()) {
		_, err5 := db.GetInfoByToken(token)
		if err5 != nil {
			c.AbortWithStatus(401)
			return
		}
	}

	var text, tag, typ, filePath string
	var timestamp, likenum, replynum int
	_, text, timestamp, tag, typ, filePath, likenum, replynum, _, err = db.GetOnePost(pid)
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "没有这条树洞")
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"pid":       pid,
				"text":      text,
				"type":      typ,
				"timestamp": timestamp,
				"reply":     replynum,
				"likenum":   likenum,
				"url":       utils.GetHashedFilePath(filePath),
				"tag":       utils.IfThenElse(len(tag) != 0, tag, nil),
			},
			"timestamp": utils.GetTimeStamp(),
		})
		return
	}
}

func getComment(c *gin.Context) {
	pid, err := strconv.Atoi(c.Query("pid"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "获取失败，pid不合法")
		return
	}
	token := c.Query("user_token")
	attention := 0
	if len(token) == 32 {
		emailHash, err := db.GetInfoByToken(token)
		if err == nil {
			attention, _ = db.IsAttention(emailHash, pid)
		} else if !viper.GetBool("allow_unregistered_access") && !utils.IsInAllowedSubnet(c.ClientIP()) {
			c.AbortWithStatus(401)
			return
		}
	} else if !viper.GetBool("allow_unregistered_access") && !utils.IsInAllowedSubnet(c.ClientIP()) {
		c.AbortWithStatus(401)
		return
	}

	_, isAdmin := utils.ContainsString(viper.GetStringSlice("admins_tokens"), token)
	_, _, _, _, _, _, _, _, _, err3 := db.GetOnePost(pid)
	if err3 != nil && !isAdmin {
		utils.HttpReturnWithCodeOne(c, "pid不存在")
		return
	}
	data, err2 := db.GetSavedComments(pid)
	if err2 != nil {
		log.Printf("dbGetSavedComments failed: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      utils.IfThenElse(data != nil, data, []string{}),
			"attention": attention,
		})
		return
	}
}

func getList(c *gin.Context) {
	p, err := strconv.Atoi(c.Query("p"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "获取失败，参数p不合法")
		return
	}

	token := c.Query("user_token")
	if !viper.GetBool("allow_unregistered_access") && !utils.IsInAllowedSubnet(c.ClientIP()) {
		_, err5 := db.GetInfoByToken(token)
		if err5 != nil {
			//c.AbortWithStatus(401)
			utils.HttpReturnWithCodeOne(c, "登录凭据过期，请使用邮箱重新登录。")
			return
		}
	}

	var maxPid int
	maxPid, err = db.GetMaxPid()
	if err != nil {
		log.Printf("dbGetMaxPid failed: %s\n", err)
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      []string{},
			"timestamp": utils.GetTimeStamp(),
			"count":     0,
		})
		return
	}
	pidLeft := maxPid - p*consts.PageSize
	pidRight := maxPid - (p-1)*consts.PageSize
	data, err2 := db.GetSavedPosts(pidLeft, pidRight)
	if err2 != nil {
		log.Printf("dbGetSavedPosts failed while getList: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		pinnedPids := viper.GetIntSlice("pin_pids")
		if len(pinnedPids) > 0 && p == 1 {
			pinnedData, err3 := db.GetPostsByPidList(pinnedPids)
			if err3 != nil {
				log.Printf("get pinned post failed: %s\n", err2)
				utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
				return
			} else {
				rtnData := append(pinnedData, data...)
				c.JSON(http.StatusOK, gin.H{
					"code":      0,
					"data":      rtnData,
					"timestamp": utils.GetTimeStamp(),
					"count":     utils.IfThenElse(data != nil, len(rtnData), 0),
				})
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":      0,
				"data":      utils.IfThenElse(data != nil, data, []string{}),
				"timestamp": utils.GetTimeStamp(),
				"count":     utils.IfThenElse(data != nil, len(data), 0),
			})
			return
		}
	}
}

func httpReturnInfo(c *gin.Context, text string) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": []map[string]interface{}{gin.H{
			"pid":       66666666,
			"text":      text,
			"type":      "text",
			"timestamp": utils.GetTimeStamp(),
			"reply":     0,
			"likenum":   0,
			"url":       "",
			"tag":       nil,
		}},
		"timestamp": utils.GetTimeStamp(),
		"count":     1,
	})
}

var HotPosts []interface{}

func searchPost(c *gin.Context) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page > consts.SearchMaxPage || page <= 0 {
		utils.HttpReturnWithCodeOne(c, "获取失败，参数page不合法")
		return
	}
	pageSize, err := strconv.Atoi(c.Query("pagesize"))
	if err != nil || pageSize > consts.SearchMaxPageSize || pageSize <= 0 {
		utils.HttpReturnWithCodeOne(c, "获取失败，参数pagesize不合法")
		return
	}

	token := c.Query("user_token")
	if !viper.GetBool("allow_unregistered_access") && !utils.IsInAllowedSubnet(c.ClientIP()) {
		_, err5 := db.GetInfoByToken(token)
		if err5 != nil {
			c.AbortWithStatus(401)
			return
		}
	}

	keywords := c.Query("keywords")

	if len(keywords) > consts.SearchMaxLength {
		utils.HttpReturnWithCodeOne(c, "搜索内容过长")
		return
	}

	if keywords == "热榜" {
		rtn := utils.SafeSubSlice(HotPosts, (page-1)*pageSize, page*pageSize)
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      utils.IfThenElse(rtn != nil, rtn, []string{}),
			"timestamp": utils.GetTimeStamp(),
			"count":     utils.IfThenElse(rtn != nil, len(rtn), 0),
		})
		return
	}

	// Admin function
	setTagRe := regexp.MustCompile(`^settag (.*) (pid=|cid=|)(\d+)$`)
	_, isAdmin := utils.ContainsString(viper.GetStringSlice("admins_tokens"), token)
	if isAdmin && setTagRe.MatchString(keywords) {
		log.Printf("admin search action: token=%s, keywords=%s\n", token, keywords)
		strs := setTagRe.FindStringSubmatch(keywords)
		tag := strs[1]
		typ := strs[2]
		id, err2 := strconv.Atoi(strs[3])
		if err2 != nil {
			httpReturnInfo(c, strs[3]+" not valid")
			return
		}
		if typ == "pid=" || typ == "" {
			r, err := db.SetPostTagIns.Exec(tag, id)
			if err != nil {
				httpReturnInfo(c, "failed")
				return
			}
			rowsAffected, err2 := r.RowsAffected()
			httpReturnInfo(c, "rows affected = "+strconv.Itoa(int(rowsAffected))+"\nsuccess = "+strconv.FormatBool(err2 == nil))
			return
		} else if typ == "cid=" {
			r, err := db.SetCommentTagIns.Exec(tag, id)
			if err != nil {
				httpReturnInfo(c, "failed")
				return
			}
			rowsAffected, err2 := r.RowsAffected()
			httpReturnInfo(c, "rows affected = "+strconv.Itoa(int(rowsAffected))+"\nsuccess = "+strconv.FormatBool(err2 == nil))
			return
		}
	}

	delCommentRe := regexp.MustCompile(`^del (\d+) (.*)$`)
	if isAdmin && delCommentRe.MatchString(keywords) {
		log.Printf("admin search action: token=%s, keywords=%s\n", token, keywords)
		strs := delCommentRe.FindStringSubmatch(keywords)
		reason := strs[2]
		id, err2 := strconv.Atoi(strs[1])
		if err2 != nil {
			httpReturnInfo(c, strs[3]+" not valid")
			return
		}
		_, czEmailHash, text, _, _, _, err := db.GetOneComment(id)
		if err != nil {
			log.Printf("GetOneComment failed while delComment: %s\n", err)
			httpReturnInfo(c, "cid不存在")
			return
		}
		bannedTimes, err := db.BannedTimesPost(czEmailHash, -1)
		if err != nil {
			log.Printf("BannedTimesPost failed while delComment: %s\n", err)
			httpReturnInfo(c, "error while getting banned times")
			return
		}
		_, err = db.PlusCommentReportIns.Exec(666, id)
		if err != nil {
			log.Printf("PlusCommentReportIns failed while delComment: %s\n", err)
			httpReturnInfo(c, "error while updating reportnum")
			return
		}
		msg := "您的树洞评论#" + strconv.Itoa(id) + "\n\"" + text + "\"\n被管理员删除。管理员的删除理由是：【" + reason + "】。这是您第" +
			strconv.Itoa(bannedTimes+1) + "次被举报，在" + strconv.Itoa(bannedTimes+1) + "天之内您将无法发布树洞。"
		err = db.SaveBanUser(czEmailHash, msg, (1+bannedTimes)*86400)
		if err != nil {
			log.Printf("error dbSaveBanUser while delComment: %s\n", err)
			httpReturnInfo(c, "error while saving ban info")
			return
		}
		httpReturnInfo(c, "success")
		return
	}

	if isAdmin && keywords == "statistics" {
		httpReturnInfo(c, fmt.Sprintf("24h内注册用户：%d\n总注册用户：%d	", db.GetNewRegisterCountIn24h(), db.GetUserCount()))
		return
	}

	var data []interface{}
	if isAdmin && keywords == "deleted" {
		data, err = db.GetDeletedPosts((page-1)*pageSize, pageSize)
	} else if isAdmin && keywords == "bans" {
		data, err = db.GetBans((page-1)*pageSize, pageSize)
	} else if isAdmin && keywords == "reports" {
		data, err = db.GetReports((page-1)*10, 10)
	} else {
		data, err = db.SearchSavedPosts("+"+strings.ReplaceAll(keywords, " ", " +"), keywords, keywords, (page-1)*pageSize, pageSize)
	}
	if err != nil {
		log.Printf("dbSearchSavedPosts or dbGetDeletedPosts failed while searchList: %s\n", err)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      utils.IfThenElse(data != nil, data, []string{}),
			"timestamp": utils.GetTimeStamp(),
			"count":     utils.IfThenElse(data != nil, len(data), 0),
		})
		return
	}
}

func getAttention(c *gin.Context) {
	token := c.Query("user_token")
	emailHash, err := db.GetInfoByToken(token)

	if err != nil {
		utils.HttpReturnWithCodeOne(c, "操作失败，请检查登录状态")
		return
	}

	pids, err3 := db.GetAttentionPids(emailHash)
	if err3 != nil {
		log.Printf("dbGetAttentionPids failed while getAttention: %s\n", err3)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}

	if len(pids) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      []string{},
			"timestamp": utils.GetTimeStamp(),
			"count":     0,
		})
		return
	}

	data, err2 := db.GetPostsByPidList(pids)
	if err2 != nil {
		log.Printf("dbGetPostsByPidList failed while getAttention: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      utils.IfThenElse(data != nil, data, []string{}),
			"timestamp": utils.GetTimeStamp(),
			"count":     utils.IfThenElse(data != nil, len(data), 0),
		})
		return
	}
}

func apiGet(c *gin.Context) {
	action := c.Query("action")

	switch action {
	case "getone":
		getOne(c)
		return
	case "getcomment":
		getComment(c)
		return
	case "getlist":
		getList(c)
		return
	case "getattention":
		getAttention(c)
		return
	case "search":
		searchPost(c)
		return
	default:
		c.AbortWithStatus(403)
	}
}
