package route

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"strconv"
	"thuhole-go-backend/pkg/config"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/permissions"
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
	"unicode/utf8"
)

func commentToJson(comment structs.Comment, user structs.User) gin.H {
	offset := utils.CalcExtra(user.EmailHash, strconv.Itoa(int(comment.ID)))
	return gin.H{
		"cid":         comment.ID,
		"pid":         comment.PostID,
		"text":        "[" + comment.Name + "] " + comment.Text,
		"type":        comment.Type,
		"timestamp":   comment.CreatedAt.Unix() - offset,
		"url":         utils.GetHashedFilePath(comment.FilePath),
		"tag":         utils.IfThenElse(len(comment.Tag) != 0, comment.Tag, nil),
		"permissions": permissions.GetPermissionsByComment(user, comment),
		"deleted":     comment.DeletedAt.Valid,
		"name":        comment.Name,
	}
}

func commentsToJson(comments []structs.Comment, user structs.User) []gin.H {
	var data []gin.H
	for _, comment := range comments {
		data = append(data, commentToJson(comment, user))
	}
	return data
}

func detailPost(c *gin.Context) {
	pid, err := strconv.Atoi(c.Query("pid"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "获取失败，pid不合法")
		return
	}

	user := c.MustGet("user").(structs.User)
	canViewDelete := permissions.CanViewDeletedPost(user)

	var post structs.Post
	err3 := db.GetDb(canViewDelete).First(&post, int32(pid)).Error
	if err3 != nil {
		utils.HttpReturnWithCodeOne(c, "找不到这条树洞")
		return
	}
	var attention int64
	_ = db.GetDb(false).Model(&structs.Attention{}).Where(&structs.Attention{PostID: post.ID, UserID: user.ID}).Count(&attention).Error

	var comments []structs.Comment
	err2 := db.GetDb(canViewDelete).Where("post_id = ?", int32(pid)).Order("id asc").Find(&comments).Error
	if err2 != nil {
		log.Printf("dbGetSavedComments failed: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		data := commentsToJson(comments, user)
		post.ReplyNum = int32(len(comments))
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": utils.IfThenElse(data != nil, data, []string{}),
			"post": postToJson(post, user, attention == 1),
		})
		return
	}
}

func postToJson(post structs.Post, user structs.User, attention bool) gin.H {
	offset := utils.CalcExtra(user.EmailHash, strconv.Itoa(int(post.ID)))
	return gin.H{
		"pid":         post.ID,
		"text":        post.Text,
		"type":        post.Type,
		"timestamp":   post.CreatedAt.Unix() - offset,
		"updated_at":  post.UpdatedAt.Unix() - offset,
		"reply":       post.ReplyNum,
		"likenum":     post.LikeNum,
		"attention":   attention,
		"permissions": permissions.GetPermissionsByPost(user, post),
		"deleted":     post.DeletedAt.Valid,
		"url":         utils.GetHashedFilePath(post.FilePath),
		"tag":         utils.IfThenElse(len(post.Tag) != 0, post.Tag, nil),
	}
}

func postsToJson(posts []structs.Post, user structs.User, attentionPids []int32) []gin.H {
	var data []gin.H
	pidsSet := utils.Int32SliceToSet(attentionPids)
	for _, post := range posts {
		data = append(data, postToJson(post, user, utils.Int32IsInSet(post.ID, pidsSet)))
	}
	return data
}

func getAttentionPidsInPosts(user structs.User, posts []structs.Post) (attentionPids []int32, err error) {
	var pids []int32
	for _, post := range posts {
		pids = append(pids, post.ID)
	}
	err = db.GetDb(false).Model(&structs.Attention{}).Where("user_id = ? and post_id in ?", user.ID, pids).
		Pluck("post_id", &attentionPids).Error
	return
}

