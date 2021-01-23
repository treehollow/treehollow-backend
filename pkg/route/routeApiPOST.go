package route

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/permissions"
	"thuhole-go-backend/pkg/s3"
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
	"time"
)

func generateTag(text string) string {
	re := regexp.MustCompile(`[#＃](性相关|政治相关|引战|未经证实的传闻|令人不适|NSFW|nsfw|折叠)`)
	if re.MatchString(text) {
		return strings.ToUpper(re.FindStringSubmatch(text)[1])
	}
	re1, err := regexp.Compile(viper.GetString("fold_regex"))
	if err == nil && re1.MatchString(text) {
		return "折叠"
	}
	re2, err2 := regexp.Compile(viper.GetString("sex_related_regex"))
	if err2 == nil && re2.MatchString(text) {
		return "性相关"
	}
	return ""
}

func sendPost(c *gin.Context) {
	text := c.PostForm("text")
	typ := c.PostForm("type")
	img := c.PostForm("data")
	user := c.MustGet("user").(structs.User)

	tag := generateTag(text)

	var pid int32
	var err error
	var imgPath string
	var uploadChan chan bool
	uploadChan = nil
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if typ == "image" {
		imgPath = utils.GenToken()
		sDec, suffix, err2 := utils.SaveImage(img, imgPath)
		if err2 != nil {
			utils.HttpReturnWithCodeOne(c, err2.Error())
			return
		}

		pid, err = db.SavePost(user.ID, text, tag, typ, imgPath+suffix)
		if err == nil && len(viper.GetString("DCSecretKey")) > 0 {
			uploadChan = make(chan bool, 1)
			go func() {
				err4 := s3.Upload(imgPath[:2]+"/"+imgPath+suffix, bytes.NewReader(sDec))
				if err4 != nil {
					log.Printf("S3 upload failed, err=%s\n", err4)
				}

				select {
				default:
					uploadChan <- true
				case <-ctx.Done():
					return
				}
			}()
		}
	} else {
		pid, err = db.SavePost(user.ID, text, tag, typ, "")
	}

	if err != nil {
		log.Printf("error dbSavePost! %s\n", err)
		utils.HttpReturnWithCodeOne(c, "数据库写入失败，请联系管理员")
		return
	} else {
		if uploadChan != nil {
			//wait until upload complete.
			select {
			case <-uploadChan:

			case <-time.After(15 * time.Second):
				log.Println("image upload timeout")
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": pid,
		})
		return
	}
}

var commentMux sync.Mutex

func sendComment(c *gin.Context) {
	text := c.PostForm("text")
	typ := c.PostForm("type")
	img := c.PostForm("data")
	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "发送失败，pid不合法")
		return
	}

	user := c.MustGet("user").(structs.User)
	canViewDelete := permissions.CanViewDeletedPost(&user)

	var post structs.Post
	err = db.GetDb(canViewDelete).First(&post, int32(pid)).Error
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "发送失败，pid不存在")
		return
	}

	var name string
	names0, names1 := consts.Names0, consts.Names1
	commentMux.Lock()

	name, err = db.GenCommenterName(post.UserID, user.ID, post.ID, names0, names1)
	if err != nil {
		log.Printf("error GenCommenterName in doComment(), err=%s\n", err.Error())
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		commentMux.Unlock()
		return
	}

	var imgPath string
	var uploadChan chan bool
	uploadChan = nil
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if typ == "image" {
		imgPath = utils.GenToken()
		sDec, suffix, err2 := utils.SaveImage(img, imgPath)
		if err2 != nil {
			utils.HttpReturnWithCodeOne(c, err2.Error())
			commentMux.Unlock()
			return
		}

		_, err = db.SaveComment(user.ID, text, "", typ, imgPath+suffix, int32(pid), name)
		if err == nil && len(viper.GetString("DCSecretKey")) > 0 {
			uploadChan = make(chan bool, 1)
			go func() {
				err4 := s3.Upload(imgPath[:2]+"/"+imgPath+suffix, bytes.NewReader(sDec))
				if err4 != nil {
					log.Printf("S3 upload failed, err=%s\n", err4)
				}

				select {
				default:
					uploadChan <- true
				case <-ctx.Done():
					return
				}
			}()
		}
	} else {
		_, err = db.SaveComment(user.ID, text, "", "text", "", int32(pid), name)
	}
	commentMux.Unlock()

	if err != nil {
		utils.HttpReturnWithCodeOne(c, "数据库写入失败，请联系管理员")
		return
	} else {

		if user.ID == post.UserID {
			re := regexp.MustCompile(`[#＃](性相关|政治相关|引战|未经证实的传闻|令人不适|NSFW|nsfw|折叠|重复内容)`)
			if re.MatchString(text) {
				tag := strings.ToUpper(re.FindStringSubmatch(text)[1])
				_ = db.GetDb(canViewDelete).Model(&structs.Post{}).Where("id = ?", post.ID).Update("tag", tag)
			}
		}

		if uploadChan != nil {
			//wait until upload complete.
			select {
			case <-uploadChan:

			case <-time.After(15 * time.Second):
				log.Println("image upload timeout")
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": pid,
		})
	}
}

