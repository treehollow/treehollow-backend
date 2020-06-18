package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
)

func getOne(c *gin.Context) {
	pid, err := strconv.Atoi(c.Query("pid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "获取失败，pid不合法",
		})
		return
	}
	var text, tag, typ, filePath string
	var timestamp, likenum, replynum int
	_, text, timestamp, tag, typ, filePath, likenum, replynum, err = getOnePost(pid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "获取失败，pid不存在",
		})
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
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "获取失败，pid不合法",
		})
		return
	}
	token := c.Query("user_token")
	attention := 0
	if len(token) == 32 {
		s, _, err := getInfoByToken(token)
		if err == nil {
			pids := hexToIntSlice(s)
			if _, ok := contains(pids, pid); ok {
				attention = 1
			}
		} else {
			log.Printf("getInfoByToken failed: %s\n", err)
		}
	}
	data, err2 := getSavedComments(pid)
	if err2 != nil {
		log.Printf("getSavedComments failed: %s\n", err2)
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "数据库读取失败，请联系管理员",
		})
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
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "获取失败，参数p不合法",
		})
		return
	}
	var maxPid int
	maxPid, err = getMaxPid()
	if err != nil {
		log.Printf("getMaxPid failed: %s\n", err)
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
	data, err2 := getSavedPosts(pidLeft, pidRight)
	if err2 != nil {
		log.Printf("getSavedPosts failed: %s\n", err2)
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "数据库读取失败，请联系管理员",
		})
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
	attentions, _, err := getInfoByToken(token)

	if err != nil {
		log.Printf("getInfoByToken failed: %s\n", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "操作失败，请检查登陆状态",
		})
		return
	}
	pids := hexToIntSlice(attentions)
	data, err2 := getPostsByPidList(pids)
	if err2 != nil {
		log.Printf("getSavedPosts failed: %s\n", err2)
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "数据库读取失败，请联系管理员",
		})
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

	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,Date,Content-Length")
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
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "搜索功能将在树洞条数达到几百条后开启！",
		})
		return
	default:
		c.AbortWithStatus(403)
	}
}
