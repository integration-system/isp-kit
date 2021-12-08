package rmq

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type ConsumerUnit struct {
	cfg           Consumer
	workersWg     *sync.WaitGroup
	unexpectedErr chan *amqp.Error
	workersStop   chan struct{}
	closeConsumer chan struct{}
	observer      Observer

	ch         *amqp.Channel
	deliveries <-chan amqp.Delivery
}

func NewConsumerUnit(cfg Consumer, observer Observer) *ConsumerUnit {
	return &ConsumerUnit{
		cfg:           cfg,
		workersWg:     &sync.WaitGroup{},
		unexpectedErr: make(chan *amqp.Error),
		workersStop:   make(chan struct{}),
		closeConsumer: make(chan struct{}),
		observer:      observer,
	}
}

func (c *ConsumerUnit) Run(ch *amqp.Channel) error {
	c.ch = ch
	c.unexpectedErr = ch.NotifyClose(c.unexpectedErr)
	go c.runUnexpectedErrorListener()

	if c.cfg.PrefetchCount > 0 {
		err := ch.Qos(c.cfg.PrefetchCount, 0, false)
		if err != nil {
			return errors.WithMessagef(err, "set qos")
		}
	}

	deliveries, err := ch.Consume(c.cfg.Queue, c.cfg.Name, false, false, false, false, nil)
	if err != nil {
		return errors.WithMessage(err, "begin consume")
	}
	c.deliveries = deliveries

	for i := 0; i < c.cfg.Concurrency; i++ {
		c.workersWg.Add(1)
		go c.runWorker()
	}

	return nil
}

func (c *ConsumerUnit) runWorker() {
	defer c.workersWg.Done()

	for {
		select {
		case delivery, ok := <-c.deliveries:
			if !ok { //normal close
				return
			}
			c.cfg.Handler.Handle(context.Background(), delivery)
		case <-c.workersStop:
			return //unexpected error occurred
		}
	}
}

func (c *ConsumerUnit) runUnexpectedErrorListener() {
	for {
		select {
		case <-c.closeConsumer: //close consumer
			return
		case err, ok := <-c.unexpectedErr:
			if !ok { //normal close
				return
			}
			if err != nil {
				c.observer.ConsumerError(c.cfg, err)
				close(c.workersStop) //stop all workers because of unexpected error
				return
			}
		}
	}
}

func (c *ConsumerUnit) Close() error {
	err := c.ch.Cancel(c.cfg.Name, false)
	if err != nil {
		return errors.WithMessage(err, "channel cancel")
	}

	close(c.closeConsumer)
	c.workersWg.Wait()

	err = c.ch.Close()
	return errors.WithMessage(err, "channel close")
}
