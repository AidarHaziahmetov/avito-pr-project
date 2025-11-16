package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/aidar/avito-pr-project/internal/service"
)

// ContextKey это кастомный тип для ключей контекста
type ContextKey string

const (
	// UserIDKey ключ контекста для ID пользователя
	UserIDKey ContextKey = "user_id"
	// TeamNameKey ключ контекста для названия команды
	TeamNameKey ContextKey = "team_name"
)

// AuthMiddleware создает middleware для валидации JWT токенов
func AuthMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing authorization header"}}`, http.StatusUnauthorized)
				return
			}

			// Проверяем формат Bearer
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid authorization header format"}}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Валидируем токен
			claims, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid or expired token"}}`, http.StatusUnauthorized)
				return
			}

			// Добавляем claims в контекст
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, TeamNameKey, claims.TeamName)

			// Вызываем следующий обработчик
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext извлекает ID пользователя из контекста
func GetUserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetTeamNameFromContext извлекает название команды из контекста
func GetTeamNameFromContext(ctx context.Context) string {
	teamName, ok := ctx.Value(TeamNameKey).(string)
	if !ok {
		return ""
	}
	return teamName
}