func getReportType(typ string) structs.ReportType {
	switch typ {
	case "report":
		return structs.UserReport
	case "fold":
		return structs.UserReportFold
	case "set_tag":
		return structs.AdminTag
	case "delete":
		return structs.UserDelete
	case "undelete_unban":
		return structs.AdminUndelete
	case "delete_ban":
		return structs.AdminDeleteAndBan
	case "unban":
		return structs.AdminUnban
	default:
		return structs.UserReport
	}
}

func getPostOrCommentText(c *gin.Context, isComment bool) string {
	if isComment {
		return c.MustGet("comment").(structs.Comment).Text
	}
	return c.MustGet("post").(structs.Post).Text
}

func handleReport(c *gin.Context) {
	report := c.MustGet("report").(structs.Report)
	user := c.MustGet("user").(structs.User)
	err := db.GetDb(false).Create(&report).Error
	if err == nil {
		switch report.Type {
		case structs.UserReport:
			var reportScore int64
			err = db.GetDb(false).Model(&structs.Report{}).Select("SUM(weight)").Where(&structs.Report{
				PostID:    report.PostID,
				CommentID: report.CommentID,
				IsComment: report.IsComment,
				Type:      structs.UserReport,
			}).First(&reportScore).Error
			if reportScore >= 100 && err == nil {
				err = db.DeleteAndBan(report, utils.TrimText(getPostOrCommentText(c, report.IsComment), 20))
			}
		case structs.UserReportFold:
			if report.ReportedUserID == report.UserID && !report.IsComment {
				err = db.SetTagByReport(report)
			} else {
				var reportScore int64
				err = db.GetDb(false).Model(&structs.Report{}).Where(&structs.Report{
					PostID:    report.PostID,
					CommentID: report.CommentID,
					IsComment: report.IsComment,
					Reason:    report.Reason,
					Type:      structs.UserReportFold,
				}).Count(&reportScore).Error
				if reportScore == 2 && err == nil {
					err = db.SetTagByReport(report)
				}
			}
		case structs.UserDelete:
			err = db.DeleteByReport(report)
		case structs.AdminTag:
			err = db.SetTagByReport(report)
		case structs.AdminDeleteAndBan:
			uidStr := strconv.Itoa(int(user.ID))
			ctx, _ := deleteBanLimiter.Peek(c, uidStr)
			limit := permissions.GetDeletePostRateLimitIn24h(user.Role)
			if ctx.Limit-ctx.Remaining >= limit {
				utils.HttpReturnWithCodeOne(c, "您的24h内的删帖数量已经达到系统限制")
				return
			}
			err = db.DeleteAndBan(report, utils.TrimText(getPostOrCommentText(c, report.IsComment), 20))
			if err == nil {
				_, _ = deleteBanLimiter.Get(c, uidStr)
			}
		case structs.AdminUndelete:
			_ = db.UnbanByReport(report)
			if report.IsComment {
				err = db.GetDb(false).Model(&structs.Comment{}).
					Where("id = ?", report.CommentID).Update("deleted_at", nil).Error
				db.DelCommentCache(int(report.PostID))
			} else {
				err = db.GetDb(false).Model(&structs.Post{}).
					Where("id = ?", report.PostID).Update("deleted_at", nil).Error
			}
		case structs.AdminUnban:
			err = db.UnbanByReport(report)
			if err != nil {
				utils.HttpReturnWithCodeOne(c, "没有找到相关的封禁，可能已经被解封了")
				return
			}
		}
	}

	if err != nil {
		utils.HttpReturnWithCodeOne(c, "数据库写入失败，请联系管理员")
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
		})
	}
	return

}

func editAttention(c *gin.Context) {
	user := c.MustGet("user").(structs.User)
	canViewDelete := permissions.CanViewDeletedPost(&user)

	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "关注操作失败，pid不合法")
		return
	}
	var post structs.Post
	err3 := db.GetDb(canViewDelete).First(&post, int32(pid)).Error
	if err3 != nil {
		utils.HttpReturnWithCodeOne(c, "关注失败，pid不存在")
		return
	}
	s := c.PostForm("switch")

	var isAttention int64
	err2 := db.GetDb(false).Model(&structs.Attention{}).Where(&structs.Attention{PostID: post.ID, UserID: user.ID}).Count(&isAttention).Error
	if err2 != nil {
		log.Printf("error dbIsAttention while doAttention: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}
	if isAttention == 0 && s == "0" {
		utils.HttpReturnWithCodeOne(c, "您已经取消关注了")
		return
	}
	if isAttention == 1 && s == "1" {
		utils.HttpReturnWithCodeOne(c, "您已经关注过了")
		return
	}
	if isAttention == 0 {
		_ = db.GetDb(false).Create(&structs.Attention{UserID: user.ID, PostID: post.ID}).Error
		post.LikeNum += 1
	}
	if isAttention == 1 {
		_ = db.GetDb(false).Delete(&structs.Attention{UserID: user.ID, PostID: post.ID}).Error
		post.LikeNum -= 1
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": postToJson(&post, &user, isAttention == 0),
	})
}
