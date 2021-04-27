package contents

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/iancoleman/orderedmap"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/config"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/model"
	"treehollow-v3-backend/pkg/utils"
	"unicode/utf8"
)

func commentToJson(comment *base.Comment, user *base.User) gin.H {
	offset := utils.CalcExtra(user.ForgetPwNonce, strconv.Itoa(int(comment.ID)))
	imageMetadata := map[string]int{}
	err2 := json.Unmarshal([]byte(comment.FileMetadata), &imageMetadata)
	if err2 != nil {
		log.Printf("bad image metadata in cid=%d: err=%s\n", comment.ID, err2)
	}
	return gin.H{
		"cid":            comment.ID,
		"pid":            comment.PostID,
		"text":           comment.Text,
		"type":           comment.Type,
		"timestamp":      comment.CreatedAt.Unix() - offset,
		"reply_to":       comment.ReplyTo,
		"url":            utils.GetHashedFilePath(comment.FilePath),
		"tag":            utils.IfThenElse(len(comment.Tag) != 0, comment.Tag, nil),
		"permissions":    base.GetPermissionsByComment(user, comment),
		"deleted":        comment.DeletedAt.Valid,
		"name":           comment.Name,
		"is_dz":          comment.Name == consts.DzName,
		"image_metadata": imageMetadata,
	}
}

func commentsToJson(comments []base.Comment, user *base.User) []gin.H {
	data := make([]gin.H, 0, len(comments))
	for _, comment := range comments {
		if !comment.DeletedAt.Valid || base.CanViewDeletedPost(user) {
			data = append(data, commentToJson(&comment, user))
		}
	}
	return data
}

func detailPost(c *gin.Context) {
	pid, err := strconv.Atoi(c.Query("pid"))
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("DetailPostPidNotInt", "获取失败，pid不合法", logger.WARN))
		return
	}

	user := c.MustGet("user").(base.User)
	canViewDelete := base.CanViewDeletedPost(&user)

	var post base.Post
	err3 := base.GetDb(canViewDelete).First(&post, int32(pid)).Error
	if err3 != nil {
		if errors.Is(err3, gorm.ErrRecordNotFound) {
			base.HttpReturnWithErr(c, -101, logger.NewSimpleError("DetailPostPidNotFound", "找不到这条树洞", logger.WARN))
		} else {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "DetailPostError", consts.DatabaseReadFailedString))
		}
		return
	}

	offset := utils.CalcExtra(user.ForgetPwNonce, strconv.Itoa(int(post.ID)))
	var attention int64
	_ = base.GetDb(false).Model(&base.Attention{}).Where(&base.Attention{PostID: post.ID, UserID: user.ID}).Count(&attention).Error

	votes, err4 := getVotesInPosts(base.GetDb(false), &user, []base.Post{post})
	if err4 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err4, "GetVotesInPostsFailed", consts.DatabaseReadFailedString))
		return
	}

	if (c.Query("include_comment") == "0") ||
		(c.Query("old_updated_at") == strconv.Itoa(int(post.UpdatedAt.Unix()-offset))) {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"data": nil,
			"post": postToJson(&post, &user, attention == 1, votes[post.ID]),
		})
		return
	}
	comments, err2 := base.GetCommentsWithCache(&post, time.Now())
	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "GetCommentsWithCacheFailed", consts.DatabaseReadFailedString))
		return
	}

	data := commentsToJson(comments, &user)
	post.ReplyNum = int32(len(data))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": utils.IfThenElse(data != nil, data, []string{}),
		"post": postToJson(&post, &user, attention == 1, votes[post.ID]),
	})
	return
}

func postToJson(post *base.Post, user *base.User, attention bool, voted string) gin.H {
	offset := utils.CalcExtra(user.ForgetPwNonce, strconv.Itoa(int(post.ID)))
	imageMetadata := map[string]int{}
	err2 := json.Unmarshal([]byte(post.FileMetadata), &imageMetadata)
	if err2 != nil {
		log.Printf("bad image metadata in pid=%d: err=%s\n", post.ID, err2)
	}
	tag := post.Tag
	if post.ReportNum >= 3 && !post.DeletedAt.Valid && tag == "" {
		tag = "举报较多"
	}
	vote := gin.H{}
	if len(post.VoteData) > 2 {
		voteData := orderedmap.New()
		err := json.Unmarshal([]byte(post.VoteData), &voteData)
		if err == nil {
			if len(voted) == 0 {
				for _, k := range voteData.Keys() {
					voteData.Set(k, -1)
				}
			}
			vote = gin.H{
				"voted":        voted,
				"vote_options": voteData.Keys(),
				"vote_data":    voteData,
			}
		} else {
			log.Printf("bad vote_data in pid=%d: err=%s\n", post.ID, err)
		}
	}
	return gin.H{
		"pid":            post.ID,
		"text":           post.Text,
		"type":           post.Type,
		"timestamp":      post.CreatedAt.Unix() - offset,
		"updated_at":     post.UpdatedAt.Unix() - offset,
		"reply":          post.ReplyNum,
		"likenum":        post.LikeNum,
		"attention":      attention,
		"permissions":    base.GetPermissionsByPost(user, post),
		"deleted":        post.DeletedAt.Valid,
		"url":            utils.GetHashedFilePath(post.FilePath),
		"tag":            utils.IfThenElse(len(tag) == 0, nil, tag),
		"image_metadata": imageMetadata,
		"vote":           vote,
	}
}

