package ackqueue

import (
	"context"
	"go-im/internal/access/types"
	"go-im/internal/pkg/utils"
	"sync"
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
	msg        *types.Message
	next       *node
	pre        *node
	t          *time.Timer
	retryCount int
}

type AckQueue struct {
	head     *node
	tail     *node
	entryMap map[int64]*node

	retry   chan *types.Message
	timeout time.Duration
	isClose bool

	mutex *sync.Mutex
	cond  *sync.Cond
}

func NewAckQueue(timeout time.Duration, retry chan *types.Message) *AckQueue {
	a := &AckQueue{
		head:     &node{},
		entryMap: make(map[int64]*node, 1000),
		retry:    retry,
		timeout:  timeout,
		mutex:    &sync.Mutex{},
		cond:     &sync.Cond{},
	}
	a.cond.L = a.mutex
	utils.SafeGo(func() {
		a.run()
	})
	return a
}

func (a *AckQueue) Put(msg *types.Message) {
	a.cond.L.Lock()
	defer a.cond.L.Unlock()
	defer a.cond.Broadcast()

	if a.isClose {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	t := timepool.Get().(*time.Timer)
	t.Reset(a.timeout)
	e := &node{
		ctx:    ctx,
		cancel: cancel,
		msg:    msg,
		t:      t,
	}
	a.entryMap[e.msg.Id] = e
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
			delete(a.entryMap, n.msg.Id)
		case <-n.t.C:
			a.retry <- n.msg
			n.retryCount++
			if n.retryCount >= 3 {
				n.t.Stop()
				timepool.Put(n.t)
				delete(a.entryMap, n.msg.Id)
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

func (a *AckQueue) Ack(id int64) {
	a.cond.L.Lock()
	defer a.cond.L.Unlock()

	if a.isClose {
		return
	}
	e, ok := a.entryMap[id]
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
