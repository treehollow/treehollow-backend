package contents

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/model"
)

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func getPush(c *gin.Context) {
	user := c.MustGet("user").(base.User)
	var pushSettings base.PushSettings
	err := base.GetDb(false).First(&pushSettings, user.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": gin.H{
					"push_system_msg": 1,
					"push_reply_me":   1,
					"push_favorited":  0,
				},
			})
		} else {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "GetPushSettingsFailed", consts.DatabaseReadFailedString))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"push_system_msg": boolToInt((pushSettings.Settings & model.SystemMessage) > 0),
			"push_reply_me":   boolToInt((pushSettings.Settings & model.ReplyMeComment) > 0),
			"push_favorited":  boolToInt((pushSettings.Settings & model.CommentInFavorited) > 0),
		},
	})
}

func setPush(c *gin.Context) {
	pushSystemMsg := c.PostForm("push_system_msg")
	pushReplyMe := c.PostForm("push_reply_me")
	pushFavorited := c.PostForm("push_favorited")
	user := c.MustGet("user").(base.User)

	var pushSettings model.PushType
	if pushSystemMsg == "1" {
		pushSettings += model.SystemMessage
	}
	if pushReplyMe == "1" {
		pushSettings += model.ReplyMeComment
	}
	if pushFavorited == "1" {
		pushSettings += model.CommentInFavorited
	}

	err := base.GetDb(false).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&base.PushSettings{
		UserID:   user.ID,
		Settings: pushSettings,
	}).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SavePushSettingsFailed", consts.DatabaseWriteFailedString))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}
