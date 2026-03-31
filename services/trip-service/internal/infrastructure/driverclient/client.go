package driverclient

import (
	"context"
	"log"
	"os"
	"strings"

	pb "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	pb.DriverServiceClient
	conn *grpc.ClientConn
}

func New() (*Client, error) {
	url := os.Getenv("DRIVER_SERVICE_URL")
	if url == "" {
		url = "driver-service:8080"
	}
	if !strings.HasPrefix(url, "dns:///") {
		url = "dns:///" + url
	}

	opts := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	conn, err := grpc.NewClient(url, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		DriverServiceClient: pb.NewDriverServiceClient(conn),
		conn:                conn,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) NotifyTripAcceptedSeats(ctx context.Context, driverID, tripID string, seats int32) {
	if c == nil || driverID == "" || seats < 1 {
		return
	}
	_, err := c.DriverServiceClient.NotifyTripAccepted(ctx, &pb.NotifyTripAcceptedRequest{
		DriverID:       driverID,
		TripID:         tripID,
		RequestedSeats: seats,
	})
	if err != nil {
		log.Printf("driverclient: NotifyTripAccepted failed: %v", err)
	}
}

func (c *Client) NotifyTripCompletedSeats(ctx context.Context, driverID, tripID string, seats int32) {
	if c == nil || driverID == "" || seats < 1 {
		return
	}
	_, err := c.DriverServiceClient.NotifyTripCompleted(ctx, &pb.NotifyTripCompletedRequest{
		DriverID:      driverID,
		TripID:        tripID,
		ReleasedSeats: seats,
	})
	if err != nil {
		log.Printf("driverclient: NotifyTripCompleted failed: %v", err)
	}
}
