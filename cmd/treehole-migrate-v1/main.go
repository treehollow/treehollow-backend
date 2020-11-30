package main

import (
	"gorm.io/gorm"
	"thuhole-go-backend/pkg/config"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/logger"
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
	"time"
)

func main() {
	logger.InitLog(consts.ServicesApiLogFile)
	config.InitConfigFile()

	var err error
	db.InitDb()
	//TODO: uncomment these when migrating
	//TODO: disable hooks
	//err = db.GetDb(false).Migrator().RenameTable("user_info", "v1_users")
	//utils.FatalErrorHandle(&err, "error rename table")
	//err = db.GetDb(false).Migrator().RenameTable("verification_codes", "v1_verification_codes")
	//utils.FatalErrorHandle(&err, "error rename table")
	//err = db.GetDb(false).Migrator().RenameTable("posts", "v1_posts")
	//utils.FatalErrorHandle(&err, "error rename table")
	//err = db.GetDb(false).Migrator().RenameTable("comments", "v1_comments")
	//utils.FatalErrorHandle(&err, "error rename table")
	//err = db.GetDb(false).Migrator().RenameTable("attentions", "v1_attentions")
	//utils.FatalErrorHandle(&err, "error rename table")
	//err = db.GetDb(false).Migrator().RenameTable("reports", "v1_reports")
	//utils.FatalErrorHandle(&err, "error rename table")
	//err = db.GetDb(false).Migrator().RenameTable("banned", "v1_banned")
	//utils.FatalErrorHandle(&err, "error rename table")
	//
	//err = db.GetDb(false).AutoMigrate(&structs.User{}, &structs.VerificationCode{}, &structs.Post{}, &structs.Comment{}, &structs.Attention{}, &structs.Report{}, &structs.SystemMessage{}, structs.Ban{})
	//utils.FatalErrorHandle(&err, "error migrating database!")

	emailHashToId := make(map[string]int32)
	var results []map[string]interface{}

	err = db.GetDb(false).Table("v1_users").Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_users!")
	var users []structs.User
	for _, result := range results {
		user := structs.User{
			EmailHash: result["email_hash"].(string),
			Token:     result["token"].(string),
			CreatedAt: time.Unix(int64(result["timestamp"].(int32)), 0),
		}
		users = append(users, user)
		emailHashToId[user.EmailHash] = user.ID
	}
	err = db.GetDb(false).CreateInBatches(&users, 1000).Error
	utils.FatalErrorHandle(&err, "error writing v1_users!")
	users = nil

	results = nil
	err = db.GetDb(false).Table("v1_verification_codes").Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_verification_codes!")
	var vcs []structs.VerificationCode
	for _, result := range results {
		vc := structs.VerificationCode{
			EmailHash:   result["email_hash"].(string),
			FailedTimes: int(result["failed_times"].(int64)),
			Code:        result["code"].(string),
			CreatedAt:   time.Unix(int64(result["timestamp"].(int32)), 0),
		}
		vcs = append(vcs, vc)
	}
	err = db.GetDb(false).CreateInBatches(&vcs, 1000).Error
	utils.FatalErrorHandle(&err, "error writing v1_verification_codes!")
	vcs = nil

	results = nil
	err = db.GetDb(false).Table("v1_attentions").Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_attentions!")
	var attentions []structs.Attention
	for _, result := range results {
		at := structs.Attention{
			UserID: emailHashToId[result["email_hash"].(string)],
			PostID: result["pid"].(int32),
		}
		attentions = append(attentions, at)
	}
	err = db.GetDb(false).CreateInBatches(&attentions, 1000).Error
	utils.FatalErrorHandle(&err, "error writing v1_attentions!")
	attentions = nil

	results = nil
	err = db.GetDb(false).Table("v1_posts").Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_posts!")
	var posts []structs.Post
	for _, result := range results {
		var deletedAt gorm.DeletedAt
		if result["reportnum"].(int64) >= 10 {
			deletedAt = gorm.DeletedAt{
				Time:  time.Now(),
				Valid: true,
			}
		} else {
			deletedAt = gorm.DeletedAt{
				Valid: false,
			}
		}
		post := structs.Post{
			ID:        result["pid"].(int32),
			UserID:    emailHashToId[result["email_hash"].(string)],
			Text:      result["text"].(string),
			Tag:       result["tag"].(string),
			Type:      result["type"].(string),
			FilePath:  result["file_path"].(string),
			LikeNum:   result["likenum"].(int32),
			ReplyNum:  result["replynum"].(int32),
			ReportNum: int32(result["reportnum"].(int64)),
			CreatedAt: time.Unix(int64(result["timestamp"].(int32)), 0),
			DeletedAt: deletedAt,
		}
		posts = append(posts, post)
	}
	err = db.GetDb(false).CreateInBatches(&posts, 1000).Error
	utils.FatalErrorHandle(&err, "error writing v1_posts!")
	posts = nil

	results = nil
	var comments []structs.Comment
	err = db.GetDb(false).Table("v1_comments").Where("cid < 300000").Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_comments!")
	for _, result := range results {
		var deletedAt gorm.DeletedAt
		if result["reportnum"].(int64) >= 10 {
			deletedAt = gorm.DeletedAt{
				Time:  time.Now(),
				Valid: true,
			}
		} else {
			deletedAt = gorm.DeletedAt{
				Valid: false,
			}
		}
		comment := structs.Comment{
			ID:        result["cid"].(int32),
			PostID:    result["pid"].(int32),
			UserID:    emailHashToId[result["email_hash"].(string)],
			Text:      result["text"].(string),
			Tag:       result["tag"].(string),
			Type:      result["type"].(string),
			FilePath:  result["file_path"].(string),
			CreatedAt: time.Unix(int64(result["timestamp"].(int32)), 0),
			DeletedAt: deletedAt,
		}
		comments = append(comments, comment)
	}
	err = db.GetDb(false).CreateInBatches(&comments, 1000).Error
	utils.FatalErrorHandle(&err, "error writing v1_posts!")
	results = nil
	comments = nil
	err = db.GetDb(false).Table("v1_comments").Where("cid >= 300000").Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_comments!")
	for _, result := range results {
		var deletedAt gorm.DeletedAt
		if result["reportnum"].(int32) >= 10 {
			deletedAt = gorm.DeletedAt{
				Time:  time.Now(),
				Valid: true,
			}
		} else {
			deletedAt = gorm.DeletedAt{
				Valid: false,
			}
		}
		comment := structs.Comment{
			ID:        result["cid"].(int32),
			PostID:    result["pid"].(int32),
			UserID:    emailHashToId[result["email_hash"].(string)],
			Text:      result["text"].(string),
			Tag:       result["tag"].(string),
			Type:      result["type"].(string),
			FilePath:  result["file_path"].(string),
			Name:      result["name"].(string),
			CreatedAt: time.Unix(int64(result["timestamp"].(int32)), 0),
			DeletedAt: deletedAt,
		}
		comments = append(comments, comment)
	}
	err = db.GetDb(false).CreateInBatches(&comments, 1000).Error
	utils.FatalErrorHandle(&err, "error writing v1_posts!")

	//TODO: bans, reports, system messages

	//log.Println("start time: ", time.Now().Format("01-02 15:04:05"))
	//if false == viper.GetBool("is_debug") {
	//	gin.SetMode(gin.ReleaseMode)
	//}
	//
	//route.HotPosts, _ = db.GetHotPosts()
	//c := cron.New()
	//_, _ = c.AddFunc("*/1 * * * *", func() {
	//	route.HotPosts, _ = db.GetHotPosts()
	//	//log.Println("refreshed hotPosts ,err=", err)
	//})
	//c.Start()
	//
	//route.ServicesApiListenHttp()
}
