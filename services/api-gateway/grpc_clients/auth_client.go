package grpc_clients

import (
	"os"
	pb "ride-sharing/shared/proto/auth"
	"ride-sharing/shared/tracing"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserAuthServiceClient struct {
	Client pb.UserAuthServiceClient
	conn   *grpc.ClientConn
}

func NewUserAuthServiceClient() (*UserAuthServiceClient, error) {
	url := os.Getenv("USER_AUTH_SERVICE_URL")
	if url == "" {
		url = "user-auth-service:9095"
	}
	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if !strings.HasPrefix(url, "dns:///") {
		url = "dns:///" + url
	}
	conn, err := grpc.NewClient(url, dialOptions...)
	if err != nil {
		return nil, err
	}
	return &UserAuthServiceClient{
		Client: pb.NewUserAuthServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *UserAuthServiceClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
