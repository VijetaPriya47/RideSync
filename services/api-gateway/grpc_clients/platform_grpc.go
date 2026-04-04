package grpc_clients

import (
	"os"
	authpb "ride-sharing/shared/proto/auth"
	finpb "ride-sharing/shared/proto/finance"
	"ride-sharing/shared/tracing"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// FinanceServiceClient wraps the finance gRPC stub (optional dedicated connection).
type FinanceServiceClient struct {
	Client finpb.FinanceServiceClient
	conn   *grpc.ClientConn
}

// UserAuthServiceClient wraps the user-auth gRPC stub (optional dedicated connection).
type UserAuthServiceClient struct {
	Client authpb.UserAuthServiceClient
	conn   *grpc.ClientConn
}

// PlatformGRPC holds finance and auth clients sharing one connection to platform-service.
type PlatformGRPC struct {
	Finance *FinanceServiceClient
	Auth    *UserAuthServiceClient
	conn    *grpc.ClientConn
}

func resolvePlatformServiceURL() string {
	if u := os.Getenv("PLATFORM_SERVICE_URL"); u != "" {
		return u
	}
	if u := os.Getenv("FINANCE_SERVICE_URL"); u != "" {
		return u
	}
	if u := os.Getenv("USER_AUTH_SERVICE_URL"); u != "" {
		return u
	}
	return "platform-service:9094"
}

func dialInsecure(url string) (*grpc.ClientConn, error) {
	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if !strings.HasPrefix(url, "dns:///") {
		url = "dns:///" + url
	}
	return grpc.NewClient(url, dialOptions...)
}

// NewPlatformGRPC dials platform-service once and exposes both protobuf clients.
func NewPlatformGRPC() (*PlatformGRPC, error) {
	url := resolvePlatformServiceURL()
	conn, err := dialInsecure(url)
	if err != nil {
		return nil, err
	}
	return &PlatformGRPC{
		Finance: &FinanceServiceClient{Client: finpb.NewFinanceServiceClient(conn), conn: nil},
		Auth:    &UserAuthServiceClient{Client: authpb.NewUserAuthServiceClient(conn), conn: nil},
		conn:    conn,
	}, nil
}

// Close closes the shared connection.
func (p *PlatformGRPC) Close() {
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
}

// Close is a no-op when the client shares a PlatformGRPC connection (conn nil).
func (c *FinanceServiceClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

// Close is a no-op when the client shares a PlatformGRPC connection (conn nil).
func (c *UserAuthServiceClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}
