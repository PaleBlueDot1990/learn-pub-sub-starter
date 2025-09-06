package pubsub

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// DeclareAndBind declares a queue on the provided AMQP connection and binds
// it to the specified exchange using the given routing key. It returns the
// opened channel and the declared queue.
func DeclareAndBind(
	conn *amqp.Connection,
	exchangeName,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {

	// Open a channel on the connection.
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	// Configure queue options based on the requested queue type.
	durable := queueType == Durable
	autoDelete := queueType == Transient
	exclusive := queueType == Transient

	// Declare the queue with the computed options.
	queue, err := channel.QueueDeclare(queueName, durable, autoDelete, exclusive, false, amqp.Table{
		"x-dead-letter-exchange" : "peril_dlx",
	})
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	// Bind the declared queue to the exchange with the routing key.
	err = channel.QueueBind(queue.Name, key, exchangeName, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return channel, queue, nil
}
