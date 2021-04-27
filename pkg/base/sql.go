package base

import (
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/model"
	"treehollow-v3-backend/pkg/utils"
)

var db *gorm.DB

func AutoMigrateDb() {
	err := db.AutoMigrate(&User{}, &DecryptionKeyShares{}, &Email{},
		&Device{}, &PushSettings{}, &Vote{},
		&VerificationCode{}, &Post{}, &PushMessage{},
		&Comment{}, &Attention{}, &Report{}, &SystemMessage{}, Ban{})
	utils.FatalErrorHandle(&err, "error migrating database!")
}

func InitDb() {
	err2 := initRedis()
	utils.FatalErrorHandle(&err2, "error init redis")
	initCache()

	logFile, err := os.OpenFile("sql.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	utils.FatalErrorHandle(&err, "error init sql log file")
	mw := io.MultiWriter(os.Stdout, logFile)
	logLevel := logger.Warn
	if viper.GetBool("is_debug") {
		logLevel = logger.Info
	}
	newLogger := logger.New(
		log.New(mw, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Millisecond * 500, // Slow SQL threshold
			LogLevel:      logLevel,               // Log level
			Colorful:      false,
		},
	)

	db, err = gorm.Open(mysql.Open(
		viper.GetString("sql_source")+"?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   newLogger,
	})
	utils.FatalErrorHandle(&err, "error opening sql db")
}

func GetDb(unscoped bool) *gorm.DB {
	if unscoped {
		return db.Unscoped()
	}
	return db
}

func ListPosts(tx *gorm.DB, p int, user *User) (posts []Post, err error) {
	offset := (p - 1) * consts.PageSize
	limit := consts.PageSize
	pinnedPids := viper.GetIntSlice("pin_pids")
	if CanViewDeletedPost(user) {
		tx = tx.Unscoped()
	}
	if len(pinnedPids) == 0 {
		err = tx.Order("id desc").Limit(limit).Offset(offset).Find(&posts).Error
	} else {
		err = tx.Where("id not in ?", pinnedPids).Order("id desc").Limit(limit).Offset(offset).
			Find(&posts).Error
	}
	return
}

func ListMsgs(p int, minId int32, userId int32, pushOnly bool) (msgs []PushMessage, err error) {
	offset := (p - 1) * consts.MsgPageSize
	limit := consts.MsgPageSize
	tx := db
	if pushOnly {
		tx = tx.Where("do_push = ?", true)
	}
	err = tx.Where("user_id = ? and id > ?", userId, minId).Order("id desc").Limit(limit).Offset(offset).
		Find(&msgs).Error
	return
}

func GetComments(pid int32) ([]Comment, error) {
	var comments []Comment
	err := db.Unscoped().Where("post_id = ?", pid).Order("id asc").Find(&comments).Error
	return comments, err
}

func GetMultipleComments(tx *gorm.DB, pids []int32) ([]Comment, error) {
	var comments []Comment
	err := tx.Unscoped().Where("post_id in (?)", pids).Order("id asc").Find(&comments).Error
	return comments, err
}

func SearchPosts(page int, keywords string, limitPids []int32, user User, order model.SearchOrder,
	includeComment bool, beforeTimestamp int64, afterTimestamp int64) (posts []Post, err error) {
	canViewDelete := CanViewDeletedPost(&user)
	var thePost Post
	var err2 error
	pid := -1
	if page == 1 {
		if strings.HasPrefix(keywords, "#") {
			pid, err2 = strconv.Atoi(keywords[1:])
		} else {
			pid, err2 = strconv.Atoi(keywords)
		}
		if err2 == nil {
			err2 = GetDb(canViewDelete).First(&thePost, int32(pid)).Error
		}
	}
	offset := (page - 1) * consts.SearchPageSize
	limit := consts.SearchPageSize

	tx := GetDb(canViewDelete)
	if limitPids != nil {
		tx = tx.Where("id in ?", limitPids)
	}

	subSearch := func(tx0 *gorm.DB, isTag bool) *gorm.DB {
		if isTag {
			return tx0.Where("tag = ?", keywords[1:])
		}
		replacedKeywords := "+" + strings.ReplaceAll(keywords, " ", " +")
		return tx0.Where("match(text) against(? IN BOOLEAN MODE)", replacedKeywords)
	}

	if canViewDelete && keywords == "dels" {
		subQuery1 := db.Unscoped().Model(&Report{}).Distinct().
			Where("type in (?) and user_id != reported_user_id and post_id = posts.id",
				[]ReportType{UserDelete, AdminDeleteAndBan}).Select("post_id")
		err = db.Unscoped().Where("id in (?)", subQuery1).
			Order(order.ToString()).Limit(limit).Offset(offset).Find(&posts).Error
	} else {
		var subQuery2 *gorm.DB
		if includeComment {
			subQuery := subSearch(GetDb(canViewDelete).Model(&Comment{}).Distinct(),
				strings.HasPrefix(keywords, "#")).
				Select("post_id")
			subQuery2 = subSearch(GetDb(canViewDelete), strings.HasPrefix(keywords, "#")).
				Or("id in (?)", subQuery)
		} else {
			subQuery2 = subSearch(GetDb(canViewDelete), strings.HasPrefix(keywords, "#"))
		}

		if beforeTimestamp > 0 {
			tx = tx.Where("created_at < ?", time.Unix(beforeTimestamp, 0).In(consts.TimeLoc))
		}
		if afterTimestamp > 0 {
			tx = tx.Where("created_at >= ?", time.Unix(afterTimestamp, 0).In(consts.TimeLoc))
		}
		if pid > 0 {
			tx = tx.Where("id != ?", pid)
		}

		err = tx.Where(subQuery2).Order(order.ToString()).Limit(limit).Offset(offset).Find(&posts).Error
	}

	if err2 == nil && page == 1 {
		posts = append([]Post{thePost}, posts...)
	}
	return
}

func GetVerificationCode(emailHash string) (string, int64, int, error) {
	var vc VerificationCode
	err := db.Where("email_hash = ?", emailHash).First(&vc).Error
	return vc.Code, vc.UpdatedAt.Unix(), vc.FailedTimes, err
}

func SavePost(uid int32, text string, tag string, typ string, filePath string, metaStr string, voteData string) (id int32, err error) {
	post := Post{Tag: tag, UserID: uid, Text: text, Type: typ, FilePath: filePath, LikeNum: 0, ReplyNum: 0,
		ReportNum: 0, FileMetadata: metaStr, VoteData: voteData}
	err = db.Save(&post).Error
	id = post.ID
	return
}

func GetHotPosts() (posts []Post, err error) {
	err = db.Where("id>(SELECT MAX(id)-2000 FROM posts)").
		Order("like_num*3+reply_num+UNIX_TIMESTAMP(created_at)/1800-report_num*10 DESC").
		Limit(200).Find(&posts).Error
	return
}

func SaveComment(tx *gorm.DB, uid int32, text string, tag string, typ string, filePath string, pid int32, replyTo int32, name string,
	metaStr string) (id int32, err error) {
	comment := Comment{Tag: tag, UserID: uid, PostID: pid, ReplyTo: replyTo, Text: text, Type: typ, FilePath: filePath,
		Name: name, FileMetadata: metaStr}
	err = tx.Save(&comment).Error
	id = comment.ID
	if err == nil {
		err = DelCommentCache(int(pid))
	}
	return
}

func GenCommenterName(tx *gorm.DB, dzUserID int32, czUserID int32, postID int32, names0 []string, names1 []string) (string, error) {
	var name string
	var err error
	if dzUserID == czUserID {
		name = consts.DzName
	} else {
		var comment Comment
		err = tx.Unscoped().Where("user_id = ? AND post_id=?", czUserID, postID).First(&comment).Error
		if err != nil { // token is not in comments
			var count int64
			err = tx.Unscoped().Model(&Comment{}).Where("user_id != ? AND post_id=?", dzUserID, postID).
				Distinct("user_id").Count(&count).Error
			if err != nil {
				return "", err
			}
			name = utils.GetCommenterName(int(count)+1, names0, names1)
		} else {
			name = comment.Name
		}
	}
	return name, nil
}

func GetBannedTime(tx *gorm.DB, uid int32, startTime int64) (times int64, err error) {
	err = tx.Model(&Ban{}).Where("user_id = ? and expire_at > ?", uid, startTime).Count(&times).Error
	return
}

func calcBanExpireTime(times int64) int64 {
	return utils.GetTimeStamp() + (times+1)*86400
}

func generateBanReason(report Report, originalText string) (rtn string) {
	var pre string
	if report.IsComment {
		pre = "您的树洞评论#" + strconv.Itoa(int(report.PostID)) + "-" + strconv.Itoa(int(report.CommentID))
	} else {
		pre = "您的树洞#" + strconv.Itoa(int(report.PostID))
	}
	switch report.Type {
	case UserReport:
		rtn = pre + "\n\"" + originalText + "\"\n因为用户举报过多被删除。"
	case AdminDeleteAndBan:
		rtn = pre + "\n\"" + originalText + "\"\n被管理员删除。管理员的删除理由是：【" + report.Reason + "】。"
	}
	return
}

func DeleteByReport(tx *gorm.DB, report Report) (err error) {
	if report.IsComment {
		err = tx.Where("id = ?", report.CommentID).Delete(&Comment{}).Error
		if err == nil {
			err = tx.Model(&Post{}).Where("id = ?", report.PostID).Update("reply_num",
				gorm.Expr("reply_num - 1")).Error
			if err == nil {
				err = DelCommentCache(int(report.PostID))
				go func() {
					SendDeletionToPushService(report.CommentID)
				}()
			}
		}
	} else {
		err = tx.Where("id = ?", report.PostID).Delete(&Post{}).Error
	}
	return
}

func DeleteAndBan(tx *gorm.DB, report Report, text string) (err error) {
	err = DeleteByReport(tx, report)
	if err == nil {
		times, err2 := GetBannedTime(tx, report.ReportedUserID, 0)
		if err2 == nil {
			tx.Create(&Ban{
				UserID:   report.ReportedUserID,
				ReportID: report.ID,
				Reason:   generateBanReason(report, text),
				ExpireAt: calcBanExpireTime(times),
			})
		}
	}
	return
}

func SetTagByReport(tx *gorm.DB, report Report) (err error) {
	if report.IsComment {
		err = tx.Model(&Comment{}).Where("id = ?", report.CommentID).
			Update("tag", report.Reason).Error
		if err == nil {
			err = tx.Model(&Post{}).Where("id = ?", report.PostID).
				Update("updated_at", time.Now()).Error
			if err == nil {
				err = DelCommentCache(int(report.PostID))
			}
		}
	} else {
		err = tx.Model(&Post{}).Where("id = ?", report.PostID).
			Update("tag", report.Reason).Error
	}
	return
}

func UnbanByReport(tx *gorm.DB, report Report) (err error) {
	var ban Ban
	subQuery := tx.Model(&Report{}).Distinct().
		Where("post_id = ? and comment_id = ? and is_comment = ? and type in (?)",
			report.PostID, report.CommentID, report.IsComment,
			[]ReportType{UserReport, AdminDeleteAndBan}).
		Select("id")
	err = tx.Model(&Ban{}).Where("report_id in (?)", subQuery).First(&ban).Error
	if err == nil {
		err = tx.Delete(&ban).Error
	}
	return
}
