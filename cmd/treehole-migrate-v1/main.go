package main

import (
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"log"
	"thuhole-go-backend/pkg/config"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/logger"
	"thuhole-go-backend/pkg/utils"
	"time"
)

type UserRole int32

const (
	BannedUserRole UserRole = -100
	NormalUserRole          = 50
)

type ReportType string

type User struct {
	ID             int32  `gorm:"primaryKey;autoIncrement;not null"`
	EmailHash      string `gorm:"index;type:char(64) NOT NULL"`
	Token          string `gorm:"index;type:char(32) NOT NULL"`
	Role           UserRole
	SystemMessages []SystemMessage
	Bans           []Ban
	Posts          []Post
	Comments       []Comment
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type VerificationCode struct {
	EmailHash   string `gorm:"primaryKey;type:char(64) NOT NULL"`
	Code        string `gorm:"type:varchar(20) NOT NULL"`
	FailedTimes int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Post struct {
	ID        int32 `gorm:"primaryKey;autoIncrement;not null"`
	User      User
	UserID    int32
	Text      string `gorm:"index:,class:FULLTEXT,option:WITH PARSER ngram;type: varchar(10000) NOT NULL"`
	Tag       string `gorm:"index;type:varchar(60) NOT NULL"`
	Type      string `gorm:"type:varchar(20) NOT NULL"`
	FilePath  string `gorm:"type:varchar(60) NOT NULL"`
	LikeNum   int32
	ReplyNum  int32
	ReportNum int32
	Comments  []Comment
	CreatedAt time.Time
	UpdatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Comment struct {
	ID        int32 `gorm:"primaryKey;autoIncrement;not null"`
	Post      Post
	PostID    int32
	User      User
	UserID    int32
	Text      string `gorm:"index:,class:FULLTEXT,option:WITH PARSER ngram;type: varchar(10000) NOT NULL"`
	Tag       string `gorm:"index;type:varchar(60) NOT NULL"`
	Type      string `gorm:"type:varchar(20) NOT NULL"`
	FilePath  string `gorm:"type:varchar(60) NOT NULL"`
	Name      string `gorm:"type:varchar(60) NOT NULL"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Report struct {
	ID             int32 `gorm:"primaryKey;autoIncrement;not null"`
	User           User
	UserID         int32
	ReportedUser   User
	ReportedUserID int32
	Post           Post
	PostID         int32
	Comment        Comment
	CommentID      int32
	Reason         string     `gorm:"type: varchar(1000) NOT NULL"`
	Type           ReportType `gorm:"type:varchar(20) NOT NULL"`
	IsComment      bool
	Weight         int32
	CreatedAt      time.Time `gorm:"index"`
}

type Attention struct {
	User   User
	UserID int32 `gorm:"primaryKey;index"`
	Post   Post
	PostID int32 `gorm:"primaryKey"`
}

type SystemMessage struct {
	ID        int32 `gorm:"primaryKey;autoIncrement;not null"`
	User      User
	UserID    int32
	Text      string `gorm:"type: varchar(11000) NOT NULL"`
	Title     string `gorm:"type: varchar(100) NOT NULL"`
	Ban       Ban
	BanID     int32     `gorm:"index"`
	CreatedAt time.Time `gorm:"index"`
}

type Ban struct {
	ID        int32 `gorm:"primaryKey;autoIncrement;not null"`
	User      User
	UserID    int32
	Report    Report
	ReportID  int32
	Reason    string `gorm:"type: varchar(11000) NOT NULL"`
	ExpireAt  int64
	CreatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

var emailHashToId = make(map[string]int32)

const batchSize = 3000

func migrateAttentions(page int) (count int) {
	var results []map[string]interface{}
	err := db.GetDb(false).Table("v1_attentions").Limit(batchSize).Offset((page - 1) * batchSize).
		Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_attentions!")
	var attentions []Attention
	for _, result := range results {
		at := Attention{
			UserID: emailHashToId[result["email_hash"].(string)],
			PostID: result["pid"].(int32),
		}
		attentions = append(attentions, at)
	}
	count = len(results)
	if count > 0 {
		err = db.GetDb(false).Create(&attentions).Error
		utils.FatalErrorHandle(&err, "error writing v1_attentions!")
	}
	return
}

func migratePost(page int) (count int) {
	var results []map[string]interface{}
	err := db.GetDb(false).Table("v1_posts").Where("pid <= ? and pid > ?", page*batchSize,
		(page-1)*batchSize).Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_posts!")
	var posts []Post
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
		post := Post{
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
	count = len(results)
	if count > 0 {
		err = db.GetDb(false).Create(&posts).Error
		utils.FatalErrorHandle(&err, "error writing v1_posts!")
	}
	return
}

func migrateComment(page int) (count int) {
	var results []map[string]interface{}
	var comments []Comment
	err := db.GetDb(false).Table("v1_comments").Where("cid <= ? and cid > ?", page*batchSize,
		(page-1)*batchSize).Find(&results).Error
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
		comment := Comment{
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
	count = len(results)
	if count > 0 {
		err = db.GetDb(false).Create(&comments).Error
		utils.FatalErrorHandle(&err, "error writing v1_comments!")
	}
	return
}

func migrate(foo func(int) int) {
	count := -1
	page := 1
	for count != 0 {
		count = foo(page)
		page += 1
	}

}

func main() {
	logger.InitLog("migration.log")
	config.InitConfigFile()
	log.Println("starting migration...")

	var err error
	db.InitDb()

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

	err = db.GetDb(false).AutoMigrate(&User{}, &VerificationCode{}, &Post{}, &Comment{}, &Attention{}, &Report{}, &SystemMessage{}, Ban{})
	utils.FatalErrorHandle(&err, "error migrating database!")

	var results []map[string]interface{}

	err = db.GetDb(false).Table("v1_users").Order("timestamp asc").Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_users!")
	var users []User
	for _, result := range results {
		user := User{
			EmailHash: result["email_hash"].(string),
			Token:     result["token"].(string),
			Role:      NormalUserRole,
			CreatedAt: time.Unix(int64(result["timestamp"].(int32)), 0),
		}
		users = append(users, user)
		emailHashToId[user.EmailHash] = user.ID
	}
	for _, emailHash := range viper.GetStringSlice("banned_email_hashes") {
		user := User{
			EmailHash: emailHash,
			Token:     utils.GenToken(),
			Role:      BannedUserRole,
		}
		users = append(users, user)
	}
	err = db.GetDb(false).CreateInBatches(&users, 10000).Error
	utils.FatalErrorHandle(&err, "error writing v1_users!")
	for _, user := range users {
		emailHashToId[user.EmailHash] = user.ID
	}
	users = nil
	log.Println("done migrating users")

	results = nil
	err = db.GetDb(false).Table("v1_verification_codes").Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v1_verification_codes!")
	var vcs []VerificationCode
	for _, result := range results {
		vc := VerificationCode{
			EmailHash:   result["email_hash"].(string),
			FailedTimes: int(result["failed_times"].(int64)),
			Code:        result["code"].(string),
			CreatedAt:   time.Unix(int64(result["timestamp"].(int32)), 0),
		}
		vcs = append(vcs, vc)
	}
	err = db.GetDb(false).CreateInBatches(&vcs, 10000).Error
	utils.FatalErrorHandle(&err, "error writing v1_verification_codes!")
	vcs = nil
	log.Println("done migrating verification_codes")

	migrate(migrateAttentions)
	log.Println("done migrating attentions")

	migrate(migratePost)
	log.Println("done migrating posts")

	migrate(migrateComment)
	log.Println("done migrating comments")
	log.Println("done all migration")

	//	No need to migrate banned and reports db
}
