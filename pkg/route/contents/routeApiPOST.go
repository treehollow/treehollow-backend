package contents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/bot"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/model"
	"treehollow-v3-backend/pkg/s3"
	"treehollow-v3-backend/pkg/utils"
)

//TODO: (low priority)config, webhook

func generateTag(text string) string {
	//re := regexp.MustCompile(`[#＃](性相关|政治相关|引战|未经证实的传闻|令人不适|刷屏|NSFW|nsfw)`)
	//if re.MatchString(text) {
	//	return strings.ToUpper(re.FindStringSubmatch(text)[1])
	//}
	re1, err := regexp.Compile(viper.GetString("fold_regex"))
	if err == nil && re1.MatchString(text) {
		return "刷屏"
	}
	re2, err2 := regexp.Compile(viper.GetString("sex_related_regex"))
	if err2 == nil && re2.MatchString(text) {
		return "性相关"
	}
	return ""
}

func containRiskWords(text string) (string, bool) {
	riskWords := viper.GetStringSlice("risk_words")
	for _, word := range riskWords {
		if strings.Contains(text, word) {
			return word, true
		}
	}
	return "", false
}

func sendPost(c *gin.Context) {
	text := c.PostForm("text")
	typ := c.PostForm("type")
	img := c.PostForm("data")
	user := c.MustGet("user").(base.User)

	strVoteData := c.MustGet("vote_data").(string)

	tag := c.PostForm("tag")
	if _, b := utils.ContainsString(viper.GetStringSlice("sendable_tags"), tag); !b {
		tag = generateTag(text)
	}

	var pid int32
	var err error
	var imgPath string
	var suffix2 string
	var uploadChan chan bool
	uploadChan = nil
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if typ == "image" {
		imgPath = utils.GenToken()
		sDec, suffix, metaStr, err2 := utils.SaveImage(img, imgPath)
		suffix2 = suffix
		if err2 != nil {
			base.HttpReturnWithCodeMinusOne(c, err2)
			return
		}

		pid, err = base.SavePost(user.ID, text, tag, typ, imgPath+suffix, metaStr, strVoteData)
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
		pid, err = base.SavePost(user.ID, text, tag, typ, "", "{}", strVoteData)
	}

	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SendPostSaveFailed", consts.DatabaseWriteFailedString))
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
			"code":    0,
			"post_id": pid,
		})

		if word, b := containRiskWords(text); viper.GetBool("enable_telegram") && b {
			fullImgPath := ""
			if typ == "image" {
				fullImgPath = filepath.Join(viper.GetString("images_path"), imgPath[:2], imgPath+suffix2)
			}
			bot.TgMessageChannel <- bot.TgMessage{
				Text: fmt.Sprintf("New post contains risk word:'%s'\n#%d\n %s", word, pid, text), ImagePath: fullImgPath,
			}
		}

		return
	}
}

