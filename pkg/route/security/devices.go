package security

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

func devicesToJson(devices []base.Device) []gin.H {
	var data []gin.H
	for _, device := range devices {
		data = append(data, gin.H{
			"device_uuid": device.ID,
			"login_date":  device.CreatedAt.Format("2006-01-02"),
			"device_info": device.DeviceInfo,
			"device_type": int32(device.Type),
		})
	}
	return data
}

func listDevices(c *gin.Context) {
	token := c.GetHeader("TOKEN")
	var device base.Device
	err := base.GetDb(false).Model(&base.Device{}).
		Where("token = ? and created_at > ?", token, utils.GetEarliestAuthenticationTime()).
		First(&device).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			base.HttpReturnWithErrAndAbort(c, -100, logger.NewSimpleError("TokenExpired",
				"登录凭据过期，请使用邮箱重新登录。", logger.INFO))
		} else {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetDeviceByTokenFailed", consts.DatabaseReadFailedString))
		}
		return
	}

	var devices []base.Device
	err = base.GetDb(false).Model(&base.Device{}).
		Where("user_id = ? and created_at > ?", device.UserID, utils.GetEarliestAuthenticationTime()).
		Find(&devices).
		Error
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetDevicesByUserIDFailed", consts.DatabaseReadFailedString))
	}
	data := devicesToJson(devices)
	c.JSON(http.StatusOK, gin.H{
		"code":        0,
		"data":        data,
		"this_device": device.ID,
	})
}

func terminateDevice(c *gin.Context) {
	token := c.GetHeader("TOKEN")
	var device base.Device
	err := base.GetDb(false).Model(&base.Device{}).
		Where("token = ? and created_at > ?", token, utils.GetEarliestAuthenticationTime()).
		First(&device).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			base.HttpReturnWithErrAndAbort(c, -100, logger.NewSimpleError("TokenExpired",
				"登录凭据过期，请使用邮箱重新登录。", logger.INFO))
		} else {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetDeviceByTokenFailed", consts.DatabaseReadFailedString))
		}
		return
	}

	deviceUUID := c.PostForm("device_uuid")
	result := base.GetDb(false).
		Where("user_id = ? and id = ? and created_at > ?", device.UserID, deviceUUID, utils.GetEarliestAuthenticationTime()).
		Delete(&base.Device{})
	if result.Error != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(result.Error, "DeleteDeviceByUUIDFailed", consts.DatabaseWriteFailedString))
		return
	}
	if result.RowsAffected != 1 {
		base.HttpReturnWithErrAndAbort(c, -100, logger.NewSimpleError("NoDeviceFound", "找不到这个设备。", logger.WARN))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
	_ = base.DelUserCache(device.Token)
}
