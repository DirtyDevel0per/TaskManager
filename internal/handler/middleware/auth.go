package middleware

import (
	"context"
	"net/http"
	"strings"
	"task-manager/pkg/jwt"

	"github.com/google/uuid"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// AuthMiddleware проверяет JWT токен и добавляет userID в контекст
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем заголовок Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Проверяем формат: Bearer <token>
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization format. Use: Bearer <token>", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Валидируем токен
			claims, err := jwt.ValidateToken(token, jwtSecret)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Добавляем userID в контекст
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID извлекает userID из контекста
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, false
	}

	return id, true
}
