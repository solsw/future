//go:build go1.18

package future

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrPromiseTimeout = errors.New("promise timeout")
)

// Future returns the result sometime in the future.
type Future[Result any] struct {
	ctx     context.Context
	promise func(ctx context.Context) (Result, error)
	timeout time.Duration
	lazy    bool

	once sync.Once
	wg   sync.WaitGroup

	hasResult bool
	res       Result
	err       error
}

// New creates a new Future.
//
// 'ctx' - initial context (is passed to 'promise' as is if 'timeout' is not provided).
// 'promise' produces the Future's result. 'promise' is called exactly once.
// 'timeout' - timeout to wait for 'promise' to complete (pass zero if timeout is not needed).
// If 'timeout' is provided, new context (that is canceled when 'timeout' elapses)
// is derived from 'ctx' and passed to 'promise'. 'promise' must respect context cancelling to not leak goroutine.
// If 'promise' does not complete before 'timeout' elapses, 'promise' is discarded and ErrPromiseTimeout is returned.
// If 'lazy', 'promise' will be called synchronously by the first Result() call, otherwise - asynchronously immediately.
func New[Result any](ctx context.Context, promise func(context.Context) (Result, error), timeout time.Duration, lazy bool) *Future[Result] {
	f := Future[Result]{
		ctx:     ctx,
		promise: promise,
		timeout: timeout,
		lazy:    lazy,
	}
	f.wg.Add(1)
	if !f.lazy {
		go f.getResult()
	}
	return &f
}

func (f *Future[Result]) getResult() {
	defer func() {
		f.hasResult = true
		f.wg.Done()
	}()
	if f.timeout <= 0 {
		f.res, f.err = f.promise(f.ctx)
		return
	}
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		// initial context canceled or deadlined
		var r0 Result
		f.res, f.err = r0, ctx.Err()
	case <-time.After(f.timeout):
		var r0 Result
		f.res, f.err = r0, ErrPromiseTimeout
	case <-func() <-chan struct{} {
		ch := make(chan struct{})
		go func() {
			defer close(ch)
			res, err := f.promise(ctx)
			// promise may be already canceled or timed out here
			if f.err != nil {
				return
			}
			f.res, f.err = res, err
		}()
		return ch
	}():
	}
}

// Result returns the Future's result or/and error.
//
// If a result or/and an error is not obtained yet, Result blocks until the Future runs to depletion.
// Result is threadsafe.
func (f *Future[Result]) Result() (Result, error) {
	if f.lazy {
		f.once.Do(func() { f.getResult() })
	}
	f.wg.Wait()
	return f.res, f.err
}

// Depleted reports whether the Future already has a result or/and an error.
func (f *Future[Result]) Depleted() bool {
	return f.hasResult
}
