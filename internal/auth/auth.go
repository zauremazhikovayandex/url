package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	SecretKey  = "supersecretkey"
	TokenExp   = time.Hour * 3
	CookieName = "auth_token"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// Генерация JWT токена
func GenerateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: userID,
	})

	return token.SignedString([]byte(SecretKey))
}

// Считываем и валидируем JWT из куки
func GetUserIDFromRequest(r *http.Request) (string, error) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return "", errors.New("no auth cookie")
	}

	tokenStr := c.Value
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	return claims.UserID, nil
}

// Устанавливаем куку, если её нет. Если есть возвращаем UserID
func EnsureAuthCookie(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err == nil && c != nil {
		// Попробуем получить userID из токена
		tokenStr := c.Value
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(SecretKey), nil
		})

		if err == nil && token.Valid {
			return claims.UserID
		}
	}

	// создаём нового пользователя и устанавливаем токен
	newUserID := generateUserID()
	token, _ := GenerateToken(newUserID)

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(TokenExp),
		HttpOnly: true,
	})

	return newUserID
}

// Генерация userID (можно UUID, а пока — random base64)
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