func sendComment(c *gin.Context) {
	text := c.PostForm("text")
	typ := c.PostForm("type")
	img := c.PostForm("data")
	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SendCommentInvalidPid", "发送失败，pid不合法"))
		return
	}
	replyToCommentID, err2 := strconv.Atoi(c.PostForm("reply_to_cid"))
	if err2 != nil {
		replyToCommentID = -1
	}

	user := c.MustGet("user").(base.User)
	canViewDelete := base.CanViewDeletedPost(&user)
	var imgPath string
	var uploadChan chan bool
	var post base.Post
	var suffix2 string
	uploadChan = nil

	var commentID int32
	err7 := base.GetDb(false).Transaction(func(tx *gorm.DB) error {
		err = utils.UnscopedTx(tx, canViewDelete).Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, int32(pid)).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				base.HttpReturnWithErr(c, -101, logger.NewSimpleError("SendCommentNoPid", "找不到这条树洞", logger.WARN))
			} else {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "CommentGetPostFailed", consts.DatabaseReadFailedString))
			}
			return err
		}

		var replyToComment base.Comment
		if replyToCommentID > 0 {
			err = utils.UnscopedTx(tx, canViewDelete).Model(&base.Comment{}).
				Where("id = ? and post_id = ?", replyToCommentID, pid).First(&replyToComment).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("SendCommentNoReplyToPid", "找不到你要评论的树洞", logger.WARN))
				} else {
					base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "CommentGetReplyToPostFailed", consts.DatabaseReadFailedString))
				}
				return err
			}
		}

		var name string
		names0, names1 := consts.Names0, consts.Names1

		name, err = base.GenCommenterName(tx, post.UserID, user.ID, post.ID, names0, names1)
		if err != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "GenCommenterNameFailed", consts.DatabaseReadFailedString))
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if typ == "image" {
			imgPath = utils.GenToken()
			sDec, suffix, metaStr, err3 := utils.SaveImage(img, imgPath)
			suffix2 = suffix
			if err3 != nil {
				base.HttpReturnWithCodeMinusOne(c, err3)
				return err3.Err
			}

			commentID, err = base.SaveComment(tx, user.ID, text, "", typ, imgPath+suffix, int32(pid), int32(replyToCommentID), name, metaStr)
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
			commentID, err = base.SaveComment(tx, user.ID, text, "", "text", "", int32(pid), int32(replyToCommentID), name, "{}")
		}
		if err != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SaveCommentFailed", consts.DatabaseWriteFailedString))
			return err
		}

		// Push Notification
		if !post.DeletedAt.Valid {
			var attentions []base.Attention
			err = tx.Model(&base.Attention{}).Where("post_id = ?", int32(pid)).Find(&attentions).Error
			if err != nil {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "GetAttentionsByPidFailed", consts.DatabaseReadFailedString))
				return err
			}

			pushMessages := make([]base.PushMessage, 0, len(attentions)+1)
			replyToUserID := post.UserID
			if replyToCommentID > 0 {
				replyToUserID = replyToComment.UserID
			}
			if replyToUserID != user.ID {
				pushMessages = append(pushMessages, base.PushMessage{
					Message:   utils.TrimText(text, 100),
					Title:     name + "回复了树洞#" + strconv.Itoa(pid),
					PostID:    int32(pid),
					CommentID: commentID,
					Type:      model.ReplyMeComment,
					UserID:    replyToUserID,
					UpdatedAt: time.Now(),
				})
			}
			for _, attention := range attentions {
				if replyToUserID != user.ID && attention.UserID == replyToUserID {
					pushMessages[0].Type |= model.CommentInFavorited
				} else if attention.UserID != user.ID {
					pushMessages = append(pushMessages, base.PushMessage{
						Message:   utils.TrimText(text, 100),
						Title:     name + "回复了树洞#" + strconv.Itoa(pid),
						PostID:    int32(pid),
						CommentID: commentID,
						Type:      model.CommentInFavorited,
						UserID:    attention.UserID,
						UpdatedAt: time.Now(),
					})
				}
			}
			err = base.PreProcessPushMessages(tx, pushMessages)
			if err != nil {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SaveCommentFailed", consts.DatabaseWriteFailedString))
				return err
			}
			go func() {
				base.SendToPushService(pushMessages)
			}()
		}

		return nil
	})

	if err7 == nil {
		if user.ID == post.UserID {
			//TODO: (low priority) save this in config
			re := regexp.MustCompile(`[#＃](性相关|政治相关|引战|未经证实的传闻|令人不适|刷屏|NSFW|nsfw|重复内容)`)
			if re.MatchString(text) {
				tag := strings.ToUpper(re.FindStringSubmatch(text)[1])
				_ = base.GetDb(canViewDelete).Model(&base.Post{}).Where("id = ?", post.ID).Update("tag", tag)
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
			"code":       0,
			"comment_id": commentID,
		})

		if word, b := containRiskWords(text); viper.GetBool("enable_telegram") && b {
			fullImgPath := ""
			if typ == "image" {
				fullImgPath = filepath.Join(viper.GetString("images_path"), imgPath[:2], imgPath+suffix2)
			}
			bot.TgMessageChannel <- bot.TgMessage{
				Text: fmt.Sprintf("New comment contains risk word:'%s'\n#%d-%d\n %s", word, pid, commentID, text), ImagePath: fullImgPath,
			}
		}
	}
}

func getReportType(typ string) base.ReportType {
	switch typ {
	case "report":
		return base.UserReport
	case "fold":
		return base.UserReportFold
	case "set_tag":
		return base.AdminTag
	case "delete":
		return base.UserDelete
	case "undelete_unban":
		return base.AdminUndelete
	case "delete_ban":
		return base.AdminDeleteAndBan
	case "unban":
		return base.AdminUnban
	default:
		return base.UserReport
	}
}

func getPostOrCommentText(post *base.Post, comment *base.Comment, isComment bool) string {
	if isComment {
		return comment.Text
	}
	return post.Text
}

