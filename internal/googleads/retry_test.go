package googleads

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRetryUnaryInterceptor_DisabledWhenMaxRetriesZero(t *testing.T) {
	var calls int32
	interceptor := retryUnaryInterceptor(0, 250*time.Millisecond)
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		atomic.AddInt32(&calls, 1)
		return status.Error(codes.Unavailable, "transient")
	}
	err := interceptor(context.Background(), "/m", nil, nil, nil, invoker)
	if err == nil {
		t.Fatal("expected error to bubble up")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (retries disabled)", calls)
	}
}

func TestRetryUnaryInterceptor_RetriesUnavailable(t *testing.T) {
	var calls int32
	interceptor := retryUnaryInterceptor(3, time.Millisecond) // tiny backoff so the test stays fast
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return status.Error(codes.Unavailable, "transient")
		}
		return nil // succeed on 3rd attempt
	}
	if err := interceptor(context.Background(), "/m", nil, nil, nil, invoker); err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetryUnaryInterceptor_GivesUpAfterMaxRetries(t *testing.T) {
	var calls int32
	interceptor := retryUnaryInterceptor(2, time.Millisecond)
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		atomic.AddInt32(&calls, 1)
		return status.Error(codes.Unavailable, "always failing")
	}
	err := interceptor(context.Background(), "/m", nil, nil, nil, invoker)
	if err == nil {
		t.Fatal("expected final error")
	}
	// 1 initial + 2 retries = 3 total attempts
	if calls != 3 {
		t.Errorf("calls = %d, want 3 (1 initial + 2 retries)", calls)
	}
}

func TestRetryUnaryInterceptor_DoesNotRetryNonRetryable(t *testing.T) {
	var calls int32
	interceptor := retryUnaryInterceptor(5, time.Millisecond)
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		atomic.AddInt32(&calls, 1)
		return status.Error(codes.InvalidArgument, "bad request")
	}
	err := interceptor(context.Background(), "/m", nil, nil, nil, invoker)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (InvalidArgument is not retryable)", calls)
	}
}

func TestRetryUnaryInterceptor_RespectsContextCancellation(t *testing.T) {
	var calls int32
	interceptor := retryUnaryInterceptor(10, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	invoker := func(_ context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			// Trigger cancel before the backoff sleep would finish.
			cancel()
		}
		return status.Error(codes.Unavailable, "transient")
	}
	err := interceptor(ctx, "/m", nil, nil, nil, invoker)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (cancellation should abort the backoff loop)", calls)
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("plain"), false},
		{status.Error(codes.Unavailable, ""), true},
		{status.Error(codes.DeadlineExceeded, ""), true},
		{status.Error(codes.ResourceExhausted, ""), true},
		{status.Error(codes.InvalidArgument, ""), false},
		{status.Error(codes.NotFound, ""), false},
		{status.Error(codes.PermissionDenied, ""), false},
	}
	for _, c := range cases {
		if got := isRetryable(c.err); got != c.want {
			t.Errorf("isRetryable(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}
