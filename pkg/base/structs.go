package base

import (
	"fmt"
	"gorm.io/gorm"
	"time"
	"treehollow-v3-backend/pkg/model"
	"treehollow-v3-backend/pkg/utils"
)

type UserRole int32

const (
	BannedUserRole   UserRole = -100
	SuperUserRole             = 0
	AdminRole                 = 1
	DeleterRole               = 2
	UnDeleterRole             = 3
	Deleter2Role              = 20
	Deleter3Role              = 21
	NormalUserRole            = 50
	UnregisteredRole          = 100
)

type ReportType string

const (
	UserReport        ReportType = "UserReport"
	UserReportFold    ReportType = "UserReportFold"
	UserDelete        ReportType = "UserDelete" // delete, no ban
	AdminTag          ReportType = "AdminTag"
	AdminDeleteAndBan ReportType = "AdminDeleteBan" // delete, ban
	AdminUndelete     ReportType = "Undelete"       // undelete + unban
	AdminUnban        ReportType = "AdminUnban"     // delete + unban
	//	For now, there's no "undelete + no unban" option
)

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
	DeletedAt gorm.DeletedAt `gorm:"index"`
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

const (
	WebDevice     DeviceType = 0
	AndroidDevice            = 1
	IOSDevice                = 2
)

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

type PushMessage struct {
	ID        int32  `gorm:"primaryKey;autoIncrement;not null"`
	Message   string `gorm:"type: varchar(10000) NOT NULL"`
	Title     string `gorm:"type: varchar(200) NOT NULL"`
	UserID    int32  `gorm:"index"`
	PostID    int32
	CommentID int32 `gorm:"index"`
	BanID     int32 `gorm:"index"`
	DoPush    bool  `gorm:"index"`
	Type      model.PushType
	UpdatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

//type Messages struct {
//	ID        int32 `gorm:"primaryKey;autoIncrement;not null"`
//	UserID    int32 `gorm:"index"`
//	CommentID int32
//}

func (report *Report) ToString() string {
	rtn := ""
	var name string
	if report.IsComment {
		name = fmt.Sprintf("To:树洞回复#%d-%d", report.PostID, report.CommentID)
	} else {
		name = fmt.Sprintf("To:树洞#%d", report.PostID)
	}
	rtn = fmt.Sprintf("%s\n***\nReason: %s", name, report.Reason)
	return rtn
}

func (typ *ReportType) ToString() string {
	switch *typ {
	case UserReport:
		return "用户举报"
	case UserReportFold:
		return "用户举报折叠"
	case AdminTag:
		return "管理员打Tag"
	case UserDelete:
		return "撤回或管理员删除"
	case AdminUndelete:
		return "撤销删除并解禁"
	case AdminDeleteAndBan:
		return "删帖禁言"
	case AdminUnban:
		return "解禁"
	default:
		return "unknown"
	}
}

func (report *Report) ToDetailedString() string {
	typeStr := report.Type.ToString()
	if report.Type == UserDelete {
		typeStr = utils.IfThenElse(report.UserID == report.ReportedUserID, "撤回", "管理员删除").(string)
	}
	rtn := fmt.Sprintf("From User ID:%d\nTo User ID:%d\nType:%s\n%s", report.UserID, report.ReportedUserID,
		typeStr, report.ToString())
	return rtn
}

func (msg *SystemMessage) ToString() string {
	return fmt.Sprintf("User ID:%d\nTitle:%s\n***\n%s", msg.UserID, msg.Title, msg.Text)
}
