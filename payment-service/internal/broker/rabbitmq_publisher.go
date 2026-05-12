package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueName    = "payment.completed"
	DLXName      = "payment.dlx"
	DLQName      = "payment.dead"
	ExchangeName = ""
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[RabbitMQ] connect attempt %d/10 failed: %v – retrying in 3s", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		err := conn.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		DLXName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, fmt.Errorf("declare dlx: %w", err)
	}

	if _, err := ch.QueueDeclare(
		QueueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange": DLXName,
		},
	); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	if err := ch.QueueBind(DLQName, "", DLXName, false, nil); err != nil {
		return nil, fmt.Errorf("bind dlq: %w", err)
	}

	if _, err := ch.QueueDeclare(
		QueueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-queue-type":           "quorum",
			"x-dead-letter-exchange": DLXName,
			"x-delivery-limit":       int32(3),
		},
	); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	log.Printf("[RabbitMQ] publisher ready – queue=%s dlq=%s", QueueName, DLQName)
	return &RabbitMQPublisher{conn: conn, channel: ch}, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, event PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	err = p.channel.PublishWithContext(ctx,
		ExchangeName,
		QueueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.EventID,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	log.Printf("[RabbitMQ] published event_id=%s order_id=%s status=%s",
		event.EventID, event.OrderID, event.Status)
	return nil
}

func (p *RabbitMQPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}
