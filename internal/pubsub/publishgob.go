package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"

	amqp "github.com/rabbitmq/amqp091-go"
)

// PublishGob encodes val to Gob and publishes it to the given exchange/key
// using the provided AMQP channel. The publish is performed with a background
// context.
func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {

	// Encode the value to Gob.
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(val)
	if err != nil {
		return err
	}

	// Build the AMQP publishing message.
	publishVal := amqp.Publishing{
		ContentType: "application/gob",
		Body:        buf.Bytes(),
	}

	// Publish the message to the exchange with the routing key.
	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, publishVal)
	return err
}
