# Package future
[![Go Reference](https://pkg.go.dev/badge/github.com/solsw/future.svg)](https://pkg.go.dev/github.com/solsw/future/v2)
[![GitHub](https://img.shields.io/badge/github--green?logo=github)](https://github.com/solsw/future)

Package **future** contains a generic implementation of
[explicit](https://en.wikipedia.org/wiki/Futures_and_promises#Implicit_vs._explicit)
[future](https://en.wikipedia.org/wiki/Futures_and_promises).

A `Future[Result]` represents a value (or an error) that will become available
sometime in the future. The value is produced by a *promise* — a function
`func(context.Context) (Result, error)` — that is called **exactly once**.

Inspired by *Cloud Native Go* by Matthew A. Titmus.

## Installation

```sh
go get github.com/solsw/future/v2
```

```go
import "github.com/solsw/future/v2"
```

## API

### New

```go
func New[Result any](
    ctx context.Context,
    promise func(context.Context) (Result, error),
    timeout time.Duration,
    lazy bool,
) *Future[Result]
```

Creates a new `Future`:

- **ctx** — initial context (is passed to `promise` as is if `timeout` is not provided);
- **promise** — produces the Future's result. `promise` is called exactly once.
  `promise` must respect cancelling of the context it receives, otherwise
  the goroutine running it may leak;
- **timeout** — timeout to wait for `promise` to complete (pass zero if timeout is not needed).
  If `timeout` is provided, a new context (that is canceled when `timeout` elapses)
  is derived from `ctx` and passed to `promise`. If `promise` does not complete before
  `timeout` elapses, its result is discarded and `ErrPromiseTimeout` is returned;
- **lazy** — if `true`, `promise` is called synchronously by the first `Result()` call;
  otherwise `promise` starts asynchronously (in its own goroutine) immediately.

### Result

```go
func (f *Future[Result]) Result() (Result, error)
```

Returns the Future's result or/and error. If the result is not obtained yet,
`Result` blocks until the Future runs to depletion. `Result` is threadsafe —
it may be called any number of times from any number of goroutines;
every call returns the same result.

### Depleted

```go
func (f *Future[Result]) Depleted() bool
```

Reports whether the Future already has a result or/and an error
(i.e. whether `Result` would return without blocking).

### ErrPromiseTimeout

```go
var ErrPromiseTimeout = errors.New("promise timeout")
```

Returned by `Result` if `promise` does not complete before `timeout` elapses.

## Usage

### Eager evaluation

The promise starts immediately in a background goroutine;
`Result` waits for it to finish.

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/solsw/future/v2"
)

func main() {
    f := future.New(context.Background(),
        func(ctx context.Context) (int, error) {
            time.Sleep(100 * time.Millisecond) // some long computation
            return 42, nil
        },
        0,     // no timeout
        false, // eager: the promise starts right now
    )

    // do other work here while the promise is running...

    r, err := f.Result() // blocks until the promise completes
    fmt.Println(r, err)  // 42 <nil>
}
```

### Lazy evaluation

The promise is not started until `Result` is called for the first time;
that call runs the promise synchronously.

```go
f := future.New(ctx, promise, 0, true) // nothing happens yet
// ...
r, err := f.Result() // the promise runs here
```

### With timeout

```go
f := future.New(context.Background(),
    func(ctx context.Context) (string, error) {
        select {
        case <-time.After(500 * time.Millisecond):
            return "done", nil
        case <-ctx.Done(): // respect cancellation to not leak the goroutine
            return "", ctx.Err()
        }
    },
    100*time.Millisecond, // the promise is given 100 ms to complete
    false,
)

_, err := f.Result()
fmt.Println(errors.Is(err, future.ErrPromiseTimeout)) // true
```

### Polling with Depleted

```go
f := future.New(ctx, promise, 0, false)
for !f.Depleted() {
    // the result is not ready yet, do something else
}
r, err := f.Result() // returns immediately
```

## Cancellation and timeout semantics

Which error `Result` returns depends on what happens first:

| What happens first                          | `Result` returns                                  |
|---------------------------------------------|---------------------------------------------------|
| `promise` completes                         | whatever `promise` returned                       |
| the Future's `timeout` elapses              | zero `Result` value and `ErrPromiseTimeout`       |
| the initial `ctx` is canceled or deadlined  | zero `Result` value and `ctx.Err()`               |

Notes:

- If `timeout` is zero (not provided), the Future does not watch `ctx` itself —
  `ctx` is passed to `promise` as is, and the outcome is entirely up to
  how `promise` handles it.
- The promise should always honor cancelling of the context it receives.
  When the Future gives up (timeout or canceled context), the context passed
  to the promise is canceled, but the promise's goroutine keeps running
  until the promise itself returns.
