package structs

import (
	"fmt"
	"gorm.io/gorm"
	"time"
)

type UserRole int32

const (
	BannedUserRole   UserRole = -100
	SuperUserRole             = 0
	AdminRole                 = 1
	DeleterRole               = 2
	UnDeleterRole             = 3
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
	EmailHash      string `gorm:"uniqueIndex;type:char(64) NOT NULL"`
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
	CreatedAt time.Time      `gorm:"index"`
	UpdatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Comment struct {
	ID        int32 `gorm:"primaryKey;autoIncrement;not null"`
	Post      Post
	PostID    int32 `gorm:"index"`
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
	PostID int32 `gorm:"primaryKey;index"`
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

func (report *Report) ToString() string {
	rtn := ""
	var name string
	if report.IsComment {
		name = fmt.Sprintf("To:树洞回复#%d-%d", report.PostID, report.CommentID)
	} else {
		name = fmt.Sprintf("To:树洞#%d", report.PostID)
	}
	rtn = fmt.Sprintf("%s\n***\n Reason: %s", name, report.Reason)
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
	rtn := fmt.Sprintf("From User ID:%d\nTo User ID:%d\nType:%s\n%s", report.UserID, report.ReportedUserID,
		report.Type.ToString(), report.ToString())
	return rtn
}

func (msg *SystemMessage) ToString() string {
	return fmt.Sprintf("User ID:%d\nTitle:%s\n***\n%s", msg.UserID, msg.Title, msg.Text)
}
