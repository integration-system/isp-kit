package rmq

import (
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type Declarator struct {
	cfg Declarations
	ch  *amqp.Channel
}

func NewDeclarator(cfg Declarations) *Declarator {
	return &Declarator{
		cfg: cfg,
	}
}

func (c *Declarator) Run(ch *amqp.Channel) error {
	c.ch = ch

	for _, exchange := range c.cfg.Exchanges {
		err := ch.ExchangeDeclare(exchange.Name, exchange.Type, true, false, false, false, exchange.Args)
		if err != nil {
			return errors.WithMessagef(err, "declare exchange '%s'", exchange.Name)
		}
	}

	for _, queue := range c.cfg.Queues {
		_, err := ch.QueueDeclare(queue.Name, true, false, false, false, queue.Args)
		if err != nil {
			return errors.WithMessagef(err, "declare queue '%s'", queue.Name)
		}
	}

	for _, binding := range c.cfg.Bindings {
		err := ch.QueueBind(binding.QueueName, binding.RoutingKey, binding.ExchangeName, false, binding.Args)
		if err != nil {
			return errors.WithMessagef(err, "declare binding for queue '%s' to exchange '%s'", binding.QueueName, binding.ExchangeName)
		}
	}

	return nil
}

func (c *Declarator) Close() error {
	err := c.ch.Close()
	return errors.WithMessage(err, "channel close")
}
