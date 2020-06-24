package main

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func getOne(c *gin.Context) {
	pid, err := strconv.Atoi(c.Query("pid"))
	if err != nil {
		httpReturnWithCodeOne(c, "获取失败，pid不合法")
		return
	}
	var text, tag, typ, filePath string
	var timestamp, likenum, replynum int
	_, text, timestamp, tag, typ, filePath, likenum, replynum, _, err = dbGetOnePost(pid)
	if err != nil {
		httpReturnWithCodeOne(c, "获取失败，pid不存在")
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
				"tag":       IfThenElse(len(tag) != 0, tag, nil),
			},
			"timestamp": getTimeStamp(),
		})
		return
	}
}

func getComment(c *gin.Context) {
	pid, err := strconv.Atoi(c.Query("pid"))
	if err != nil {
		httpReturnWithCodeOne(c, "获取失败，pid不合法")
		return
	}
	token := c.Query("user_token")
	attention := 0
	if len(token) == 32 {
		s, _, err := dbGetInfoByToken(token)
		if err == nil {
			pids := hexToIntSlice(s)
			if _, ok := containsInt(pids, pid); ok {
				attention = 1
			}
		}
	}
	data, err2 := dbGetSavedComments(pid)
	if err2 != nil {
		log.Printf("dbGetSavedComments failed: %s\n", err2)
		httpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      IfThenElse(data != nil, data, []string{}),
			"attention": attention,
		})
		return
	}
}

func getList(c *gin.Context) {
	p, err := strconv.Atoi(c.Query("p"))
	if err != nil {
		httpReturnWithCodeOne(c, "获取失败，参数p不合法")
		return
	}
	var maxPid int
	maxPid, err = dbGetMaxPid()
	if err != nil {
		log.Printf("dbGetMaxPid failed: %s\n", err)
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      []string{},
			"timestamp": getTimeStamp(),
			"count":     0,
		})
		return
	}
	pidLeft := maxPid - p*pageSize
	pidRight := maxPid - (p-1)*pageSize
	data, err2 := dbGetSavedPosts(pidLeft, pidRight)
	if err2 != nil {
		log.Printf("dbGetSavedPosts failed while getList: %s\n", err2)
		httpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		pinnedPids := getPinnedPids()
		if len(pinnedPids) > 0 && p == 1 {
			pinnedData, err3 := dbGetPostsByPidList(pinnedPids)
			if err3 != nil {
				log.Printf("get pinned post failed: %s\n", err2)
				httpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
				return
			} else {
				rtnData := append(pinnedData, data...)
				c.JSON(http.StatusOK, gin.H{
					"code":      0,
					"data":      rtnData,
					"timestamp": getTimeStamp(),
					"count":     IfThenElse(data != nil, len(rtnData), 0),
				})
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":      0,
				"data":      IfThenElse(data != nil, data, []string{}),
				"timestamp": getTimeStamp(),
				"count":     IfThenElse(data != nil, len(data), 0),
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
			"timestamp": getTimeStamp(),
			"reply":     0,
			"likenum":   0,
			"url":       "",
			"tag":       nil,
		}},
		"timestamp": getTimeStamp(),
		"count":     1,
	})
}

var hotPosts []interface{}

func searchPost(c *gin.Context) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page > searchMaxPage || page <= 0 {
		httpReturnWithCodeOne(c, "获取失败，参数page不合法")
		return
	}
	pageSize, err := strconv.Atoi(c.Query("pagesize"))
	if err != nil || pageSize > searchMaxPageSize || pageSize <= 0 {
		httpReturnWithCodeOne(c, "获取失败，参数pagesize不合法")
		return
	}
	keywords := c.Query("keywords")

	if keywords == "热榜" {
		rtn := safeSubSlice(hotPosts, (page-1)*pageSize, page*pageSize)
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      IfThenElse(rtn != nil, rtn, []string{}),
			"timestamp": getTimeStamp(),
			"count":     IfThenElse(rtn != nil, len(rtn), 0),
		})
		return
	}

	// Admin function
	token := c.Query("user_token")
	setTagRe := regexp.MustCompile(`^settag (.*) (pid|cid)=(\d+)$`)
	if strings.Contains(viper.GetString("report_admin_tokens"), token) && setTagRe.MatchString(keywords) {
		strs := setTagRe.FindStringSubmatch(keywords)
		tag := strs[1]
		typ := strs[2]
		id, err2 := strconv.Atoi(strs[3])
		if err2 != nil {
			httpReturnInfo(c, typ+"not valid")
			return
		}
		if typ == "pid" {
			r, err := setPostTagIns.Exec(tag, id)
			if err != nil {
				httpReturnInfo(c, "failed")
				return
			}
			rowsAffected, err2 := r.RowsAffected()
			httpReturnInfo(c, "rows affected = "+strconv.Itoa(int(rowsAffected))+"\nsuccess = "+strconv.FormatBool(err2 == nil))
			return
		} else if typ == "cid" {
			r, err := setCommentTagIns.Exec(tag, id)
			if err != nil {
				httpReturnInfo(c, "failed")
				return
			}
			rowsAffected, err2 := r.RowsAffected()
			httpReturnInfo(c, "rows affected = "+strconv.Itoa(int(rowsAffected))+"\nsuccess = "+strconv.FormatBool(err2 == nil))
			return
		}
	}

	data, err2 := dbSearchSavedPosts(strings.ReplaceAll(keywords, " ", " +"), (page-1)*pageSize, pageSize)
	if err2 != nil {
		log.Printf("dbSearchSavedPosts failed while searchList: %s\n", err2)
		httpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      IfThenElse(data != nil, data, []string{}),
			"timestamp": getTimeStamp(),
			"count":     IfThenElse(data != nil, len(data), 0),
		})
		return
	}
}

func getAttention(c *gin.Context) {
	token := c.Query("user_token")
	attentions, _, err := dbGetInfoByToken(token)

	if err != nil {
		log.Printf("dbGetInfoByToken failed: %s\n", err)
		httpReturnWithCodeOne(c, "操作失败，请检查登录状态")
		return
	}

	pids := hexToIntSlice(attentions)
	if len(pids) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      []string{},
			"timestamp": getTimeStamp(),
			"count":     0,
		})
		return
	}
	data, err2 := dbGetPostsByPidList(pids)
	if err2 != nil {
		log.Printf("dbGetPostsByPidList failed while getAttention: %s\n", err2)
		httpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":      0,
			"data":      IfThenElse(data != nil, data, []string{}),
			"timestamp": getTimeStamp(),
			"count":     IfThenElse(data != nil, len(data), 0),
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
