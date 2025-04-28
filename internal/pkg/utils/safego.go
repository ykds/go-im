package utils

import (
	"go-im/internal/pkg/log"
	"runtime/debug"
)

func SafeGo(f func()) {
	go func() {
		defer func() {
			if e := recover(); e != nil {
				log.Errorf("panic: %+v, Stack: %v", e, string(debug.Stack()))
			}
		}()
		f()
	}()
}
