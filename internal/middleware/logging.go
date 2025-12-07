package middleware

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware логирует все HTTP запросы
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Логируем входящий запрос
		log.Printf("[REQUEST] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		log.Printf("[HEADERS] %+v", r.Header)

		// Создаем wrapper для ResponseWriter, чтобы перехватить статус код
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Выполняем следующий handler
		next.ServeHTTP(wrapped, r)

		// Логируем результат
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s %s - Status: %d - Duration: %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter обертка для ResponseWriter для перехвата статус кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}


