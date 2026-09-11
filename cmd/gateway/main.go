package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/SouaadMa/peril-rabbitmq/internal/config"
	"github.com/SouaadMa/peril-rabbitmq/internal/gamelogic"
	"github.com/SouaadMa/peril-rabbitmq/internal/gateway"
	"github.com/SouaadMa/peril-rabbitmq/internal/pubsub"
	"github.com/SouaadMa/peril-rabbitmq/internal/routing"
	"github.com/SouaadMa/peril-rabbitmq/internal/topology"
	"github.com/SouaadMa/peril-rabbitmq/internal/world"
)

func main() {
	err := run()
	if err != nil {
		log.Fatalf("peril gateway: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := pubsub.Dial(cfg.AMQPURL)
	if err != nil {
		return err
	}
	defer client.Close()

	err = topology.Declare(client.Channel())
	if err != nil {
		return err
	}

	w := world.New(cfg.LogLimit)
	var wg sync.WaitGroup

	err = subscribe(ctx, &wg, client, w, cfg.Prefetch)
	if err != nil {
		return err
	}

	client.Supervise(ctx, &wg, topology.Declare)
	dropStale(ctx, &wg, w, cfg.PlayerTTL)

	err = gateway.Serve(ctx, &wg, w, cfg.GatewayAddr)
	if err != nil {
		return err
	}

	<-ctx.Done()
	wg.Wait()
	log.Println("peril gateway stopped")
	return nil
}

func subscribe(ctx context.Context, wg *sync.WaitGroup, c *pubsub.Client, w *world.World, prefetch int) error {
	deadLetter := pubsub.WithDeadLetterExchange(routing.ExchangePerilDLX)

	err := pubsub.SubscribeJSON(
		ctx, wg, c,
		routing.ExchangePerilTopic,
		"gateway_state",
		routing.PlayerStatePrefix+".*",
		pubsub.TransientQueue,
		prefetch,
		func(p gamelogic.Player) pubsub.AckType {
			w.ApplyPlayerState(p, time.Now())
			return pubsub.Ack
		},
		deadLetter,
	)
	if err != nil {
		return err
	}

	err = pubsub.SubscribeJSON(
		ctx, wg, c,
		routing.ExchangePerilTopic,
		"gateway_moves",
		routing.ArmyMovesPrefix+".*",
		pubsub.TransientQueue,
		prefetch,
		func(m gamelogic.ArmyMove) pubsub.AckType {
			w.ApplyMove(m, time.Now())
			return pubsub.Ack
		},
		deadLetter,
	)
	if err != nil {
		return err
	}

	err = pubsub.SubscribeJSON(
		ctx, wg, c,
		routing.ExchangePerilTopic,
		"gateway_wars",
		routing.WarRecognitionsPrefix+".*",
		pubsub.TransientQueue,
		prefetch,
		func(r gamelogic.RecognitionOfWar) pubsub.AckType {
			result, fought := w.ApplyWar(r, time.Now())
			if fought {
				log.Printf("war in %s: %s beat %s", result.Location, result.Winner, result.Loser)
			}
			return pubsub.Ack
		},
		deadLetter,
	)
	if err != nil {
		return err
	}

	return pubsub.SubscribeGob(
		ctx, wg, c,
		routing.ExchangePerilTopic,
		"gateway_logs",
		routing.GameLogSlug+".*",
		pubsub.TransientQueue,
		prefetch,
		func(gl routing.GameLog) pubsub.AckType {
			w.ApplyLog(world.LogEntry{
				Time:     gl.CurrentTime,
				Username: gl.Username,
				Message:  gl.Message,
			})
			return pubsub.Ack
		},
		deadLetter,
	)
}

func dropStale(ctx context.Context, wg *sync.WaitGroup, w *world.World, ttl time.Duration) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(ttl / 3)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				for _, username := range w.DropStale(now, ttl) {
					log.Printf("player %s went quiet", username)
				}
			}
		}
	}()
}
