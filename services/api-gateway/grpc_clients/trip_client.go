package grpc_clients

import (
	"log"
	"os"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/tracing"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type tripServiceClient struct {
	Client pb.TripServiceClient
	conn   *grpc.ClientConn
}

func NewTripServiceClient() (*tripServiceClient, error) {
	tripServiceURL := os.Getenv("TRIP_SERVICE_URL")
	if tripServiceURL == "" {
		tripServiceURL = "trip-service:8080"
	}

	var dialOptions []grpc.DialOption
	if strings.HasPrefix(tripServiceURL, "https://") {
		// Use TLS for HTTPS URLs (Render public endpoints)
		tripServiceURL = strings.TrimPrefix(tripServiceURL, "https://")
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

	log.Printf("DEBUG: Dialing Trip Service at URL: %s", tripServiceURL)

	conn, err := grpc.NewClient(tripServiceURL, dialOptions...)
	if err != nil {
		return nil, err
	}

	client := pb.NewTripServiceClient(conn)

	return &tripServiceClient{
		Client: client,
		conn:   conn,
	}, nil
}

func (c *tripServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}
