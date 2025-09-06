package pubsub

import (
	"bytes"
	"encoding/gob"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SubscribeGob sets up a consumer that receives Gob messages from the given
// exchange/key and delivers deserialized values of type T to handler.
func SubscribeGob[T any](
	conn *amqp.Connection,
	exchangeName,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {

	// Declare the queue and bind it to the exchange with the routing key.
	channel, queue, err := DeclareAndBind(conn, exchangeName, queueName, key, queueType)
	if err != nil {
		return err
	} 

	//Limiting prefetch count to 10 
	err = channel.Qos(10, 0, false)
	if err != nil {
		return err
	}

	// Start consuming messages from the queue.
	deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	// Deliver messages to the handler in a separate goroutine.
	go deliverMessageGob(deliveries, handler)
	return nil
}

// deliverMessage reads from the AMQP deliveries channel, deserializes each
// delivery body into type T, invokes handler, and acknowledges the delivery.
func deliverMessageGob[T any](deliveries <-chan amqp.Delivery, handler func(T) AckType) {
	for delivery := range deliveries {
		var buf bytes.Buffer
		buf.Write(delivery.Body)
		dec := gob.NewDecoder(&buf)

		var message T
		err := dec.Decode(&message)
		if err != nil {
			// log the error and ACK to avoid requeues
			fmt.Printf("failed to decode message body: %v — acking to discard\n", err)
			delivery.Ack(false)
			continue
		}

		acktype := handler(message)
		
		switch acktype {
		case Ack:
			delivery.Ack(false)
		case NackRequeue:
			delivery.Nack(false, true)
		case NackDiscard:
			delivery.Nack(false, false)
		}
	}
}