package security

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/logger"
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
	email := c.PostForm("email")
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
