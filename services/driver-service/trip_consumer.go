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
			var tripStatus struct {
				Status string `json:"status"`
				Driver interface{} `json:"driver"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&tripStatus); err == nil {
				if tripStatus.Driver != nil {
					log.Printf("Trip %s already has a driver assigned. Stopping search.", payload.Trip.Id)
					return nil
				}
				if tripStatus.Status == "completed" || tripStatus.Status == "cancelled" {
					log.Printf("Trip %s is %s. Stopping search.", payload.Trip.Id, tripStatus.Status)
					return nil
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
	allSuitableIDs := c.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug, reqSeats, payload.Trip.Route)

	// Filter out already tried drivers
	var suitableIDs []string
	triedMap := make(map[string]bool)
	for _, id := range payload.TriedDriverIDs {
		triedMap[id] = true
	}
	for _, id := range allSuitableIDs {
		if !triedMap[id] {
			suitableIDs = append(suitableIDs, id)
		}
	}

	log.Printf("Found suitable drivers: current=%d, remaining=%d, tried=%d", len(allSuitableIDs), len(suitableIDs), len(payload.TriedDriverIDs))

	if len(suitableIDs) == 0 {
		// No more untried suitable drivers found. Notify the rider.
		if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("Failed to publish message to exchange: %v", err)
			return err
		}
		return nil
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
		// We reached the retry limit. Notify the rider that no drivers were found.
		if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound, contracts.AmqpMessage{
			OwnerID: payload.Trip.UserID,
		}); err != nil {
			log.Printf("Failed to publish message to exchange: %v", err)
			return err
		}
	}

	return nil
}