//TODO: (low priority) test this function
func handleReport(isComment bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = base.GetDb(false).Transaction(func(tx *gorm.DB) error {
			user := c.MustGet("user").(base.User)
			canViewDelete := base.CanViewDeletedPost(&user)
			reason := c.PostForm("reason")
			typ := c.PostForm("type")
			id := c.MustGet("id").(int)
			var post base.Post
			var comment base.Comment

			var err3 error
			if isComment {
				err3 = utils.UnscopedTx(tx, canViewDelete).Clauses(clause.Locking{Strength: "UPDATE"}).
					First(&comment, int32(id)).Error
			} else {
				err3 = utils.UnscopedTx(tx, canViewDelete).Clauses(clause.Locking{Strength: "UPDATE"}).
					First(&post, int32(id)).Error
			}
			if err3 != nil {
				if errors.Is(err3, gorm.ErrRecordNotFound) {
					base.HttpReturnWithErrAndAbort(c, -101, logger.NewSimpleError("ReportNoId", "找不到这条树洞", logger.WARN))
				} else {
					base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "GetReportPostOrCommentFailed", consts.DatabaseReadFailedString))
				}
				return err3
			}

			var userPermissions []string
			if isComment {
				userPermissions = base.GetPermissionsByComment(&user, &comment)
			} else {
				userPermissions = base.GetPermissionsByPost(&user, &post)
			}
			if _, ok := utils.ContainsString(userPermissions, typ); !ok {
				base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("PermissionDenied",
					"操作失败，权限不足", logger.WARN))
				return errors.New("操作失败，权限不足")
			}

			if typ == "fold" {
				if _, ok := utils.ContainsString(viper.GetStringSlice("reportable_tags"), reason); !ok {
					base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("TagNotExist",
						"操作失败，不存在这个tag", logger.WARN))
					return errors.New("操作失败，不存在这个tag")
				}
			}

			reportType := getReportType(typ)
			if typ == "report" || typ == "fold" {
				var reported int64
				if isComment {
					utils.UnscopedTx(tx, canViewDelete).Model(&base.Report{}).
						Where("comment_id = ? and user_id = ? and is_comment = ? and type = ?",
							comment.ID, user.ID, isComment, reportType).Count(&reported)
				} else {
					utils.UnscopedTx(tx, canViewDelete).Model(&base.Report{}).
						Where("post_id = ? and user_id = ? and is_comment = ? and type = ?",
							post.ID, user.ID, isComment, reportType).Count(&reported)
				}
				if reported == 1 {
					base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("AlreadyReported",
						"已经举报过了，举报失败。", logger.WARN))
					return errors.New("已经举报过了，举报失败。")
				}
			}

			var report base.Report
			if isComment {
				report = base.Report{
					UserID:         user.ID,
					ReportedUserID: comment.UserID,
					PostID:         comment.PostID,
					CommentID:      comment.ID,
					Reason:         reason,
					Type:           reportType,
					IsComment:      true,
					Weight:         base.GetReportWeight(&user),
				}

				if viper.GetBool("enable_telegram") {
					fullImgPath := ""
					if comment.Type == "image" {
						fullImgPath = filepath.Join(viper.GetString("images_path"), comment.FilePath[:2], comment.FilePath)
					}
					bot.TgMessageChannel <- bot.TgMessage{
						Text: fmt.Sprintf("New user report for comment #%d-%d\nReason: %s\n\nOriginal text:\n%s", comment.PostID, comment.ID, reason, comment.Text), ImagePath: fullImgPath,
					}
				}
			} else {
				report = base.Report{
					UserID:         user.ID,
					ReportedUserID: post.UserID,
					PostID:         post.ID,
					CommentID:      0,
					Reason:         reason,
					Type:           reportType,
					IsComment:      false,
					Weight:         base.GetReportWeight(&user),
				}

				if viper.GetBool("enable_telegram") {
					fullImgPath := ""
					if post.Type == "image" {
						fullImgPath = filepath.Join(viper.GetString("images_path"), post.FilePath[:2], post.FilePath)
					}
					bot.TgMessageChannel <- bot.TgMessage{
						Text: fmt.Sprintf("New user report for post #%d\nReason: %s\n\nOriginal text:\n%s", post.ID, reason, post.Text), ImagePath: fullImgPath,
					}
				}
			}

			err := tx.Create(&report).Error

			if err == nil {
				switch report.Type {
				case base.UserReport:
					var reportScore int64
					err = tx.Model(&base.Report{}).Select("SUM(weight)").Where(&base.Report{
						PostID:    report.PostID,
						CommentID: report.CommentID,
						IsComment: isComment,
						Type:      base.UserReport,
					}).First(&reportScore).Error
					if reportScore >= 100 && err == nil {
						err = base.DeleteAndBan(tx, report, utils.TrimText(getPostOrCommentText(&post, &comment, isComment), 20))
					}
				case base.UserReportFold:
					if report.ReportedUserID == report.UserID && !isComment {
						err = base.SetTagByReport(tx, report)
					} else {
						var reportScore int64
						err = tx.Model(&base.Report{}).Where(&base.Report{
							PostID:    report.PostID,
							CommentID: report.CommentID,
							IsComment: isComment,
							Reason:    report.Reason,
							Type:      base.UserReportFold,
						}).Count(&reportScore).Error
						if reportScore == 2 && err == nil {
							err = base.SetTagByReport(tx, report)
						}
					}
				case base.UserDelete:
					err = base.DeleteByReport(tx, report)
				case base.AdminTag:
					err = base.SetTagByReport(tx, report)
				case base.AdminDeleteAndBan:
					uidStr := strconv.Itoa(int(user.ID))
					ctx, _ := deleteBanLimiter.Peek(c, uidStr)
					limit := base.GetDeletePostRateLimitIn24h(user.Role)
					if ctx.Limit-ctx.Remaining >= limit {
						base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("DeletePostLimitReached",
							"您的24h内的删帖数量已经达到系统限制", logger.WARN))
						return errors.New("您的24h内的删帖数量已经达到系统限制")
					}
					err = base.DeleteAndBan(tx, report, utils.TrimText(getPostOrCommentText(&post, &comment, isComment), 20))
					if err == nil {
						_, _ = deleteBanLimiter.Get(c, uidStr)
					}
				case base.AdminUndelete:
					_ = base.UnbanByReport(tx, report)
					if isComment {
						err = tx.Model(&base.Comment{}).
							Where("id = ?", report.CommentID).
							Updates(map[string]interface{}{"deleted_at": nil}).Error
						if err == nil {
							err = base.DelCommentCache(int(report.PostID))
							if err == nil {
								err = tx.Model(&base.Post{}).Where("id = ?", report.PostID).
									Update("reply_num", gorm.Expr("reply_num + 1")).Error
							}
						}
					} else {
						err = tx.Model(&base.Post{}).
							Where("id = ?", report.PostID).
							Updates(map[string]interface{}{"deleted_at": nil, "report_num": 0}).Error
					}
				case base.AdminUnban:
					err = base.UnbanByReport(tx, report)
					if err != nil {
						base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("NoBanFound",
							"没有找到相关的封禁，可能已经被解封了", logger.WARN))
						return errors.New("没有找到相关的封禁，可能已经被解封了")
					}
				}
			}

			if err != nil {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "ReportError", consts.DatabaseWriteFailedString))
			} else {
				c.JSON(http.StatusOK, gin.H{
					"code": 0,
				})
			}

			return nil
		})
	}
}

