package server

import (
	"context"
	"go-im/api/access"
	"go-im/api/message"
	"go-im/api/user"
	"go-im/internal/access/pkg/ackqueue"
	"go-im/internal/common/mkafka"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mjson"
	"go-im/internal/pkg/utils"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ReSendMsg struct {
	Bytes []byte
	Count int
}

type Conn struct {
	ctx    context.Context
	cancel context.CancelFunc
	userId int64
	*websocket.Conn
	wrch          chan []byte
	svc           *WsServer
	hb            chan struct{}
	retry         chan *access.Message
	ackQueue      ackqueue.AckQueue
	unackMsg      map[int64]*ReSendMsg
	unackMsgMutex *sync.Mutex
	acked         int64
	pollMutext    *sync.Mutex
}

func newConn(svc *WsServer, userId int64, conn *websocket.Conn) *Conn {
	retry := make(chan *access.Message, 512)
	ctx, cancel := context.WithCancel(svc.ctx)
	return &Conn{
		ctx:           ctx,
		cancel:        cancel,
		userId:        userId,
		Conn:          conn,
		wrch:          make(chan []byte, 1000),
		svc:           svc,
		hb:            make(chan struct{}, 1),
		ackQueue:      *ackqueue.NewAckQueue(100, retry),
		unackMsg:      make(map[int64]*ReSendMsg, 1000),
		unackMsgMutex: &sync.Mutex{},
		retry:         retry,
		pollMutext:    &sync.Mutex{},
	}
}

func (c *Conn) run() {
	utils.SafeGo(func() {
		c.read()
	})
	utils.SafeGo(func() {
		c.write()
	})
	utils.SafeGo(func() {
		c.heartbeat()
	})
	utils.SafeGo(func() {
		c.reSend()
	})
}

func (c *Conn) read() {
	defer c.Close()
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		mt, p, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		switch mt {
		case websocket.TextMessage:
			msg := access.Message{}
			err := mjson.Unmarshal(p, &msg)
			if err != nil {
				log.Errorf("decode message failed, %v", err)
				return
			}
			err = c.dealMessage(&msg)
			if err != nil {
				log.Errorf("deal message failed, %v", err)
			}
		}
	}
}

func (c *Conn) dealMessage(msg *access.Message) error {
	switch int(msg.Type) {
	case mkafka.AckMsg:
		ack := &access.AckMessage{}
		err := mjson.Unmarshal([]byte(msg.Data), ack)
		if err != nil {
			return err
		}
		switch int(ack.Type) {
		case mkafka.FriendApplyMsg:
			fallthrough
		case mkafka.FriendApplyResultMsg:
			fallthrough
		case mkafka.FriendInfoUpdatedMsg:
			fallthrough
		case mkafka.GroupApplyMsg:
			fallthrough
		case mkafka.GroupAppluResultMsg:
			fallthrough
		case mkafka.GroupInfoUpdatedMsg:
			fallthrough
		case mkafka.NewMessageMsg:
			if ack.AckId != nil {
				c.ackQueue.Ack(*ack.AckId)
			}
		case mkafka.MessageMsg:
			if ack.Kind != nil && ack.Id != nil && ack.Seq != nil {
				c.pollMutext.Lock()
				if *ack.Seq > c.acked {
					c.svc.msgbox.Ack(*ack.Kind, *ack.Id, *ack.Seq)
					c.acked = *ack.Seq
				}
				c.pollMutext.Unlock()
			}
			_, err = c.svc.MessageRpc.AckMessage(context.Background(), &message.AckMessageReq{
				SessionId: *ack.Id,
				Seq:       *ack.Seq,
			})
			return err
		}
	case mkafka.HeartbeatMsg:
		_, err := c.svc.UserRpc.Heartbeat(context.Background(), &user.HeartBeatReq{UserId: c.userId})
		if err != nil {
			return err
		}
		c.hb <- struct{}{}
		return nil
	case mkafka.MessageMsg:
		req := &access.PollMessageReq{}
		err := mjson.Unmarshal([]byte(msg.Data), req)
		if err != nil {
			c.pollMutext.Unlock()
			return err
		}
		msgs := c.svc.msgbox.List(req.Kind, req.SessionId, req.Seq)
		b, _ := mjson.Marshal(msgs)
		resp := &access.Message{
			Type: int64(mkafka.MessageMsg),
			Data: string(b),
		}
		c.Send(resp)
	}
	return nil
}

func (c *Conn) write() {
	defer c.Close()
	for b := range c.wrch {
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		err := c.Conn.WriteMessage(websocket.TextMessage, b)
		if err != nil {
			log.Errorf("[conn:%d]write message failed, %v", c.userId, err)
			return
		}
	}
}

func (c *Conn) heartbeat() {
	t := time.NewTimer(time.Second * 60)
	defer func() {
		c.Close()
		t.Stop()
	}()
	for {
		select {
		case <-c.hb:
			t.Reset(time.Second * 60)
		case <-t.C:
			return
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Conn) reSend() {
	for msg := range c.retry {
		b, _ := mjson.Marshal(msg)
		c.wrch <- b
	}
}

func (c *Conn) Close() {
	c.cancel()
	c.Conn.Close()
	c.ackQueue.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c.svc.UserRpc.DisConnect(ctx, &user.DisConnectReq{UserId: c.userId})
	c.svc.m.Lock()
	delete(c.svc.conns, c.userId)
	c.svc.m.Unlock()
}

func (c *Conn) Send(msg *access.Message) {
	b, _ := mjson.Marshal(msg)
	c.wrch <- b
}
