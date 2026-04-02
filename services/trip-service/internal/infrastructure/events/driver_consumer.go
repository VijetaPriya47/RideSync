package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pbd "ride-sharing/shared/proto/driver"

	"github.com/rabbitmq/amqp091-go"
)

// SeatNotifier notifies driver-service when seats are reserved or released.
type SeatNotifier interface {
	NotifyTripAcceptedSeats(ctx context.Context, driverID, tripID string, seats int32)
}

func generateOTP() string {
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

type driverConsumer struct {
	rabbitmq     *messaging.RabbitMQ
	service      domain.TripService
	repo         domain.TripRepository
	seatNotifier SeatNotifier
}

func NewDriverConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService, repo domain.TripRepository, seatNotifier SeatNotifier) *driverConsumer {
	return &driverConsumer{
		rabbitmq:     rabbitmq,
		service:      service,
		repo:         repo,
		seatNotifier: seatNotifier,
	}
}

func (c *driverConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.DriverTripResponseQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		var payload messaging.DriverTripResponseData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}

		log.Printf("driver response received message: %+v", payload)

		switch msg.RoutingKey {
		case contracts.DriverCmdTripAccept:
			if err := c.handleTripAccepted(ctx, payload.TripID, payload.Driver); err != nil {
				log.Printf("Failed to handle the trip accept: %v", err)
				return err
			}
		case contracts.DriverCmdTripDecline:
			if err := c.handleTripDeclined(ctx, payload.TripID, payload.RiderID, payload.TriedDriverIDs); err != nil {
				log.Printf("Failed to handle the trip decline: %v", err)
				return err
			}
			return nil
		}
		log.Printf("unknown trip event: %+v", payload)

		return nil
	})
}

func (c *driverConsumer) handleTripDeclined(ctx context.Context, tripID, riderID string, triedDriverIDs []string) error {
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if trip == nil {
		return fmt.Errorf("trip not found: %s", tripID)
	}

	newPayload := messaging.TripEventData{
		Trip:           trip.ToProto(),
		TriedDriverIDs: triedDriverIDs,
	}

	marshalledPayload, err := json.Marshal(newPayload)
	if err != nil {
		return err
	}

	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverNotInterested,
		contracts.AmqpMessage{
			OwnerID: riderID,
			Data:    marshalledPayload,
		},
	); err != nil {
		return err
	}

	return nil
}

func (c *driverConsumer) handleTripAccepted(ctx context.Context, tripID string, driver *pbd.Driver) error {
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	if trip == nil {
		return fmt.Errorf("Trip was not found %s", tripID)
	}

	if err := c.service.UpdateTrip(ctx, tripID, "accepted", driver); err != nil {
		log.Printf("Failed to update the trip: %v", err)
		return err
	}

	trip, err = c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	seats := int32(1)
	if trip.RideFare != nil && trip.RideFare.RequestedSeats > 0 {
		seats = trip.RideFare.RequestedSeats
	}
	if c.seatNotifier != nil && driver != nil {
		c.seatNotifier.NotifyTripAcceptedSeats(ctx, driver.Id, tripID, seats)
	}

	// Generate and persist an OTP for this trip
	otp := generateOTP()
	if err := c.repo.SetTripOTP(ctx, tripID, otp); err != nil {
		log.Printf("WARN: failed to set OTP for trip %s: %v", tripID, err)
		// Non-fatal: continue
	}
	trip.OTP = otp

	marshalledTrip, err := json.Marshal(trip)
	if err != nil {
		return err
	}

	// Notify rider: driver assigned + OTP embedded
	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverAssigned, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    marshalledTrip,
	}); err != nil {
		return err
	}

	// NOTE: Payment session is now created AFTER the driver verifies the OTP via POST /trip/{id}/verify-otp
	return nil
}