func editAttention(c *gin.Context) {
	user := c.MustGet("user").(base.User)
	canViewDelete := base.CanViewDeletedPost(&user)

	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "EditAttentionInvalidPid", "关注操作失败，pid不合法"))
		return
	}

	_ = base.GetDb(false).Transaction(func(tx *gorm.DB) error {
		var post base.Post
		err3 := utils.UnscopedTx(tx, canViewDelete).Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, int32(pid)).Error

		if err3 != nil {
			if errors.Is(err3, gorm.ErrRecordNotFound) {
				base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("FailedAttentionNoPid", "关注失败，pid不存在", logger.WARN))
			} else {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "GetPostFailedEditAttention", consts.DatabaseReadFailedString))
			}
			return err3
		}

		s := c.PostForm("switch")

		var isAttention int64
		err2 := tx.Model(&base.Attention{}).Where(&base.Attention{PostID: post.ID, UserID: user.ID}).Count(&isAttention).Error
		if err2 != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err2, "GetIsAttentionFailed", consts.DatabaseReadFailedString))
			return err2
		}
		if isAttention == 0 && s == "0" {
			base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("AlreadyNotAttention", "您已经取消关注了", logger.WARN))
			return nil
		}
		if isAttention == 1 && s == "1" {
			base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("AlreadyAttention", "您已经关注过了", logger.WARN))
			return nil
		}

		if isAttention == 0 {
			_ = tx.Create(&base.Attention{UserID: user.ID, PostID: post.ID}).Error
			post.LikeNum += 1
		}
		if isAttention == 1 {
			_ = tx.Delete(&base.Attention{UserID: user.ID, PostID: post.ID}).Error
			post.LikeNum -= 1
		}

		votes, err4 := getVotesInPosts(tx, &user, []base.Post{post})
		if err4 != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "GetVotesInPostsFailed", consts.DatabaseReadFailedString))
			return err4
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": postToJson(&post, &user, isAttention == 0, votes[post.ID]),
		})

		return nil
	})
}
