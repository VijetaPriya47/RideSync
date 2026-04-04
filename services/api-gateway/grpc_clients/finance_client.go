package grpc_clients

import (
	"os"
	pb "ride-sharing/shared/proto/finance"
	"ride-sharing/shared/tracing"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type FinanceServiceClient struct {
	Client pb.FinanceServiceClient
	conn   *grpc.ClientConn
}

func NewFinanceServiceClient() (*FinanceServiceClient, error) {
	url := os.Getenv("FINANCE_SERVICE_URL")
	if url == "" {
		url = "finance-service:9094"
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
	return &FinanceServiceClient{
		Client: pb.NewFinanceServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *FinanceServiceClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
