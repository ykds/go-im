package errcode

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidParam        = NewError(400, "参数错误")
	ErrUnAuthorized        = NewError(401, "未授权")
	ErrServerInternalError = NewError(500, "服务器异常")
)

// user
var (
	ErrUserNotExists = NewError(10001, "用户不存在")
	ErrPasswordWrong = NewError(10002, "密码错误")
	ErrPhoneRegisted = NewError(10003, "手机号已注册")
	ErrTokenExpired  = NewError(10004, "token已失效")
)

// friends
var (
	ErrFriendExists    = NewError(20001, "好友已存在")
	ErrApplyExists     = NewError(20002, "申请已存在")
	ErrApplyNotFound   = NewError(20003, "申请不存在")
	ErrApplyNotPending = NewError(20004, "申请已处理")
	ErrFriendNotExists = NewError(20005, "好友不存在")
)

// session
var (
	ErrSessionExists    = NewError(30001, "会话已存在")
	ErrSessionNotExists = NewError(30002, "会话不存在")
	ErrCreateMessage    = NewError(30003, "创建消息失败")
	ErrNotFriend        = NewError(30004, "非好友关系")
)

// message
var (
	ErrMessageExists = NewError(40001, "消息重复")
)

// group
var (
	ErrGroupNoExists  = NewError(50001, "群组不存在")
	ErrWasGroupMember = NewError(50002, "已是群组成员")
	ErrHadApplied     = NewError(50003, "已存在申请")
	ErrApplyHandled   = NewError(50004, "申请已处理")
	ErrGroupOwnerOnly = NewError(50005, "仅群主可操作")
	ErrNotGroupMember = NewError(50006, "非群组成员")
)

var (
	ErrAvatarExtNotSupported = NewError(60001, "头像文件格式不支持")
)

var (
	codeMap = make(map[int]*Error)
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("code: %d, message: %v", e.Code, e.Message)
}

func NewError(code int, message string) *Error {
	if _, ok := codeMap[code]; ok {
		panic("code has defined")
	}
	e := &Error{Code: code, Message: message}
	codeMap[e.Code] = e
	return e
}

func ToRpcError(err error) error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return status.New(codes.Code(e.Code), e.Message).Err()
	} else {
		return status.New(500, "服务异常").Err()
	}
}

func FromRpcError(err error) error {
	if err == nil {
		return nil
	}
	s, ok := status.FromError(err)
	if !ok {
		return &Error{Code: 500, Message: s.Message()}
	}
	e, ok := codeMap[int(s.Code())]
	if !ok {
		return &Error{Code: 500, Message: s.Message()}
	}
	return e
}
