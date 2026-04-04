package grpc_clients

import (
	"os"
	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/tracing"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DriverServiceClient wraps a long-lived gRPC connection to Driver Service. Create once at
// process startup with NewDriverServiceClient, reuse for all requests, and Close on shutdown.
type DriverServiceClient struct {
	Client pb.DriverServiceClient
	conn   *grpc.ClientConn
}

func NewDriverServiceClient() (*DriverServiceClient, error) {
	driverServiceURL := os.Getenv("DRIVER_SERVICE_URL")
	if driverServiceURL == "" {
		driverServiceURL = "driver-service:8080"
	}

	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if !strings.HasPrefix(driverServiceURL, "dns:///") {
		driverServiceURL = "dns:///" + driverServiceURL
	}

	conn, err := grpc.NewClient(driverServiceURL, dialOptions...)
	if err != nil {
		return nil, err
	}

	client := pb.NewDriverServiceClient(conn)

	return &DriverServiceClient{
		Client: client,
		conn:   conn,
	}, nil
}

func (c *DriverServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}
