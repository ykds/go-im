package mkafka

import (
	"context"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/utils"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	MessageTopic           = "message"
	GroupApplyNotifyTopic  = "group-apply-notify"
	FriendApplyNotifyTopic = "apply-friend-notify"
)

type MsgType int

const (
	AckMsg       MsgType = 1
	HeartbeatMsg MsgType = 2

	FriendNotifyMsg      MsgType = 3
	FriendApplyMsg       MsgType = 4
	FriendApplyResultMsg MsgType = 5
	MessageMsg           MsgType = 6
	NewMessageMsg        MsgType = 7
	GroupNotifyMsg       MsgType = 8
	GroupApplyMsg        MsgType = 9
	GroupAppluResultMsg  MsgType = 10
)

type (
	WriterOption func(opt *kafka.Writer)
	ReaderOption func(opt *kafka.Reader)
)

type Writer struct {
	*kafka.Writer
	req chan kafka.Message
}

type Consumer struct {
	Topic string `json:"topic" yaml:"topic"`
	Group string `json:"group" yaml:"group"`
}

type Config struct {
	Brokers       []string   `json:"brokers"`
	Topic         string     `json:"topic" yaml:"topic"`
	ConsumerGroup []Consumer `json:"consumer_group" yaml:"consumer_group"`
}

func NewProducer(c Config, opts ...WriterOption) *Writer {
	if len(c.Brokers) == 0 {
		panic("brokers empty")
	}
	kw := &kafka.Writer{
		BatchTimeout: time.Millisecond * 10,
		Addr:         kafka.TCP(c.Brokers...),
		Balancer:     &kafka.RoundRobin{},
		Compression:  kafka.Lz4,
		RequiredAcks: kafka.RequireOne,
		Topic:        c.Topic,
	}
	for _, opt := range opts {
		opt(kw)
	}
	w := &Writer{
		Writer: kw,
		req:    make(chan kafka.Message, 1000),
	}
	utils.SafeGo(func() {
		for item := range w.req {
			err := w.WriteMessages(context.Background(), item)
			if err != nil {
				log.Errorf("发送kafka消息失败, %v", err)
			}
		}
	})
	return w
}

func (w *Writer) Send(msg ...kafka.Message) {
	for _, item := range msg {
		w.req <- item
	}
}

func NewGroupReader(c Config, groupId string, topic string, opts ...ReaderOption) *kafka.Reader {
	rc := kafka.ReaderConfig{
		Brokers: c.Brokers,
		Topic:   topic,
		GroupID: groupId,
	}
	r := kafka.NewReader(rc)
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func NewReader(c Config, opts ...ReaderOption) *kafka.Reader {
	rc := kafka.ReaderConfig{
		Brokers: c.Brokers,
		Topic:   c.Topic,
	}
	r := kafka.NewReader(rc)
	for _, opt := range opts {
		opt(r)
	}
	return r
}
