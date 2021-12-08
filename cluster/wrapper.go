package cluster

import (
	"context"
	"time"

	etpclient "github.com/integration-system/isp-etp-go/v2/client"
	"github.com/integration-system/isp-kit/json"
	"github.com/integration-system/isp-kit/log"
)

type clientWrapper struct {
	etpclient.Client
	ctx    context.Context
	logger log.Logger
}

func newClientWrapper(ctx context.Context, cli etpclient.Client, logger log.Logger) *clientWrapper {
	return &clientWrapper{
		Client: cli,
		ctx:    ctx,
		logger: logger,
	}
}

func (w *clientWrapper) On(event string, handler func(data []byte)) {
	w.Client.On(event, func(data []byte) {
		w.logger.Info(
			w.ctx,
			"event received",
			log.String("event", event),
			log.Any("data", json.RawMessage(data)),
		)
		handler(data)
	})
}

func (w *clientWrapper) EmitWithAck(ctx context.Context, event string, data []byte) ([]byte, error) {
	w.logger.Info(
		ctx,
		"send event",
		log.String("event", event),
		log.Any("data", json.RawMessage(data)),
	)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	resp, err := w.Client.EmitWithAck(ctx, event, data)
	if err != nil {
		w.logger.Error(ctx, "error", log.Any("error", err))
		return resp, err
	}

	w.logger.Info(ctx, "event acknowledged", log.String("event", event), log.String("response", string(resp)))
	return resp, err
}

func (w *clientWrapper) EventChan(event string) chan []byte {
	ch := make(chan []byte)
	w.On(event, func(data []byte) {
		select {
		case <-w.ctx.Done():
		case ch <- data:
		}
	})
	return ch
}
