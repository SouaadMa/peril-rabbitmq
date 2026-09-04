package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/SouaadMa/peril-rabbitmq/internal/config"
	"github.com/SouaadMa/peril-rabbitmq/internal/gamelogic"
	"github.com/SouaadMa/peril-rabbitmq/internal/pubsub"
	"github.com/SouaadMa/peril-rabbitmq/internal/routing"
	"github.com/SouaadMa/peril-rabbitmq/internal/topology"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	err := run()
	if err != nil {
		log.Fatalf("peril server: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("Starting Peril server...")

	connection, err := amqp.Dial(cfg.AMQPURL)
	if err != nil {
		return fmt.Errorf("connect to broker: %w", err)
	}
	defer connection.Close()
	fmt.Println("Connected successfully")

	channel, err := connection.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer channel.Close()

	err = topology.Declare(channel)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	err = pubsub.SubscribeGob(
		ctx,
		&wg,
		connection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.DurableQueue,
		cfg.Prefetch,
		func(gameLog routing.GameLog) pubsub.AckType {
			defer fmt.Print("> ")
			err := gamelogic.WriteLog(gameLog, cfg.WriteLogWait)
			if err != nil {
				fmt.Println(err)
				return pubsub.Nack
			}
			return pubsub.Ack
		},
		pubsub.WithDeadLetterExchange(routing.ExchangePerilDLX),
	)
	if err != nil {
		return err
	}

	gamelogic.PrintServerHelp()
	runREPL(ctx, channel)

	stop()
	wg.Wait()
	fmt.Println("Peril server stopped")
	return nil
}

func runREPL(ctx context.Context, ch *amqp.Channel) {
	lines := gamelogic.InputLines(ctx)
	for {
		select {
		case <-ctx.Done():
			fmt.Println()
			return
		case words, ok := <-lines:
			if !ok {
				return
			}
			switch words[0] {
			case "pause":
				fmt.Println("Pausing the game")
				publishPauseState(ctx, ch, true)
			case "resume":
				fmt.Println("Resuming the game")
				publishPauseState(ctx, ch, false)
			case "quit":
				fmt.Println("Quitting the game")
				return
			default:
				fmt.Println("Invalid command")
			}
		}
	}
}

func publishPauseState(ctx context.Context, ch *amqp.Channel, paused bool) {
	err := pubsub.PublishJSON(ctx, ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: paused,
	})
	if err != nil {
		fmt.Println(err)
	}
}
