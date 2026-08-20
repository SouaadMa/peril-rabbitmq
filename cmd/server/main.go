package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	//TODO: Declare this in env
	connection_string := "amqp://guest:guest@localhost:5672/"
	fmt.Println("Starting Peril server...")

	connection, err := amqp.Dial(connection_string)
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

	err = channel.ExchangeDeclare("peril_dlx", "fanout", true, false, false, false, nil)
	if err != nil {
		fmt.Println("Failed to declare dlx")
		return
	}

	_, err = channel.QueueDeclare("peril_dlq", true, false, false, false, nil)
	if err != nil {
		fmt.Println("Failed to declare dlq")
		return
	}

	err = channel.QueueBind("peril_dlq", "", "peril_dlx", false, nil)
	if err != nil {
		fmt.Println("Failed to bind dlq")
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
		func(log routing.GameLog) pubsub.AckType {
			defer fmt.Println(">")
			gamelogic.WriteLog(log)
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
