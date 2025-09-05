package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

// PublishJSON marshals val to JSON and publishes it to the given exchange/key
// using the provided AMQP channel. The publish is performed with a background
// context.
func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {

	// Marshal the value to JSON.
	jsonVal, err := json.Marshal(val)
	if err != nil {
		return err
	}

	// Build the AMQP publishing message.
	publishVal := amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonVal,
	}

	// Publish the message to the exchange with the routing key.
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, publishVal)
	return err
}
