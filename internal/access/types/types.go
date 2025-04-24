package types

import (
	"go-im/internal/pkg/mkafka"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Message struct {
	Id        int64
	SessionId int64
	FromId    int64
	ToId      int64
	Seq       int64
	Type      mkafka.MsgType `json:"type"`
	Content   string         `json:"content,omitempty"`
}

func (r *Message) Encode() []byte {
	b, _ := json.Marshal(r)
	return b
}

func (r *Message) Decode(b []byte) error {
	return json.Unmarshal(b, r)
}

type AckMessage struct {
	Type      mkafka.MsgType `json:"type"`
	Kind      string         `json:"kind"`
	SessionId int64          `json:"sessionId"`
	Id        int64          `json:"id"`
	Seq       int64          `json:"seq"`
}

func (a *AckMessage) Decode(b []byte) error {
	return json.Unmarshal(b, a)
}

type ReSendMsg struct {
	Bytes []byte
	Count int
}

type PollMessageReq struct {
	Kind      string `json:"kind"`
	SessionId int64  `json:"sessionId"`
	Seq       int64  `json:"seq"`
}

func (p *PollMessageReq) Decode(b []byte) error {
	return json.Unmarshal(b, p)
}

type NewMessageResp struct {
	Kind      string `json:"kind"`
	SessionId int64  `json:"sessionId"`
	Seq       int64  `json:"seq"`
	ToId      int64  `json:"toId"`
}

func (nm *NewMessageResp) Encode() []byte {
	b, _ := json.Marshal(nm)
	return b
}

func (nm *NewMessageResp) Decode(b []byte) error {
	return json.Unmarshal(b, nm)
}
