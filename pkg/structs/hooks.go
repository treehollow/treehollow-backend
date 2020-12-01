package structs

import (
	"gorm.io/gorm"
	"thuhole-go-backend/pkg/utils"
)

//Set the first registered user to be superuser
func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	if u.ID == 1 {
		err = tx.Model(u).Update("role", SuperUserRole).Error
	}
	return
}

func (post *Post) AfterCreate(tx *gorm.DB) (err error) {
	err = tx.Create(&Attention{UserID: post.UserID, PostID: post.ID}).Error
	return
}

func (comment *Comment) AfterCreate(tx *gorm.DB) (err error) {
	var attention int64
	err = tx.Model(&Attention{}).Where(&Attention{UserID: comment.UserID, PostID: comment.PostID}).Count(&attention).Error
	if err == nil && attention == 0 {
		err = tx.Create(&Attention{UserID: comment.UserID, PostID: comment.PostID}).Error
	}
	if err == nil {
		err = tx.Model(&Post{ID: comment.PostID}).
			Update("reply_num", gorm.Expr("reply_num + 1")).Error
	}
	return
}

func (attention *Attention) AfterCreate(tx *gorm.DB) (err error) {
	err = tx.Table("posts").Where("id = ?", attention.PostID).
		UpdateColumn("like_num", gorm.Expr("like_num + 1")).Error
	return
}

func (attention *Attention) AfterDelete(tx *gorm.DB) (err error) {
	err = tx.Table("posts").Where("id = ?", attention.PostID).
		UpdateColumn("like_num", gorm.Expr("like_num - 1")).Error
	return
}

func (report *Report) AfterCreate(tx *gorm.DB) (err error) {
	if report.Type == UserReport && !report.IsComment {
		err = tx.Table("posts").Where("id = ?", report.PostID).
			UpdateColumn("report_num", gorm.Expr("report_num + 1")).Error
	}
	return
}

func (ban *Ban) AfterCreate(tx *gorm.DB) (err error) {
	err = tx.Create(&SystemMessage{
		UserID: ban.UserID,
		BanID:  ban.ID,
		Title:  "封禁提示",
		Text:   ban.Reason + "\n\n在" + utils.TimestampToString(ban.ExpireAt) + "之前您将无法发布树洞。",
	}).Error
	return
}

//TODO: maybe, show reason here?
func (ban *Ban) AfterDelete(tx *gorm.DB) (err error) {
	err = tx.Create(&SystemMessage{
		UserID: ban.UserID,
		BanID:  ban.ID,
		Title:  "解除封禁提示",
		Text:   "您的以下封禁已被管理员手动解除：\n\n\"" + ban.Reason + "\"",
	}).Error
	return
}
