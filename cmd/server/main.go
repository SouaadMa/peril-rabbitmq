package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/config"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/topology"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Starting Peril server...")

	connection, err := amqp.Dial(cfg.AMQPURL)
	if err != nil {
		fmt.Println("Failed to connect")
		return
	}
	fmt.Println("Connected successfully")
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		fmt.Println("Failed to create a channel")
	}
	fmt.Println("Created a channel successfully")
	defer channel.Close()

	err = topology.Declare(channel)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, _, err = pubsub.DeclareAndBind(connection, routing.ExchangePerilTopic, routing.GameLogSlug, "game_logs.*", pubsub.DurableQueue)
	if err != nil {
		fmt.Println("Failed to declare and bind queue")
	}
	fmt.Println("Declared and binded queue game_logs.*")

	err = pubsub.SubscribeGob(
		connection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		"game_logs.*",
		pubsub.DurableQueue,
		func(gameLog routing.GameLog) pubsub.AckType {
			defer fmt.Println(">")
			err := gamelogic.WriteLog(gameLog, cfg.WriteLogWait)
			if err != nil {
				fmt.Println(err)
				return pubsub.Nack
			}
			return pubsub.Ack
		},
	)
	if err != nil {
		fmt.Println("Failed to subscribe to queue")
	}

	gamelogic.PrintServerHelp()

REPL:
	for {
		args := gamelogic.GetInput()
		if len(args) == 0 {
			continue
		}
		first := args[0]
		switch first {
		case "pause":
			fmt.Println("Pausing the game")
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
		case "resume":
			fmt.Println("Resuming the game")
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
		case "quit":
			fmt.Println("Quitting the game")
			break REPL
		default:
			fmt.Println("Invalid command")
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("Received ctrl+c... shutting down")
}
