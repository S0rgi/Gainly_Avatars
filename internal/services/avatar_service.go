package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/S0rgi/Gainly_Avatars/internal/clients"
	"github.com/google/uuid"
)

type AvatarService struct {
	r2Client    *clients.R2Client
	redisClient *clients.RedisClient
}

func NewAvatarService(r2Client *clients.R2Client, redisClient *clients.RedisClient) *AvatarService {
	return &AvatarService{
		r2Client:    r2Client,
		redisClient: redisClient,
	}
}

// AddAvatar добавляет новую аватарку
func (s *AvatarService) AddAvatar(ctx context.Context, username string, file io.Reader, filename string, contentType string, size int64) (string, error) {
	// Генерируем новый GUID
	guid := uuid.New().String()

	// Загружаем файл в R2
	if err := s.r2Client.UploadAvatar(ctx, guid, file, contentType, size); err != nil {
		return "", fmt.Errorf("failed to upload avatar: %w", err)
	}

	// Сохраняем метаданные в Redis
	metadata := &clients.AvatarMetadata{
		GUID:       guid,
		Username:   username,
		Filename:   filename,
		Size:       size,
		MimeType:   contentType,
		UploadedAt: time.Now(),
	}

	if err := s.redisClient.SetAvatarMetadata(ctx, metadata); err != nil {
		// Если не удалось сохранить метаданные, удаляем файл из R2
		_ = s.r2Client.DeleteAvatar(ctx, guid)
		return "", fmt.Errorf("failed to save metadata: %w", err)
	}

	// Обновляем связь username -> GUID
	if err := s.redisClient.SetGUIDByUsername(ctx, username, guid); err != nil {
		// Если не удалось сохранить связь, удаляем метаданные и файл
		_ = s.redisClient.DeleteAvatarMetadata(ctx, guid)
		_ = s.r2Client.DeleteAvatar(ctx, guid)
		return "", fmt.Errorf("failed to save username mapping: %w", err)
	}

	return guid, nil
}

// GetAvatarByUsername получает аватарку по username
func (s *AvatarService) GetAvatarByUsername(ctx context.Context, username string) (string, error) {
	guid, err := s.redisClient.GetGUIDByUsername(ctx, username)
	if err != nil {
		return "", err
	}

	// Генерируем presigned URL (действителен 1 час)
	url, err := s.r2Client.GetAvatarPresignedURL(ctx, guid, 3600)
	if err != nil {
		return "", fmt.Errorf("failed to generate avatar URL: %w", err)
	}

	return url, nil
}

// GetAvatarsByUsernames получает аватарки для списка username
func (s *AvatarService) GetAvatarsByUsernames(ctx context.Context, usernames []string) (map[string]string, error) {
	// Получаем GUIDs для всех username
	guidMap, err := s.redisClient.GetGUIDsByUsernames(ctx, usernames)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for username, guid := range guidMap {
		url, err := s.r2Client.GetAvatarPresignedURL(ctx, guid, 3600)
		if err != nil {
			// Пропускаем ошибки генерации URL
			continue
		}
		result[username] = url
	}

	return result, nil
}

// GetMyAvatar получает аватарку текущего пользователя
func (s *AvatarService) GetMyAvatar(ctx context.Context, username string) (string, error) {
	return s.GetAvatarByUsername(ctx, username)
}

// DeleteMyAvatar удаляет аватарку текущего пользователя
func (s *AvatarService) DeleteMyAvatar(ctx context.Context, username string) error {
	// Получаем GUID по username
	guid, err := s.redisClient.GetGUIDByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("avatar not found for username: %s", username)
	}

	// Удаляем файл из R2
	if err := s.r2Client.DeleteAvatar(ctx, guid); err != nil {
		return fmt.Errorf("failed to delete avatar from R2: %w", err)
	}

	// Удаляем метаданные из Redis
	if err := s.redisClient.DeleteAvatarMetadata(ctx, guid); err != nil {
		// Логируем ошибку, но не возвращаем её, так как файл уже удален
		fmt.Printf("warning: failed to delete metadata: %v\n", err)
	}

	// Удаляем связь username -> GUID
	if err := s.redisClient.DeleteUsernameMapping(ctx, username); err != nil {
		fmt.Printf("warning: failed to delete username mapping: %v\n", err)
	}

	return nil
}
