package logic

import (
	"go-im/api/user"
	"go-im/internal/common/errcode"
	"go-im/internal/common/middleware/mhttp"
	"go-im/internal/common/response"
	"go-im/internal/gateway/server"
	"go-im/internal/gateway/types"

	"github.com/gin-gonic/gin"
)

type FriendApi struct {
	s *server.Server
}

func NewFriendApi(s *server.Server) *FriendApi {
	return &FriendApi{s}
}

func (api *FriendApi) RegisterRouter(engine *gin.RouterGroup) {
	friend := engine.Group("/friends", mhttp.AuthMiddleware())
	{
		friend.DELETE("", api.DeleteFriend)
		friend.GET("", api.ListFriend)
		friend.GET("/apply", api.ListApply)
		friend.POST("/apply", api.FriendApply)
		friend.PUT("/apply", api.HandleApply)
	}
}

func (api *FriendApi) DeleteFriend(c *gin.Context) {
	var (
		req types.DeleteFriendReq
		err error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, nil)
		}
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	_, err = api.s.UserRpc.DeleteFriend(c.Request.Context(), &user.DeleteFriendReq{
		UserId:   c.GetInt64("user_id"),
		FriendId: req.FriendId,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *FriendApi) FriendApply(c *gin.Context) {
	var (
		req types.FriendApplyReq
		err error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, nil)
		}
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	_, err = api.s.UserRpc.FriendApply(c.Request.Context(), &user.FriendApplyReq{UserId: c.GetInt64("user_id"), FriendId: req.FriendId})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *FriendApi) HandleApply(c *gin.Context) {
	var (
		req types.HandleApplyReq
		err error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, nil)
		}
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	_, err = api.s.UserRpc.HandleApply(c.Request.Context(), &user.HandleApplyReq{ApplyId: req.ApplyId, UserId: c.GetInt64("user_id"), Status: int32(req.Status)})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *FriendApi) ListApply(c *gin.Context) {
	var (
		resp types.ListApplyResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, resp)
		}
	}()
	rpcResp, err := api.s.UserRpc.ListApply(c.Request.Context(), &user.ListApplyReq{UserId: c.GetInt64("user_id")})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	applyResp := make([]types.ApplyInfo, 0)
	for _, apply := range rpcResp.List {
		applyResp = append(applyResp, types.ApplyInfo{
			ApplyId:  int64(apply.ApplyId),
			UserId:   int64(apply.FriendId),
			Username: apply.Username,
			Avatar:   apply.Avatar,
		})
	}
	resp = types.ListApplyResp{
		List: applyResp,
	}
}

func (api *FriendApi) ListFriend(c *gin.Context) {
	var (
		resp types.ListFriendsResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, resp)
		}
	}()
	rpcResp, err := api.s.UserRpc.ListFriends(c.Request.Context(), &user.ListFriendsReq{UserId: c.GetInt64("user_id")})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
	friendInfo := make([]types.FriendInfo, 0)
	for _, friend := range rpcResp.List {
		friendInfo = append(friendInfo, types.FriendInfo{
			UserId:   int64(friend.UserId),
			Username: friend.Username,
			Avatar:   friend.Avatar,
		})
	}
	resp = types.ListFriendsResp{
		List: friendInfo,
	}
}
