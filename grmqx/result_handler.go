package grmqx

import (
	"context"
	"time"

	"github.com/integration-system/grmq/consumer"
	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-kit/metrics"
	rabbitmq_metircs "github.com/integration-system/isp-kit/metrics/rabbitmq_metrics"
)

type ConsumerMetricStorage interface {
	ObserveConsumeDuration(exchange string, routingKey string, t time.Duration)
	ObserveConsumeMsgSize(exchange string, routingKey string, size int)
	IncRequeueCount(exchange string, routingKey string)
	IncDlqCount(exchange string, routingKey string)
	IncSuccessCount(exchange string, routingKey string)
}

type Result struct {
	ack            bool
	requeue        bool
	requeueTimeout time.Duration
	moveToDlq      bool
	err            error
}

type HandlerAdapter interface {
	Handle(ctx context.Context, body []byte) Result
}

type AdapterFunc func(ctx context.Context, body []byte) Result

func (a AdapterFunc) Handle(ctx context.Context, body []byte) Result {
	return a(ctx, body)
}

func Ack() Result {
	return Result{ack: true}
}

func Requeue(after time.Duration, err error) Result {
	return Result{requeue: true, requeueTimeout: after, err: err}
}

// MoveToDlq
// if there is no DLQ, the message will be dropped
func MoveToDlq(err error) Result {
	return Result{moveToDlq: true, err: err}
}

type ResultHandler struct {
	logger        log.Logger
	adapter       HandlerAdapter
	metricStorage ConsumerMetricStorage
}

func NewResultHandler(logger log.Logger, adapter HandlerAdapter) ResultHandler {
	return ResultHandler{
		logger:        logger,
		adapter:       adapter,
		metricStorage: rabbitmq_metircs.NewConsumerStorage(metrics.DefaultRegistry),
	}
}

func (r ResultHandler) Handle(ctx context.Context, delivery *consumer.Delivery) {
	exchange := delivery.Source().Exchange
	routingKey := delivery.Source().RoutingKey
	start := time.Now()
	result := r.adapter.Handle(ctx, delivery.Source().Body)
	r.metricStorage.ObserveConsumeDuration(exchange, routingKey, time.Since(start))
	r.metricStorage.ObserveConsumeMsgSize(exchange, routingKey, len(delivery.Source().Body))

	switch {
	case result.ack:
		r.metricStorage.IncSuccessCount(exchange, routingKey)
		r.logger.Debug(ctx, "rmq client: message will be acknowledged")
		err := delivery.Ack()
		if err != nil {
			r.logger.Error(ctx, "rmq client: ack message error", log.Any("error", err))
		}
	case result.requeue:
		r.metricStorage.IncRequeueCount(exchange, routingKey)
		r.logger.Error(
			ctx,
			"rmq client: message will be requeued",
			log.Any("error", result.err),
			log.String("requeueTimeout", result.requeueTimeout.String()),
		)
		select {
		case <-time.After(result.requeueTimeout):
		case <-ctx.Done():
		}
		err := delivery.Nack(true)
		if err != nil {
			r.logger.Error(ctx, "rmq client: nack message error", log.Any("error", err))
		}
	case result.moveToDlq:
		r.metricStorage.IncDlqCount(exchange, routingKey)
		r.logger.Error(
			ctx,
			"rmq client: message will be moved to DLQ or dropped",
			log.Any("error", result.err),
		)
		err := delivery.Nack(false)
		if err != nil {
			r.logger.Error(ctx, "rmq client: nack message error", log.Any("error", err))
		}
	}
}
