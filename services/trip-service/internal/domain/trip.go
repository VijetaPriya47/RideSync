package domain

import (
	"context"
	"ride-sharing/shared/types"

	tripTypes "ride-sharing/services/trip-service/pkg/types"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TripDriver struct {
	ID             string `bson:"id"`
	Name           string `bson:"name"`
	ProfilePicture string `bson:"profilePicture"`
	CarPlate       string `bson:"carPlate"`
}

func (d *TripDriver) ToProto() *pb.TripDriver {
	if d == nil {
		return nil
	}
	return &pb.TripDriver{
		Id:             d.ID,
		Name:           d.Name,
		ProfilePicture: d.ProfilePicture,
		CarPlate:       d.CarPlate,
	}
}

type TripModel struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	UserID   string             `bson:"userID"`
	Status   string             `bson:"status"`
	RideFare *RideFareModel     `bson:"rideFare"`
	Driver   *TripDriver        `bson:"driver"`
}

func (t *TripModel) ToProto() *pb.Trip {
	return &pb.Trip{
		Id:           t.ID.Hex(),
		UserID:       t.UserID,
		SelectedFare: t.RideFare.ToProto(),
		Status:       t.Status,
		Driver:       t.Driver.ToProto(),
		Route:        t.RideFare.Route.ToProto(),
	}
}

type TripRepository interface {
	CreateTrip(ctx context.Context, trip *TripModel) (*TripModel, error)
	SaveRideFare(ctx context.Context, f *RideFareModel) error
	GetRideFareByID(ctx context.Context, id string) (*RideFareModel, error)
	GetTripByID(ctx context.Context, id string) (*TripModel, error)
	UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error
	UpdateRideFareTotal(ctx context.Context, fareID string, totalPriceInCents float64) error
	UpdateTripRideFareTotal(ctx context.Context, tripID string, totalPriceInCents float64) error
	UpdateRideFareSeats(ctx context.Context, fareID string, seats int32) error
	// ListTripsForUser returns trips where the user is the rider (userID) or assigned driver (driver.id), newest first.
	ListTripsForUser(ctx context.Context, userID string, limit int32) ([]*TripModel, error)
}

type TripService interface {
	CreateTrip(ctx context.Context, fare *RideFareModel) (*TripModel, error)
	GetRoute(ctx context.Context, waypoints []*types.Coordinate, useOsrmApi bool) (*tripTypes.OsrmApiResponse, error)
	EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*RideFareModel
	GenerateTripFares(
		ctx context.Context,
		fares []*RideFareModel,
		userID string,
		Route *tripTypes.OsrmApiResponse,
		requestedSeats int32,
	) ([]*RideFareModel, error)
	GetAndValidateFare(ctx context.Context, fareID, userID string) (*RideFareModel, error)
	GetTripByID(ctx context.Context, id string) (*TripModel, error)
	UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error
	IncreaseTripFare(ctx context.Context, tripID, userID string, totalPriceInCents float64) (*TripModel, error)
	UpdateRideFareSeats(ctx context.Context, fareID string, seats int32) error
	ListMyTrips(ctx context.Context, userID string, limit int32) ([]*TripModel, error)
}
