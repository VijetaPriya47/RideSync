package consumer

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/services/finance-service/internal/repo"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ListenPaymentSuccess(rmq *messaging.RabbitMQ, rep *repo.Repo) error {
	return rmq.ConsumeMessages(messaging.FinancePaymentSuccessQueue, func(ctx context.Context, msg amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &envelope); err != nil {
			log.Printf("finance consumer: unmarshal envelope: %v", err)
			return err
		}
		var pay messaging.PaymentStatusUpdateData
		if err := json.Unmarshal(envelope.Data, &pay); err != nil {
			log.Printf("finance consumer: unmarshal payment: %v", err)
			return err
		}
		if err := rep.InsertPaymentDebit(ctx, pay.UserID, pay.AmountCents, pay.Currency, pay.Region, pay.TripID); err != nil {
			log.Printf("finance consumer: insert: %v", err)
			return err
		}
		return nil
	})
}
