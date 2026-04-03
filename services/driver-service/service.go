package main

import (
	"fmt"
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

func (s *Service) FindAvailableDrivers(packageType string, requestedSeats int32, tripRoute *pb.Route) []string {
	if requestedSeats < 1 {
		requestedSeats = 1
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var matchingDrivers []string
	
	if tripRoute != nil && len(tripRoute.Geometry) > 0 && len(tripRoute.Geometry[0].Coordinates) > 0 {
		// tripStart is no longer needed since we relaxed the geohash check
	}

	for _, d := range s.drivers {
		if d.Driver.PackageSlug != packageType {
			continue
		}
		if d.Driver.AvailableSeats < requestedSeats {
			continue
		}
		
		// No proximity check for carpool drivers on active trips for now.
		// The frontend handles accurate route overlap filtering, and
		// this allows matching even if mocked driver locations are stale.

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
		Name:           "Anya",
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

	if d.PackageSlug != carpoolPackageSlug {
		d.AvailableSeats = 0
	} else {
		d.AvailableSeats -= requestedSeats
	}

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

	if d.PackageSlug != carpoolPackageSlug {
		d.AvailableSeats = d.Capacity
	} else {
		d.AvailableSeats += releasedSeats
		if d.AvailableSeats > d.Capacity {
			d.AvailableSeats = d.Capacity
		}
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
