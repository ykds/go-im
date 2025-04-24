package server

import (
	"context"
	"encoding/json"
	"go-im/api/message"
	"go-im/api/user"
	"go-im/internal/access/pkg/ackqueue"
	"go-im/internal/access/types"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mkafka"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	ctx    context.Context
	cancel context.CancelFunc
	userId int64
	*websocket.Conn
	wrch                chan []byte
	svc                 *WsServer
	hb                  chan struct{}
	retry               chan *types.Message
	friendApplyackQueue ackqueue.AckQueue
	unackMsg            map[int64]*types.ReSendMsg
	unackMsgMutex       *sync.Mutex
}

func newConn(svc *WsServer, userId int64, conn *websocket.Conn) *Conn {
	retry := make(chan *types.Message, 512)
	ctx, cancel := context.WithCancel(svc.ctx)
	return &Conn{
		ctx:                 ctx,
		cancel:              cancel,
		userId:              userId,
		Conn:                conn,
		wrch:                make(chan []byte, 1000),
		svc:                 svc,
		hb:                  make(chan struct{}, 1),
		friendApplyackQueue: *ackqueue.NewAckQueue(100, retry),
		unackMsg:            make(map[int64]*types.ReSendMsg, 1000),
		unackMsgMutex:       &sync.Mutex{},
		retry:               retry,
	}
}

func (c *Conn) run() {
	go c.read()
	go c.write()
	go c.heartbeat()
	go c.reSend()
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
			msg := types.Message{}
			err := msg.Decode(p)
			if err != nil {
				log.Errorf("decode message failed, %v", err)
				return
			}
			err = c.dealMessage(msg)
			if err != nil {
				log.Errorf("deal message failed, %v", err)
			}
		}
	}
}

func (c *Conn) dealMessage(msg types.Message) error {
	switch msg.Type {
	case mkafka.AckMsg:
		ack := &types.AckMessage{}
		err := ack.Decode([]byte(msg.Content))
		if err != nil {
			return err
		}
		switch ack.Type {
		case mkafka.FriendApplyMsg:
			// c.friendApplyackQueue.Ack(ack.Id)
		case mkafka.FriendApplyResultMsg: // TODO
		case mkafka.GroupApplyMsg:
		case mkafka.GroupAppluResultMsg:
		case mkafka.MessageMsg:
			c.svc.msgbox.Ack(ack.Kind, ack.SessionId, ack.Seq)
			_, err = c.svc.MessageRpc.AckMessage(context.Background(), &message.AckMessageReq{
				SessionId: ack.SessionId,
				Seq:       ack.Seq,
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
		req := &types.PollMessageReq{}
		err := req.Decode([]byte(msg.Content))
		if err != nil {
			return err
		}
		msgs := c.svc.msgbox.List(req.Kind, req.SessionId, req.Seq)
		// TODO if msgs is empty, pull from message server
		b, _ := json.Marshal(msgs)
		resp := &types.Message{
			Type:    mkafka.MessageMsg,
			Content: string(b),
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
		c.wrch <- msg.Encode()
	}
}

func (c *Conn) Close() {
	c.cancel()
	c.Conn.Close()
	c.friendApplyackQueue.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c.svc.UserRpc.DisConnect(ctx, &user.DisConnectReq{UserId: c.userId})
	c.svc.m.Lock()
	delete(c.svc.conns, c.userId)
	c.svc.m.Unlock()
}

func (c *Conn) Send(msg *types.Message) {
	c.wrch <- msg.Encode()
}
