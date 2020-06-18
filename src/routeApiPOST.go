package main

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func doPost(c *gin.Context) {
	text := c.PostForm("text")
	typ := c.PostForm("type")
	token := c.PostForm("user_token")
	img := c.PostForm("data")
	if len(text) > postMaxLength {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "字数过长！字数限制为" + strconv.Itoa(postMaxLength) + "字。",
		})
		return
	} else if typ != "text" && typ != "image" {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "未知类型的树洞",
		})
		return
	} else if int(float64(len(img))/Base64Rate) > imgMaxLength {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "图片大小超出限制！",
		})
		return
	}

	var pid int
	var err error
	var imgPath string
	if typ == "image" {
		imgPath = genToken()
		pid, err = savePost(token, text, "", typ, imgPath+".jpeg")
	} else {
		pid, err = savePost(token, text, "", typ, "")
	}

	if err != nil {
		log.Printf("error savePost! %s\n", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "发送失败，请检查登陆状态",
		})
		return
	} else {
		if typ == "image" {
			sDec, err2 := base64.StdEncoding.DecodeString(img)
			if err2 != nil {
				c.JSON(http.StatusOK, gin.H{
					"code": 1,
					"msg":  "发送失败，图片数据不合法",
				})
				return
			}
			err3 := ioutil.WriteFile(viper.GetString("images_path")+imgPath+".jpeg", sDec, 0644)
			if err3 != nil {
				c.JSON(http.StatusOK, gin.H{
					"code": 1,
					"msg":  "图片写入失败，请联系管理员",
				})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": pid,
		})
		_, err = addAttention(token, pid)
		if err != nil {
			log.Printf("error add attention while sending post! %s\n", err)
		}
		return
	}
}

func doComment(c *gin.Context) {
	text := c.PostForm("text")
	if len(text) > commentMaxLength {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "字数过长！字数限制为" + strconv.Itoa(commentMaxLength) + "字。",
		})
		return
	}
	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "发送失败，pid不合法",
		})
		return
	}
	token := c.PostForm("user_token")
	s, emailHash, err := getInfoByToken(token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "发送失败，请检查登陆状态",
		})
		return
	}
	var dzEmailHash string
	dzEmailHash, _, _, _, _, _, _, _, err = getOnePost(pid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "发送失败，pid不存在",
		})
		return
	}

	var name string
	if dzEmailHash == emailHash {
		name = dzName
	} else {
		name, err = getCommentNameByToken(token, pid)
		if err != nil { // token is not in comments
			var i int
			i, err = getCommentCount(pid, dzEmailHash)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"code": 1,
					"msg":  "数据库读取失败，请联系管理员",
				})
				return
			}
			name = getCommenterName(i + 1)
		}
	}
	_, err = saveComment(token, text, "", pid, name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "数据库写入失败，请联系管理员",
		})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": pid,
		})

		_, err = plusOneCommentIns.Exec(pid)
		if err != nil {
			log.Printf("error plusOneCommentIns while commenting: %s\n", err)
		}
		_, err = addAttention2(s, token, pid)
		if err != nil {
			log.Printf("error addAttention2 while commenting: %s\n", err)
		}
	}
}

func doAttention(c *gin.Context) {
	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "关注操作失败，pid不合法",
		})
		return
	}
	s := c.PostForm("switch")
	token := c.PostForm("user_token")
	var success bool
	if s == "1" {
		success, err = addAttention(token, pid)
	} else {
		success, err = removeAttention(token, pid)
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "关注操作失败，请检查登陆状态",
		})
		return
	} else if !success {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "关注操作失败，重复操作",
		})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
		})
		return
	}
}

func apiPost(c *gin.Context) {
	action := c.Query("action")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Access-Control-Allow-Origin,Content-Type,Date,Content-Length")
	switch action {
	case "docomment":
		doComment(c)
		return
	case "dopost":
		doPost(c)
		return
	case "attention":
		doAttention(c)
		return
	case "report":
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"msg":  "额，这个功能还在开发。。。",
		})
		return
	default:
		c.AbortWithStatus(403)
	}
}
