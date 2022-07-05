package bgjobx_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/integration-system/bgjob"
	"github.com/integration-system/isp-kit/bgjobx"
	"github.com/integration-system/isp-kit/dbx"
	"github.com/integration-system/isp-kit/test"
	"github.com/integration-system/isp-kit/test/dbt"
)

func TestClient(t *testing.T) {
	test, assert := test.New(t)
	testDb := dbt.New(test, dbx.WithMigration("./migration"))
	cli := bgjobx.NewClient(testDb, test.Logger())
	t.Cleanup(func() {
		cli.Close()
	})
	callCount := int32(0)
	worker := bgjobx.WorkerConfig{
		Queue: "test",
		Handle: bgjob.HandlerFunc(func(ctx context.Context, job bgjob.Job) bgjob.Result {
			atomic.AddInt32(&callCount, 1)
			return bgjob.Complete()
		}),
		PollInterval: 500 * time.Millisecond,
	}
	err := cli.Upgrade(context.Background(), []bgjobx.WorkerConfig{worker})
	assert.NoError(err)

	err = cli.Enqueue(context.Background(), bgjob.EnqueueRequest{
		Queue: "test",
		Type:  "type",
	})
	assert.NoError(err)

	time.Sleep(1 * time.Second)
	assert.EqualValues(1, atomic.LoadInt32(&callCount))
}
