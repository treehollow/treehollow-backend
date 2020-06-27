package route

import (
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
	var text, tag, typ, filePath string
	var timestamp, likenum, replynum int
	_, text, timestamp, tag, typ, filePath, likenum, replynum, _, err = db.GetOnePost(pid)
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "获取失败，pid不存在")
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
				"url":       filePath,
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
		}
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
		pinnedPids := utils.GetPinnedPids()
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
	keywords := c.Query("keywords")

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
	token := c.Query("user_token")
	setTagRe := regexp.MustCompile(`^settag (.*) (pid|cid)=(\d+)$`)
	isAdmin := strings.Contains(viper.GetString("report_admin_tokens"), token) &&
		len(token) == 32 && !strings.Contains(token, ",")
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
		if typ == "pid" {
			r, err := db.SetPostTagIns.Exec(tag, id)
			if err != nil {
				httpReturnInfo(c, "failed")
				return
			}
			rowsAffected, err2 := r.RowsAffected()
			httpReturnInfo(c, "rows affected = "+strconv.Itoa(int(rowsAffected))+"\nsuccess = "+strconv.FormatBool(err2 == nil))
			return
		} else if typ == "cid" {
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

	var data []interface{}
	if isAdmin && keywords == "deleted" {
		data, err = db.GetDeletedPosts((page-1)*pageSize, pageSize)
	} else if isAdmin && keywords == "bans" {
		data, err = db.GetBans((page-1)*pageSize, pageSize)
	} else if isAdmin && keywords == "reports" {
		data, err = db.GetReports((page-1)*pageSize, pageSize)
	} else {
		data, err = db.SearchSavedPosts(strings.ReplaceAll(keywords, " ", " +"), (page-1)*pageSize, pageSize)
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
		log.Printf("dbGetInfoByToken failed: %s\n", err)
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
