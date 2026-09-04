package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string

const (
	DurableQueue   SimpleQueueType = "durable"
	TransientQueue SimpleQueueType = "transient"
)

type AckType string

const (
	Ack    AckType = "ack"
	Nack   AckType = "nack"
	Reject AckType = "reject"
)

type QueueOption func(amqp.Table)

func WithDeadLetterExchange(exchange string) QueueOption {
	return func(args amqp.Table) {
		args["x-dead-letter-exchange"] = exchange
	}
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	return publish(ch, exchange, key, val, "application/json", encodeJSON[T])
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	return publish(ch, exchange, key, val, "application/gob", encodeGob[T])
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
	prefetch int,
	handler func(T) AckType,
	opts ...QueueOption,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, prefetch, decodeJSON[T], handler, opts...)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
	prefetch int,
	handler func(T) AckType,
	opts ...QueueOption,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, prefetch, decodeGob[T], handler, opts...)
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
	opts ...QueueOption,
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("open channel: %w", err)
	}

	args := amqp.Table{}
	for _, opt := range opts {
		opt(args)
	}

	durable := queueType == DurableQueue
	autoDelete := queueType == TransientQueue
	exclusive := queueType == TransientQueue

	queue, err := channel.QueueDeclare(queueName, durable, autoDelete, exclusive, false, args)
	if err != nil {
		channel.Close()
		return nil, amqp.Queue{}, fmt.Errorf("declare queue %q: %w", queueName, err)
	}

	err = channel.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		channel.Close()
		return nil, amqp.Queue{}, fmt.Errorf("bind queue %q to %q with key %q: %w", queueName, exchange, key, err)
	}

	return channel, queue, nil
}

func publish[T any](
	ch *amqp.Channel,
	exchange, key string,
	val T,
	contentType string,
	encode func(T) ([]byte, error),
) error {
	body, err := encode(val)
	if err != nil {
		return fmt.Errorf("encode %s message: %w", contentType, err)
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType:  contentType,
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish to %q with key %q: %w", exchange, key, err)
	}
	return nil
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
	prefetch int,
	decode func([]byte, *T) error,
	handler func(T) AckType,
	opts ...QueueOption,
) error {
	channel, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType, opts...)
	if err != nil {
		return err
	}

	err = channel.Qos(prefetch, 0, false)
	if err != nil {
		channel.Close()
		return fmt.Errorf("set prefetch on %q: %w", queueName, err)
	}

	messages, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		channel.Close()
		return fmt.Errorf("consume from %q: %w", queueName, err)
	}

	go func() {
		defer channel.Close()
		for msg := range messages {
			var val T
			err := decode(msg.Body, &val)
			if err != nil {
				// Retrying will never make it decode, so drop it to the
				// dead-letter exchange rather than requeue it.
				log.Printf("discarding undecodable message from %s: %v", queueName, err)
				msg.Nack(false, false)
				continue
			}
			switch handler(val) {
			case Ack:
				msg.Ack(false)
			case Nack:
				msg.Nack(false, true)
			case Reject:
				msg.Reject(false)
			}
		}
	}()

	return nil
}

func encodeJSON[T any](val T) ([]byte, error) {
	return json.Marshal(val)
}

func encodeGob[T any](val T) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(val)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeJSON[T any](data []byte, val *T) error {
	return json.Unmarshal(data, val)
}

func decodeGob[T any](data []byte, val *T) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(val)
}
