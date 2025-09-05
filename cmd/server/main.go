package main

import (
	"fmt"
	pubsub "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	routing "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	gamelogic "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"

	amqp "github.com/rabbitmq/amqp091-go"
)

// pausePublish is the message sent to clients to indicate the game is paused.
var pausePublish = routing.PlayingState{
	IsPaused: true,
}

// resumePublish is the message sent to clients to indicate the game is resumed.
var resumePublish = routing.PlayingState{
	IsPaused: false,
}

func main() {
	fmt.Println("Starting Peril server...")

	// Connect to RabbitMQ.
	connectionString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		fmt.Printf("failed to connect to RabbitMQ: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("connected to RabbitMQ at %s\n", connectionString)

	// Open a channel on the connection.
	channel, err := conn.Channel()
	if err != nil {
		fmt.Printf("failed to open channel: %v\n", err)
		return
	}
	defer channel.Close()
	fmt.Printf("channel opened on RabbitMQ connection\n")

	// Declare a durable queue for game logs.
	queue, err := channel.QueueDeclare(routing.GameLogSlug, true, false, false, false, nil)
	if err != nil {
		fmt.Printf("failed to declare queue %s: %v\n", routing.GameLogSlug, err)
		return
	}

	// Bind the queue to the topic exchange using the routing key pattern.
	bindingKey := routing.GameLogSlug + ".*"
	err = channel.QueueBind(queue.Name, bindingKey, routing.ExchangePerilTopic, false, nil)
	if err != nil {
		fmt.Printf("failed to bind queue %s to exchange %s with key %s: %v\n", queue.Name, routing.ExchangePerilTopic, bindingKey, err)
		return
	}

	fmt.Printf("queue declared and bound successfully\n")
	fmt.Printf("|Exchange: %s| -> |Binding Key: %s| -> |Queue: %s|\n", routing.ExchangePerilTopic, bindingKey, queue.Name)

	// Print REPL help and start accepting commands.
	gamelogic.PrintServerHelp()

	// REPL loop: accept commands and publish pause/resume messages.
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "pause":
			// Publish a pause message to all clients.
			fmt.Printf("publishing pause message to clients\n")
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, pausePublish)
			if err != nil {
				fmt.Printf("failed to publish pause message: %v\n", err)
				return
			}
			fmt.Printf("published to exchange %s\n", routing.ExchangePerilDirect)

		case "resume":
			// Publish a resume message to all clients.
			fmt.Printf("publishing resume message to clients\n")
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, resumePublish)
			if err != nil {
				fmt.Printf("failed to publish resume message: %v\n", err)
				return
			}
			fmt.Printf("published to exchange %s\n", routing.ExchangePerilDirect)

		case "quit":
			fmt.Printf("exiting REPL\n")
			return 

		default:
			fmt.Printf("unrecognized command: %s\n", words[0])
		}
	}
}