func postsToJson(posts []base.Post, user *base.User, attentionPids []int32, voted map[int32]string) []gin.H {
	data := make([]gin.H, 0, len(posts))
	attentionPidsSet := utils.Int32SliceToSet(attentionPids)
	for _, post := range posts {
		data = append(data, postToJson(&post, user, utils.Int32IsInSet(post.ID, attentionPidsSet), voted[post.ID]))
	}
	return data
}

func getAttentionPidsInPosts(tx *gorm.DB, user *base.User, posts []base.Post) (attentionPids []int32, err error) {
	pids := make([]int32, 0, len(posts))
	for _, post := range posts {
		pids = append(pids, post.ID)
	}
	err = tx.Model(&base.Attention{}).Where("user_id = ? and post_id in ?", user.ID, pids).
		Pluck("post_id", &attentionPids).Error
	return
}

func getVotesInPosts(tx *gorm.DB, user *base.User, posts []base.Post) (map[int32]string, error) {
	pids := make([]int32, 0, len(posts))
	for _, post := range posts {
		if len(post.VoteData) > 2 {
			pids = append(pids, post.ID)
		}
	}
	if len(pids) == 0 {
		return make(map[int32]string), nil
	}

	var votes []base.Vote
	err := tx.Model(&base.Vote{}).
		Where("user_id = ? and post_id in ?", user.ID, pids).
		Find(&votes).Error
	if err != nil {
		return nil, err
	}
	rtn := make(map[int32]string)
	for _, vote := range votes {
		rtn[vote.PostID] = vote.Option
	}
	return rtn, nil
}

func appendPostDetail(tx *gorm.DB, posts []base.Post, user *base.User) ([]gin.H, *logger.InternalError) {
	attentionPids, err3 := getAttentionPidsInPosts(tx, user, posts)
	if err3 != nil {
		return nil, logger.NewError(err3, "getAttentionPidsInPosts failed", consts.DatabaseReadFailedString)
	}
	votes, err4 := getVotesInPosts(tx, user, posts)
	if err4 != nil {
		return nil, logger.NewError(err4, "getVotesInPosts failed", consts.DatabaseReadFailedString)
	}
	jsPosts := postsToJson(posts, user, attentionPids, votes)
	return jsPosts, nil
}

func getCommentsByPosts(posts []base.Post, user *base.User) (map[int32][]gin.H, *logger.InternalError) {
	comments := make(map[int32][]gin.H)
	commentsMap, err4 := base.GetMultipleCommentsWithCache(base.GetDb(false), posts, time.Now())
	if err4 != nil {
		return nil, err4
	}
	//TODO: (low priority) update reply_num
	for pid, tmp := range commentsMap {
		if len(tmp) > 3 {
			tmp = tmp[:3]
		}
		if len(tmp) > 0 {
			comments[pid] = commentsToJson(tmp, user)
		}
	}
	return comments, nil
}

func listPost(c *gin.Context) {
	user := c.MustGet("user").(base.User)
	canViewDelete := base.CanViewDeletedPost(&user)
	page := c.MustGet("page").(int)
	posts, err2 := base.ListPosts(base.GetDb(false), page, &user)
	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "ListPostsFailed", consts.DatabaseReadFailedString))
		return
	}

	pinnedPids := viper.GetIntSlice("pin_pids")

	var configInfo gin.H
	if page == 1 {
		configInfo = config.GetFrontendConfigInfo()
		if len(pinnedPids) > 0 {
			var pinnedPosts []base.Post
			err3 := base.GetDb(canViewDelete).Where(pinnedPids).Order("id desc").Find(&pinnedPosts).Error
			if err3 != nil {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "GetPinnedPostsFailed", consts.DatabaseReadFailedString))
				return
			} else {
				posts = append(pinnedPosts, posts...)
			}
		}
	}

	jsPosts, err := appendPostDetail(base.GetDb(false), posts, &user)
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, err)
		return
	}

	comments, err4 := getCommentsByPosts(posts, &user)
	if err4 != nil {
		base.HttpReturnWithCodeMinusOne(c, err4)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   0,
		"data":   utils.IfThenElse(jsPosts != nil, jsPosts, []string{}),
		"config": configInfo,
		//"timestamp": utils.GetTimeStamp(),
		"count":    utils.IfThenElse(jsPosts != nil, len(jsPosts), 0),
		"comments": comments,
	})
	return
}

