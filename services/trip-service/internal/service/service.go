package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	"ride-sharing/shared/env"
	pbd "ride-sharing/shared/proto/driver"
	"ride-sharing/shared/types"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const carpoolPackageSlug = "carpool"

type service struct {
	repo domain.TripRepository
}

func NewService(repo domain.TripRepository) *service {
	return &service{
		repo: repo,
	}
}

func (s *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
		Driver:   nil,
	}

	return s.repo.CreateTrip(ctx, t)
}

func (s *service) GetRoute(ctx context.Context, waypoints []*types.Coordinate, useOSRMApi bool) (*tripTypes.OsrmApiResponse, error) {
	if len(waypoints) < 2 {
		return nil, fmt.Errorf("at least two waypoints are required")
	}

	if !useOSRMApi {
		coords := make([][]float64, 0, len(waypoints))
		for _, w := range waypoints {
			coords = append(coords, []float64{w.Longitude, w.Latitude})
		}
		return &tripTypes.OsrmApiResponse{
			Routes: []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Geometry struct {
					Coordinates [][]float64 `json:"coordinates"`
				} `json:"geometry"`
			}{
				{
					Distance: 5.0,
					Duration: 600,
					Geometry: struct {
						Coordinates [][]float64 `json:"coordinates"`
					}{
						Coordinates: coords,
					},
				},
			},
		}, nil
	}

	baseURL := env.GetString("OSRM_API", "http://router.project-osrm.org")

	var b strings.Builder
	for i, w := range waypoints {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(fmt.Sprintf("%f,%f", w.Longitude, w.Latitude))
	}

	url := fmt.Sprintf("%s/route/v1/driving/%s?overview=full&geometries=geojson", baseURL, b.String())
	log.Printf("Started Fetching from OSRM API: URL: %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSRM request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch route from OSRM API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSRM API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OSRM response body: %w", err)
	}

	log.Printf("Got response from OSRM API %s", string(body))

	var routeResp tripTypes.OsrmApiResponse
	if err := json.Unmarshal(body, &routeResp); err != nil {
		return nil, fmt.Errorf("failed to parse OSRM response: %w", err)
	}

	return &routeResp, nil
}

func (s *service) EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*domain.RideFareModel {
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))

	for i, f := range baseFares {
		estimatedFares[i] = estimateFareRoute(f, route)
	}

	return estimatedFares
}

func (s *service) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string, route *tripTypes.OsrmApiResponse, requestedSeats int32) ([]*domain.RideFareModel, error) {
	if requestedSeats < 1 {
		requestedSeats = 1
	}

	fares := make([]*domain.RideFareModel, len(rideFares))

	for i, f := range rideFares {
		id := primitive.NewObjectID()

		fare := &domain.RideFareModel{
			UserID:            userID,
			ID:                id,
			TotalPriceInCents: f.TotalPriceInCents,
			PackageSlug:       f.PackageSlug,
			RequestedSeats:    requestedSeats,
			Route:             route,
		}

		if err := s.repo.SaveRideFare(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save trip fare: %w", err)
		}

		fares[i] = fare
	}

	return fares, nil
}

func (s *service) GetAndValidateFare(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	fare, err := s.repo.GetRideFareByID(ctx, fareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip fare: %w", err)
	}

	if fare == nil {
		return nil, fmt.Errorf("fare does not exist")
	}

	if userID != fare.UserID {
		return nil, fmt.Errorf("fare does not belong to the user")
	}

	return fare, nil
}

func estimateFareRoute(f *domain.RideFareModel, route *tripTypes.OsrmApiResponse) *domain.RideFareModel {
	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := f.TotalPriceInCents

	distanceKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	distanceFare := distanceKm * pricingCfg.PricePerUnitOfDistance
	timeFare := durationInMinutes * pricingCfg.PricingPerMinute
	totalPrice := carPackagePrice + distanceFare + timeFare

	if f.PackageSlug == carpoolPackageSlug {
		totalPrice *= 0.5
	}

	return &domain.RideFareModel{
		TotalPriceInCents: totalPrice,
		PackageSlug:       f.PackageSlug,
	}
}

func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 200,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 350,
		},
		{
			PackageSlug:       "van",
			TotalPriceInCents: 400,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 1000,
		},
		{
			PackageSlug:       carpoolPackageSlug,
			TotalPriceInCents: 350,
		},
	}
}

func (s *service) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	return s.repo.GetTripByID(ctx, id)
}

func (s *service) ListMyTrips(ctx context.Context, userID string, limit int32) ([]*domain.TripModel, error) {
	return s.repo.ListTripsForUser(ctx, userID, limit)
}

func (s *service) UpdateRideFareSeats(ctx context.Context, fareID string, seats int32) error {
	return s.repo.UpdateRideFareSeats(ctx, fareID, seats)
}

func (s *service) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	return s.repo.UpdateTrip(ctx, tripID, status, driver)
}

func (s *service) IncreaseTripFare(ctx context.Context, tripID, userID string, totalPriceInCents float64) (*domain.TripModel, error) {
	if totalPriceInCents <= 0 {
		return nil, fmt.Errorf("invalid total price")
	}

	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}
	if trip == nil {
		return nil, fmt.Errorf("trip not found")
	}
	if trip.UserID != userID {
		return nil, fmt.Errorf("trip does not belong to the user")
	}
	if trip.Status != "pending" {
		return nil, fmt.Errorf("trip is not pending")
	}
	if trip.Driver != nil {
		return nil, fmt.Errorf("driver already assigned")
	}
	if trip.RideFare == nil {
		return nil, fmt.Errorf("trip has no fare")
	}

	fareID := trip.RideFare.ID.Hex()
	if err := s.repo.UpdateRideFareTotal(ctx, fareID, totalPriceInCents); err != nil {
		return nil, fmt.Errorf("update ride fare: %w", err)
	}
	if err := s.repo.UpdateTripRideFareTotal(ctx, tripID, totalPriceInCents); err != nil {
		return nil, fmt.Errorf("update trip fare: %w", err)
	}

	return s.repo.GetTripByID(ctx, tripID)
}
