package auth

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("TOKEN")
		user, err := base.GetUserWithCache(token)
		if err != nil {
			fmt.Println(err.Error())
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewError(err, "AuthDbFailed", consts.DatabaseReadFailedString))
				return
			}
			if !viper.GetBool("allow_unregistered_access") && !utils.IsInAllowedSubnet(c.ClientIP()) {
				base.HttpReturnWithErrAndAbort(c, -100, logger.NewSimpleError("TokenExpired",
					"登录凭据过期，请使用邮箱重新登录。", logger.INFO))
				return
			} else {
				c.Set("user", base.User{ID: -1, Role: base.UnregisteredRole, EmailEncrypted: ""})
				c.Next()
			}
		} else {
			if user.Role == base.BannedUserRole {
				base.HttpReturnWithCodeMinusOneAndAbort(c, logger.NewSimpleError("AccountFrozen",
					"您的账户已被冻结。如果需要解冻，请联系"+
						viper.GetString("contact_email")+"。", logger.ERROR))

				return
			}
			c.Set("user", user)
			c.Next()
		}
	}
}

func DisallowUnregisteredUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(base.User)
		if user.Role == base.UnregisteredRole {
			base.HttpReturnWithErrAndAbort(c, -100, logger.NewSimpleError("TokenExpired",
				"登录凭据过期，请使用邮箱重新登录。", logger.INFO))
			return
		}
		c.Next()
	}
}
