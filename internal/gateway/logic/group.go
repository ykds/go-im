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

type GroupApi struct {
	s *server.Server
}

func NewGroupApi(s *server.Server) *GroupApi {
	return &GroupApi{s}
}

func (api *GroupApi) RegisterRouter(engine *gin.RouterGroup) {
	group := engine.Group("/group", mhttp.AuthMiddleware())
	{
		group.POST("", api.CreateGroup)
		group.GET("", api.ListGroup)
		group.DELETE("", api.DismissGroup)
		group.PUT("", api.UpdateGroupInfo)
		group.POST("/apply", api.ApplyInGroup)
		group.GET("/apply", api.ListGroupApply)
		group.POST("/exit", api.ExitGroup)
		group.PUT("/handle/apply", api.HandlerGroupApply)
		group.PUT("/member/invite", api.InviteMember)
		group.PUT("/member/move-out", api.MoveOutMember)
		group.GET("/members", api.ListGroupMember)
		group.GET("/search", api.SearchGroup)
	}
}

func (api *GroupApi) ApplyInGroup(c *gin.Context) {
	var (
		req types.ApplyInGroupReq
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
	_, err = api.s.MessageRpc.ApplyInGroup(c, &message.ApplyInGroupReq{
		GroupNo: req.GroupNo,
		UserId:  c.GetInt64("user_id"),
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *GroupApi) CreateGroup(c *gin.Context) {
	var (
		req  types.CreateGroupReq
		resp types.CreateGroupResq
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, resp)
		}
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	rpcResp, err := api.s.MessageRpc.CreateGroup(c, &message.CreateGroupReq{
		UserId: c.GetInt64("user_id"),
		Name:   req.Name,
		Avatar: req.Avatar,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	resp = types.CreateGroupResq{}
	resp.Id = rpcResp.Id
	resp.Name = rpcResp.Name
	resp.GroupNo = rpcResp.GroupNo
	resp.Avatar = rpcResp.Avatar
}

func (api *GroupApi) DismissGroup(c *gin.Context) {
	var (
		req types.DismissGroupReq
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
	_, err = api.s.MessageRpc.DismissGroup(c, &message.DismissGroupReq{
		UserId:  c.GetInt64("user_id"),
		GroupId: req.GroupId,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *GroupApi) ExitGroup(c *gin.Context) {
	var (
		req types.ExitGroupReq
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
	_, err = api.s.MessageRpc.ExitGroup(c, &message.ExitGroupReq{
		GroupId: req.GroupId,
		UserId:  c.GetInt64("user_id"),
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *GroupApi) HandlerGroupApply(c *gin.Context) {
	var (
		req types.HandleGroupApplyReq
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
	_, err = api.s.MessageRpc.HandleGroupApply(c, &message.HandleGroupApplyReq{
		ApplyId: req.ApplyId,
		UserId:  c.GetInt64("user_id"),
		Status:  req.Status,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *GroupApi) InviteMember(c *gin.Context) {
	var (
		req types.InviteMemberReq
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
	_, err = api.s.MessageRpc.InviteMember(c, &message.InviteMemberReq{
		GroupId:   req.GroupId,
		InvitedId: req.InvitedId,
		UserId:    c.GetInt64("user_id"),
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *GroupApi) ListGroupApply(c *gin.Context) {
	var (
		resp types.ListGroupApplyResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, resp)
		}
	}()
	rpcResp, err := api.s.MessageRpc.ListGroupApply(c, &message.ListGroupApplyReq{
		UserId: c.GetInt64("user_id"),
	})
	if err != nil {
		err = errcode.ToRpcError(err)
		return
	}
	resp = types.ListGroupApplyResp{}
	for _, item := range rpcResp.List {
		ag := types.ApplyGroup{
			Name:   item.Name,
			Avatar: item.Avatar,
		}
		for _, apply := range item.Apply {
			ag.Apply = append(ag.Apply, types.UserApply{
				ApplyId: apply.ApplyId,
				Name:    apply.Name,
				Avatar:  apply.Avatar,
			})
		}
		resp.List = append(resp.List, ag)
	}
}

func (api *GroupApi) ListGroup(c *gin.Context) {
	var (
		resp types.ListGroupResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, resp)
		}
	}()
	rpcResp, err := api.s.MessageRpc.ListGroup(c, &message.ListGroupReq{
		UserId: c.GetInt64("user_id"),
	})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	infos := make([]types.GroupInfo, 0, len(rpcResp.Groups))
	for _, group := range rpcResp.Groups {
		members := make([]types.GroupMember, 0, len(group.Members))
		for _, mem := range group.Members {
			members = append(members, types.GroupMember{
				Id:     mem.Id,
				Name:   mem.Name,
				Avatar: mem.Avatar,
			})
		}
		infos = append(infos, types.GroupInfo{
			Id:      group.Id,
			GroupNo: group.GroupNo,
			Name:    group.Name,
			Avatar:  group.Avatar,
			Members: members,
		})
	}
	resp = types.ListGroupResp{
		Groups: infos,
	}
}

func (api *GroupApi) ListGroupMember(c *gin.Context) {
	var (
		req  types.ListGroupMemberReq
		resp types.ListGroupMemberResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, resp)
		}
	}()
	if err = c.BindQuery(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	rpcResp, err := api.s.MessageRpc.ListGroupMember(c, &message.ListGroupMemberReq{
		GroupId: req.GroupId,
		UserId:  c.GetInt64("user_id"),
	})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	members := make([]types.GroupMember, 0, len(rpcResp.Members))
	for _, member := range rpcResp.Members {
		members = append(members, types.GroupMember{
			Id:     member.Id,
			Name:   member.Name,
			Avatar: member.Avatar,
		})
	}
	resp = types.ListGroupMemberResp{
		Members: members,
	}
}

func (api *GroupApi) MoveOutMember(c *gin.Context) {
	var (
		req types.MoveOutMemberReq
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
	_, err = api.s.MessageRpc.MoveOutMember(c, &message.MoveOutMemberReq{
		GroupId: req.GroupId,
		UserId:  c.GetInt64("user_id"),
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *GroupApi) SearchGroup(c *gin.Context) {
	var (
		req  types.SearchGroupReq
		resp types.SearchGroupResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, resp)
		}
	}()
	if err = c.BindQuery(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	rpcResp, err := api.s.MessageRpc.SearchGroup(c, &message.SearchGroupReq{
		GroupNo: req.GroupNo,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	resp = types.SearchGroupResp{
		Infos: make([]types.SearchGroupInfo, 0),
	}
	for _, info := range rpcResp.Infos {
		resp.Infos = append(resp.Infos, types.SearchGroupInfo{
			Id:          info.Id,
			GroupNo:     info.GroupNo,
			Name:        info.Name,
			Avatar:      info.Avatar,
			MemberCount: info.MemberCount,
			OwnerId:     info.OwnerId,
		})
	}
}

func (api *GroupApi) UpdateGroupInfo(c *gin.Context) {
	var (
		req types.UpdateGroupInfoReq
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
	_, err = api.s.MessageRpc.UpdateGroupInfo(c, &message.UpdateGroupInfoReq{
		UserId:  c.GetInt64("user_id"),
		GroupId: req.GroupId,
		Name:    req.Name,
		Avatar:  req.Avatar,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}
