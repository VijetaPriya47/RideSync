package main

import (
	"context"
	"log"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type driverGrpcHandler struct {
	pb.UnimplementedDriverServiceServer

	service *Service
}

func NewGrpcHandler(s *grpc.Server, service *Service) {
	handler := &driverGrpcHandler{
		service: service,
	}

	pb.RegisterDriverServiceServer(s, handler)
}

func (h *driverGrpcHandler) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	driver, err := h.service.RegisterDriver(req.GetDriverID(), req.GetPackageSlug(), req.GetCapacity())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register driver")
	}

	return &pb.RegisterDriverResponse{
		Driver: driver,
	}, nil
}

func (h *driverGrpcHandler) UnregisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	h.service.UnregisterDriver(req.GetDriverID())

	return &pb.RegisterDriverResponse{
		Driver: &pb.Driver{
			Id: req.GetDriverID(),
		},
	}, nil
}

func (h *driverGrpcHandler) NotifyTripAccepted(ctx context.Context, req *pb.NotifyTripAcceptedRequest) (*pb.NotifyTripAcceptedResponse, error) {
	if err := h.service.NotifyTripAccepted(req.GetDriverID(), req.GetTripID(), req.GetRequestedSeats()); err != nil {
		log.Printf("NotifyTripAccepted: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return &pb.NotifyTripAcceptedResponse{}, nil
}

func (h *driverGrpcHandler) NotifyTripCompleted(ctx context.Context, req *pb.NotifyTripCompletedRequest) (*pb.NotifyTripCompletedResponse, error) {
	if err := h.service.NotifyTripCompleted(req.GetDriverID(), req.GetTripID(), req.GetReleasedSeats()); err != nil {
		log.Printf("NotifyTripCompleted: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return &pb.NotifyTripCompletedResponse{}, nil
}
