package tests

import (
	"context"
	"testing"
	"time"

	"github.com/integration-system/isp-kit/grmqx"
	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-kit/requestid"
	"github.com/integration-system/isp-kit/test"
	"github.com/integration-system/isp-kit/test/grmqt"
	"github.com/rabbitmq/amqp091-go"
)

func TestRequestIdChain(t *testing.T) {
	test, require := test.New(t)

	expectedRequestId := requestid.Next()

	pubCfg1 := grmqx.Publisher{
		Exchange:   "",
		RoutingKey: "queue1",
	}
	pub1 := pubCfg1.DefaultPublisher(grmqx.PublisherLog(test.Logger()))
	consumerCfg1 := grmqx.Consumer{
		Queue: "queue1",
	}
	pubCfg2 := grmqx.Publisher{
		RoutingKey: "queue2",
	}
	pub2 := pubCfg2.DefaultPublisher(grmqx.PublisherLog(test.Logger()))
	consumerCfg2 := grmqx.Consumer{
		Queue: "queue2",
	}

	handler1 := grmqx.NewResultHandler(
		test.Logger(),
		grmqx.AdapterFunc(func(ctx context.Context, body []byte) grmqx.Result {
			err := pub2.Publish(ctx, &amqp091.Publishing{})
			require.NoError(err)
			return grmqx.Ack()
		}),
	)
	consumer1 := consumerCfg1.DefaultConsumer(handler1, grmqx.ConsumerLog(test.Logger()))

	await := make(chan struct{})
	handler2 := grmqx.NewResultHandler(
		test.Logger(),
		grmqx.AdapterFunc(func(ctx context.Context, body []byte) grmqx.Result {
			requestId := requestid.FromContext(ctx)
			require.EqualValues(expectedRequestId, requestId)
			close(await)
			return grmqx.Ack()
		}),
	)
	consumer2 := consumerCfg2.DefaultConsumer(handler2, grmqx.ConsumerLog(test.Logger()))

	testCli := grmqt.New(test)
	cli := grmqx.New(test.Logger())
	t.Cleanup(func() {
		cli.Close()
	})
	cfg := grmqx.NewConfig(
		testCli.ConnectionConfig().Url(),
		grmqx.WithPublishers(pub1, pub2),
		grmqx.WithConsumers(consumer1, consumer2),
		grmqx.WithDeclarations(grmqx.TopologyFromConsumers(consumerCfg1, consumerCfg2)),
	)
	err := cli.Upgrade(context.Background(), cfg)
	require.NoError(err)

	ctx := requestid.ToContext(context.Background(), expectedRequestId)
	ctx = log.ToContext(ctx, log.String("requestId", expectedRequestId))
	err = pub1.Publish(ctx, &amqp091.Publishing{})
	require.NoError(err)

	select {
	case <-await:
	case <-time.After(5 * time.Second):
		require.Fail("handler wasn't called")
	}
}
