package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/S0rgi/Gainly_Avatars/internal/clients"
	"github.com/S0rgi/Gainly_Avatars/internal/config"
	"github.com/S0rgi/Gainly_Avatars/internal/handlers"
	"github.com/S0rgi/Gainly_Avatars/internal/middleware"
	"github.com/S0rgi/Gainly_Avatars/internal/services"
)

// @title Gainly Avatars API
// @version 1.0
// @description API для управления аватарками пользователей
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите токен в формате: Bearer {token}
func main() {
	// Загружаем переменные окружения из .env файла (если существует)
	// Игнорируем ошибку, если файл не найден
	_ = godotenv.Load()

	// Загружаем конфигурацию
	cfg := config.Load()

	// Инициализируем клиенты
	grpcClient, err := clients.NewGRPCClient(cfg.GRPCUserServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create gRPC client: %v", err)
	}
	defer grpcClient.Close()

	redisClient, err := clients.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	r2Client, err := clients.NewR2Client(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2SecretKey,
		cfg.R2BucketName,
		cfg.R2Endpoint,
	)
	if err != nil {
		log.Fatalf("Failed to create R2 client: %v", err)
	}

	// Создаем сервисы
	avatarService := services.NewAvatarService(r2Client, redisClient)

	// Создаем handlers
	handlers := handlers.NewHandlers(avatarService)

	// Настраиваем роутер
	router := mux.NewRouter()

	// Применяем логирование ко всем запросам
	router.Use(middleware.LoggingMiddleware)

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Применяем middleware для аутентификации ко всем API routes
	// (GetAvatarsByUsernames пропускается внутри middleware)
	api.Use(middleware.AuthMiddleware(grpcClient))

	// Avatar routes
	api.HandleFunc("/avatar", handlers.AddAvatar).Methods("POST")
	api.HandleFunc("/avatars", handlers.GetAvatarsByUsernames).Methods("POST")
	api.HandleFunc("/avatar/me", handlers.GetMyAvatar).Methods("GET")
	api.HandleFunc("/avatar/me", handlers.DeleteMyAvatar).Methods("DELETE")

	// Swagger JSON - загружаем из файла (должен быть перед Swagger UI)
	router.HandleFunc("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Пробуем несколько возможных путей
		possiblePaths := []string{
			filepath.Join("docs", "swagger.json"),      // Относительно рабочей директории
			filepath.Join(".", "docs", "swagger.json"), // Текущая директория
		}

		// Добавляем путь относительно рабочей директории
		if wd, err := os.Getwd(); err == nil {
			possiblePaths = append(possiblePaths, filepath.Join(wd, "docs", "swagger.json"))
		}

		var data []byte
		var err error
		var foundPath string

		for _, path := range possiblePaths {
			data, err = os.ReadFile(path)
			if err == nil {
				foundPath = path
				break
			}
		}

		if err != nil {
			log.Printf("Failed to load swagger.json. Tried paths: %v. Error: %v", possiblePaths, err)
			// Возвращаем базовую структуру, если файл не найден
			basicSwagger := map[string]interface{}{
				"swagger": "2.0",
				"info": map[string]interface{}{
					"title":       "Gainly Avatars API",
					"version":     "1.0",
					"description": "API для управления аватарками пользователей",
				},
				"host":     "localhost:" + cfg.ServerPort,
				"basePath": "/api",
				"schemes":  []string{"http", "https"},
				"paths":    map[string]interface{}{},
			}
			if err := json.NewEncoder(w).Encode(basicSwagger); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		log.Printf("Loaded swagger.json from: %s", foundPath)

		// Валидируем JSON
		var swaggerDoc map[string]interface{}
		if err := json.Unmarshal(data, &swaggerDoc); err != nil {
			log.Printf("Invalid swagger.json: %v", err)
			http.Error(w, "Invalid swagger documentation", http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	// Swagger UI
	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:"+cfg.ServerPort+"/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Настраиваем HTTP сервер
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("REST API server starting on port %s", cfg.ServerPort)
		log.Printf("Swagger UI available at http://localhost:%s/swagger/", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Ожидаем сигнал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
