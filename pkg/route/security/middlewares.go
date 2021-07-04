package security

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

func loginCheckIOSToken(c *gin.Context) {
	deviceType := c.MustGet("device_type").(base.DeviceType)
	if deviceType == base.IOSDevice {
		iosDeviceToken := c.PostForm("ios_device_token")
		//if len(iosDeviceToken) < 1 || len(iosDeviceToken) > 100 {
		if len(iosDeviceToken) > 100 {
			base.HttpReturnWithErrAndAbort(c, -11, logger.NewSimpleError("NoIOSDeviceToken", "获取iOS推送口令失败", logger.WARN))
			return
		}
	}
	c.Next()
}

func loginParamsCheckMiddleware(c *gin.Context) {
	pwHashed := c.PostForm("password_hashed")
	email := strings.ToLower(c.PostForm("email"))
	deviceTypeStr := c.PostForm("device_type")
	deviceInfo := c.PostForm("device_info")
	iosDeviceToken := c.PostForm("ios_device_token")

	if len(email) > 100 || len(pwHashed) > 64 || len(deviceInfo) > 100 || len(iosDeviceToken) > 100 {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("LoginParamsOutOfBound", "参数错误", logger.WARN))
		return
	}
	deviceTypeInt, err := strconv.Atoi(deviceTypeStr)
	deviceType := base.DeviceType(deviceTypeInt)
	if err != nil || (deviceType != base.AndroidDevice &&
		deviceType != base.IOSDevice &&
		deviceType != base.WebDevice) {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("DeviceTypeError", "参数device_type错误", logger.WARN))
		return
	}

	c.Set("device_type", deviceType)
	c.Next()
}

func checkAccountNotRegistered(c *gin.Context) {
	email := strings.ToLower(c.PostForm("email"))
	emailHash := utils.HashEmail(email)

	var count int64
	err := base.GetDb(false).Where("email_hash = ?", emailHash).
		Model(&base.Email{}).Count(&count).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "CheckAccountRegisteredFailed", consts.DatabaseReadFailedString))
		return
	}
	if count == 1 {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("AlreadyRegisteredError", "你已经注册过了！", logger.WARN))
		return
	}

	c.Set("email_hash", emailHash)
	c.Next()
}

func checkAccountIsRegistered(c *gin.Context) {
	email := strings.ToLower(c.PostForm("email"))
	emailHash := utils.HashEmail(email)

	var count int64
	err := base.GetDb(false).Where("email_hash = ?", emailHash).
		Model(&base.Email{}).Count(&count).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "CheckAccountUnRegisteredFailed", consts.DatabaseReadFailedString))
		return
	}
	if count != 1 {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("NotRegisteredError", "你还没有注册过！", logger.WARN))
		return
	}

	c.Set("email_hash", emailHash)
	c.Next()
}
