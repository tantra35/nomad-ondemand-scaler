package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTimeoutedContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func(_ctx context.Context) {
		defer wg.Done()

		select {
		case <-_ctx.Done():
			t.Logf("done")
			if _ctx.Err() == context.DeadlineExceeded {
				t.Logf("deadline")
			}
		case <-time.After(10 * time.Second):
			t.Logf("action done")
		}
	}(ctx)

	time.Sleep(5 * time.Second)
	cancel()
	wg.Wait()
}
