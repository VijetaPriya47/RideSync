package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type tripConsumer struct {
	rabbitmq     *messaging.RabbitMQ
	service      *Service
	tripSvcURL   string
}

func NewTripConsumer(rabbitmq *messaging.RabbitMQ, service *Service) *tripConsumer {
	return &tripConsumer{
		rabbitmq:   rabbitmq,
		service:    service,
		tripSvcURL: env.GetString("TRIP_SERVICE_HTTP_URL", "http://ridesync:8080"),
	}
}

func (c *tripConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var tripEvent contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &tripEvent); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		var payload messaging.TripEventData
		if err := json.Unmarshal(tripEvent.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		log.Printf("driver search consumer: routingKey=%s, tried=%d, tripID=%s", msg.RoutingKey, len(payload.TriedDriverIDs), payload.Trip.Id)

		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return c.handleFindAndNotifyDrivers(ctx, payload)
		}

		log.Printf("unknown trip event: %s", msg.RoutingKey)

		return nil
	})
}

// Bounding box heuristic adapted from the frontend: prevents dispatching irrelevant carpool trips.
type tripStatusResponse struct {
	Status   string      `json:"Status"`
	Driver   interface{} `json:"Driver"`
	RideFare *struct {
		TotalPriceInCents float64 `json:"TotalPriceInCents"`
		Route             *struct {
			Routes []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Geometry struct {
					Coordinates [][]float64 `json:"coordinates"` // [lon, lat]
				} `json:"geometry"`
			} `json:"routes"`
		} `json:"Route"`
	} `json:"RideFare"`
}

