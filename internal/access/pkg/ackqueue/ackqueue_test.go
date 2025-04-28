package ackqueue

import (
	"go-im/api/access"
	"testing"
	"time"
)

func TestAckQueue(t *testing.T) {
	retry := make(chan *access.Message, 10)
	go func() {
		for item := range retry {
			t.Logf("retry: %v", item)
		}
	}()
	q := NewAckQueue(100*time.Millisecond, retry)
	m := &access.Message{
		Type: 3,
		Data: "test",
	}
	q.Put(m)
	time.Sleep(time.Millisecond*30)
	q.Ack(m.AckId)

	m1 := &access.Message{
		Type: 3,
		Data: "test",
	}
	q.Put(m1)
	time.Sleep(time.Millisecond*30)

	q.Ack(m1.AckId)
}
