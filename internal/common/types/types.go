package types

import jsoniter "github.com/json-iterator/go"

var (
	CacheOnlineKey = "online:%d"
	json           = jsoniter.ConfigCompatibleWithStandardLibrary
)

type Message struct {
	Id        int64  `json:"id"`
	SessionId int64  `json:"sessionId"`
	FromId    int64  `json:"fromId"`
	ToId      int64  `json:"toId"`
	Content   string `json:"content"`
	Seq       int64  `json:"seq"`
	Kind      string `json:"kind"`
	CreatedAt int64  `json:"createdAt"`
}

func (m *Message) Encode() []byte {
	b, _ := json.Marshal(m)
	return b
}

func (m *Message) Decode(b []byte) error {
	return json.Unmarshal(b, m)
}
