package server

import (
	"go-im/internal/access/types"
	"go-im/internal/pkg/mkafka"
	"testing"
)

func TestMsgBox(t *testing.T) {
	b := NewMsgBox()
	for i := 0; i < 20; i++ {
		b.Append(&types.Message{
			FromId:  1,
			ToId:    2,
			Seq:     int64(i),
			Type:    mkafka.MessageMsg,
			Content: "test",
		}, "single", 1)
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
