package main

import (
	"fmt"
	"strconv"
	"time"

	gamelogic "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	pubsub "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	routing "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

var conn *amqp.Connection

func main() {
	var err error 
	fmt.Println("Starting Peril client...")

	// Connect to RabbitMQ.
	connectionString := "amqp://guest:guest@localhost:5672/"
	conn, err = amqp.Dial(connectionString)
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
	
	// Create a new local game state for this client.
	gameState := gamelogic.NewGameState(userName)

	// Create a transient queue that subscribes to pause/resume messages.
	queueName := routing.PauseKey + "." + userName
	key := routing.PauseKey
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		key,
		pubsub.Transient,
		handlerPause(gameState),
	)
	if err != nil {
		fmt.Printf("failed to subscribe to RabbitMQ: %v\n", err)
		return
	}

	// Create a transient queue that subscribes to move messages.
	queueName2 := routing.ArmyMovesPrefix + "." + userName
    key2 := routing.ArmyMovesPrefix + ".*"
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		queueName2,
		key2,
		pubsub.Transient,
		handlerMove(gameState),
	)
	if err != nil {
		fmt.Printf("failed to subscribe to RabbitMQ: %v\n", err)
		return
	}

	// Create a durable queue that subscribes to war messages.
	queueName3 := routing.WarRecognitionsPrefix
    key3 := routing.WarRecognitionsPrefix + ".*"
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		queueName3,
		key3,
		pubsub.Durable,
		handlerWar(gameState),
	)
	if err != nil {
		fmt.Printf("failed to subscribe to RabbitMQ: %v\n", err)
		return
	}

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
			move, _ := gameState.CommandMove(words)
			channel, _ := conn.Channel()
			key := routing.ArmyMovesPrefix + "." + userName
			pubsub.PublishJSON(channel, routing.ExchangePerilTopic, key, move)

		case "status":
			gameState.CommandStatus()

		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			if len(words) == 1 {
				continue
			} 
			n, err := strconv.Atoi(words[1])
			if err != nil {
				continue 
			}
			for range n {
				mallog := gamelogic.GetMaliciousLog()
				err = publishLog(gameState, mallog)
				if err != nil {
					fmt.Printf("error publishing spam message\n")
				}
			}
		case "quit":
			gamelogic.PrintQuit()
			return

		default:
			fmt.Printf("unrecognized command: %s\n", words[0])
		}
	}
}

// Handler function to execute when pause/resume messages are consumed. 
func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
    return func(s routing.PlayingState) pubsub.AckType {
        gs.HandlePause(s)     
        fmt.Print("> ")  
	    return pubsub.Ack
    }
}

// Handler function to execute when move messages are consumed. 
func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
    return func(move gamelogic.ArmyMove) pubsub.AckType {
        moveoutCome := gs.HandleMove(move)     
        fmt.Print("> ")  
		
		switch moveoutCome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack

		case gamelogic.MoveOutcomeMakeWar:
			war := gamelogic.RecognitionOfWar{
				Attacker: move.Player,
				Defender: gs.Player,
			}
			channel, _ := conn.Channel()
			key := routing.WarRecognitionsPrefix + "." + gs.GetUsername()
			err := pubsub.PublishJSON(channel, routing.ExchangePerilTopic, key, war)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack

		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard

		default:
			return pubsub.NackDiscard
		}
    }
}

// Handler function to execute when war messages are consumed. 
func handlerWar(gs *gamelogic.GameState) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(recWar gamelogic.RecognitionOfWar) pubsub.AckType {
		outcome, winner, loser := gs.HandleWar(recWar)
		fmt.Print("> ")

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue

		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard

		case gamelogic.WarOutcomeOpponentWon:
			logMessage := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishLog(gs, logMessage)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack

		case gamelogic.WarOutcomeYouWon:
			logMessage := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishLog(gs, logMessage)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack

		case gamelogic.WarOutcomeDraw:
			logMessage := fmt.Sprintf("%s and %s resulted in a draw", winner, loser)
			err := publishLog(gs, logMessage)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
			
		default:
			fmt.Print("war outcome is unidentfied\n")
			return pubsub.NackDiscard
		}
	}
}


func publishLog(gs *gamelogic.GameState, logMessage string) error {
	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	log := routing.GameLog{
		CurrentTime: time.Now(),
		Message: logMessage,
		Username: gs.GetUsername(),
	}

	key := routing.GameLogSlug + "." + gs.GetUsername()
	return pubsub.PublishGob(channel, routing.ExchangePerilTopic, key, log)
}