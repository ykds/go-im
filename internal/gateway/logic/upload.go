package logic

import (
	"go-im/internal/common/errcode"
	"go-im/internal/common/middleware/mhttp"
	"go-im/internal/common/response"
	"go-im/internal/gateway/server"
	"go-im/internal/gateway/types"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var (
	dir = "D:\\Project\\Go\\go-im-zero\\api_gateway\\static"
)

type UploadApi struct {
	s *server.Server
}

func NewUploadApi(s *server.Server) *UploadApi {
	return &UploadApi{s}
}

func (api *UploadApi) RegisterRouter(engine *gin.RouterGroup) {
	g := engine.Group("/upload", mhttp.AuthMiddleware())
	g.POST("", api.Upload)
}

func (api *UploadApi) Upload(c *gin.Context) {
	var (
		resp types.UploadResp
		err  error
	)
	defer func() {
		if err != nil {
			response.Error(c, err)
		} else {
			response.Success(resp)
		}
	}()
	header, err := c.FormFile("file")
	if err != nil {
		return
	}
	err = c.SaveUploadedFile(header, filepath.Join(dir, header.Filename))
	if err != nil {
		return
	}
	ext := filepath.Ext(header.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		err = errcode.ErrAvatarExtNotSupported
		return
	}
	resp = types.UploadResp{
		Url: path.Join("/static", header.Filename),
	}
}
