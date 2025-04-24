package msglist

import (
	"go-im/internal/access/types"
	"sync"
)

type node struct {
	Content *types.Message
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

func (l *MsgList) List(seq int64) []*types.Message {
	head := l.head.next
	for head != nil && head.Content.Seq < seq {
		head = head.next
	}
	result := make([]*types.Message, 0, 10)
	for head != nil {
		result = append(result, head.Content)
		head = head.next
	}
	return result
}

func (l *MsgList) Insert(msg *types.Message, unread int) error {
	l.m.Lock()
	defer l.m.Unlock()

	newNode := &node{
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
	l.loc[msg.Seq] = newNode
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
