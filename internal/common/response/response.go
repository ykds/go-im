package response

import (
	"errors"
	"go-im/internal/common/errcode"
	"go-im/internal/pkg/log"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(200, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, err error) {
	log.Errorf("err: %v", err)
	code := 500
	message := "服务器异常"
	var e *errcode.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}
	c.JSON(200, Response{
		Code:    code,
		Message: message,
	})
}
