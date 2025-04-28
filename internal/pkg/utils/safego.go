package utils

import (
	"runtime/debug"

	"github.com/zeromicro/go-zero/core/logx"
)

func SafeGo(f func()) {
	go func() {
		defer func() {
			if e := recover(); e != nil {
				logx.Errorf("panic: %+v, Stack: %v", e, string(debug.Stack()))
			}
		}()
		f()
	}()
}
