package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SubscribeJSON sets up a consumer that receives JSON messages from the given
// exchange/key and delivers deserialized values of type T to handler.
func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchangeName,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T),
) (*amqp.Channel, error) {

	// Declare the queue and bind it to the exchange with the routing key.
	channel, queue, err := DeclareAndBind(conn, exchangeName, queueName, key, queueType)
	if err != nil {
		return nil, err
	}

	// Start consuming messages from the queue.
	deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	// Deliver messages to the handler in a separate goroutine.
	go deliverMessage(deliveries, handler)
	return channel, nil
}

// deliverMessage reads from the AMQP deliveries channel, unmarshals each
// delivery body into type T, invokes handler, and acknowledges the delivery.
func deliverMessage[T any](deliveries <-chan amqp.Delivery, handler func(T)) {
	for delivery := range deliveries {
		var message T

		err := json.Unmarshal(delivery.Body, &message)
		if err != nil {
			// log the error and ACK to avoid requeues
			fmt.Printf("failed to unmarshal message body: %v — acking to discard\n", err)
			delivery.Ack(false)
			continue
		}

		handler(message)
		delivery.Ack(false)
	}
}
