// Package auth реализует аутентификацию и работу с пользовательским контекстом.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"github.com/zauremazhikovayandex/url/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims - UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// GenerateToken - Генерация JWT токена
func GenerateToken(userID string) (string, error) {
	conf := config.AppConfig
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(conf.JWTTokenExp)),
		},
		UserID: userID,
	})

	return token.SignedString([]byte(conf.JWTSecretKey))
}

// GetUserID - Получить userID
func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(UserIDKey).(string)
	return userID
}

// Генерация userID (можно UUID, а пока — random base64)
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
