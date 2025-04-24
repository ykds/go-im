package response

import (
	"context"
	"errors"
	"go-im/internal/common/errcode"
	"go-im/internal/pkg/log"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(data any) Response {
	return Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}
}

func Error(ctx context.Context, err error) Response {
	log.Errorf("err: %v", err)
	code := 500
	message := "服务器异常"
	var e *errcode.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}
	return Response{
		Code:    code,
		Message: message,
	}
}
