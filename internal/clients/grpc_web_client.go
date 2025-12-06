package clients

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	pb "github.com/S0rgi/Gainly_Avatars/pkg/proto"
	"google.golang.org/protobuf/proto"
)

type GRPCWebClient struct {
	baseURL string
	client  *http.Client
}

func NewGRPCWebClient(addr string) (*GRPCWebClient, error) {
	// Убираем https:// из адреса, если есть, но сохраняем протокол
	baseURL := addr
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		// Если протокол не указан, добавляем https://
		baseURL = "https://" + baseURL
	}

	log.Printf("[GRPC-WEB] Creating gRPC-Web client for: %s", baseURL)

	return &GRPCWebClient{
		baseURL: baseURL,
		client:  &http.Client{
			// Таймауты
		},
	}, nil
}

// ValidateToken валидирует токен через gRPC-Web
func (c *GRPCWebClient) ValidateToken(ctx context.Context, token string) (*pb.UserResponse, error) {
	log.Printf("[GRPC-WEB] Validating token (length: %d, first 20 chars: %s...)", len(token), token[:min(20, len(token))])

	// Создаем запрос
	req := &pb.TokenRequest{
		AccessToken: token,
	}

	// Сериализуем protobuf сообщение
	messageData, err := proto.Marshal(req)
	if err != nil {
		log.Printf("[GRPC-WEB] ERROR: Failed to marshal request: %v", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Формируем gRPC-Web запрос в правильном формате
	// gRPC-Web формат: [flags:1 byte][length:4 bytes][message data]
	msgLen := uint32(len(messageData))
	flags := byte(0) // 0 = данные, 1 = трайлеры

	// Создаем буфер с правильным форматом
	var buf bytes.Buffer
	buf.WriteByte(flags)                         // Флаги
	binary.Write(&buf, binary.BigEndian, msgLen) // Длина сообщения (4 байта)
	buf.Write(messageData)                       // Само сообщение

	url := fmt.Sprintf("%s/user.UserService/ValidateToken", c.baseURL)

	log.Printf("[GRPC-WEB] Sending request to: %s (message size: %d bytes)", url, len(messageData))

	// Создаем HTTP запрос с gRPC-Web заголовками
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		log.Printf("[GRPC-WEB] ERROR: Failed to create HTTP request: %v", err)
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Устанавливаем gRPC-Web заголовки
	httpReq.Header.Set("Content-Type", "application/grpc-web+proto")
	httpReq.Header.Set("Accept", "application/grpc-web+proto")
	httpReq.Header.Set("X-Grpc-Web", "1")
	httpReq.Header.Set("X-User-Agent", "grpc-web-go/1.0")

	// Отправляем запрос
	resp, err := c.client.Do(httpReq)
	if err != nil {
		log.Printf("[GRPC-WEB] ERROR: HTTP request failed: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[GRPC-WEB] Response status: %d %s", resp.StatusCode, resp.Status)
	log.Printf("[GRPC-WEB] Response headers: %+v", resp.Header)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[GRPC-WEB] ERROR: Non-200 status. Body: %s", string(body))
		return nil, fmt.Errorf("gRPC-Web request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[GRPC-WEB] ERROR: Failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// gRPC-Web ответ может быть в base64 или бинарном формате
	// Проверяем заголовок Content-Type
	contentType := resp.Header.Get("Content-Type")
	log.Printf("[GRPC-WEB] Response Content-Type: %s", contentType)

	var responseData []byte
	if strings.Contains(contentType, "application/grpc-web-text") {
		// Base64 encoded
		decoded, decodeErr := base64.StdEncoding.DecodeString(string(body))
		if decodeErr != nil {
			log.Printf("[GRPC-WEB] ERROR: Failed to decode base64 response: %v", decodeErr)
			return nil, fmt.Errorf("failed to decode base64 response: %w", decodeErr)
		}
		responseData = decoded
	} else {
		// Binary format
		responseData = body
	}

	// Парсим gRPC-Web формат
	// gRPC-Web формат: [flags:1 byte][length:4 bytes][message data]
	if len(responseData) < 5 {
		log.Printf("[GRPC-WEB] ERROR: Response too short: %d bytes", len(responseData))
		return nil, fmt.Errorf("response too short: %d bytes", len(responseData))
	}

	// Пропускаем флаги (1 байт) и читаем длину (4 байта)
	responseMsgLen := binary.BigEndian.Uint32(responseData[1:5])
	if len(responseData) < int(5+responseMsgLen) {
		log.Printf("[GRPC-WEB] ERROR: Response incomplete. Expected %d bytes, got %d", 5+responseMsgLen, len(responseData))
		return nil, fmt.Errorf("response incomplete")
	}

	// Извлекаем сообщение
	msgData := responseData[5 : 5+responseMsgLen]

	// Десериализуем protobuf ответ
	userResp := &pb.UserResponse{}
	if err := proto.Unmarshal(msgData, userResp); err != nil {
		log.Printf("[GRPC-WEB] ERROR: Failed to unmarshal response: %v", err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Printf("[GRPC-WEB] SUCCESS: Token validated. User ID: %s, Username: %s, Email: %s",
		userResp.Id, userResp.Username, userResp.Email)

	return userResp, nil
}

func (c *GRPCWebClient) GetUserById(ctx context.Context, userId string) (*pb.UserResponse, error) {
	req := &pb.UserRequest{
		Id: userId,
	}

	messageData, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Формируем gRPC-Web формат
	msgLen := uint32(len(messageData))
	flags := byte(0)

	var buf bytes.Buffer
	buf.WriteByte(flags)
	binary.Write(&buf, binary.BigEndian, msgLen)
	buf.Write(messageData)

	url := fmt.Sprintf("%s/user.UserService/GetUserById", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/grpc-web+proto")
	httpReq.Header.Set("Accept", "application/grpc-web+proto")
	httpReq.Header.Set("X-Grpc-Web", "1")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gRPC-Web request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	var responseData []byte
	if strings.Contains(contentType, "application/grpc-web-text") {
		decoded, decodeErr := base64.StdEncoding.DecodeString(string(body))
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode base64 response: %w", decodeErr)
		}
		responseData = decoded
	} else {
		responseData = body
	}

	if len(responseData) < 5 {
		return nil, fmt.Errorf("response too short")
	}

	responseMsgLen := binary.BigEndian.Uint32(responseData[1:5])
	if len(responseData) < int(5+responseMsgLen) {
		return nil, fmt.Errorf("response incomplete")
	}

	msgData := responseData[5 : 5+responseMsgLen]

	userResp := &pb.UserResponse{}
	if err := proto.Unmarshal(msgData, userResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return userResp, nil
}

func (c *GRPCWebClient) Close() error {
	// HTTP клиент не требует закрытия
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
