package mhttp

import (
	"go-im/internal/common/errcode"
	"go-im/internal/common/jwt"
	"go-im/internal/common/response"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("token")
		if token == "" {
			var ok bool
			token, ok = c.GetQuery("token")
			if !ok {
				c.Abort()
				response.Error(c, errcode.ErrUnAuthorized)
				return
			}
		}
		userId, err := jwt.GetUserIDFromToken(token)
		if err != nil {
			c.Abort()
			response.Error(c, errcode.ErrTokenExpired)
			return
		}
		c.Set("user_id", userId)
		c.Next()
	}
}