func wanderListPost(c *gin.Context) {
	user := c.MustGet("user").(base.User)
	canViewDelete := base.CanViewDeletedPost(&user)
	var posts []base.Post

	var maxId int32
	err2 := base.GetDb(canViewDelete).Model(&base.Post{}).Select("max(id)").First(&maxId).Error
	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "GetMaxPidFailed", consts.DatabaseReadFailedString))
		return
	}
	pids := make([]int32, 0, consts.WanderPageSize)
	for i := 0; i < consts.WanderPageSize; i++ {
		pids = append(pids, 1+int32(rand.Intn(int(maxId))))
	}
	err2 = base.GetDb(canViewDelete).Where("id in (?)", pids).Order("RAND()").Find(&posts).Error
	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "GetWanderPosts", consts.DatabaseReadFailedString))
		return
	}

	var posts2 []base.Post

	reportableTags := viper.GetStringSlice("reportable_tags")
	inactiveRangeStart := viper.GetIntSlice("inactive_pid_range_start")
	inactiveRangeEnd := viper.GetIntSlice("inactive_pid_range_end")
	checkID := func(id int) bool {
		if len(inactiveRangeStart) != len(inactiveRangeEnd) {
			return false
		}
		for i := range inactiveRangeStart {
			if id >= inactiveRangeStart[i] && id < inactiveRangeEnd[i] {
				return true
			}
		}
		return false
	}
	for i := 0; i < 10; i++ {
		posts2 = make([]base.Post, 0, len(posts))
		for _, post := range posts {
			if _, b := utils.ContainsString(reportableTags, post.Tag); b || checkID(int(post.ID)) {
				if rand.Float32() > 0.1 {
					continue
				}
			}
			posts2 = append(posts2, post)
		}
		if len(posts2) > 0 {
			break
		}
	}
	jsPosts, err := appendPostDetail(base.GetDb(false), posts2, &user)
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": utils.IfThenElse(jsPosts != nil, jsPosts, []string{}),
		//"timestamp": utils.GetTimeStamp(),
		"count": utils.IfThenElse(jsPosts != nil, len(jsPosts), 0),
	})
	return
}

func searchPost(c *gin.Context) {
	page := c.MustGet("page").(int)
	user := c.MustGet("user").(base.User)
	keywords := c.Query("keywords")
	includeComment := c.Query("include_comment") != "false"
	beforeDate := c.Query("before")
	beforeTimestamp, err := strconv.ParseInt(beforeDate, 10, 64)
	if err != nil {
		beforeTimestamp = -1
	}
	afterDate := c.Query("after")
	afterTimestamp, err := strconv.ParseInt(afterDate, 10, 64)
	if err != nil {
		afterTimestamp = -1
	}

	if utf8.RuneCountInString(keywords) > consts.SearchMaxLength {
		base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("TooLongKeywords", "搜索内容过长", logger.WARN))
		return
	}

	posts, err2 := base.SearchPosts(page, keywords, nil, user,
		model.SearchOrderFromString(c.Query("order")), includeComment, beforeTimestamp, afterTimestamp)
	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "SearchPostsFailed", consts.DatabaseReadFailedString))
		return
	}

	jsPosts, err3 := appendPostDetail(base.GetDb(false), posts, &user)
	if err3 != nil {
		base.HttpReturnWithCodeMinusOne(c, err3)
		return
	}

	keywordsSlice := strings.Split(keywords, " ")
	comments := make(map[int32][]gin.H)
	commentsMap, err5 := base.GetMultipleCommentsWithCache(base.GetDb(false), posts, time.Now())
	if err5 != nil {
		base.HttpReturnWithCodeMinusOne(c, err5)
		return
	}
	//TODO: (low priority) update reply_num
	for pid, tmp := range commentsMap {
		var commentsContainsKeywords []base.Comment
		for _, comment := range tmp {
			//TODO: (low priority) check if keyword is #tag
			for _, keyword := range keywordsSlice {
				if strings.Contains(comment.Text, keyword) {
					commentsContainsKeywords = append(commentsContainsKeywords, comment)
					break
				}
			}
			//if len(commentsContainsKeywords) >= 3 {
			//	break
			//}
		}
		if len(commentsContainsKeywords) > 0 {
			comments[pid] = commentsToJson(commentsContainsKeywords, &user)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": utils.IfThenElse(jsPosts != nil, jsPosts, []string{}),
		//"timestamp": utils.GetTimeStamp(),
		"count":    utils.IfThenElse(jsPosts != nil, len(jsPosts), 0),
		"comments": comments,
	})
	return
}

