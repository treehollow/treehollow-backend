package base

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"treehollow-v3-backend/pkg/logger"
)

func HttpReturnWithCodeMinusOne(c *gin.Context, e *logger.InternalError) {
	HttpReturnWithErr(c, -1, e)
}

func HttpReturnWithErr(c *gin.Context, code int, e *logger.InternalError) {
	user, exists := c.Get("user")
	if exists {
		e.InternalMsg = "(UserID=" + strconv.Itoa(int(user.(User).ID)) + ")" + e.InternalMsg
	}
	e.Log()
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  e.DisplayMsg,
	})
}

func HttpReturnWithErrAndAbort(c *gin.Context, code int, e *logger.InternalError) {
	HttpReturnWithErr(c, code, e)
	c.Abort()
}

func HttpReturnWithCodeMinusOneAndAbort(c *gin.Context, e *logger.InternalError) {
	HttpReturnWithErrAndAbort(c, -1, e)
	c.Abort()
}
