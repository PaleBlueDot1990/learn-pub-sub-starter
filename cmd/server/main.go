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

var conn *amqp.Connection

func main() {
	var err error 
	fmt.Println("Starting Peril server...")

	// Connect to RabbitMQ.
	connectionString := "amqp://guest:guest@localhost:5672/"
	conn, err = amqp.Dial(connectionString)
	if err != nil {
		fmt.Printf("failed to connect to RabbitMQ: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("connected to RabbitMQ at %s\n", connectionString)

	// Create a durable queue that subscribes to log messages.
	queueName := routing.GameLogSlug
	key := routing.GameLogSlug + ".*"
	err = pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		queueName,
		key,
		pubsub.Durable,
		handlerLog(),
	)
	if err != nil {
		fmt.Printf("failed to subscribe to RabbitMQ: %v\n", err)
		return
	}

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
		    channel, _ := conn.Channel()
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, pausePublish)
			if err != nil {
				fmt.Printf("failed to publish pause message: %v\n", err)
				return
			}
			fmt.Printf("published to exchange %s\n", routing.ExchangePerilDirect)

		case "resume":
			// Publish a resume message to all clients.
			fmt.Printf("publishing resume message to clients\n")
			channel, _ := conn.Channel()
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

func handlerLog() func(gamelog routing.GameLog) pubsub.AckType {
	return func(gamelog routing.GameLog) pubsub.AckType {
		defer fmt.Print("> ")

		err := gamelogic.WriteLog(gamelog)
		if err != nil {
			fmt.Printf("error writing log: %v\n", err)
			return pubsub.NackRequeue
		}
		return pubsub.Ack
	}
}