func routesOverlap(activeTripRoute *struct {
	Routes []struct {
		Distance float64 `json:"distance"`
		Duration float64 `json:"duration"`
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"routes"`
}, newRoute *pb.Route) bool {
	if activeTripRoute == nil || len(activeTripRoute.Routes) == 0 || newRoute == nil {
		return true // Optimistic match if routing data is missing
	}

	var minLat, minLon, maxLat, maxLon float64
	minLat, minLon = 1e9, 1e9
	maxLat, maxLon = -1e9, -1e9

	pointsCount := 0
	for _, coord := range activeTripRoute.Routes[0].Geometry.Coordinates {
		if len(coord) < 2 {
			continue
		}
		lon, lat := coord[0], coord[1]
		pointsCount++
		if lat < minLat {
			minLat = lat
		}
		if lat > maxLat {
			maxLat = lat
		}
		if lon < minLon {
			minLon = lon
		}
		if lon > maxLon {
			maxLon = lon
		}
	}

	if pointsCount == 0 {
		return true
	}

	tolerance := 0.005 // Approx 0.5km
	minLat -= tolerance
	maxLat += tolerance
	minLon -= tolerance
	maxLon += tolerance

	for _, g := range newRoute.Geometry {
		for _, c := range g.Coordinates {
			if c.Latitude >= minLat && c.Latitude <= maxLat && c.Longitude >= minLon && c.Longitude <= maxLon {
				return true
			}
		}
	}

	return false
}

func (c *tripConsumer) checkDriverOverlap(driverID string, newRoute *pb.Route) bool {
	activeTrips := c.service.GetDriverActiveTrips(driverID)
	if len(activeTrips) == 0 {
		return true // No active trips, automatically overlapping/available
	}

	// Fetch trip status from trip-service for active trips to test overlaps
	for _, tripID := range activeTrips {
		url := fmt.Sprintf("%s/trips/%s", c.tripSvcURL, tripID)
		resp, err := http.Get(url)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				var status tripStatusResponse
				if err := json.NewDecoder(resp.Body).Decode(&status); err == nil {
					if status.RideFare != nil && status.RideFare.Route != nil {
						if routesOverlap(status.RideFare.Route, newRoute) {
							resp.Body.Close()
							return true
						}
					} else {
						resp.Body.Close()
						// log.Printf("checkDriverOverlap: missing route data for trip %s", tripID)
						continue 
					}
				}
			}
			resp.Body.Close()
		}
	}
	return false
}

func (c *tripConsumer) handleFindAndNotifyDrivers(ctx context.Context, payload messaging.TripEventData) error {
	if payload.Trip == nil || payload.Trip.Id == "" || payload.Trip.SelectedFare == nil {
		return nil
	}

	// 1. Check if the trip is still active/unassigned using simple HTTP
	url := fmt.Sprintf("%s/trips/%s", c.tripSvcURL, payload.Trip.Id)
	resp, err := http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var tripStatus tripStatusResponse
			if err := json.NewDecoder(resp.Body).Decode(&tripStatus); err == nil {
				if tripStatus.Driver != nil {
					log.Printf("Trip %s already has a driver assigned. Stopping search.", payload.Trip.Id)
					return nil
				}
				if tripStatus.Status == "completed" || tripStatus.Status == "cancelled" || tripStatus.Status == "accepted" {
					log.Printf("Trip %s is %s. Stopping search.", payload.Trip.Id, tripStatus.Status)
					return nil
				}
				if tripStatus.RideFare != nil && tripStatus.RideFare.TotalPriceInCents > payload.Trip.SelectedFare.TotalPriceInCents {
					log.Printf("Trip %s fare was increased (old: %v, new: %v). Throwing old request to dlq.", payload.Trip.Id, payload.Trip.SelectedFare.TotalPriceInCents, tripStatus.RideFare.TotalPriceInCents)
					return fmt.Errorf("outdated_fare")
				}
			}
		}
	} else {
		log.Printf("WARN: failed to check trip status: %v", err)
	}

	reqSeats := int32(1)
	if n := payload.Trip.SelectedFare.GetRequestedSeats(); n > 0 {
		reqSeats = n
	}
	allSuitableIDs := c.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug, reqSeats, payload.Trip.Route, len(payload.TriedDriverIDs))

	// Filter out already tried drivers AND check overlapping logic if they have active trips
	var suitableIDs []string
	triedMap := make(map[string]bool)
	for _, id := range payload.TriedDriverIDs {
		triedMap[id] = true
	}
	for _, id := range allSuitableIDs {
		if !triedMap[id] {
			if payload.Trip.SelectedFare.PackageSlug == "carpool" {
				if c.checkDriverOverlap(id, payload.Trip.Route) {
					suitableIDs = append(suitableIDs, id)
				}
			} else {
				suitableIDs = append(suitableIDs, id)
			}
		}
	}

	log.Printf("Found suitable drivers: current=%d, remaining=%d, tried=%d", len(allSuitableIDs), len(suitableIDs), len(payload.TriedDriverIDs))

	if len(suitableIDs) == 0 {
		return fmt.Errorf("exhausted_all_drivers")
	}

	// Get a random index from the matching drivers
	randomIndex := rand.Intn(len(suitableIDs))
	suitableDriverID := suitableIDs[randomIndex]

	// Add to tried list
	payload.TriedDriverIDs = append(payload.TriedDriverIDs, suitableDriverID)

	marshalledEventData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Notify the driver about a potential trip
	if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID,
		Data:    marshalledEventData,
	}); err != nil {
		log.Printf("Failed to publish message to exchange: %v", err)
		return err
	}

	// 120s TTL on FindAvailableDriversQueue handles the total timeout.
	// But we manually schedule a retry in 10s intervals across multiple drivers.
	if len(payload.TriedDriverIDs) < 12 { // Limit to 12 drivers (120s total at 10s intervals)
		if err := c.rabbitmq.PublishDelayMessage(ctx, contracts.AmqpMessage{
			Data: marshalledEventData,
		}); err != nil {
			log.Printf("Failed to schedule search retry: %v", err)
		}
	} else {
		log.Printf("Reached maximum driver notifications (12) for trip %s", payload.Trip.Id)
		// Return an error to force this message into the DLQ due to retries exhaustion
		return fmt.Errorf("max_driver_retries_reached")
	}

	return nil
}
