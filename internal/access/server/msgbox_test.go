package server

import (
	"go-im/api/access"
	"go-im/internal/common/protocol"
	"testing"
)

func TestMsgBox(t *testing.T) {
	b := NewMsgBox()
	for i := 0; i < 20; i++ {
		b.Append(&access.Message{
			Type: int64(protocol.MessageMsg),
			Data: "test",
		}, nil, 1)
	}

	list := b.List("", 1, 0)
	for _, item := range list {
		t.Logf("%+v\n", item)
	}

	t.Log("----------------------")

	b.Ack("", 1, 10)

	list = b.List("", 1, 10)
	for _, item := range list {
		t.Logf("%+v\n", item)
	}
}
