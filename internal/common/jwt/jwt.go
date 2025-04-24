package jwt

import (
	"errors"
	"go-im/internal/common/errcode"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var defaultJWT *JWT

// Config JWT配置
type Config struct {
	Key    string
	Expire int
}

// JWT 处理结构
type JWT struct {
	config Config
}

// Claims 自定义的JWT声明结构
type claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func Init(config Config) {
	if config.Key == "" {
		panic("SigningKey is required")
	}
	if config.Expire <= 0 {
		config.Expire = 7 * 24 * 60
	}
	defaultJWT = &JWT{
		config: config,
	}
}

// GenerateToken 生成JWT令牌
func GenerateToken(userID int64) (string, error) {
	now := time.Now()
	claims := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(defaultJWT.config.Expire) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// 使用HS256算法创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(defaultJWT.config.Key))
}

// ParseToken 解析JWT令牌
func parseToken(tokenString string) (*claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(defaultJWT.config.Key), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errcode.ErrTokenExpired
		}
		return nil, errcode.ErrTokenExpired
	}

	claims, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return nil, errcode.ErrTokenExpired
	}

	return claims, nil
}

// ValidateToken 验证JWT令牌
func ValidateToken(tokenString string) bool {
	_, err := parseToken(tokenString)
	return err == nil
}

// GetUserIDFromToken 从令牌中获取用户ID
func GetUserIDFromToken(tokenString string) (int64, error) {
	claims, err := parseToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}
