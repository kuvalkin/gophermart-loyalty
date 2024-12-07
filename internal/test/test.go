package test

import (
	"context"
	"testing"
)

func Int64Pointer(x int64) *int64 {
	return &x
}

func Context(t *testing.T) (context.Context, context.CancelFunc) {
	deadline, ok := t.Deadline()
	if !ok {
		return context.WithCancel(context.Background())
	}

	return context.WithDeadline(context.Background(), deadline)
}
