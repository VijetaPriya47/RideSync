package main

import (
	"encoding/json"
	"log"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

// isDriverSearchTTLExpired returns true when the message was dead-lettered from
// find_available_drivers because it sat past x-message-ttl (driver search timeout).
func isDriverSearchTTLExpired(d amqp.Delivery) bool {
	raw, ok := d.Headers["x-death"]
	if !ok {
		return false
	}
	death, ok := raw.([]interface{})
	if !ok || len(death) == 0 {
		return false
	}
	first, ok := death[0].(amqp.Table)
	if !ok {
		return false
	}
	reason, _ := first["reason"].(string)
	queue, _ := first["queue"].(string)
	return (reason == "expired" || reason == "rejected") && queue == messaging.FindAvailableDriversQueue
}

func startDriverSearchExpiredConsumer(rb *messaging.RabbitMQ) {
	ch, err := rb.Conn.Channel()
	if err != nil {
		log.Printf("dlq consumer: channel: %v", err)
		return
	}

	msgs, err := ch.Consume(
		messaging.DeadLetterQueue,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("dlq consumer: consume: %v", err)
		_ = ch.Close()
		return
	}

	go func() {
		defer ch.Close()
		for msg := range msgs {
			if !isDriverSearchTTLExpired(msg) {
				continue
			}
			var body contracts.AmqpMessage
			if err := json.Unmarshal(msg.Body, &body); err != nil {
				log.Printf("dlq consumer: unmarshal: %v", err)
				continue
			}
			if body.OwnerID == "" {
				continue
			}
			ws := contracts.WSMessage{
				Type: contracts.TripEventNoDriversFound,
				Data: nil,
			}
			if err := connManager.SendMessage(body.OwnerID, ws); err != nil {
				log.Printf("dlq consumer: ws to %s: %v", body.OwnerID, err)
			}
		}
	}()
}
