package googleads

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestConcurrencyUnaryInterceptor_NilSemIsPassthrough(t *testing.T) {
	interceptor := concurrencyUnaryInterceptor(nil)
	var called bool
	err := interceptor(context.Background(), "/m", nil, nil, nil, func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !called {
		t.Fatal("invoker not called")
	}
}

func TestConcurrencyUnaryInterceptor_CapsConcurrentCalls(t *testing.T) {
	const limit = 3
	const total = 20

	sem := make(chan struct{}, limit)
	interceptor := concurrencyUnaryInterceptor(sem)

	var inFlight, peak int64
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		now := atomic.AddInt64(&inFlight, 1)
		for {
			old := atomic.LoadInt64(&peak)
			if now <= old || atomic.CompareAndSwapInt64(&peak, old, now) {
				break
			}
		}
		// Hold the slot long enough that, if the cap weren't honored,
		// goroutines would pile up. 5ms is plenty for 20 goroutines.
		time.Sleep(5 * time.Millisecond)
		atomic.AddInt64(&inFlight, -1)
		return nil
	}

	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = interceptor(context.Background(), "/m", nil, nil, nil, invoker)
		}()
	}
	wg.Wait()

	if peak > int64(limit) {
		t.Fatalf("peak concurrency = %d, want <= %d", peak, limit)
	}
	if peak == 0 {
		t.Fatal("interceptor never let any call through")
	}
}

func TestConcurrencyUnaryInterceptor_RespectsContextCancellation(t *testing.T) {
	sem := make(chan struct{}, 1)
	sem <- struct{}{} // fill it so the next call must wait

	interceptor := concurrencyUnaryInterceptor(sem)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	err := interceptor(ctx, "/m", nil, nil, nil, func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		t.Fatal("invoker should not be called when ctx is canceled before acquiring slot")
		return nil
	})
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
}
