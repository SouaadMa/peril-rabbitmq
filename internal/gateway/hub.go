package gateway

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/SouaadMa/peril-rabbitmq/internal/world"
	"github.com/coder/websocket"
)

const (
	broadcastInterval = 100 * time.Millisecond
	sendBuffer        = 8
	writeTimeout      = 5 * time.Second
)

type Hub struct {
	world *world.World

	register   chan *conn
	unregister chan *conn
	notify     chan struct{}
	done       chan struct{}
}

type conn struct {
	send chan []byte
}

func NewHub(w *world.World) *Hub {
	return &Hub{
		world:      w,
		register:   make(chan *conn),
		unregister: make(chan *conn),
		notify:     make(chan struct{}, 1),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Notify() {
	select {
	case h.notify <- struct{}{}:
	default:
	}
}

func (h *Hub) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(h.done)

		clients := map[*conn]struct{}{}
		ticker := time.NewTicker(broadcastInterval)
		defer ticker.Stop()
		dirty := false

		for {
			select {
			case <-ctx.Done():
				for c := range clients {
					close(c.send)
				}
				return

			case c := <-h.register:
				clients[c] = struct{}{}
				payload, err := h.encode()
				if err != nil {
					log.Printf("encode snapshot: %v", err)
					continue
				}
				c.send <- payload

			case c := <-h.unregister:
				if _, ok := clients[c]; ok {
					delete(clients, c)
					close(c.send)
				}

			case <-h.notify:
				dirty = true

			case <-ticker.C:
				if !dirty || len(clients) == 0 {
					dirty = false
					continue
				}
				dirty = false

				payload, err := h.encode()
				if err != nil {
					log.Printf("encode snapshot: %v", err)
					continue
				}
				for c := range clients {
					select {
					case c.send <- payload:
					default:
						delete(clients, c)
						close(c.send)
					}
				}
			}
		}
	}()
}

func (h *Hub) ServeWS(rw http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Accept(rw, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("websocket accept: %v", err)
		return
	}
	defer ws.CloseNow()

	ctx := ws.CloseRead(r.Context())
	c := &conn{send: make(chan []byte, sendBuffer)}

	select {
	case h.register <- c:
	case <-ctx.Done():
		return
	case <-h.done:
		return
	}

	defer func() {
		select {
		case h.unregister <- c:
		case <-h.done:
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-c.send:
			if !ok {
				ws.Close(websocket.StatusPolicyViolation, "client too slow")
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := ws.Write(writeCtx, websocket.MessageText, payload)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (h *Hub) encode() ([]byte, error) {
	return json.Marshal(h.world.Snapshot())
}
