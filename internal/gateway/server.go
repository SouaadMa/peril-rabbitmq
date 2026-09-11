package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/SouaadMa/peril-rabbitmq/internal/world"
)

const shutdownTimeout = 5 * time.Second

func Serve(ctx context.Context, wg *sync.WaitGroup, w *world.World, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(rw).Encode(w.Snapshot())
		if err != nil {
			log.Printf("encode snapshot: %v", err)
		}
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	srv := &http.Server{Handler: mux}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := srv.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			log.Printf("http shutdown: %v", err)
		}
	}()

	log.Printf("gateway listening on %s", listener.Addr())
	return nil
}
