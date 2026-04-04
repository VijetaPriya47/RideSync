package auditconsumer

import (
	"context"
	"encoding/json"
	"log"

	"ride-sharing/services/user-auth-service/internal/repo"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Listen(rmq *messaging.RabbitMQ, rep *repo.Repo) error {
	return rmq.ConsumeMessages(messaging.AuditLogsQueue, func(ctx context.Context, msg amqp.Delivery) error {
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
		if err := rep.InsertAuditLog(ctx, pl.Method, pl.Path, pl.ActorUserID, pl.Role, pl.IP, detail); err != nil {
			log.Printf("audit consumer: insert: %v", err)
			return err
		}
		return nil
	})
}
