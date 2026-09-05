package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/SouaadMa/peril-rabbitmq/internal/config"
	"github.com/SouaadMa/peril-rabbitmq/internal/gamelogic"
	"github.com/SouaadMa/peril-rabbitmq/internal/pubsub"
	"github.com/SouaadMa/peril-rabbitmq/internal/routing"
	"github.com/SouaadMa/peril-rabbitmq/internal/topology"
)

func main() {
	err := run()
	if err != nil {
		log.Fatalf("peril client: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("Starting Peril client...")

	client, err := pubsub.Dial(cfg.AMQPURL)
	if err != nil {
		return fmt.Errorf("connect to broker: %w", err)
	}
	defer client.Close()
	fmt.Println("Connected successfully")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		return err
	}

	gameState := gamelogic.NewGameState(username)
	deadLetter := pubsub.WithDeadLetterExchange(routing.ExchangePerilDLX)
	var wg sync.WaitGroup

	err = pubsub.SubscribeJSON(
		ctx, &wg, client,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.TransientQueue,
		cfg.Prefetch,
		handlerPause(gameState),
		deadLetter,
	)
	if err != nil {
		return err
	}

	err = pubsub.SubscribeJSON(
		ctx, &wg, client,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+username,
		routing.ArmyMovesPrefix+".*",
		pubsub.TransientQueue,
		cfg.Prefetch,
		handlerMove(ctx, gameState, client),
		deadLetter,
	)
	if err != nil {
		return err
	}

	err = pubsub.SubscribeJSON(
		ctx, &wg, client,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.DurableQueue,
		cfg.Prefetch,
		handlerWar(ctx, gameState, client),
		deadLetter,
	)
	if err != nil {
		return err
	}

	client.Supervise(ctx, &wg, topology.Declare)

	runREPL(ctx, gameState, client, username)

	stop()
	wg.Wait()
	return nil
}

func runREPL(ctx context.Context, gs *gamelogic.GameState, c *pubsub.Client, username string) {
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
			case "spawn":
				err := gs.CommandSpawn(words)
				if err != nil {
					fmt.Println(err)
				}
			case "move":
				commandMove(ctx, gs, c, words, username)
			case "status":
				gs.CommandStatus()
			case "spam":
				commandSpam(ctx, c, words, username)
			case "help":
				gamelogic.PrintClientHelp()
			case "quit":
				gamelogic.PrintQuit()
				return
			default:
				fmt.Println("Invalid command")
			}
		}
	}
}

func commandMove(ctx context.Context, gs *gamelogic.GameState, c *pubsub.Client, words []string, username string) {
	move, err := gs.CommandMove(words)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = pubsub.PublishJSON(ctx, c, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+username, move)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Published move event")
}

func commandSpam(ctx context.Context, c *pubsub.Client, words []string, username string) {
	if len(words) < 2 {
		fmt.Println("usage: spam <n>")
		return
	}
	n, err := strconv.Atoi(words[1])
	if err != nil {
		fmt.Printf("%q is not a valid spam count\n", words[1])
		return
	}
	for range n {
		err := publishGameLog(ctx, c, routing.GameLog{
			CurrentTime: time.Now(),
			Message:     gamelogic.GetMaliciousLog(),
			Username:    username,
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	fmt.Printf("Published %d spam logs\n", n)
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(ctx context.Context, gs *gamelogic.GameState, c *pubsub.Client) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		switch gs.HandleMove(am) {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(ctx, c, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix+"."+gs.GetUsername(), gamelogic.RecognitionOfWar{
				Attacker: am.Player,
				Defender: gs.GetPlayerSnap(),
			})
			if err != nil {
				fmt.Println(err)
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

func handlerWar(ctx context.Context, gs *gamelogic.GameState, c *pubsub.Client) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(r gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(r)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.Nack
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.Reject
		case gamelogic.WarOutcomeOpponentWon, gamelogic.WarOutcomeYouWon:
			err := publishGameLog(ctx, c, routing.GameLog{
				CurrentTime: time.Now(),
				Username:    gs.GetUsername(),
				Message:     fmt.Sprintf("%s won a war against %s", winner, loser),
			})
			if err != nil {
				fmt.Println(err)
				return pubsub.Nack
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			err := publishGameLog(ctx, c, routing.GameLog{
				CurrentTime: time.Now(),
				Username:    gs.GetUsername(),
				Message:     fmt.Sprintf("The war between %s and %s resulted in a draw", winner, loser),
			})
			if err != nil {
				fmt.Println(err)
				return pubsub.Nack
			}
			return pubsub.Ack
		default:
			return pubsub.Nack
		}
	}
}

func publishGameLog(ctx context.Context, c *pubsub.Client, gl routing.GameLog) error {
	return pubsub.PublishGob(ctx, c, routing.ExchangePerilTopic, routing.GameLogSlug+"."+gl.Username, gl)
}
