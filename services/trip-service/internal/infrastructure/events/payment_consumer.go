package events

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	"github.com/rabbitmq/amqp091-go"
)

type TripPaidNotifier interface {
	NotifyTripCompletedSeats(ctx context.Context, driverID, tripID string, seats int32)
}

type paymentConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
	paid     TripPaidNotifier
}

func NewPaymentConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService, paid TripPaidNotifier) *paymentConsumer {
	return &paymentConsumer{
		rabbitmq: rabbitmq,
		service:  service,
		paid:     paid,
	}
}

func (c *paymentConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.NotifyPaymentSuccessQueue, func(ctx context.Context, msg amqp091.Delivery) error {
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			return err
		}
		var payload messaging.PaymentStatusUpdateData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			log.Printf("Failed to unmarshal payload: %v", err)
			return err
		}

		log.Printf("Trip has been completed and payed.")

		trip, tripErr := c.service.GetTripByID(ctx, payload.TripID)
		if tripErr != nil {
			log.Printf("payment consumer: get trip: %v", tripErr)
		}

		if err := c.service.UpdateTrip(
			ctx,
			payload.TripID,
			"payed",
			nil,
		); err != nil {
			return err
		}

		if c.paid != nil && trip != nil && trip.Driver != nil {
			seats := int32(1)
			if trip.RideFare != nil && trip.RideFare.RequestedSeats > 0 {
				seats = trip.RideFare.RequestedSeats
			}
			c.paid.NotifyTripCompletedSeats(ctx, trip.Driver.ID, payload.TripID, seats)
		}

		return nil
	})
}
