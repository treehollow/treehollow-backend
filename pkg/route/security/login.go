package security

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

func loginGetUserMiddleware(c *gin.Context) {
	pwHashed := c.PostForm("password_hashed")
	email := strings.ToLower(c.PostForm("email"))

	emailEncrypted, err := utils.AESEncrypt(email, pwHashed)
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "AESEncryptFailedInMiddleware", consts.DatabaseEncryptFailedString))
		return
	}

	var user base.User

	err = base.GetDb(false).Where("email_encrypted = ?", emailEncrypted).
		Model(&base.User{}).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("MiddlewareNoAuth", "用户名或密码错误", logger.WARN))
		} else {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetUserByEmailEncryptedFailed", consts.DatabaseReadFailedString))
		}
		return
	}
	if user.Role == base.BannedUserRole {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("AccountFrozen",
			"您的账户已被冻结。如果需要解冻，请联系"+
				viper.GetString("contact_email")+"。", logger.ERROR))

		return
	}

	c.Set("user", user)
	c.Next()
}

func loginCheckMaxDevices(c *gin.Context) {
	user := c.MustGet("user").(base.User)

	var count int64
	err := base.GetDb(false).
		Where("user_id = ? and created_at > ?", user.ID, utils.GetEarliestAuthenticationTime()).
		Model(&base.Device{}).Count(&count).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "GetEarliestDeviceFailed", consts.DatabaseReadFailedString))
		return
	}
	if count >= consts.MaxDevicesPerUser {
		log.Printf("user login more than max allowed: %d\n", user.ID)
		_ = base.GetDb(false).
			Where("user_id = ? and created_at > ?", user.ID, utils.GetEarliestAuthenticationTime()).
			Order("created_at asc").Limit(1).
			Delete(&base.Device{}).Error
		return
	}
	c.Next()
}

func login(c *gin.Context) {
	user := c.MustGet("user").(base.User)
	token := utils.GenToken()
	deviceUUID := uuid.New().String()
	deviceType := c.MustGet("device_type").(base.DeviceType)
	deviceInfo := c.PostForm("device_info")
	city := "Unknown"

	if geoDb := utils.GeoDb.Get(); geoDb != nil {
		ip := net.ParseIP(c.ClientIP())
		record, err5 := geoDb.City(ip)
		if err5 == nil {
			country := record.Country.Names["zh-CN"]
			if len(country) == 0 {
				country = record.Country.Names["en"]
			}
			if len(country) > 0 {
				cityName := record.City.Names["zh-CN"]
				if len(cityName) == 0 {
					cityName = record.City.Names["en"]
				}
				if len(cityName) > 0 {
					city = cityName + ", " + country
				} else {
					city = country
				}
			}
		}
	}

	err := base.GetDb(false).Create(&base.Device{
		ID:             deviceUUID,
		UserID:         user.ID,
		Token:          token,
		DeviceInfo:     deviceInfo,
		Type:           deviceType,
		LoginIP:        c.ClientIP(),
		LoginCity:      city,
		IOSDeviceToken: c.PostForm("ios_device_token"),
	}).Error
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SaveDeviceWhileLoginFailed", consts.DatabaseWriteFailedString))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"token": token,
		"uuid":  deviceUUID,
	})
	_ = base.GetDb(false).Create(&base.SystemMessage{
		UserID: user.ID,
		Title:  "新的登录",
		Text: fmt.Sprintf("您好，您的账户在%s于%s使用设备\"%s\"登录。\n\n如果这不是您本人所为，请您立刻修改密码。",
			time.Now().Format("2006-01-02 15:04"), city, deviceInfo),
		BanID: -1,
	}).Error
	//TODO: (middle priority) send email
	return
}

func logout(c *gin.Context) {
	token := c.GetHeader("TOKEN")
	result := base.GetDb(false).
		Where("token = ? and created_at > ?", token, utils.GetEarliestAuthenticationTime()).
		Delete(&base.Device{})
	if result.Error != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(result.Error, "DeleteDeviceFailed", consts.DatabaseWriteFailedString))
		return
	}
	if result.RowsAffected != 1 {
		base.HttpReturnWithErrAndAbort(c, -100, logger.NewSimpleError("TokenExpired",
			"登录凭据过期，请使用邮箱重新登录。", logger.INFO))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}
