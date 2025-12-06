package clients

import (
	"context"

	pb "github.com/S0rgi/Gainly_Avatars/pkg/proto"
)

// GRPCClient интерфейс для gRPC клиента (может быть обычный gRPC или gRPC-Web)
type GRPCClient interface {
	ValidateToken(ctx context.Context, token string) (*pb.UserResponse, error)
	GetUserById(ctx context.Context, userId string) (*pb.UserResponse, error)
	Close() error
}

// NewGRPCClient создает gRPC-Web клиент (так как сервер требует grpc-web)
func NewGRPCClient(addr string) (GRPCClient, error) {
	return NewGRPCWebClient(addr)
}
