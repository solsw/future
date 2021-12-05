package future

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFuture_Result(t *testing.T) {
	promise := func(ctx context.Context) (interface{}, error) {
		select {
		case <-time.After(500 * time.Millisecond):
			return 1, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	tests := []struct {
		name        string
		ctxTimeout  time.Duration
		f           *Future
		want        interface{}
		wantErr     bool
		expectedErr error
	}{
		{name: "1 - initial context with timeout, future without timeout",
			ctxTimeout:  100 * time.Millisecond,
			f:           New(context.Background(), promise, 0, true),
			wantErr:     true,
			expectedErr: context.DeadlineExceeded,
		},
		{name: "2 - initial context with timeout, future with longer timeout",
			ctxTimeout:  100 * time.Millisecond,
			f:           New(context.Background(), promise, 300*time.Millisecond, true),
			wantErr:     true,
			expectedErr: context.DeadlineExceeded,
		},
		{name: "3 - initial context with longer timeout, future with timeout",
			ctxTimeout:  300 * time.Millisecond,
			f:           New(context.Background(), promise, 100*time.Millisecond, true),
			wantErr:     true,
			expectedErr: ErrPromiseTimeout,
		},
		{name: "4 - initial context without timeout, future with timeout",
			f:           New(context.Background(), promise, 100*time.Millisecond, true),
			wantErr:     true,
			expectedErr: ErrPromiseTimeout,
		},
		{name: "5 - initial context without timeout, future without timeout",
			f:    New(context.Background(), promise, 0, true),
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ctxTimeout > 0 {
				ctx, cancel := context.WithTimeout(tt.f.ctx, tt.ctxTimeout)
				defer cancel()
				tt.f.ctx = ctx
			}
			got, err := tt.f.Result()
			if (err != nil) != tt.wantErr {
				t.Errorf("Future.Result() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err != tt.expectedErr {
					t.Errorf("Future.Result() error = %v, expectedErr %v", err, tt.expectedErr)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Future.Result() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuture_Depleted(t *testing.T) {
	promise := func(_ context.Context) (interface{}, error) {
		time.Sleep(500 * time.Millisecond)
		return 1, nil
	}
	future := New(context.Background(), promise, 300*time.Millisecond, false)
	tests := []struct {
		name string
		f    *Future
		want bool
	}{
		// future's timeout (300 ms) has not elapsed yet
		{name: "first HasResult() call", f: future, want: false},
		{name: "second HasResult() call", f: future, want: false},
		// future's timeout (300 ms) has already elapsed
		{name: "third HasResult() call", f: future, want: true},
		{name: "fourth HasResult() call", f: future, want: true},
	}
	for _, tt := range tests {
		if !strings.HasPrefix(tt.name, "first") {
			time.Sleep(200 * time.Millisecond)
		}
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.Depleted(); got != tt.want {
				t.Errorf("Future.HasResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
