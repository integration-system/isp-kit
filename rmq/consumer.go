package rmq

import (
	"context"

	"github.com/streadway/amqp"
)

type Handler interface {
	Handle(ctx context.Context, delivery amqp.Delivery)
}

type HandleFunc func(ctx context.Context, delivery amqp.Delivery)

func (f HandleFunc) Handle(ctx context.Context, delivery amqp.Delivery) {
	f(ctx, delivery)
}

type Consumer struct {
	Handler       Handler
	Queue         string
	Name          string
	Concurrency   int
	PrefetchCount int
}
