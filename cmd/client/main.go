package main

import (
	"fmt"

	gamelogic "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	pubsub "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	routing "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	// Connect to RabbitMQ.
	connectionString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		fmt.Printf("failed to connect to RabbitMQ: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("connected to RabbitMQ at %s\n", connectionString)

	// Get the player's username.
	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Printf("failed to get username: %v\n", err)
		return
	}
	queueName := routing.PauseKey + "." + userName

	// Create a new local game state for this client.
	gameState := gamelogic.NewGameState(userName)

	// Subscribe to direct exchange for pause/resume messages (transient consumer).
	channel, err := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gameState),
	)
	if err != nil {
		fmt.Printf("failed to subscribe to RabbitMQ: %v\n", err)
		return
	}
	defer channel.Close()

	// Print REPL help and start accepting commands.
	gamelogic.PrintClientHelp()

	// REPL loop: read commands and dispatch to game state handlers.
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			gameState.CommandSpawn(words)
		case "move":
			gameState.CommandMove(words)
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Printf("spamming not allowed\n")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Printf("unrecognized command: %s\n", words[0])
		}
	}
}

// handlerPause returns a function compatible with SubscribeJSON's handler signature.
// The deferred fmt.Print call restores the REPL prompt after handling a pause message.
func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
    return func(s routing.PlayingState) {
        gs.HandlePause(s)     
        fmt.Print("> ")       
    }
}