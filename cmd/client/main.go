package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/config"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err)
		return
	}
	connection_string := cfg.AMQPURL
	fmt.Println("Starting Peril client...")

	connection, err := amqp.Dial(connection_string)
	if err != nil {
		fmt.Println("Failed to connect")
		return
	}
	fmt.Print("Connected successfully")
	defer connection.Close()

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		fmt.Println("Failed to welcome")
		return
	}

	channel, err := connection.Channel()
	if err != nil {
		fmt.Println("Failed to create a channel")
		return
	}
	fmt.Println("Created a channel successfully")
	defer channel.Close()

	pauseQueueName := routing.PauseKey + "." + username
	gameState := gamelogic.NewGameState(username)

	deadLetter := pubsub.WithDeadLetterExchange(routing.ExchangePerilDLX)

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilDirect, pauseQueueName, routing.PauseKey, pubsub.TransientQueue, cfg.Prefetch, handlerPause(gameState), deadLetter)
	if err != nil {
		fmt.Println(err)
		return
	}

	armyMovesQueueName := routing.ArmyMovesPrefix + "." + username
	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, armyMovesQueueName, routing.ArmyMovesPrefix+".*", pubsub.TransientQueue, cfg.Prefetch, handlerMove(gameState, channel), deadLetter)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, routing.WarRecognitionsPrefix+".*", pubsub.DurableQueue, cfg.Prefetch, handlerWar(gameState, channel), deadLetter)
	if err != nil {
		fmt.Println(err)
		return
	}

REPL:
	for {
		args := gamelogic.GetInput()
		if len(args) == 0 {
			continue
		}
		first := args[0]
		switch first {
		case "spawn":
			gameState.CommandSpawn(args)
		case "move":
			am, err := gameState.CommandMove(args)
			if err != nil {
				fmt.Println("Failed to move")
				continue
			}
			err = pubsub.PublishJSON(channel, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, am)
			if err != nil {
				fmt.Println("Failed to publish move")
				continue
			}
			fmt.Print("Published move event")
		case "status":
			gameState.CommandStatus()
		case "spam":
			second := args[1]
			n, err := strconv.Atoi(second)
			if err != nil {
				fmt.Println("Invalid spam count")
				continue
			}
			for range n {
				mlog := gamelogic.GetMaliciousLog()
				err = pubsub.PublishGob(channel, routing.ExchangePerilTopic, routing.GameLogSlug+"."+username, routing.GameLog{
					CurrentTime: time.Now(),
					Message:     mlog,
					Username:    username,
				})
				if err != nil {
					fmt.Println("Failed to publish spam")
					continue
				}
				fmt.Print("Published spam event")
			}
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			break REPL
		default:
			fmt.Println("Invalid command")
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		switch gs.HandleMove(am) {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix+"."+gs.GetUsername(), gamelogic.RecognitionOfWar{
				Attacker: am.Player,
				Defender: gs.GetPlayerSnap(),
			})
			if err != nil {
				fmt.Println("Failed to publish war move")
				return pubsub.Nack
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.Reject
		default:
			return pubsub.Reject
		}
	}
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(r gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(r)
		fmt.Println(outcome, winner, loser)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.Nack
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.Reject
		case gamelogic.WarOutcomeOpponentWon:
			log := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishGameLog(ch, routing.GameLog{Username: gs.GetUsername(), Message: log})
			if err != nil {
				fmt.Println("Failed to publish war log")
				return pubsub.Nack
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			log := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishGameLog(ch, routing.GameLog{Username: gs.GetUsername(), Message: log})
			if err != nil {
				fmt.Println("Failed to publish war log")
				return pubsub.Nack
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			log := fmt.Sprintf("The war between %s and %s resulted in a draw", winner, loser)
			err := publishGameLog(ch, routing.GameLog{Username: gs.GetUsername(), Message: log})
			if err != nil {
				fmt.Println("Failed to publish war log")
				return pubsub.Nack
			}
			return pubsub.Ack
		default:
			return pubsub.Nack
		}
	}
}

func publishGameLog(ch *amqp.Channel, gl routing.GameLog) error {
	err := pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gl.Username, gl)
	if err != nil {
		return err
	}
	return nil
}
