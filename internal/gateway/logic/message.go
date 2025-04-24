package logic

import (
	"go-im/api/message"
	"go-im/internal/common/errcode"
	"go-im/internal/common/middleware/mhttp"
	"go-im/internal/common/response"
	"go-im/internal/gateway/server"
	"go-im/internal/gateway/types"

	"github.com/gin-gonic/gin"
)

type MessageApi struct {
	s *server.Server
}

func NewMessageApi(s *server.Server) *MessageApi {
	return &MessageApi{s}
}

func (api *MessageApi) RegisterRouter(engine *gin.RouterGroup) {
	msg := engine.Group("/message", mhttp.AuthMiddleware())
	{
		msg.POST("", api.SendMessage)
		msg.PUT("", api.AckMessage)
		msg.GET("/session", api.ListSession)
		msg.POST("/session", api.CreateSession)
		msg.GET("/unread", api.UnreadMessage)
	}
}

func (api *MessageApi) AckMessage(c *gin.Context) {
	var (
		req types.AckMessageReq
		err error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(nil)
		}
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	_, err = api.s.MessageRpc.AckMessage(c, &message.AckMessageReq{
		UserId:    c.GetInt64("user_id"),
		SessionId: req.SessionId,
		Seq:       req.Seq,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *MessageApi) CreateSession(c *gin.Context) {
	var (
		req  types.CreateSessionReq
		resp types.CreateSessionResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(resp)
		}
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	rpcResp, err := api.s.MessageRpc.CreateSession(c, &message.CreateSessionReq{
		UserId:   c.GetInt64("user_id"),
		FriendId: req.FriendId,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	resp = types.CreateSessionResp{
		SessionId: rpcResp.SessionId,
	}
}

func (api *MessageApi) ListSession(c *gin.Context) {
	var (
		resp types.ListSessionResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(resp)
		}
	}()
	rpcResp, err := api.s.MessageRpc.ListSession(c, &message.ListSessionReq{
		UserId: c.GetInt64("user_id"),
	})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	infos := make([]types.SessionInfo, 0, len(rpcResp.List))
	for _, item := range rpcResp.List {
		si := types.SessionInfo{
			SessionId: item.SessionId,
			Kind:      item.Kind,
			Seq:       item.Seq,
		}
		if item.GroupId != nil {
			si.GroupId = *item.GroupId
		}
		if item.GroupName != nil {
			si.GroupName = *item.GroupName
		}
		if item.GroupAvatar != nil {
			si.GroupAvatar = *item.GroupAvatar
		}
		if item.FriendId != nil {
			si.FriendId = *item.FriendId
		}
		if item.FriendName != nil {
			si.FrienName = *item.FriendName
		}
		if item.FriendAvatar != nil {
			si.FriendAvatar = *item.FriendAvatar
		}
		infos = append(infos, si)
	}
	resp = types.ListSessionResp{List: infos}
}

func (api *MessageApi) SendMessage(c *gin.Context) {
	var (
		req types.SendMessageReq
		err error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(nil)
		}
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	_, err = api.s.MessageRpc.SendMessage(c, &message.SendMessageReq{
		UserId:  c.GetInt64("user_id"),
		Kind:    req.Kind,
		ToId:    req.ToId,
		Message: req.Message,
		Seq:     req.Seq,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *MessageApi) UnreadMessage(c *gin.Context) {
	var (
		req  types.ListUnReadMessageReq
		resp types.ListUnReadMessageResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(resp)
		}
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	respRpc, err := api.s.MessageRpc.ListUnReadMessage(c, &message.ListUnReadMessageReq{
		UserId:  c.GetInt64("user_id"),
		GroupId: req.GroupId,
		FromId:  req.FromId,
		Seq:     req.Seq,
		Kind:    req.Kind,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	infos := make([]types.MessageInfo, 0, len(respRpc.List))
	for _, item := range respRpc.List {
		infos = append(infos, types.MessageInfo{
			Id:      item.Id,
			Content: item.Content,
			Seq:     item.Seq,
			Kind:    item.Kind,
			FromId:  item.FromId,
		})
	}
	resp = types.ListUnReadMessageResp{
		List: infos,
	}
}
