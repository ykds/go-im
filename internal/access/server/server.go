package server

import (
	"context"
	"go-im/api/message"
	"go-im/api/user"
	"go-im/internal/access/config"
	"go-im/internal/access/types"
	"go-im/internal/common/middleware/mgrpc"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mkafka"
	"go-im/internal/pkg/utils"
	"strconv"
	"strings"

	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
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
	c      *config.Config
	ctx    context.Context
	cancel context.CancelFunc

	conns  map[int64]*Conn
	msgCh  chan *kafka.Message
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
		msgCh:      make(chan *kafka.Message, 1000),
		msgbox:     NewMsgBox(),
	}
	go ws.consume()
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

func (ws *WsServer) handleMsg() {
	for {
		select {
		case <-ws.ctx.Done():
			return
		case kafkaMsg := <-ws.msgCh:
			var msg types.Message
			switch kafkaMsg.Topic {
			case mkafka.MessageTopic:
				split := strings.Split(string(kafkaMsg.Key), "-")
				kind := split[0]
				toIdStr := split[1]
				toId, _ := strconv.ParseInt(toIdStr, 10, 64)
				commsg := types.Message{}
				err := commsg.Decode(kafkaMsg.Value)
				if err != nil {
					log.Errorf("decode msg failed, err: %v", err)
					continue
				}

				if kind == "group" {
					resp, err := ws.MessageRpc.ListGroupMember(context.Background(), &message.ListGroupMemberReq{
						GroupId: int64(toId),
						UserId:  commsg.FromId,
					})
					if err != nil {
						log.Errorf("list group member failed, err: %v", err)
						continue
					}
					msg = types.Message{
						Id:      commsg.Id,
						FromId:  commsg.FromId,
						ToId:    commsg.ToId,
						Seq:     commsg.Seq,
						Type:    mkafka.MessageMsg,
						Content: string(kafkaMsg.Value),
					}
					content := &types.NewMessageResp{
						Kind: kind,
						Seq:  commsg.Seq,
						ToId: commsg.ToId,
					}
					ws.m.Lock()
					for _, member := range resp.Members {
						if member.Id == msg.FromId {
							continue
						}
						msg.SessionId = member.SessionId
						content.SessionId = member.SessionId
						ws.msgbox.Append(&msg, kind, len(resp.Members))
						c, ok := ws.conns[member.Id]
						if ok {
							c.Send(&types.Message{
								Type:    mkafka.NewMessageMsg,
								Content: string(content.Encode()),
							})
						}
					}
					ws.m.Unlock()
				} else {
					msg = types.Message{
						Id:        commsg.Id,
						SessionId: commsg.SessionId,
						FromId:    commsg.FromId,
						ToId:      commsg.ToId,
						Seq:       commsg.Seq,
						Type:      mkafka.MessageMsg,
						Content:   string(kafkaMsg.Value),
					}
					content := &types.NewMessageResp{
						Kind:      kind,
						SessionId: commsg.SessionId,
						Seq:       commsg.Seq,
						ToId:      commsg.ToId,
					}
					ws.msgbox.Append(&msg, kind, 1)
					ws.m.Lock()
					c, ok := ws.conns[toId]
					ws.m.Unlock()
					if !ok {
						continue
					}
					c.Send(&types.Message{
						Type:    mkafka.NewMessageMsg,
						Content: string(content.Encode()),
					})
				}
			case mkafka.FriendApplyNotifyTopic:
				toId, _ := strconv.ParseInt(string(kafkaMsg.Key), 10, 64)
				msg = types.Message{
					Type:    mkafka.FriendNotifyMsg,
					Content: string(kafkaMsg.Value),
				}
				ws.m.Lock()
				c, ok := ws.conns[toId]
				ws.m.Unlock()
				if !ok {
					continue
				}
				// id, _ := strconv.ParseInt(string(kafkaMsg.Value), 10, 64)
				// c.friendApplyackQueue.Put(&types.Message{
				// 	Id: id,
				// })
				c.Send(&msg)
			case mkafka.GroupApplyNotifyTopic:
				toId, _ := strconv.ParseInt(string(kafkaMsg.Key), 10, 64)
				msg = types.Message{
					Type:    mkafka.GroupNotifyMsg,
					Content: string(kafkaMsg.Value),
				}
				ws.m.Lock()
				c, ok := ws.conns[toId]
				ws.m.Unlock()
				if !ok {
					continue
				}
				// id, _ := strconv.ParseInt(string(kafkaMsg.Value), 10, 64)
				// c.friendApplyackQueue.Put(&types.Message{
				// 	Id: id,
				// })
				c.Send(&msg)
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
			r := mkafka.NewGroupReader(ws.c.Kafka, group.Group, group.Topic)
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
					ws.Send(&m)
				}
			})
		}
	}
	if ws.c.Kafka.Topic != "" {
		r := mkafka.NewReader(ws.c.Kafka)
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
				ws.Send(&m)
			}
		})
	}
}

func (ws *WsServer) Stop() {
	ws.cancel()
}

func (ws *WsServer) Send(m *kafka.Message) {
	ws.msgCh <- m
}
