package grpc_clients

import (
	"log"
	"os"
	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/tracing"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type driverServiceClient struct {
	Client pb.DriverServiceClient
	conn   *grpc.ClientConn
}

func NewDriverServiceClient() (*driverServiceClient, error) {
	driverServiceURL := os.Getenv("DRIVER_SERVICE_URL")
	if driverServiceURL == "" {
		driverServiceURL = "driver-service:8080"
	}

	var dialOptions []grpc.DialOption
	if strings.HasPrefix(driverServiceURL, "https://") {
		// Use TLS for HTTPS URLs (Render public endpoints)
		driverServiceURL = strings.TrimPrefix(driverServiceURL, "https://")
		dialOptions = append(
			tracing.DialOptionsWithTracing(),
			grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
		)
	} else {
		// Use insecure for internal/local connections
		dialOptions = append(
			tracing.DialOptionsWithTracing(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	log.Printf("DEBUG: Dialing Driver Service at URL: %s", driverServiceURL)

	conn, err := grpc.NewClient(driverServiceURL, dialOptions...)
	if err != nil {
		return nil, err
	}

	client := pb.NewDriverServiceClient(conn)

	return &driverServiceClient{
		Client: client,
		conn:   conn,
	}, nil
}

func (c *driverServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}
