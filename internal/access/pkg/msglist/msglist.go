package msglist

import (
	"go-im/api/access"
	"sync"
)

type node struct {
	Content *access.Message
	seq     int64
	Unread  int
	next    *node
	pre     *node
}

type MsgList struct {
	loc  map[int64]*node
	head *node
	tail *node
	size int
	m    *sync.Mutex
}

func NewMsgList() *MsgList {
	return &MsgList{
		loc:  make(map[int64]*node),
		head: &node{},
		m:    &sync.Mutex{},
	}
}

func (l *MsgList) List(seq int64) []*access.Message {
	head := l.head.next
	for head != nil && head.seq < seq {
		head = head.next
	}
	result := make([]*access.Message, 0, 10)
	for head != nil {
		result = append(result, head.Content)
		head = head.next
	}
	return result
}

func (l *MsgList) Insert(msg *access.Message, msgBody *access.MessageBody, unread int) error {
	l.m.Lock()
	defer l.m.Unlock()

	newNode := &node{
		seq:     msgBody.Seq,
		Content: msg,
		Unread:  unread,
	}
	if l.tail == nil {
		l.head.next = newNode
		newNode.pre = l.head
		l.tail = newNode
	} else {
		l.tail.next = newNode
		newNode.pre = l.tail
		l.tail = newNode
	}
	l.loc[msgBody.Seq] = newNode
	l.size++
	return nil
}

func (l *MsgList) AckMsg(seq int64) {
	l.m.Lock()
	defer l.m.Unlock()

	node, ok := l.loc[seq]
	if !ok {
		return
	}
	for node != l.head {
		node.Unread--
		if node.Unread == 0 {
			pre := node.pre
			if node == l.tail {
				l.tail = nil
				pre.next = nil
			} else {
				pre.next = node.next
				if node.next != nil {
					node.next.pre = pre
				}
			}
			node = pre
			l.size--
		} else {
			node = node.pre
		}
	}
}
