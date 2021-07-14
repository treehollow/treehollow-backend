package security

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

func deleteAccount(c *gin.Context) {
	email := strings.ToLower(c.PostForm("email"))
	emailHash := utils.HashEmail(email)
	nonce := c.PostForm("nonce")
	code := c.PostForm("valid_code")
	now := utils.GetTimeStamp()
	if len(nonce) < 10 {
		base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("NonceNotEnoughLong", "Nonce错误", logger.INFO))
		return
	}

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

	_ = base.GetDb(false).Transaction(func(tx *gorm.DB) error {
		var user base.User
		err := tx.Model(&base.User{}).Where("forget_pw_nonce = ?", nonce).First(&user).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("NonceNotFound",
					"没有找到nonce对应的账户。请你重新查看刚刚注册树洞后收到的欢迎邮件中的“找回密码口令”(nonce)。"+
						"如果仍然无法解决问题，请联系"+viper.GetString("contact_email")+"。", logger.WARN))
				return errors.New("NonceNotFound")
			}
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "DeleteNonceFailed", consts.DatabaseReadFailedString))
			return err
		}

		if user.Role == base.BannedUserRole {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("DeleteAccountFrozen",
				"您的账户已被冻结，无法注销。如果需要解冻，请联系"+
					viper.GetString("contact_email")+"。", logger.ERROR))

			return errors.New("DeleteBannedAccount")
		}

		if user.CreatedAt.After(utils.GetEarliestAuthenticationTime()) {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("DeleteJustRegisteredAccount",
				"注销失败，账户需要注册"+strconv.Itoa(consts.TokenExpireDays)+"天以上才可以注销。", logger.ERROR))

			return errors.New("DeleteJustRegisteredAccount")
		}

		timestamp := utils.GetTimeStamp()
		var count int64
		err3 := tx.Model(&base.Ban{}).Where("user_id = ? and expire_at > ?", user.ID, timestamp).Count(&count).Error
		if err3 != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "GetBanFailed", consts.DatabaseReadFailedString))
			return err3
		}

		if count > 0 {
			base.HttpReturnWithCodeMinusOneAndAbort(c,
				logger.NewSimpleError("DisallowDeleteWhileBan", "很抱歉，您当前处于禁言状态，无法注销。", logger.ERROR))
			return errors.New("DisallowDeleteWhileBan")
		}

		result := tx.Where("forget_pw_nonce = ?", nonce).
			Delete(&base.User{})

		if result.Error != nil {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(result.Error, "DeleteNonceFailed", consts.DatabaseWriteFailedString))
			return result.Error
		}

		result = tx.Where("email_hash = ?", emailHash).
			Delete(&base.Email{})

		if result.Error != nil {
			base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(result.Error, "DeleteEmailHashFailed", consts.DatabaseWriteFailedString))
			return result.Error
		}

		if result.RowsAffected == 0 {
			base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("EmailNotFound",
				"没有找到此邮箱对应的账户", logger.WARN))
			return errors.New("EmailNotFound")
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
		})
		return nil
	})
}
