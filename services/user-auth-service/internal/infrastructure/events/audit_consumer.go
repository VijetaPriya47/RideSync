package events

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/services/user-auth-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type auditConsumer struct {
	rabbitmq *messaging.RabbitMQ
	svc      domain.AuthService
}

// NewAuditConsumer builds a consumer that persists gateway audit events.
func NewAuditConsumer(rabbitmq *messaging.RabbitMQ, svc domain.AuthService) *auditConsumer {
	return &auditConsumer{rabbitmq: rabbitmq, svc: svc}
}

// Listen subscribes to the audit logs queue.
func (c *auditConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.AuditLogsQueue, func(ctx context.Context, msg amqp.Delivery) error {
		var envelope contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &envelope); err != nil {
			log.Printf("audit consumer: envelope: %v", err)
			return err
		}
		var pl messaging.AuditLogPayload
		if err := json.Unmarshal(envelope.Data, &pl); err != nil {
			log.Printf("audit consumer: payload: %v", err)
			return err
		}
		detail := "{}"
		if b, err := json.Marshal(pl); err == nil {
			detail = string(b)
		}
		if err := c.svc.InsertAuditLog(ctx, pl.Method, pl.Path, pl.ActorUserID, pl.Role, pl.IP, detail); err != nil {
			log.Printf("audit consumer: insert: %v", err)
			return err
		}
		return nil
	})
}
