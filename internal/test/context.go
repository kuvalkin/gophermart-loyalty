package test

import (
	"context"
	"testing"
)

func Context(t *testing.T) (context.Context, context.CancelFunc) {
	deadline, ok := t.Deadline()
	if !ok {
		return context.WithCancel(context.Background())
	}

	return context.WithDeadline(context.Background(), deadline)
}
