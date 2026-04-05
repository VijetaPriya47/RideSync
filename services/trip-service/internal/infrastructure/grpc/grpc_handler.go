package grpc

import (
	"context"
	"log"
	"strings"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer

	service   domain.TripService
	publisher *events.TripEventPublisher
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService, publisher *events.TripEventPublisher) *gRPCHandler {
	handler := &gRPCHandler{
		service:   service,
		publisher: publisher,
	}

	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	fareID := req.GetRideFareID()
	userID := req.GetUserID()

	rideFare, err := h.service.GetAndValidateFare(ctx, fareID, userID)
	if err != nil {
		log.Printf("DEBUG: CreateTrip - GetAndValidateFare failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to validate the fare: %v", err)
	}

	trip, err := h.service.CreateTrip(ctx, rideFare)
	if err != nil {
		log.Printf("DEBUG: CreateTrip - service.CreateTrip failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to create the trip: %v", err)
	}

	if err := h.publisher.PublishTripCreated(ctx, trip); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish the trip created event: %v", err)
	}

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
		Trip:   trip.ToProto(),
	}, nil
}

func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
	pickup := req.GetStartLocation()
	destination := req.GetEndLocation()

	waypoints := []*types.Coordinate{
		{
			Latitude:  pickup.Latitude,
			Longitude: pickup.Longitude,
		},
		{
			Latitude:  destination.Latitude,
			Longitude: destination.Longitude,
		},
	}

	userID := req.GetUserID()
	requestedSeats := req.GetRequestedSeats()
	if requestedSeats < 1 {
		requestedSeats = 1
	}

	route, err := h.service.GetRoute(ctx, waypoints, true)
	if err != nil {
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}

	estimatedFares := h.service.EstimatePackagesPriceWithRoute(route)

	fares, err := h.service.GenerateTripFares(ctx, estimatedFares, userID, route, requestedSeats)
	if err != nil {
		log.Printf("DEBUG: PreviewTrip - GenerateTripFares failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to generate the ride fares: %v", err)
	}

	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}

func (h *gRPCHandler) IncreaseTripFare(ctx context.Context, req *pb.IncreaseTripFareRequest) (*pb.IncreaseTripFareResponse, error) {
	trip, err := h.service.IncreaseTripFare(ctx, req.GetTripID(), req.GetUserID(), req.GetTotalPriceInCents())
	if err != nil {
		log.Printf("IncreaseTripFare: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	if err := h.publisher.PublishTripCreated(ctx, trip); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to republish trip for driver search: %v", err)
	}

	return &pb.IncreaseTripFareResponse{
		Trip: trip.ToProto(),
	}, nil
}

func (h *gRPCHandler) GetTrip(ctx context.Context, req *pb.GetTripRequest) (*pb.GetTripResponse, error) {
	tripID := req.GetTripId()
	if tripID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "trip_id is required")
	}
	trip, err := h.service.GetTripByID(ctx, tripID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get trip: %v", err)
	}
	if trip == nil {
		return nil, status.Errorf(codes.NotFound, "trip not found")
	}
	return &pb.GetTripResponse{Trip: trip.ToProto()}, nil
}

func (h *gRPCHandler) UpdateFareSeats(ctx context.Context, req *pb.UpdateFareSeatsRequest) (*pb.UpdateFareSeatsResponse, error) {
	fareID := req.GetFareId()
	if fareID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "fare_id is required")
	}
	if err := h.service.UpdateRideFareSeats(ctx, fareID, req.GetSeats()); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.UpdateFareSeatsResponse{}, nil
}

func (h *gRPCHandler) ListMyTrips(ctx context.Context, req *pb.ListMyTripsRequest) (*pb.ListMyTripsResponse, error) {
	uid := strings.TrimSpace(req.GetUserId())
	if uid == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id required")
	}
	limit := req.GetLimit()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	trips, err := h.service.ListMyTrips(ctx, uid, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	var entries []*pb.RideHistoryEntry
	for _, t := range trips {
		entries = append(entries, tripModelToHistoryEntry(t, uid))
	}
	return &pb.ListMyTripsResponse{Entries: entries}, nil
}
