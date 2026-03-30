package main

import (
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"
)

type previewTripRequest struct {
	UserID          string           `json:"userID"`
	Pickup          types.Coordinate `json:"pickup"`
	Destination     types.Coordinate `json:"destination"`
	RequestedSeats  int32            `json:"requestedSeats,omitempty"`
}

func (p *previewTripRequest) toProto() *pb.PreviewTripRequest {
	return &pb.PreviewTripRequest{
		UserID: p.UserID,
		StartLocation: &pb.Coordinate{
			Latitude:  p.Pickup.Latitude,
			Longitude: p.Pickup.Longitude,
		},
		EndLocation: &pb.Coordinate{
			Latitude:  p.Destination.Latitude,
			Longitude: p.Destination.Longitude,
		},
		RequestedSeats: p.RequestedSeats,
	}
}

type startTripRequest struct {
	RideFareID string `json:"rideFareID"`
	UserID     string `json:"userID"`
}

func (c *startTripRequest) toProto() *pb.CreateTripRequest {
	return &pb.CreateTripRequest{
		RideFareID: c.RideFareID,
		UserID:     c.UserID,
	}
}

type increaseTripFareRequest struct {
	TripID             string  `json:"tripID"`
	UserID             string  `json:"userID"`
	TotalPriceInCents  float64 `json:"totalPriceInCents"`
}

func (r *increaseTripFareRequest) toProto() *pb.IncreaseTripFareRequest {
	return &pb.IncreaseTripFareRequest{
		TripID:            r.TripID,
		UserID:            r.UserID,
		TotalPriceInCents: r.TotalPriceInCents,
	}
}