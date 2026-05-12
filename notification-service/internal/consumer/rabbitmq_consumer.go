package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"notification-service/internal/worker"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueName = "payment.completed"
	DLXName   = "payment.dlx"
	DLQName   = "payment.dead"
)

type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	w       *worker.NotificationWorker
}

func New(url string, w *worker.NotificationWorker) (*RabbitMQConsumer, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[Consumer] connect attempt %d/10 failed: %v – retrying in 3s", i+1, err)
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

	if err := ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("qos: %w", err)
	}

	if err := ch.ExchangeDeclare(DLXName, "fanout", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare dlx: %w", err)
	}
	if _, err := ch.QueueDeclare(DLQName, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare dlq: %w", err)
	}
	if err := ch.QueueBind(DLQName, "", DLXName, false, nil); err != nil {
		return nil, fmt.Errorf("bind dlq: %w", err)
	}

	if _, err := ch.QueueDeclare(
		QueueName, true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange": DLXName,
		},
	); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	log.Printf("[Consumer] connected – queue=%s dlq=%s", QueueName, DLQName)
	return &RabbitMQConsumer{conn: conn, channel: ch, w: w}, nil
}

func (c *RabbitMQConsumer) Consume(ctx context.Context) error {
	msgs, err := c.channel.Consume(QueueName, "notification-consumer", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Println("[Consumer] waiting for messages…")

	for {
		select {
		case <-ctx.Done():
			log.Println("[Consumer] context cancelled – stopping")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				log.Println("[Consumer] channel closed")
				return nil
			}

			var event worker.PaymentEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("[Consumer] malformed message, discarding: %v", err)
				_ = msg.Ack(false)
				continue
			}

			if err := c.w.Process(ctx, event); err != nil {
				log.Printf("[Consumer] NACK event=%s: %v", event.EventID, err)
				_ = msg.Nack(false, false)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

func (c *RabbitMQConsumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return err
	}
	return c.conn.Close()
}