func listPost(c *gin.Context) {
	user := c.MustGet("user").(structs.User)
	canViewDelete := permissions.CanViewDeletedPost(user)
	page := c.MustGet("page").(int)
	posts, err2 := db.ListPosts(page, user)
	if err2 != nil {
		log.Printf("ListPosts failed: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}

	pinnedPids := viper.GetIntSlice("pin_pids")

	var configInfo gin.H
	if page == 1 {
		configInfo = config.GetFrontendConfigInfo()
		if len(pinnedPids) > 0 {
			var pinnedPosts []structs.Post
			err3 := db.GetDb(canViewDelete).Where(pinnedPids).Order("id desc").Find(&pinnedPosts).Error
			if err3 != nil {
				log.Printf("get pinned post failed: %s\n", err2)
				utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
				return
			} else {
				posts = append(pinnedPosts, posts...)
			}
		}
	}

	attentionPids, err3 := getAttentionPidsInPosts(user, posts)
	if err3 != nil {
		log.Printf("dbGetAttentionPids failed while list posts: %s\n", err3)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}
	jsPosts := postsToJson(posts, user, attentionPids)

	c.JSON(http.StatusOK, gin.H{
		"code":   0,
		"data":   utils.IfThenElse(jsPosts != nil, jsPosts, []string{}),
		"config": configInfo,
		//"timestamp": utils.GetTimeStamp(),
		"count": utils.IfThenElse(jsPosts != nil, len(jsPosts), 0),
	})
	return
}

var HotPosts []structs.Post

func searchPost(c *gin.Context) {
	page := c.MustGet("page").(int)
	pageSize := c.MustGet("page_size").(int)
	user := c.MustGet("user").(structs.User)
	keywords := c.Query("keywords")

	if utf8.RuneCountInString(keywords) > consts.SearchMaxLength {
		utils.HttpReturnWithCodeOne(c, "搜索内容过长")
		return
	}

	posts, err2 := db.SearchPosts(page, pageSize, keywords, nil, user)
	if err2 != nil {
		log.Printf("SearchPosts failed: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}
	attentionPids, err3 := getAttentionPidsInPosts(user, posts)
	if err3 != nil {
		log.Printf("dbGetAttentionPids failed while search posts: %s\n", err3)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}
	jsPosts := postsToJson(posts, user, attentionPids)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": utils.IfThenElse(jsPosts != nil, jsPosts, []string{}),
		//"timestamp": utils.GetTimeStamp(),
		"count": utils.IfThenElse(jsPosts != nil, len(jsPosts), 0),
	})
	return
}

func searchAttentionPost(c *gin.Context) {
	page := c.MustGet("page").(int)
	pageSize := c.MustGet("page_size").(int)
	user := c.MustGet("user").(structs.User)
	canViewDelete := permissions.CanViewDeletedPost(user)
	keywords := c.Query("keywords")

	if utf8.RuneCountInString(keywords) > consts.SearchMaxLength {
		utils.HttpReturnWithCodeOne(c, "搜索内容过长")
		return
	}

	var attentionPids []int32
	err3 := db.GetDb(canViewDelete).Model(&structs.Attention{}).
		Where("user_id = ?", user.ID).
		Pluck("post_id", &attentionPids).Error
	if err3 != nil {
		log.Printf("dbGetAttentionPids failed while search posts: %s\n", err3)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}

	posts, err2 := db.SearchPosts(page, pageSize, keywords, attentionPids, user)
	if err2 != nil {
		log.Printf("SearchPosts failed: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}
	jsPosts := postsToJson(posts, user, attentionPids)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": utils.IfThenElse(jsPosts != nil, jsPosts, []string{}),
		//"timestamp": utils.GetTimeStamp(),
		"count": utils.IfThenElse(jsPosts != nil, len(jsPosts), 0),
	})
	return
}

func attentionPosts(c *gin.Context) {
	page := c.MustGet("page").(int)

	user := c.MustGet("user").(structs.User)
	canViewDelete := permissions.CanViewDeletedPost(user)
	offset := (page - 1) * consts.PageSize
	limit := consts.PageSize

	var attentionPids []int32
	err3 := db.GetDb(canViewDelete).Model(&structs.Attention{}).
		Where("user_id = ?", user.ID).Order("post_id desc").Limit(limit).Offset(offset).
		Pluck("post_id", &attentionPids).Error
	if err3 != nil {
		log.Printf("dbGetAttentionPids failed while getAttention: %s\n", err3)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}

	var posts []structs.Post
	err2 := db.GetDb(canViewDelete).Where("id in ?", attentionPids).Order("id desc").Find(&posts).Error
	if err2 != nil {
		log.Printf("get posts failed while getAttention: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		data := postsToJson(posts, user, attentionPids)
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": utils.IfThenElse(data != nil, data, []string{}),
			//"timestamp": utils.GetTimeStamp(),
			"count": utils.IfThenElse(data != nil, len(data), 0),
		})
		return
	}
}

func systemMsg(c *gin.Context) {
	var msgs []structs.SystemMessage
	user := c.MustGet("user").(structs.User)
	err2 := db.GetDb(false).Where("user_id = ?", user.ID).Order("created_at desc").Find(&msgs).Error
	var data []gin.H
	for _, msg := range msgs {
		data = append(data, gin.H{
			"content":   msg.Text,
			"timestamp": msg.CreatedAt.Unix(),
			"title":     msg.Title,
		})
	}

	if err2 != nil {
		log.Printf("get systemMsg failed: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"error": nil,
			"result": utils.IfThenElse(data != nil, data, []gin.H{{
				"content":   "目前尚无系统消息",
				"timestamp": 0,
				"title":     "提示",
			}}),
		})
	}
}
