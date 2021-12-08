package rmq

import (
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type PublisherUnit struct {
	cfg Publisher
	ch  *atomic.Value
}

func NewPublisherUnit(cfg Publisher) *PublisherUnit {
	return &PublisherUnit{
		cfg: cfg,
		ch:  &atomic.Value{},
	}
}

func (p *PublisherUnit) Publish(exchange string, routingKey string, msg amqp.Publishing) error {
	ch, err := p.amqpChannel()
	if err != nil {
		return errors.WithMessage(err, "amqp channel")
	}
	err = ch.Publish(exchange, routingKey, true, false, msg)
	if err != nil {
		return errors.WithMessagef(err, "publish")
	}
	return nil
}

func (p *PublisherUnit) Run(ch *amqp.Channel) error {
	p.ch.Store(ch)
	return nil
}

func (p *PublisherUnit) Close() error {
	ch, err := p.amqpChannel()
	if err != nil {
		return errors.WithMessage(err, "amqp channel")
	}
	err = ch.Close()
	return errors.WithMessagef(err, "channel close")
}

func (p *PublisherUnit) amqpChannel() (*amqp.Channel, error) {
	ch, ok := p.ch.Load().(*amqp.Channel)
	if !ok {
		return nil, errors.New("publisher is not initialized")
	}
	return ch, nil
}
