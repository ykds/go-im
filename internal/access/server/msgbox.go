package server

import (
	"fmt"
	"go-im/api/access"
	"go-im/internal/access/pkg/msglist"
	"hash/crc32"
	"sync"
)

type bucket struct {
	entries map[string]*msglist.MsgList
	rwmutex *sync.RWMutex
}

func (b *bucket) Insert(key string, msg *access.Message, msgBody *access.MessageBody, unread int) {
	b.rwmutex.Lock()
	list, ok := b.entries[key]
	if !ok {
		list = msglist.NewMsgList()
		b.entries[key] = list
	}
	b.rwmutex.Unlock()
	list.Insert(msg, msgBody, unread)
}

func (b *bucket) Ack(key string, seq int64) {
	b.rwmutex.RLock()
	list, ok := b.entries[key]
	if !ok {
		b.rwmutex.RUnlock()
		return
	}
	b.rwmutex.RUnlock()
	list.AckMsg(seq)
}

func (b *bucket) List(key string, seq int64) []*access.Message {
	b.rwmutex.RLock()
	list, ok := b.entries[key]
	if !ok {
		b.rwmutex.RUnlock()
		return nil
	}
	b.rwmutex.RUnlock()
	return list.List(seq)
}

type MsgBox struct {
	box []*bucket
	rwm *sync.RWMutex
}

func NewMsgBox() *MsgBox {
	return &MsgBox{
		box: make([]*bucket, 1000),
		rwm: &sync.RWMutex{},
	}
}

func (mb *MsgBox) List(kind string, sessionId, seq int64) []*access.Message {
	k := key(kind, sessionId)
	index := hash(k)
	i := index % len(mb.box)
	mb.rwm.RLock()
	btk := mb.box[i]
	if btk == nil {
		mb.rwm.RUnlock()
		return nil
	}
	mb.rwm.RUnlock()
	return btk.List(k, seq)
}

func (mb *MsgBox) Append(msg *access.Message, msgBody *access.MessageBody, unread int) {
	k := key(msgBody.Kind, msgBody.SessionId)
	index := hash(k)
	i := index % len(mb.box)
	mb.rwm.RLock()
	btk := mb.box[i]
	if btk == nil {
		mb.rwm.RUnlock()

		mb.rwm.Lock()
		btk = &bucket{
			entries: make(map[string]*msglist.MsgList, 1000),
			rwmutex: &sync.RWMutex{},
		}
		mb.box[i] = btk
		mb.rwm.Unlock()
	} else {
		mb.rwm.RUnlock()
	}
	btk.Insert(k, msg, msgBody, unread)
}

func (mb *MsgBox) Ack(kind string, sessionId int64, seq int64) {
	k := key(kind, sessionId)
	index := hash(k)
	i := index % len(mb.box)

	mb.rwm.RLock()
	btk := mb.box[i]
	mb.rwm.RUnlock()
	if btk == nil {
		return
	}
	btk.Ack(k, seq)
}

func key(kind string, sessionId int64) string {
	return fmt.Sprintf("box-%s:%d", kind, sessionId)
}

func hash(key string) int {
	return int(crc32.ChecksumIEEE([]byte(key)))
}
