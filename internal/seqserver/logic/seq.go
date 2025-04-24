package logic

import (
	"fmt"
	"go-im/internal/common/errcode"
	"go-im/internal/common/middleware/mhttp"
	"go-im/internal/common/response"
	"go-im/internal/seqserver/server"
	"go-im/internal/seqserver/types"

	"github.com/gin-gonic/gin"
)

type SeqApi struct {
	s *server.Server
}

func NewSeqApi(s *server.Server) *SeqApi {
	return &SeqApi{s}
}

func (api *SeqApi) RegisterRouter(engine *gin.RouterGroup) {
	g := engine.Group("/seq", mhttp.AuthMiddleware())
	{
		g.GET("", api.GetSeqHandler)
	}
}

func (api *SeqApi) GetSeqHandler(c *gin.Context) {
	var (
		req  types.GetSeqReq
		resp int64
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(resp)
		}
	}()
	if err = c.BindQuery(&req); err != nil {
		err = errcode.ErrInvalidParam
		return
	}
	resp, err = api.s.SeqServer.Get(c, getSeqKey(c.GetInt64("user_id"), req.ToId, req.Kind))
}

func getSeqKey(fromId, toId int64, kind string) string {
	if kind == "single" {
		return fmt.Sprintf("seq:chat:%s:%d_%d", kind, fromId, toId)
	}
	return fmt.Sprintf("seq:chat:%s:%d", kind, toId)

}
