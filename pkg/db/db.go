package db

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
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/permissions"
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
	"time"
)

var db *gorm.DB

func InitDb() {
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
			SlowThreshold: time.Millisecond * 200, // Slow SQL threshold
			LogLevel:      logLevel,               // Log level
			Colorful:      false,
		},
	)

	db, err = gorm.Open(mysql.Open(viper.GetString("sql_source")+"?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"), &gorm.Config{
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

func ListPosts(p int, user structs.User) (posts []structs.Post, err error) {
	offset := (p - 1) * consts.PageSize
	limit := consts.PageSize
	pinnedPids := viper.GetIntSlice("pin_pids")
	tx := GetDb(permissions.CanViewDeletedPost(user))
	if len(pinnedPids) == 0 {
		err = tx.Order("id desc").Limit(limit).Offset(offset).Find(&posts).Error
	} else {
		err = tx.Where("id not in ?", pinnedPids).Order("id desc").Limit(limit).Offset(offset).Find(&posts).Error
	}
	return
}

func SearchPosts(p int, pageSize int, keywords string, limitPids []int32, user structs.User) (posts []structs.Post, err error) {
	canViewDelete := permissions.CanViewDeletedPost(user)
	var thePost structs.Post
	var err2 error
	pid := -1
	if p == 1 {
		pid, err2 = strconv.Atoi(keywords)
		if err2 == nil {
			err2 = GetDb(canViewDelete).First(&thePost, int32(pid)).Error
		}
	}
	offset := (p - 1) * pageSize
	limit := pageSize

	tx := GetDb(canViewDelete)
	if limitPids != nil {
		tx = tx.Where("id in ?", limitPids)
	}
	if strings.HasPrefix(keywords, "#") { //search tags
		subQuery := GetDb(canViewDelete).Model(&structs.Comment{}).Distinct().
			Where("tag = ?", keywords[1:]).
			Select("post_id")
		subQuery2 := GetDb(canViewDelete).
			Where("tag = ?", keywords[1:]).
			Or("id in (?)", subQuery)
		err = tx.Where(subQuery2).Where("id != ?", pid).
			Order("id desc").Limit(limit).Offset(offset).Find(&posts).Error
	} else if canViewDelete && keywords == "deleted" {
		subQuery := db.Unscoped().Model(&structs.Comment{}).Distinct().Where("deleted_at is not null").
			Select("post_id")
		subQuery2 := db.Unscoped().Where("deleted_at is not null").Or("id in (?)", subQuery)
		err = db.Unscoped().Where(subQuery2).Where("id != ?", pid).
			Order("id desc").Limit(limit).Offset(offset).Find(&posts).Error
	} else {
		replacedKeywords := "+" + strings.ReplaceAll(keywords, " ", " +")
		subQuery := GetDb(canViewDelete).Model(&structs.Comment{}).Distinct().
			Where("match(text) against(? IN BOOLEAN MODE)", replacedKeywords).
			Select("post_id")
		subQuery2 := GetDb(canViewDelete).
			Where("match(text) against(? IN BOOLEAN MODE)", replacedKeywords).
			Or("id in (?)", subQuery)
		err = tx.Where(subQuery2).Where("id != ?", pid).
			Order("id desc").Limit(limit).Offset(offset).Find(&posts).Error
	}

	if err2 == nil && p == 1 {
		posts = append([]structs.Post{thePost}, posts...)
	}
	return
}
func GetVerificationCode(emailHash string) (string, int64, int, error) {
	var vc structs.VerificationCode
	err := db.Where("email_hash = ?", emailHash).First(&vc).Error
	return vc.Code, vc.CreatedAt.Unix(), vc.FailedTimes, err
}

func SavePost(uid int32, text string, tag string, typ string, filePath string) (id int32, err error) {
	post := structs.Post{Tag: tag, UserID: uid, Text: text, Type: typ, FilePath: filePath, LikeNum: 0, ReplyNum: 0,
		ReportNum: 0}
	err = db.Save(&post).Error
	id = post.ID
	return
}

func GetHotPosts() (posts []structs.Post, err error) {
	err = db.Where("id>(SELECT MAX(id)-2000 FROM posts)").
		Order("like_num*3+reply_num+UNIX_TIMESTAMP(created_at)/1800-report_num*10 DESC").Limit(200).Find(&posts).Error
	return
}

func SaveComment(uid int32, text string, tag string, typ string, filePath string, pid int32, name string) (id int32, err error) {
	comment := structs.Comment{Tag: tag, UserID: uid, PostID: pid, Text: text, Type: typ, FilePath: filePath, Name: name}
	err = db.Save(&comment).Error
	id = comment.ID
	return
}

func GenCommenterName(dzUserID int32, czUserID int32, postID int32, names0 []string, names1 []string) (string, error) {
	var name string
	var err error
	if dzUserID == czUserID {
		name = consts.DzName
	} else {
		var comment structs.Comment
		err = db.Unscoped().Where("user_id = ? AND post_id=?", czUserID, postID).First(&comment).Error
		if err != nil { // token is not in comments
			var count int64
			err = db.Unscoped().Model(&structs.Comment{}).Where("user_id != ? AND post_id=?", dzUserID, postID).
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

func getBannedTime(uid int32) (times int64, err error) {
	err = db.Model(&structs.Ban{}).Where("user_id = ? and expire_at > ?", uid, utils.GetTimeStamp()).Count(&times).Error
	return
}

func calcBanExpireTime(times int64) int64 {
	return utils.GetTimeStamp() + (times+1)*86400
}

func generateBanReason(report structs.Report, originalText string) (rtn string) {
	var pre string
	if report.IsComment {
		pre = "您的树洞评论#" + strconv.Itoa(int(report.CommentID))
	} else {
		pre = "您的树洞#" + strconv.Itoa(int(report.PostID))
	}
	switch report.Type {
	case structs.UserReport:
		rtn = pre + "\n\"" + originalText + "\"\n因为用户举报过多被删除。"
	case structs.AdminDeleteAndBan:
		rtn = pre + "\n\"" + originalText + "\"\n被管理员删除。管理员的删除理由是：【" + report.Reason + "】。"
	}
	return
}

func DeleteByReport(report structs.Report) (err error) {
	if report.IsComment {
		err = db.Where("id = ?", report.CommentID).Delete(&structs.Comment{}).Error
	} else {
		err = db.Where("id = ?", report.PostID).Delete(&structs.Post{}).Error
	}
	return
}

func DeleteAndBan(report structs.Report, text string) (err error) {
	err = DeleteByReport(report)
	if err == nil {
		times, err := getBannedTime(report.ReportedUserID)
		if err == nil {
			db.Create(&structs.Ban{
				UserID:   report.ReportedUserID,
				ReportID: report.ID,
				Reason:   generateBanReason(report, text),
				ExpireAt: calcBanExpireTime(times),
			})
		}
	}
	return
}

func SetTagByReport(report structs.Report) (err error) {
	if report.IsComment {
		err = db.Model(&structs.Comment{}).Where("id = ?", report.CommentID).Update("tag", report.Reason).Error
	} else {
		err = db.Model(&structs.Post{}).Where("id = ?", report.PostID).Update("tag", report.Reason).Error
	}
	return
}

func UnbanByReport(report structs.Report) (err error) {
	var ban structs.Ban
	subQuery := db.Model(&structs.Report{}).Distinct().
		Where("post_id = ? and comment_id = ? and is_comment = ? and type in (?)",
			report.PostID, report.CommentID, report.IsComment,
			[]structs.ReportType{structs.UserReport, structs.AdminDeleteAndBan}).
		Select("id")
	err = db.Model(&structs.Ban{}).Where("report_id in (?)", subQuery).First(&ban).Error
	if err == nil {
		err = db.Delete(&ban).Error
	}
	return
}
