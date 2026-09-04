package topology

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Declare(ch *amqp.Channel) error {
	exchanges := []struct {
		name string
		kind string
	}{
		{routing.ExchangePerilDirect, amqp.ExchangeDirect},
		{routing.ExchangePerilTopic, amqp.ExchangeTopic},
		{routing.ExchangePerilDLX, amqp.ExchangeFanout},
	}
	for _, ex := range exchanges {
		err := ch.ExchangeDeclare(ex.name, ex.kind, true, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("declare exchange %q: %w", ex.name, err)
		}
	}

	_, err := ch.QueueDeclare(routing.QueueDeadLetter, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue %q: %w", routing.QueueDeadLetter, err)
	}

	err = ch.QueueBind(routing.QueueDeadLetter, "", routing.ExchangePerilDLX, false, nil)
	if err != nil {
		return fmt.Errorf("bind queue %q to %q: %w", routing.QueueDeadLetter, routing.ExchangePerilDLX, err)
	}

	return nil
}