func searchAttentionPost(c *gin.Context) {
	page := c.MustGet("page").(int)
	user := c.MustGet("user").(base.User)
	canViewDelete := base.CanViewDeletedPost(&user)
	keywords := c.Query("keywords")

	if utf8.RuneCountInString(keywords) > consts.SearchMaxLength {
		base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("TooLongKeywords", "搜索内容过长", logger.WARN))
		return
	}

	var attentionPids []int32
	err3 := base.GetDb(canViewDelete).Model(&base.Attention{}).
		Where("user_id = ?", user.ID).
		Pluck("post_id", &attentionPids).Error
	if err3 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "GetAttentionPidsFailed", consts.DatabaseReadFailedString))
		return
	}

	posts, err2 := base.SearchPosts(page, keywords, attentionPids, user,
		model.SearchOrderFromString(c.Query("order")), true, -1, -1)
	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "SearchPostsFailed", consts.DatabaseReadFailedString))
		return
	}
	votes, err4 := getVotesInPosts(base.GetDb(false), &user, posts)
	if err4 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err4, "GetVotesInPostsFailed", consts.DatabaseReadFailedString))
		return
	}
	jsPosts := postsToJson(posts, &user, attentionPids, votes)

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

	user := c.MustGet("user").(base.User)
	canViewDelete := base.CanViewDeletedPost(&user)
	offset := (page - 1) * consts.PageSize
	limit := consts.PageSize

	var attentionPids []int32
	err3 := base.GetDb(canViewDelete).Model(&base.Attention{}).
		Where("user_id = ?", user.ID).Order("post_id desc").Limit(limit).Offset(offset).
		Pluck("post_id", &attentionPids).Error
	if err3 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "GetAttentionPidsFailed", consts.DatabaseReadFailedString))
		return
	}

	var posts []base.Post
	err2 := base.GetDb(canViewDelete).Where("id in ?", attentionPids).Order("id desc").Find(&posts).Error
	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "GetAttentionPostsFailed", consts.DatabaseReadFailedString))
		return
	}
	votes, err4 := getVotesInPosts(base.GetDb(false), &user, posts)
	if err4 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err4, "GetAttentionPostsVoteFailed", consts.DatabaseReadFailedString))
		return
	}

	comments, err5 := getCommentsByPosts(posts, &user)
	if err5 != nil {
		base.HttpReturnWithCodeMinusOne(c, err5)
		return
	}

	data := postsToJson(posts, &user, attentionPids, votes)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": utils.IfThenElse(data != nil, data, []string{}),
		//"timestamp": utils.GetTimeStamp(),
		"count":    utils.IfThenElse(data != nil, len(data), 0),
		"comments": comments,
	})
	return

}

func systemMsg(c *gin.Context) {
	var msgs []base.SystemMessage
	user := c.MustGet("user").(base.User)
	err2 := base.GetDb(false).Where("user_id = ?", user.ID).Order("created_at desc").Find(&msgs).Error
	data := make([]gin.H, 0, len(msgs))
	for _, msg := range msgs {
		data = append(data, gin.H{
			"content":   msg.Text,
			"timestamp": msg.CreatedAt.Unix(),
			"title":     msg.Title,
		})
	}

	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "GetSysMsgFailed", consts.DatabaseReadFailedString))
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": utils.IfThenElse(data != nil, data, []gin.H{{
				"content":   "目前尚无系统消息",
				"timestamp": 0,
				"title":     "提示",
			}}),
		})
	}
}

func myMsgs(c *gin.Context) {
	user := c.MustGet("user").(base.User)
	page := c.MustGet("page").(int)
	pushOnly := c.Query("push_only") == "1"

	sinceId, err := strconv.Atoi(c.Query("since_id"))
	if err != nil {
		sinceId = -1
	}

	msgs, err2 := base.ListMsgs(page, int32(sinceId), user.ID, pushOnly)
	if err2 != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "ListMsgsFailed", consts.DatabaseReadFailedString))
		return
	}
	var data []gin.H
	for _, msg := range msgs {
		p := gin.H{
			"id":        msg.ID,
			"title":     msg.Title,
			"body":      utils.TrimText(msg.Message, 100),
			"type":      msg.Type,
			"timestamp": msg.UpdatedAt.Unix(),
		}
		if (msg.Type & (model.ReplyMeComment | model.CommentInFavorited)) > 0 {
			p["pid"] = msg.PostID
			p["cid"] = msg.CommentID
		}
		data = append(data, p)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": utils.IfThenElse(data != nil, data, []string{}),
	})
	return
}
