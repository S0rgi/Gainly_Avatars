package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/S0rgi/Gainly_Avatars/internal/clients"
	pb "github.com/S0rgi/Gainly_Avatars/pkg/proto"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware middleware для аутентификации через gRPC-Web
// Пропускает запросы к /api/avatars (GetAvatarsByUsernames не требует аутентификации)
func AuthMiddleware(grpcClient clients.GRPCClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Пропускаем запросы к /api/avatars без аутентификации
			if r.URL.Path == "/api/avatars" && r.Method == "POST" {
				log.Printf("[AUTH] Skipping authentication for /api/avatars")
				next.ServeHTTP(w, r)
				return
			}

			// Получаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			log.Printf("[AUTH] Authorization header (raw): %q", authHeader)

			if authHeader == "" {
				log.Printf("[AUTH] ERROR: Authorization header is empty")
				respondWithError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// Убираем кавычки везде (могут быть в начале, конце или везде)
			authHeader = strings.ReplaceAll(authHeader, `"`, "")
			authHeader = strings.TrimSpace(authHeader)
			log.Printf("[AUTH] Authorization header (after removing quotes): %q", authHeader)

			var token string

			// Поддерживаем два формата:
			// 1. "Bearer <token>"
			// 2. Просто "<token>" (если другие сервисы так отправляют)
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
				token = strings.TrimSpace(token)
				log.Printf("[AUTH] Token extracted from Bearer format: %q", token)
			} else {
				// Если нет префикса Bearer, считаем весь заголовок токеном
				token = authHeader
				log.Printf("[AUTH] Token used as-is (no Bearer prefix): %q", token)
			}

			// Еще раз убираем кавычки и пробелы на всякий случай
			token = strings.ReplaceAll(token, `"`, "")
			token = strings.TrimSpace(token)
			log.Printf("[AUTH] Final token (after cleanup): %q (length: %d)", token, len(token))

			if token == "" {
				log.Printf("[AUTH] ERROR: Token is empty after processing")
				respondWithError(w, http.StatusUnauthorized, "Token is empty")
				return
			}

			// Валидируем токен через gRPC
			log.Printf("[AUTH] Validating token via gRPC...")
			user, err := grpcClient.ValidateToken(r.Context(), token)
			if err != nil {
				log.Printf("[AUTH] ERROR: Token validation failed: %v", err)
				// Возвращаем детальную ошибку от gRPC
				errorMsg := fmt.Sprintf("Token validation failed: %v", err)
				respondWithError(w, http.StatusUnauthorized, errorMsg)
				return
			}

			log.Printf("[AUTH] SUCCESS: Token validated. User: ID=%s, Username=%s, Email=%s",
				user.Id, user.Username, user.Email)

			// Сохраняем информацию о пользователе в контексте
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext извлекает информацию о пользователе из контекста
func GetUserFromContext(ctx context.Context) (*pb.UserResponse, bool) {
	user, ok := ctx.Value(UserContextKey).(*pb.UserResponse)
	return user, ok
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
