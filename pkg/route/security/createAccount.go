package security

import (
	"errors"
	"fmt"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/SSSaaS/sssa-golang"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net"
	"net/http"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/mail"
	"treehollow-v3-backend/pkg/utils"
)

func saveKeyShares(user *base.User, pwHashed string, tx *gorm.DB) *logger.InternalError {
	pgpPublicKeys := viper.GetStringSlice("key_keepers_pgp_public_keys")
	minDecryptShares := viper.GetInt("min_decryption_key_count")
	if len(pgpPublicKeys) < 1 || minDecryptShares < 1 {
		return logger.NewSimpleError("ServerConfigError", "服务器加密配置错误，请联系管理员", logger.FATAL)
	}
	shares, err2 := sssa.Create(minDecryptShares, len(pgpPublicKeys), pwHashed)
	if err2 != nil {
		return logger.NewError(err2, "SSSACreateFailed", "加密失败，请联系管理员")
	}
	for i, share := range shares {
		keyRing, _ := utils.CreatePublicKeyRing(pgpPublicKeys[i])
		PGPEmail := keyRing.GetIdentities()[0].Email
		msg := fmt.Sprintf(`Hello keykeeper %s,

If you can see this message, you've successfully obtained your key slice.

The following string is the key slice that can be used to decrypt the user whose id=%d. There are %d such key slices in total and the user's personal information can be decrypted when the number of available key slices is greater than or equal to %d.

If you agree to decrypt this user's personal information, please submit the following key slice to technician for decryption. If you do not agree to the decryption, please do not disclose this key slice to anyone.

======================
%s
======================`, PGPEmail, user.ID, len(pgpPublicKeys), minDecryptShares, share)
		armor, err3 := helper.EncryptMessageArmored(pgpPublicKeys[i], msg)
		if err3 != nil {
			return logger.NewError(err3, "EncryptMessageArmoredFailed", "加密失败，请联系管理员")
		}
		err4 := tx.Create(&base.DecryptionKeyShares{
			EmailEncrypted: user.EmailEncrypted,
			PGPMessage:     armor,
			PGPEmail:       PGPEmail,
		}).Error
		if err4 != nil {
			return logger.NewError(err4, "SaveDecryptionKeySharesFailed", consts.DatabaseWriteFailedString)
		}
	}
	return nil
}

func createDevice(c *gin.Context, user *base.User, pwHashed string, tx *gorm.DB) error {
	email := strings.ToLower(c.PostForm("email"))
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

	err := tx.Create(&base.Device{
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
		rtn := logger.NewError(err, "CreateSaveDeviceFailed", consts.DatabaseWriteFailedString)
		base.HttpReturnWithCodeMinusOne(c, rtn)
		return rtn.Err
	}

	err4 := saveKeyShares(user, pwHashed, tx)
	if err4 != nil {
		base.HttpReturnWithCodeMinusOne(c, err4)
		return err4.Err
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"token": token,
		"uuid":  deviceUUID,
	})
	go func() {
		_ = mail.SendPasswordNonceEmail(user.ForgetPwNonce, email)
	}()
	return nil
}

