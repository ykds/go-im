package ackqueue

import (
	"context"
	"go-im/api/access"
	"go-im/internal/pkg/utils"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

var timepool *sync.Pool

func init() {
	timepool = &sync.Pool{
		New: func() any {
			return time.NewTimer(time.Second)
		},
	}
}

type node struct {
	ctx        context.Context
	cancel     context.CancelFunc
	id         int64
	msg        *access.Message
	next       *node
	pre        *node
	t          *time.Timer
	retryCount int
}

type AckQueue struct {
	head     *node
	tail     *node
	entryMap map[int64]*node

	retry   chan *access.Message
	timeout time.Duration
	isClose bool

	mutex *sync.Mutex
	cond  *sync.Cond

	ackId atomic.Int64
}

func NewAckQueue(timeout time.Duration, retry chan *access.Message) *AckQueue {
	a := &AckQueue{
		head:     &node{},
		entryMap: make(map[int64]*node, 1000),
		retry:    retry,
		timeout:  timeout,
		mutex:    &sync.Mutex{},
		cond:     &sync.Cond{},
	}
	a.ackId.Store(0)
	a.cond.L = a.mutex
	utils.SafeGo(func() {
		a.run()
	})
	return a
}

func (a *AckQueue) genAckId() int64 {
	i := a.ackId.Add(1)
	if i == math.MaxInt64 {
		if a.ackId.CompareAndSwap(i, 0) {
			return a.ackId.Add(1)
		} else {
			return a.genAckId()
		}
	}
	return i
}

func (a *AckQueue) Put(msg *access.Message) {
	a.cond.L.Lock()
	defer a.cond.L.Unlock()
	defer a.cond.Broadcast()

	if a.isClose {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	t := timepool.Get().(*time.Timer)
	t.Reset(a.timeout)
	ackId := a.genAckId()
	e := &node{
		ctx:    ctx,
		cancel: cancel,
		msg:    msg,
		t:      t,
		id:     ackId,
	}
	msg.AckId = ackId
	a.entryMap[ackId] = e
	if a.tail == nil {
		a.head.next = e
		e.pre = a.head
		a.tail = e
	} else {
		a.tail.next = e
		e.pre = a.tail
		a.tail = e
	}
}

func (a *AckQueue) run() {
	for {
		a.cond.L.Lock()
		if len(a.entryMap) == 0 && !a.isClose {
			a.cond.Wait()
		}

		if len(a.entryMap) == 0 {
			a.cond.L.Unlock()
			continue
		}
		n := a.head.next
		if n == a.tail {
			a.tail = nil
		}
		a.head.next = n.next
		if n.next != nil {
			n.next.pre = a.head
		}
		a.cond.L.Unlock()
		select {
		case <-n.ctx.Done():
			n.t.Stop()
			timepool.Put(n.t)
			delete(a.entryMap, n.id)
		case <-n.t.C:
			a.retry <- n.msg
			n.retryCount++
			if n.retryCount >= 3 {
				n.t.Stop()
				timepool.Put(n.t)
				delete(a.entryMap, n.id)
			} else {
				a.cond.L.Lock()
				n.t.Reset(a.timeout)
				a.tail.next = n
				n.pre = a.tail
				a.tail = n
				a.cond.L.Unlock()
			}
		}
	}

}

func (a *AckQueue) Ack(ackId int64) {
	a.cond.L.Lock()
	defer a.cond.L.Unlock()

	if a.isClose {
		return
	}
	e, ok := a.entryMap[ackId]
	if !ok {
		return
	}
	e.cancel()
	e.t.Stop()
	timepool.Put(e.t)
	e.pre.next = e.next
	if e.next != nil {
		e.next.pre = e.pre
	}
	if e == a.tail {
		a.tail = nil
	}
}

func (a *AckQueue) Close() {
	a.cond.L.Lock()
	defer a.cond.L.Unlock()
	a.isClose = true
	n := a.head.next
	for n != nil {
		n.cancel()
		n.t.Stop()
		timepool.Put(n.t)
	}
	a.entryMap = nil
}
