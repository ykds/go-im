package server

import (
	"context"
	"go-im/api/access"
	"go-im/api/message"
	"go-im/api/user"
	"go-im/internal/access/config"
	"go-im/internal/common/middleware/mgrpc"
	"go-im/internal/common/protocol"
	"go-im/internal/pkg/kafka"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mjson"
	"go-im/internal/pkg/utils"
	"strconv"

	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WsServer struct {
	access.UnimplementedAccessServer

	c      *config.Config
	ctx    context.Context
	cancel context.CancelFunc

	conns  map[int64]*Conn
	msgCh  chan *protocol.PushBody
	msgbox *MsgBox

	m sync.Mutex

	UserRpc    user.UserClient
	MessageRpc message.MessageClient
}

func NewServer(c *config.Config) *WsServer {
	userAddr := c.UserClient.ParseAddr()
	if userAddr == "" {
		panic("user service address is empty")
	}
	messageAddr := c.MessageClient.ParseAddr()
	if messageAddr == "" {
		panic("message service address is empty")
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if c.Trace.Enable {
		opts = append(opts, grpc.WithChainUnaryInterceptor(mgrpc.UnaryClientTrace()))
		opts = append(opts, grpc.WithChainStreamInterceptor(mgrpc.StreamClientTrace()))
	}
	userConn, err := grpc.NewClient(userAddr, opts...)
	if err != nil {
		panic(err)
	}
	messageConn, err := grpc.NewClient(messageAddr, opts...)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ws := &WsServer{
		c:          c,
		ctx:        ctx,
		cancel:     cancel,
		UserRpc:    user.NewUserClient(userConn),
		MessageRpc: message.NewMessageClient(messageConn),
		conns:      make(map[int64]*Conn, 1000),
		m:          sync.Mutex{},
		msgCh:      make(chan *protocol.PushBody, 1000),
		msgbox:     NewMsgBox(),
	}
	if c.Kafka.Enable {
		go ws.consume()
	}
	go ws.handleMsg()
	return ws
}

func (ws *WsServer) Handler(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Errorf("upgrade failed, %v", err)
		return
	}

	uid, ok := ctx.Get("user_id")
	if !ok {
		conn.Close()
		return
	}
	userId := uid.(int64)

	c := newConn(ws, userId, conn)
	ws.m.Lock()
	ws.conns[userId] = c
	ws.m.Unlock()
	_, err = ws.UserRpc.Connect(ws.ctx, &user.ConnectReq{UserId: userId})
	if err != nil {
		log.Errorf("Connect failed, %v", err)
		conn.Close()
		ws.m.Lock()
		delete(ws.conns, userId)
		ws.m.Unlock()
		return
	}
	c.run()
}

func (ws *WsServer) PushMessage(ctx context.Context, in *access.PushMessageReq) (*access.PushMessageResp, error) {
	ws.Send(&protocol.PushBody{
		Type: in.Type,
		Key:  in.Key,
		Body: in.Body,
	})
	return nil, nil
}

func (ws *WsServer) handleMsg() {
	for {
		select {
		case <-ws.ctx.Done():
			return
		case pushBody := <-ws.msgCh:
			var msg access.Message
			switch pushBody.Type {
			case protocol.MessageTopic:
				msgBody := access.MessageBody{}
				err := mjson.Unmarshal(pushBody.Body, &msgBody)
				if err != nil {
					log.Errorf("decode msg failed, err: %v", err)
					continue
				}

				switch msgBody.Kind {
				case "group":
					resp, err := ws.MessageRpc.ListGroupMember(context.Background(), &message.ListGroupMemberReq{
						GroupId: msgBody.ToId,
						UserId:  msgBody.FromId,
					})
					if err != nil {
						log.Errorf("list group member failed, err: %v", err)
						continue
					}
					content := &access.NewMessageNotifyMsg{
						Kind: msgBody.Kind,
						Seq:  msgBody.Seq,
					}
					msg = access.Message{
						Type: int64(protocol.MessageMsg),
						Data: string(pushBody.Body),
					}
					ws.m.Lock()
					for _, member := range resp.Members {
						if member.Id == msgBody.FromId {
							continue
						}
						content.SessionId = member.SessionId
						ws.msgbox.Append(&msg, &msgBody, len(resp.Members))
						c, ok := ws.conns[member.Id]
						if ok {
							b, _ := mjson.Marshal(content)
							c.Send(&access.Message{
								Type: int64(protocol.NewMessageMsg),
								Data: string(b),
							})
						}
					}
					ws.m.Unlock()
				case "single":
					msg = access.Message{
						Type: int64(protocol.MessageMsg),
						Data: string(pushBody.Body),
					}
					content := &access.NewMessageNotifyMsg{
						Kind:      msgBody.Kind,
						SessionId: msgBody.SessionId,
						Seq:       msgBody.Seq,
					}
					ws.msgbox.Append(&msg, &msgBody, 1)
					ws.m.Lock()
					c, ok := ws.conns[msgBody.ToId]
					ws.m.Unlock()
					if !ok {
						continue
					}

					b, _ := mjson.Marshal(content)
					msg := &access.Message{
						Type: int64(protocol.NewMessageMsg),
						Data: string(b),
					}
					c.ackQueue.Put(msg)
					c.Send(msg)
				}
			case protocol.FriendEventTopic:
				contentType, _ := strconv.Atoi(string(pushBody.Key))
				msg = access.Message{
					Type: int64(contentType),
					Data: string(pushBody.Body),
				}
				switch contentType {
				case protocol.FriendApplyMsg:
					body := access.FriendApplyMsg{}
					err := mjson.Unmarshal(pushBody.Body, &body)
					if err != nil {
						log.Errorf("unmarshal friend notify msg failed, %v", err)
						continue
					}
					ws.m.Lock()
					c, ok := ws.conns[body.UserId]
					ws.m.Unlock()
					if !ok {
						continue
					}
					c.ackQueue.Put(&msg)
					c.Send(&msg)
				case protocol.FriendApplyResultMsg:
					body := access.FriendApplyResponseMsg{}
					err := mjson.Unmarshal(pushBody.Body, &body)
					if err != nil {
						log.Errorf("unmarshal friend notify msg failed, %v", err)
						continue
					}
					ws.m.Lock()
					c, ok := ws.conns[body.UserId]
					ws.m.Unlock()
					if !ok {
						continue
					}
					c.ackQueue.Put(&msg)
					c.Send(&msg)
				case protocol.FriendInfoUpdatedMsg:
					body := access.FriendUpdatedInfoMsg{}
					err := mjson.Unmarshal(pushBody.Body, &body)
					if err != nil {
						log.Errorf("unmarshal friend notify msg failed, %v", err)
						continue
					}
					to := make([]*Conn, 0, len(body.ToId))
					ws.m.Lock()
					for _, v := range body.ToId {
						c, ok := ws.conns[v]
						if ok {
							to = append(to, c)
						}
					}
					ws.m.Unlock()
					for _, c := range to {
						c.ackQueue.Put(&msg)
						c.Send(&msg)
					}
				}
			case protocol.GroupEventTopic:
				contentType, _ := strconv.Atoi(string(pushBody.Key))
				msg = access.Message{
					Type: int64(contentType),
					Data: string(pushBody.Body),
				}
				switch contentType {
				case protocol.GroupApplyMsg:
					body := access.GroupApplyMsg{}
					err := mjson.Unmarshal(pushBody.Body, &body)
					if err != nil {
						log.Errorf("unmarshal friend notify msg failed, %v", err)
						continue
					}
					ws.m.Lock()
					c, ok := ws.conns[body.UserId]
					ws.m.Unlock()
					if !ok {
						continue
					}
					c.ackQueue.Put(&msg)
					c.Send(&msg)
				case protocol.GroupAppluResultMsg:
					body := access.GroupApplyResponseMsg{}
					err := mjson.Unmarshal(pushBody.Body, &body)
					if err != nil {
						log.Errorf("unmarshal friend notify msg failed, %v", err)
						continue
					}
					ws.m.Lock()
					c, ok := ws.conns[body.UserId]
					ws.m.Unlock()
					if !ok {
						continue
					}
					c.ackQueue.Put(&msg)
					c.Send(&msg)
				case protocol.GroupInfoUpdatedMsg:
					body := access.GroupUpdatedInfoMsg{}
					err := mjson.Unmarshal(pushBody.Body, &body)
					if err != nil {
						log.Errorf("unmarshal friend notify msg failed, %v", err)
						continue
					}
					to := make([]*Conn, 0, len(body.ToId))
					ws.m.Lock()
					for _, v := range body.ToId {
						c, ok := ws.conns[v]
						if ok {
							to = append(to, c)
						}
					}
					ws.m.Unlock()
					for _, c := range to {
						c.ackQueue.Put(&msg)
						c.Send(&msg)
					}
				}
			default:
				continue
			}
		}
	}
}

func (ws *WsServer) consume() {
	if len(ws.c.Kafka.Brokers) == 0 {
		return
	}
	if len(ws.c.Kafka.ConsumerGroup) > 0 {
		for _, group := range ws.c.Kafka.ConsumerGroup {
			r := kafka.NewGroupReader(ws.c.Kafka, group.Group, group.Topic)
			utils.SafeGo(func() {
				defer r.Close()
				for {
					select {
					case <-ws.ctx.Done():
						return
					default:
					}
					m, err := r.ReadMessage(context.Background())
					if err != nil {
						log.Errorf("fetch message failed, %v", err)
						continue
					}
					body := &protocol.PushBody{
						Type: m.Topic,
						Body: m.Value,
					}
					ws.Send(body)
				}
			})
		}
	}
	if ws.c.Kafka.Topic != "" {
		r := kafka.NewReader(ws.c.Kafka)
		utils.SafeGo(func() {
			defer r.Close()
			for {
				select {
				case <-ws.ctx.Done():
					return
				default:
				}
				m, err := r.ReadMessage(context.Background())
				if err != nil {
					log.Errorf("fetch message failed, %v", err)
					continue
				}
				body := &protocol.PushBody{
					Type: m.Topic,
					Body: m.Value,
				}
				ws.Send(body)
			}
		})
	}
}

func (ws *WsServer) Stop() {
	ws.cancel()
}

func (ws *WsServer) Send(body *protocol.PushBody) {
	ws.msgCh <- body
}
