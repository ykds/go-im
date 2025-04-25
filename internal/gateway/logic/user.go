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

type UserApi struct {
	s *server.Server
}

func NewUserApi(s *server.Server) *UserApi {
	return &UserApi{s}
}

func (api *UserApi) RegisterRouter(engine *gin.RouterGroup) {
	noauth := engine.Group("/user")
	{
		noauth.POST("/login", api.Login)
		noauth.POST("/register", api.Register)
	}
	auth := engine.Group("/user", mhttp.AuthMiddleware())
	{
		auth.GET("/info", api.UserInfo)
		auth.PUT("/info", api.UpdateInfo)
		auth.GET("/search", api.SearchUser)
	}
}

func (api *UserApi) Login(c *gin.Context) {
	var (
		req  types.LoginReq
		resp types.LoginResp
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
	rpcResp, err := api.s.UserRpc.Login(c.Request.Context(), &user.LoginReq{
		Phone:    req.Phone,
		Password: req.Password,
	})

	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	resp = types.LoginResp{
		Token: rpcResp.Token,
	}
}

func (api *UserApi) Register(c *gin.Context) {
	var (
		req types.RegisterReq
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
	_, err = api.s.UserRpc.Register(c.Request.Context(), &user.RegisterReq{
		Phone:    req.Phone,
		Username: req.Username,
		Password: req.Password,
		Avatar:   req.Avatar,
		Gender:   req.Gender,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *UserApi) SearchUser(c *gin.Context) {
	var (
		req  types.SearchUserReq
		resp types.SearchUserResp
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
	rpcResp, err := api.s.UserRpc.SearchUser(c.Request.Context(), &user.SearchUserReq{
		Phone: req.Phone,
	})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	resp = types.SearchUserResp{}
	for _, item := range rpcResp.List {
		resp.List = append(resp.List, types.SearchUserInfo{
			Id:       item.Id,
			Phone:    item.Phone,
			Username: item.Username,
			Avatar:   item.Avatar,
			Gender:   item.Gender,
		})
	}
}

func (api *UserApi) UpdateInfo(c *gin.Context) {
	var (
		req types.UpdateInfoReq
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
	_, err = api.s.UserRpc.UpdateInfo(c.Request.Context(), &user.UpdateInfoReq{UserId: c.GetInt64("user_id"), Username: req.Username})
	if err != nil {
		err = errcode.FromRpcError(err)
	}
}

func (api *UserApi) UserInfo(c *gin.Context) {
	var (
		resp types.UserInfoResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(c, resp)
		}
	}()
	rpcResp, err := api.s.UserRpc.UserInfo(c.Request.Context(), &user.UserInfoReq{UserId: c.GetInt64("user_id")})
	if err != nil {
		err = errcode.FromRpcError(err)
		return
	}
	resp = types.UserInfoResp{
		Phone:    rpcResp.Phone,
		Username: rpcResp.Username,
		Avatar:   rpcResp.Avatar,
		Gender:   rpcResp.Gender,
	}
}
