package stream

import "context"

type StreamConsumer interface {
	Setup(ctx context.Context) error
	Start(ctx context.Context) error
	Stop() error
}
