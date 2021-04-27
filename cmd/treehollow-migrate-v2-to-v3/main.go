package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/config"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/model"
	"treehollow-v3-backend/pkg/utils"
)

type UserRole int32

type ReportType string

// codebeat:disable[TOO_MANY_IVARS]
type User struct {
	ID             int32  `gorm:"primaryKey;autoIncrement;not null"`
	OldEmailHash   string `gorm:"index;type:varchar(64) NOT NULL"`
	OldToken       string `gorm:"index;type:varchar(32) NOT NULL"`
	EmailEncrypted string `gorm:"index;type:varchar(200) NOT NULL"`
	//KeyEncrypted   string `gorm:"type:varchar(200) NOT NULL"`
	ForgetPwNonce string `gorm:"type:varchar(36) NOT NULL"`
	Role          UserRole
	//SystemMessages []SystemMessage
	//Bans           []Ban
	//Posts          []Post
	//Comments       []Comment
	//Devices        []Device
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DecryptionKeyShares struct {
	EmailEncrypted string `gorm:"index;type:varchar(200) NOT NULL"`
	PGPMessage     string `gorm:"type:varchar(5000) NOT NULL"`
	PGPEmail       string `gorm:"index;type:varchar(100) NOT NULL"`
	CreatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type Email struct {
	EmailHash string `gorm:"primaryKey;type:char(64) NOT NULL"`
}

type DeviceType int32

type Device struct {
	ID             string `gorm:"type:char(36);primary_key"`
	UserID         int32  `gorm:"index;not null"`
	DeviceInfo     string `gorm:"type:varchar(100) NOT NULL"`
	Type           DeviceType
	IOSDeviceToken string         `gorm:"type:varchar(100)"`
	Token          string         `gorm:"index;type:char(32) NOT NULL"`
	LoginIP        string         `gorm:"type:varchar(50) NOT NULL"`
	LoginCity      string         `gorm:"type:varchar(50) NOT NULL"`
	CreatedAt      time.Time      `gorm:"index"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type PushSettings struct {
	UserID   int32 `gorm:"primaryKey;not null"`
	Settings model.PushType
}

type VerificationCode struct {
	EmailHash   string `gorm:"primaryKey;type:char(64) NOT NULL"`
	Code        string `gorm:"type:varchar(20) NOT NULL"`
	FailedTimes int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Post struct {
	ID int32 `gorm:"primaryKey;autoIncrement;not null"`
	//User         User
	UserID       int32
	Text         string `gorm:"index:,class:FULLTEXT,option:WITH PARSER ngram;type: varchar(10000) NOT NULL"`
	Tag          string `gorm:"index;type:varchar(60) NOT NULL"`
	Type         string `gorm:"type:varchar(20) NOT NULL"`
	FilePath     string `gorm:"type:varchar(60) NOT NULL"`
	FileMetadata string `gorm:"type:varchar(40) NOT NULL"`
	VoteData     string `gorm:"type:varchar(200) NOT NULL"`
	LikeNum      int32  `gorm:"index"`
	ReplyNum     int32  `gorm:"index"`
	ReportNum    int32
	//Comments     []Comment
	CreatedAt time.Time      `gorm:"index"`
	UpdatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Comment struct {
	ID      int32 `gorm:"primaryKey;autoIncrement;not null"`
	ReplyTo int32 `gorm:"index"`
	//Post         Post
	PostID int32 `gorm:"index"`
	//User         User
	UserID       int32
	Text         string `gorm:"index:,class:FULLTEXT,option:WITH PARSER ngram;type: varchar(10000) NOT NULL"`
	Tag          string `gorm:"index;type:varchar(60) NOT NULL"`
	Type         string `gorm:"type:varchar(20) NOT NULL"`
	FilePath     string `gorm:"type:varchar(60) NOT NULL"`
	FileMetadata string `gorm:"type:varchar(40) NOT NULL"`
	Name         string `gorm:"type:varchar(60) NOT NULL"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type Report struct {
	ID int32 `gorm:"primaryKey;autoIncrement;not null"`
	//User           User
	UserID int32
	//ReportedUser   User
	ReportedUserID int32
	//Post           Post
	PostID int32
	//Comment        Comment
	CommentID int32
	Reason    string     `gorm:"type: varchar(1000) NOT NULL"`
	Type      ReportType `gorm:"type:varchar(20) NOT NULL"`
	IsComment bool
	Weight    int32
	CreatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

//TODO: (low priority)undelete = remove user reports

type Attention struct {
	User   User
	UserID int32 `gorm:"primaryKey;index"`
	Post   Post
	PostID int32 `gorm:"primaryKey;index"`
}

type Vote struct {
	User   User
	UserID int32 `gorm:"primaryKey;index"`
	Post   Post
	PostID int32  `gorm:"primaryKey;index"`
	Option string `gorm:"type:varchar(100) NOT NULL"`
}

type SystemMessage struct {
	ID int32 `gorm:"primaryKey;autoIncrement;not null"`
	//User   User
	UserID int32
	Text   string `gorm:"type: varchar(11000) NOT NULL"`
	Title  string `gorm:"type: varchar(100) NOT NULL"`
	//Ban       Ban
	BanID     int32          `gorm:"index"`
	CreatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Ban struct {
	ID int32 `gorm:"primaryKey;autoIncrement;not null"`
	//User      User
	UserID int32
	//Report    Report
	ReportID  int32
	Reason    string `gorm:"type: varchar(11000) NOT NULL"`
	ExpireAt  int64
	CreatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

//type Messages struct {
//	ID        int32 `gorm:"primaryKey;autoIncrement;not null"`
//	UserID    int32 `gorm:"index"`
//	CommentID int32
//}

func migrateUser(page int) (count int) {
	var results []map[string]interface{}
	var users []User
	err := base.GetDb(false).Table("v2_users").Order("id asc").Offset(batchSize * page).
		Limit(batchSize).Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v2_users!")
	for _, result := range results {
		var updateAt time.Time
		if result["updated_at"] != nil {
			updateAt = result["updated_at"].(time.Time)
		} else {
			updateAt = result["created_at"].(time.Time)
		}
		user := User{
			ID:             result["id"].(int32),
			OldToken:       result["token"].(string),
			OldEmailHash:   result["email_hash"].(string),
			EmailEncrypted: "",
			ForgetPwNonce:  "",
			Role:           UserRole(result["role"].(int64)),
			CreatedAt:      result["created_at"].(time.Time),
			UpdatedAt:      updateAt,
		}
		users = append(users, user)
	}
	count = len(results)
	if count > 0 {
		err = base.GetDb(false).Create(&users).Error
		utils.FatalErrorHandle(&err, "error writing v2_users!")
	}
	return
}

func migrateComment(page int) (count int) {
	var results []map[string]interface{}
	var comments []Comment
	err := base.GetDb(false).Table("v2_comments").Order("id asc").Offset(batchSize * page).
		Limit(batchSize).Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v2_comments!")
	for _, result := range results {
		var deletedAt gorm.DeletedAt
		_ = deletedAt.Scan(result["deleted_at"])
		comment := Comment{
			ID:           result["id"].(int32),
			ReplyTo:      -1,
			PostID:       int32(result["post_id"].(int64)),
			UserID:       int32(result["user_id"].(int64)),
			Text:         result["text"].(string),
			Tag:          result["tag"].(string),
			Type:         result["type"].(string),
			FilePath:     result["file_path"].(string),
			FileMetadata: getImgMetadata(result["file_path"].(string)),
			Name:         result["name"].(string),
			CreatedAt:    result["created_at"].(time.Time),
			UpdatedAt:    result["updated_at"].(time.Time),
			DeletedAt:    deletedAt,
		}
		comments = append(comments, comment)
	}
	count = len(results)
	if count > 0 {
		err = base.GetDb(false).Create(&comments).Error
		utils.FatalErrorHandle(&err, "error writing v2_comments!")
	}
	return
}

func migratePost(page int) (count int) {
	var results []map[string]interface{}
	var posts []Post
	err := base.GetDb(false).Table("v2_posts").Order("id asc").Offset(batchSize * page).
		Limit(batchSize).Find(&results).Error
	utils.FatalErrorHandle(&err, "error reading v2_posts!")
	for _, result := range results {
		var deletedAt gorm.DeletedAt
		_ = deletedAt.Scan(result["deleted_at"])

		tag := result["tag"].(string)
		if tag == "折叠" {
			tag = "令人不适"
		}
		post := Post{
			ID:           result["id"].(int32),
			UserID:       int32(result["user_id"].(int64)),
			Text:         result["text"].(string),
			Tag:          tag,
			Type:         result["type"].(string),
			FilePath:     result["file_path"].(string),
			FileMetadata: getImgMetadata(result["file_path"].(string)),
			LikeNum:      int32(result["like_num"].(int64)),
			ReplyNum:     int32(result["reply_num"].(int64)),
			ReportNum:    int32(result["report_num"].(int64)),
			VoteData:     "{}",
			CreatedAt:    result["created_at"].(time.Time),
			UpdatedAt:    result["updated_at"].(time.Time),
			DeletedAt:    deletedAt,
		}
		posts = append(posts, post)
	}
	count = len(results)
	if count > 0 {
		err = base.GetDb(false).Create(&posts).Error
		utils.FatalErrorHandle(&err, "error writing v2_posts!")
	}
	return
}

const batchSize = 3000

var metaData map[string]string

func getImgMetadata(imgName string) (rtn string) {
	if imgName == "" {
		return "{}"
	}
	var found bool
	rtn, found = metaData[imgName]
	if !found {
		log.Printf("img %s not found\n", imgName)
	}
	return
}

func migrate(foo func(int) int) {
	count := -1
	page := 0
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
	base.InitDb()

	metaData = make(map[string]string)
	err = filepath.Walk(viper.GetString("images_path"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, "jpeg") && !info.IsDir() {
			f, err2 := os.Open(path)
			if err2 != nil {
				log.Printf("error opening file %s %s\n", path, err2)
				return err2
			}
			defer f.Close()

			im, _, err3 := image.DecodeConfig(bufio.NewReader(f))
			if err3 != nil {
				log.Printf("error decoding image %s %s\n", path, err3)
			} else {
				metadataBytes, err4 := json.Marshal(map[string]int{"w": im.Width, "h": im.Height})
				if err4 != nil {
					log.Printf("error json.Marshal while decoding image %s , err=%s\n", path, err4.Error())
					return errors.New("图片大小解析失败")
				}
				metaData[info.Name()] = string(metadataBytes)
			}
		}

		return nil
	})
	utils.FatalErrorHandle(&err, "error walking images folder")

	err = base.GetDb(false).Migrator().RenameTable("users", "v2_users")
	utils.FatalErrorHandle(&err, "error rename table")
	err = base.GetDb(false).Migrator().RenameTable("posts", "v2_posts")
	utils.FatalErrorHandle(&err, "error rename table")
	err = base.GetDb(false).Migrator().RenameTable("comments", "v2_comments")
	utils.FatalErrorHandle(&err, "error rename table")

	err = base.GetDb(false).
		AutoMigrate(&User{}, &DecryptionKeyShares{}, &Email{},
			&Device{}, &PushSettings{}, &Vote{},
			&VerificationCode{}, &Post{}, //&Messages{},
			&Comment{}, &Attention{}, &Report{}, &SystemMessage{}, Ban{})
	utils.FatalErrorHandle(&err, "error migrating database!")

	migrate(migrateUser)
	log.Println("done migrating users")

	migrate(migratePost)
	log.Println("done migrating posts")

	migrate(migrateComment)
	log.Println("done migrating comments")
	log.Println("done all migration")
}
