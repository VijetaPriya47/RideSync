package repository

import (
	"context"
	"fmt"
	"ride-sharing/services/trip-service/internal/domain"
	pbd "ride-sharing/shared/proto/driver"
)

type inmemRepository struct {
	trips     map[string]*domain.TripModel
	rideFares map[string]*domain.RideFareModel
}

func NewInmemRepository() *inmemRepository {
	return &inmemRepository{
		trips:     make(map[string]*domain.TripModel),
		rideFares: make(map[string]*domain.RideFareModel),
	}
}

func (r *inmemRepository) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	trip, ok := r.trips[id]
	if !ok {
		return nil, nil
	}
	return trip, nil
}

func (r *inmemRepository) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	trip, ok := r.trips[tripID]
	if !ok {
		return fmt.Errorf("trip not found with ID: %s", tripID)
	}

	trip.Status = status

	if driver != nil {
		trip.Driver = &domain.TripDriver{
			ID:             driver.Id,
			Name:           driver.Name,
			CarPlate:       driver.CarPlate,
			ProfilePicture: driver.ProfilePicture,
		}
	}
	return nil
}

func (r *inmemRepository) GetRideFareByID(ctx context.Context, id string) (*domain.RideFareModel, error) {
	fare, exist := r.rideFares[id]
	if !exist {
		return nil, fmt.Errorf("fare does not exist with ID: %s", id)
	}

	return fare, nil
}

func (r *inmemRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	r.trips[trip.ID.Hex()] = trip
	return trip, nil
}

func (r *inmemRepository) SaveRideFare(ctx context.Context, f *domain.RideFareModel) error {
	r.rideFares[f.ID.Hex()] = f
	return nil
}

func (r *inmemRepository) UpdateRideFareTotal(ctx context.Context, fareID string, totalPriceInCents float64) error {
	f, ok := r.rideFares[fareID]
	if !ok {
		return fmt.Errorf("ride fare not found: %s", fareID)
	}
	f.TotalPriceInCents = totalPriceInCents
	return nil
}

func (r *inmemRepository) UpdateTripRideFareTotal(ctx context.Context, tripID string, totalPriceInCents float64) error {
	trip, ok := r.trips[tripID]
	if !ok {
		return fmt.Errorf("trip not found: %s", tripID)
	}
	if trip.RideFare == nil {
		return fmt.Errorf("trip has no fare")
	}
	trip.RideFare.TotalPriceInCents = totalPriceInCents
	return nil
}

func (r *inmemRepository) UpdateRideFareSeats(ctx context.Context, fareID string, seats int32) error {
	f, ok := r.rideFares[fareID]
	if !ok {
		return fmt.Errorf("ride fare not found: %s", fareID)
	}
	f.RequestedSeats = seats
	return nil
}
