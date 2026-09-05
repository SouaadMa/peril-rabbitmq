package pubsub

import (
	"context"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
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
