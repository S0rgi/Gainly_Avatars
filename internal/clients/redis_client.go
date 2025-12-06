package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(redisURL string) (*RedisClient, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		client: client,
	}, nil
}

// GetGUIDByUsername получает GUID по username
func (r *RedisClient) GetGUIDByUsername(ctx context.Context, username string) (string, error) {
	key := fmt.Sprintf("username:%s", username)
	guid, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("username not found: %s", username)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get guid by username: %w", err)
	}
	return guid, nil
}

// SetGUIDByUsername устанавливает связь username -> GUID
func (r *RedisClient) SetGUIDByUsername(ctx context.Context, username, guid string) error {
	key := fmt.Sprintf("username:%s", username)
	return r.client.Set(ctx, key, guid, 0).Err()
}

// AvatarMetadata метаданные аватарки
type AvatarMetadata struct {
	GUID       string    `json:"guid"`
	Username   string    `json:"username"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	MimeType   string    `json:"mime_type"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// GetAvatarMetadata получает метаданные аватарки по GUID
func (r *RedisClient) GetAvatarMetadata(ctx context.Context, guid string) (*AvatarMetadata, error) {
	key := fmt.Sprintf("avatar:%s", guid)
	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("avatar metadata not found: %s", guid)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get avatar metadata: %w", err)
	}

	var metadata AvatarMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// SetAvatarMetadata устанавливает метаданные аватарки
func (r *RedisClient) SetAvatarMetadata(ctx context.Context, metadata *AvatarMetadata) error {
	key := fmt.Sprintf("avatar:%s", metadata.GUID)
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return r.client.Set(ctx, key, data, 0).Err()
}

// DeleteAvatarMetadata удаляет метаданные аватарки
func (r *RedisClient) DeleteAvatarMetadata(ctx context.Context, guid string) error {
	key := fmt.Sprintf("avatar:%s", guid)
	return r.client.Del(ctx, key).Err()
}

// GetGUIDsByUsernames получает GUIDs для списка username
func (r *RedisClient) GetGUIDsByUsernames(ctx context.Context, usernames []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, username := range usernames {
		guid, err := r.GetGUIDByUsername(ctx, username)
		if err != nil {
			// Пропускаем не найденные username
			continue
		}
		result[username] = guid
	}

	return result, nil
}

// DeleteUsernameMapping удаляет связь username -> GUID
func (r *RedisClient) DeleteUsernameMapping(ctx context.Context, username string) error {
	key := fmt.Sprintf("username:%s", username)
	return r.client.Del(ctx, key).Err()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
