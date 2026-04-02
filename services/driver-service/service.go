package main

import (
	"fmt"
	"math"
	randv2 "math/rand/v2"
	pbd "ride-sharing/shared/proto/driver"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/util"
	"sync"

	"github.com/mmcloughlin/geohash"
)

const carpoolPackageSlug = "carpool"

type driverInMap struct {
	Driver *pbd.Driver
}

type Service struct {
	drivers []*driverInMap
	mu      sync.RWMutex
}

func NewService() *Service {
	return &Service{
		drivers: make([]*driverInMap, 0),
	}
}

func defaultCapacity(packageSlug string, requested int32) int32 {
	if requested > 0 {
		return requested
	}
	if packageSlug == carpoolPackageSlug {
		return 4
	}
	return 1
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180
	
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad
	
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return earthRadiusKm * c
}

func (s *Service) FindAvailableDrivers(packageType string, requestedSeats int32, tripRoute *pb.Route, attemptIndex int) []string {
	if requestedSeats < 1 {
		requestedSeats = 1
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var matchingDrivers []string
	
	var pickupLat, pickupLon float64
	hasPickup := false
	if tripRoute != nil && len(tripRoute.Geometry) > 0 && len(tripRoute.Geometry[0].Coordinates) > 0 {
		pickupLat = tripRoute.Geometry[0].Coordinates[0].Latitude
		pickupLon = tripRoute.Geometry[0].Coordinates[0].Longitude
		hasPickup = true
	}

	var maxRadiusKm float64
	switch attemptIndex {
	case 0:
		maxRadiusKm = 1.0
	case 1:
		maxRadiusKm = 3.0
	case 2:
		maxRadiusKm = 5.0
	case 3:
		maxRadiusKm = 10.0
	default:
		maxRadiusKm = 100.0
	}

	for _, d := range s.drivers {
		if d.Driver.PackageSlug != packageType {
			continue
		}
		if d.Driver.AvailableSeats < requestedSeats {
			continue
		}
		
		// If we know the pickup location, filter by radius!
		if hasPickup && d.Driver.Location != nil {
			dist := haversineDistance(pickupLat, pickupLon, d.Driver.Location.Latitude, d.Driver.Location.Longitude)
			if dist > maxRadiusKm {
				continue
			}
		}

		matchingDrivers = append(matchingDrivers, d.Driver.Id)
	}

	return matchingDrivers
}

func (s *Service) RegisterDriver(driverID, packageSlug string, capacity int32) (*pbd.Driver, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cap := defaultCapacity(packageSlug, capacity)

	randomIndex := randv2.IntN(len(PredefinedRoutes))
	randomRoute := PredefinedRoutes[randomIndex]

	randomPlate := GenerateRandomPlate()
	randomAvatar := util.GetRandomAvatar(randomIndex)

	geohashStr := geohash.Encode(randomRoute[0][0], randomRoute[0][1])

	driver := &pbd.Driver{
		Id:             driverID,
		Geohash:        geohashStr,
		Location:       &pbd.Location{Latitude: randomRoute[0][0], Longitude: randomRoute[0][1]},
		Name:           "Lando Norris",
		PackageSlug:    packageSlug,
		ProfilePicture: randomAvatar,
		CarPlate:       randomPlate,
		Capacity:       cap,
		AvailableSeats: cap,
		ActiveTripIds:  nil,
	}

	s.drivers = append(s.drivers, &driverInMap{
		Driver: driver,
	})

	return driver, nil
}

func (s *Service) UnregisterDriver(driverID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, d := range s.drivers {
		if d.Driver.Id == driverID {
			s.drivers = append(s.drivers[:i], s.drivers[i+1:]...)
			return
		}
	}
}

func (s *Service) NotifyTripAccepted(driverID, tripID string, requestedSeats int32) error {
	if requestedSeats < 1 {
		requestedSeats = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	d := s.findDriverLocked(driverID)
	if d == nil {
		return fmt.Errorf("driver not found: %s", driverID)
	}
	if d.AvailableSeats < requestedSeats {
		return fmt.Errorf("not enough seats")
	}
	d.AvailableSeats -= requestedSeats
	d.ActiveTripIds = append(d.ActiveTripIds, tripID)
	return nil
}

func (s *Service) NotifyTripCompleted(driverID, tripID string, releasedSeats int32) error {
	if releasedSeats < 1 {
		releasedSeats = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	d := s.findDriverLocked(driverID)
	if d == nil {
		return fmt.Errorf("driver not found: %s", driverID)
	}

	d.AvailableSeats += releasedSeats
	if d.AvailableSeats > d.Capacity {
		d.AvailableSeats = d.Capacity
	}

	out := d.ActiveTripIds[:0]
	for _, id := range d.ActiveTripIds {
		if id != tripID {
			out = append(out, id)
		}
	}
	d.ActiveTripIds = out
	return nil
}

func (s *Service) GetDriverActiveTrips(driverID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := s.findDriverLocked(driverID)
	if d != nil && len(d.ActiveTripIds) > 0 {
		out := make([]string, len(d.ActiveTripIds))
		copy(out, d.ActiveTripIds)
		return out
	}
	return nil
}

func (s *Service) findDriverLocked(driverID string) *pbd.Driver {
	for _, dm := range s.drivers {
		if dm.Driver.Id == driverID {
			return dm.Driver
		}
	}
	return nil
}
