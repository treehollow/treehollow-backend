package security

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

func updateIOSToken(c *gin.Context) {
	token := c.GetHeader("TOKEN")
	iosDeviceToken := c.PostForm("ios_device_token")
	if len(iosDeviceToken) < 1 || len(iosDeviceToken) > 100 {
		base.HttpReturnWithErrAndAbort(c, -11, logger.NewSimpleError("NoIOSToken", "获取iOS推送口令失败", logger.WARN))
		return
	}
	result := base.GetDb(false).Model(&base.Device{}).
		Where("token = ? and created_at > ?", token, utils.GetEarliestAuthenticationTime()).
		Update("ios_device_token", iosDeviceToken)
	if result.Error != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(result.Error, "UpdateIOSTokenFailed", consts.DatabaseWriteFailedString))
		return
	}
	if result.RowsAffected != 1 {
		base.HttpReturnWithErrAndAbort(c, -100, logger.NewSimpleError("NoUpdateDeviceToken", "更新Device Token失败", logger.INFO))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}
