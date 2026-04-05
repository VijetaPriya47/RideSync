package events

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/services/platform-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type paymentConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.FinanceService
}

// NewPaymentConsumer consumes payment success messages for the finance ledger.
func NewPaymentConsumer(rabbitmq *messaging.RabbitMQ, svc domain.FinanceService) *paymentConsumer {
	return &paymentConsumer{rabbitmq: rabbitmq, service: svc}
}

// Listen starts consuming from the finance payment success queue.
func (c *paymentConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.FinancePaymentSuccessQueue, func(ctx context.Context, msg amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &envelope); err != nil {
			log.Printf("finance payment consumer: unmarshal envelope: %v", err)
			return err
		}
		var pay messaging.PaymentStatusUpdateData
		if err := json.Unmarshal(envelope.Data, &pay); err != nil {
			log.Printf("finance payment consumer: unmarshal payment: %v", err)
			return err
		}
		if err := c.service.InsertPaymentDebit(ctx, pay.UserID, pay.AmountCents, pay.Currency, pay.Region, pay.TripID); err != nil {
			log.Printf("finance payment consumer: insert: %v", err)
			return err
		}
		return nil
	})
}
