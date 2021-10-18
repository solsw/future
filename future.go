package future

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrPromiseTimeout = errors.New("promise timeout")

// Future returns result (interface{} and error if any) sometime in the future.
type Future struct {
	ctx     context.Context
	promise func(ctx context.Context) (interface{}, error)
	timeout time.Duration
	lazy    bool

	once sync.Once
	wg   sync.WaitGroup

	hasResult bool
	res       interface{}
	err       error
}

// New creates new Future.
//
// 'promise' will be called exactly once to produce the Future's result.
// 'ctx' is passed to 'promise', if 'timeout' is not provided.
// 'timeout' - timeout for 'promise' (pass zero if timeout is not needed).
// If 'timeout' is provided, new context (that is canceled when 'timeout' elapses)
// is derived from 'ctx' and passed to 'promise'.
// 'promise' must respect context cancelling to not leak goroutine.
// If 'lazy', 'promise' will be called synchronously by the first Result() call,
// otherwise - asynchronously immediately.
func New(promise func(context.Context) (interface{}, error), ctx context.Context, timeout time.Duration, lazy bool) *Future {
	f := Future{
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

func (f *Future) getResult() {
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
	case <-time.After(f.timeout):
		f.res, f.err = nil, ErrPromiseTimeout
	case <-func() <-chan struct{} {
		ch := make(chan struct{})
		go func() {
			defer close(ch)
			res, err := f.promise(ctx)
			// promise may be already timed out here
			if f.err == ErrPromiseTimeout {
				return
			}
			f.res, f.err = res, err
		}()
		return ch
	}():
	}
}

// Result returns the Future's result (interface{} and error if any).
//
// If the result (possibly containing only error) is not ready,
// Result blocks until the result is ready. Result is threadsafe.
func (f *Future) Result() (interface{}, error) {
	if f.lazy {
		f.once.Do(func() { f.getResult() })
	}
	f.wg.Wait()
	return f.res, f.err
}

// HasResult reports whether the Future already has a result.
func (f *Future) HasResult() bool {
	return f.hasResult
}
