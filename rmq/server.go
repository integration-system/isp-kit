package rmq

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type unit interface {
	Run(channel *amqp.Channel) error
	Close() error
}

type Client struct {
	url              string
	consumers        []Consumer
	publishers       []Publisher
	declarations     Declarations
	reconnectTimeout time.Duration
	observer         Observer

	close        chan struct{}
	shutdownDone chan struct{}
}

func New(url string) *Client {
	return &Client{
		url:              url,
		reconnectTimeout: 1 * time.Second,
		close:            make(chan struct{}),
		shutdownDone:     make(chan struct{}),
		observer:         NoopObserver{},
	}
}

// Run
// Block and wait first successfully established session
// It means all declarations were applied successfully
// All publishers were initialized
// All consumers were run
// Returns first occurred error during session opening or null
func (s *Client) Run(ctx context.Context) error {
	firstSessionErr := make(chan error, 1)
	go s.run(ctx, firstSessionErr)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-firstSessionErr:
		return err
	}
}

func (s *Client) run(ctx context.Context, firstSessionErr chan error) {
	defer func() {
		s.observer.ShutdownDone()
		close(s.shutdownDone)
	}()

	sessNum := 0
	for {
		sessNum++
		err := s.runSession(sessNum, firstSessionErr)
		if err == nil { //normal close
			return
		}

		s.observer.ClientError(err)

		select {
		case <-ctx.Done():
			return
		case <-s.close:
			return //shutdown called
		case <-time.After(s.reconnectTimeout):

		}
	}
}

func (s *Client) runSession(sessNum int, firstSessionErr chan error) (err error) {
	firstSessionErrWritten := false
	defer func() {
		if !firstSessionErrWritten && sessNum == 1 {
			firstSessionErr <- err
		}
	}()

	conn, err := amqp.Dial(s.url)
	if err != nil {
		return errors.WithMessagef(err, "dial to %s", s.url)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return errors.WithMessagef(err, "create channel for declarations")
	}
	declarations := NewDeclarator(s.declarations)
	err = declarations.Run(ch)
	if err != nil {
		return errors.WithMessagef(err, "run declarations")
	}
	err = declarations.Close()
	if err != nil {
		return errors.WithMessagef(err, "close declarations")
	}

	units := make([]unit, 0)
	defer func() {
		for i := len(units); i > 0; i-- {
			_ = units[i-1].Close()
		}
	}()

	for i, publisher := range s.publishers {
		ch, err = conn.Channel()
		if err != nil {
			return errors.WithMessagef(err, "create channel for publisher %d", i)
		}
		publisherUnit := NewPublisherUnit(publisher)
		err = publisherUnit.Run(ch)
		if err != nil {
			return errors.WithMessagef(err, "run publisher %d", i)
		}
		publisher.setDelegate(publisherUnit)
		units = append(units, publisherUnit)
	}

	for _, consumer := range s.consumers {
		ch, err = conn.Channel()
		if err != nil {
			return errors.WithMessagef(err, "create channel for consumer for '%s'", consumer.Queue)
		}
		consumerUnit := NewConsumerUnit(consumer, s.observer)
		err = consumerUnit.Run(ch)
		if err != nil {
			return errors.WithMessagef(err, "run consumer '%s'", consumer.Queue)
		}
		units = append(units, consumerUnit)
	}

	if sessNum == 1 {
		firstSessionErrWritten = true
		firstSessionErr <- nil
	}

	s.observer.ClientReady()

	connCloseChan := make(chan *amqp.Error)
	connCloseChan = conn.NotifyClose(connCloseChan)
	select {
	case err, ok := <-connCloseChan:
		if !ok {
			return nil
		}
		return err
	case <-s.close:
		return nil
	}
}

// Shutdown
// Perform graceful shutdown
func (s *Client) Shutdown() {
	close(s.close)
	s.observer.ShutdownStarted()
	<-s.shutdownDone
}
