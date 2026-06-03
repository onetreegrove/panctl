package client

import (
	"context"
	"testing"
)

func TestLimiterWaitHonorsContextCancellation(t *testing.T) {
	lim := newUserLimiter(0.001)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := lim.wait(ctx, limiterList); err == nil {
		t.Fatalf("expected canceled context error")
	}
}
