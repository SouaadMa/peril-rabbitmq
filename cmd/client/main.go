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

	queueName := routing.PauseKey + "." + username
	fmt.Println("Welcome,", queueName)

	_, _, err = pubsub.DeclareAndBind(connection, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.TransientQueue)
	if err != nil {
		fmt.Println("Failed to declare and bind")
		return
	}

	gameState := gamelogic.NewGameState(username)

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
			gameState.CommandMove(args)
		case "status":
			gameState.CommandStatus()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
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
