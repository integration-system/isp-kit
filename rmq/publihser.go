package rmq

import (
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

var (
	ErrPublisherIsNotInitialized = errors.New("publisher is not initialized")
)

type delegate interface {
	Publish(exchange string, routingKey string, msg amqp.Publishing) error
}

type Publisher struct {
	delegate *atomic.Value
}

func NewPublisher() *Publisher {
	return &Publisher{
		delegate: &atomic.Value{},
	}
}

func (p *Publisher) Publish(exchange string, routingKey string, msg amqp.Publishing) error {
	delegate, err := p.getDelegate()
	if err != nil {
		return err
	}
	return delegate.Publish(exchange, routingKey, msg)
}

func (p *Publisher) setDelegate(d delegate) {
	p.delegate.Store(d)
}

func (p *Publisher) getDelegate() (delegate, error) {
	delegate, ok := p.delegate.Load().(delegate)
	if !ok {
		return nil, ErrPublisherIsNotInitialized
	}
	return delegate, nil
}
