package config

import (
	"os"
)

type Config struct {
	ServerPort     string
	R2AccountID    string
	R2AccessKeyID  string
	R2SecretKey    string
	R2BucketName   string
	R2Endpoint     string
	RedisURL       string
	GRPCUserServiceAddr string
}

func Load() *Config {
	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		R2AccountID:    getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:  getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretKey:    getEnv("R2_SECRET_KEY", ""),
		R2BucketName:   getEnv("R2_BUCKET_NAME", ""),
		R2Endpoint:     getEnv("R2_ENDPOINT", ""),
		RedisURL:       getEnv("REDIS_URL", ""),
		GRPCUserServiceAddr: getEnv("GRPC_USER_SERVICE_ADDR", "localhost:50051"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}


