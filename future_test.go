package future

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFuture_Result(t *testing.T) {
	ctx1, cancel1 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel1()
	prom := func(ctx context.Context) (interface{}, error) {
		select {
		case <-time.After(200 * time.Millisecond):
			return 1, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	tests := []struct {
		name        string
		f           *Future
		want        interface{}
		wantErr     bool
		expectedErr error
	}{
		{name: "1 - initial context with timeout, no future's timeout",
			f:           New(prom, ctx1, 0, true),
			wantErr:     true,
			expectedErr: context.DeadlineExceeded,
		},
		{name: "2 - initial context without timeout with future's timeout",
			f:           New(prom, context.Background(), 100*time.Millisecond, true),
			wantErr:     true,
			expectedErr: ErrPromiseTimeout,
		},
		{name: "3 - initial context without timeout, no future's timeout",
			f:    New(prom, context.Background(), 0, true),
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestFuture_HasResult(t *testing.T) {
	future := New(
		func(ctx context.Context) (interface{}, error) {
			select {
			case <-time.After(1 * time.Second):
				return 1, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
		context.Background(),
		100*time.Millisecond,
		false,
	)
	tests := []struct {
		name string
		f    *Future
		want bool
	}{
		{name: "first HasResult() call", f: future, want: false},
		{name: "second HasResult() call", f: future, want: true},
	}
	for _, tt := range tests {
		if strings.HasPrefix(tt.name, "second") {
			time.Sleep(200 * time.Millisecond)
		}
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.HasResult(); got != tt.want {
				t.Errorf("Future.HasResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
