package pubsub

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	minRetryDelay = time.Second
	maxRetryDelay = 30 * time.Second
)

type starter func(context.Context, *amqp.Connection) error

type Client struct {
	url string

	mu    sync.RWMutex
	conn  *amqp.Connection
	pubCh *amqp.Channel

	subs []starter
}

func Dial(url string) (*Client, error) {
	c := &Client{url: url}
	err := c.connect()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) Supervise(ctx context.Context, wg *sync.WaitGroup, redeclare func(*amqp.Channel) error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			closed := make(chan *amqp.Error, 1)
			c.Connection().NotifyClose(closed)

			select {
			case <-ctx.Done():
				return
			case amqpErr, ok := <-closed:
				if !ok || amqpErr == nil {
					return
				}
				log.Printf("broker connection lost: %v", amqpErr)
			}

			if !c.redial(ctx) {
				return
			}
			log.Printf("reconnected to broker")

			err := redeclare(c.Channel())
			if err != nil {
				log.Printf("redeclare topology: %v", err)
				continue
			}
			for _, start := range c.starters() {
				err := start(ctx, c.Connection())
				if err != nil {
					log.Printf("resubscribe: %v", err)
				}
			}
		}
	}()
}

func (c *Client) connect() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("connect to broker: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.pubCh = ch
	c.mu.Unlock()
	return nil
}

func (c *Client) Connection() *amqp.Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

func (c *Client) Channel() *amqp.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pubCh
}

func (c *Client) Close() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) register(s starter) {
	c.mu.Lock()
	c.subs = append(c.subs, s)
	c.mu.Unlock()
}

func (c *Client) starters() []starter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]starter(nil), c.subs...)
}

func (c *Client) redial(ctx context.Context) bool {
	delay := minRetryDelay
	for {
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))
		select {
		case <-ctx.Done():
			return false
		case <-time.After(delay + jitter):
		}

		err := c.connect()
		if err == nil {
			return true
		}
		log.Printf("reconnect failed: %v", err)

		delay *= 2
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}
}
