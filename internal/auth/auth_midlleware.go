// Package auth реализует аутентификацию и работу с пользовательским контекстом.
package auth

import (
	"context"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"time"

	"github.com/zauremazhikovayandex/url/internal/config"
)

// Middleware — middleware, добавляющий userID в контекст запроса из cookie/JWT.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conf := config.AppConfig

		userID := ""
		c, err := r.Cookie(conf.JWTCookieName)
		if err == nil && c != nil {
			tokenStr := c.Value
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(conf.JWTSecretKey), nil
			})
			if err == nil && token.Valid {
				userID = claims.UserID
			}
		}

		if userID == "" {
			userID = generateUserID()
			token, _ := GenerateToken(userID)
			http.SetCookie(w, &http.Cookie{
				Name:     conf.JWTCookieName,
				Value:    token,
				Path:     "/",
				Expires:  time.Now().Add(conf.JWTTokenExp),
				HttpOnly: true,
			})
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