func createAccount(c *gin.Context) {
	oldToken := c.PostForm("old_token")
	emailHash := c.MustGet("email_hash").(string)
	email := strings.ToLower(c.PostForm("email"))
	pwHashed := c.PostForm("password_hashed")
	emailEncrypted, err := utils.AESEncrypt(email, pwHashed)

	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "AESEncryptFailedInCreateAccount", consts.DatabaseEncryptFailedString))
		return
	}

	var user base.User
	err5 := base.GetDb(false).Where("old_email_hash = ?", emailHash).
		Model(&base.User{}).First(&user).Error
	if err5 == nil && user.OldToken == oldToken {
		//	Don't need valid code
	} else {
		if err5 != nil && !errors.Is(err5, gorm.ErrRecordNotFound) {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err5, "QueryOldEmailHashFailed", consts.DatabaseReadFailedString))
			return
		}
		code := c.PostForm("valid_code")
		now := utils.GetTimeStamp()
		correctCode, timeStamp, failedTimes, err2 := base.GetVerificationCode(emailHash)
		if err2 != nil && !errors.Is(err2, gorm.ErrRecordNotFound) {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err2, "QueryValidCodeFailed", consts.DatabaseReadFailedString))
			return
		}
		if failedTimes >= 10 && now-timeStamp <= 43200 {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("ValidCodeTooMuchFailed", "验证码错误尝试次数过多，请重新发送验证码", logger.INFO))
			return
		}
		if correctCode != code || now-timeStamp > 43200 {
			base.HttpReturnWithErrAndAbort(c, -10, logger.NewSimpleError("ValidCodeInvalid", "验证码无效或过期", logger.WARN))
			_ = base.GetDb(false).Model(&base.VerificationCode{}).Where("email_hash = ?", emailHash).
				Update("failed_times", gorm.Expr("failed_times + 1")).Error
			return
		}
	}

	_ = base.GetDb(false).Transaction(func(tx *gorm.DB) error {
		if err = tx.Create(&base.Email{EmailHash: emailHash}).Error; err != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "CreateEmailHashFailed", consts.DatabaseWriteFailedString))
			return err
		}

		if err5 != nil {
			user = base.User{
				EmailEncrypted: emailEncrypted,
				ForgetPwNonce:  utils.GenNonce(),
				Role:           base.NormalUserRole,
			}
			if err = tx.Create(&user).Error; err != nil {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "CreateUserFailed", consts.DatabaseWriteFailedString))
				return err
			}
		} else {
			user.OldEmailHash = ""
			user.OldToken = ""
			user.EmailEncrypted = emailEncrypted
			user.UpdatedAt = time.Now()
			user.ForgetPwNonce = utils.GenNonce()
			if err = tx.Model(&base.User{}).Where("id = ?", user.ID).Updates(user).Error; err != nil {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "UpdateOldUserFailed", consts.DatabaseWriteFailedString))
				return err
			}
		}

		return createDevice(c, &user, pwHashed, tx)
	})
}

func changePassword(c *gin.Context) {
	oldPwHashed := c.PostForm("old_password_hashed")
	newPwHashed := c.PostForm("new_password_hashed")
	email := strings.ToLower(c.PostForm("email"))

	if len(email) > 100 || len(oldPwHashed) > 64 || len(newPwHashed) > 64 {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("ChangePasswordInvalidParam", "参数错误", logger.WARN))
		return
	}

	oldEmailEncrypted, err := utils.AESEncrypt(email, oldPwHashed)
	if err != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "AESEncryptFailed", consts.DatabaseEncryptFailedString))
		return
	}
	newEmailEncrypted, err2 := utils.AESEncrypt(email, newPwHashed)
	if err2 != nil {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err2, "AESEncryptFailed2", consts.DatabaseEncryptFailedString))
		return
	}

	_ = base.GetDb(false).Transaction(func(tx *gorm.DB) error {
		var user base.User
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email_encrypted = ?", oldEmailEncrypted).
			Model(&base.User{}).First(&user)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("ChangePasswordNoAuth", "用户名或密码错误", logger.WARN))
				return nil
			}
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(result.Error, "GetUserByEmailEncryptedFailed", consts.DatabaseReadFailedString))
			return result.Error
		}
		//if result.RowsAffected != 1 {
		//	base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("ChangePasswordNoAuth", "用户名或密码错误", logger.WARN))
		//	return nil
		//}

		result = tx.Model(&base.User{}).Where("email_encrypted = ?", oldEmailEncrypted).
			Update("email_encrypted", newEmailEncrypted)

		if result.Error != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(result.Error, "UpdateEmailEncryptedFailed", consts.DatabaseWriteFailedString))
			return result.Error
		}

		err3 := tx.Where("email_encrypted = ?", oldEmailEncrypted).
			Delete(&base.DecryptionKeyShares{}).Error
		if err3 != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "DeleteDecryptionSharesFailed", consts.DatabaseWriteFailedString))
			return err3
		}

		err4 := saveKeyShares(&base.User{
			EmailEncrypted: newEmailEncrypted,
		}, newPwHashed, tx)
		if err4 != nil {
			base.HttpReturnWithCodeMinusOne(c, err4)
			return err4.Err
		}

		err5 := tx.Where("user_id = ?", user.ID).
			Delete(&base.Device{}).Error
		if err5 != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "DeleteUserAllDevicesFailed", consts.DatabaseWriteFailedString))
			return err5
		}

		//TODO: (middle priority) send email
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
		})

		return nil
	})
}
