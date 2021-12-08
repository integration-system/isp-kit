package client

import (
	"time"

	"google.golang.org/grpc"
)

const (
	defaultMaxSizeByte = 10 * 1024 * 1024
)

func Default() (*Client, error) {
	return New(
		nil,
		WithDialOptions(
			grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(
				grpc.WaitForReady(true),
				grpc.MaxCallSendMsgSize(defaultMaxSizeByte),
				grpc.MaxCallRecvMsgSize(defaultMaxSizeByte),
			),
		),
		WithMiddlewares(
			RequestId(),
			DefaultTimeout(15*time.Second),
		),
	)
}
