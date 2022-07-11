# Signal - GoRoutine Manager

**Feature Name**: Signal - GoRoutine Manager \
**Status**: Implemented \
**Start Date**: 2020-06-23 \
**Jira Issue**: [DA-171 - [Delta & X] - Signal RFC](https://arya-analytics.atlassian.net/browse/DA-171)

# Summary

In this RFC I propose a framework for managing collections of goroutines in a CSP based
system. Virtually all of Delta's queries are served using a collection of long-lived
streams. Multiple goroutines inject values into, transform values through,
and extract values from these streams. Defining constructs for controlling and
observing the lifecycle of these routines is essential mitigating complexity
and maintaining high availability across the Delta cluster.

# Vocabulary and Abbreviations

**GR** - goroutine \
**CSP** - Communicating Sequential Processes

# Motivation

The Go language provides several primitives for managing the lifecycle of a goroutine.
`sync.WaitGroup` and `errgroup.Group` prevent a goroutine from exiting until all of
its dependent routines have exited. `context.Context`and `context.CancelFunc` allow a
GR to exit when a request/process is completed or aborted. Separating these two
constructs is useful for abstraction and modularity, but can result in the misuse of
context variables, leading to leaky goroutines and unpredictable concurrency.

A key challenge in designing a solution is guaranteeing its applicability across both
request and application scopes. Request scoped routines process values related to a
single request, while application scoped routines process values related to several
requests or application internal functionality.

These two scopes require wildly different lifecycle management. Request scoped GRs
must be able to exit when a request is complete, aborted, or an error occurs.
On the other hand, application scoped GRs *cannot* exit when a fatal error occurs for a
particular request, as other requests in the application may still be valid.

I'd like to propose a new definition called a *transient* error. In the case of a
request scoped GR, a transient error is one that may prevent a particular value
from being processed, but is *not* fatal to the request or application. In the
application scope, a transient error may be fatal to a request but not to the
application. Communicating transient errors back to the issuer of a request or the
observer of the application is essential.

# Design

This isn't a new problem, and many systems have written various ways of alleviating
the challenges of managing goroutines. Signal is not attempting to reinvent
the wheel, but rather to draw inspiration from battle tested solutions, albeit with
a few modifications to serve Delta's needs.

CockroachDB's [CtxGroup](https://github.com/cockroachdb/cockroach/tree/master/pkg/util/ctxgroup)
package merges `errgroup.Group` and `context.Context` into a single
type `ctxgroup.Group`
that injects a context into each goroutine associated with a particular
request, as opposed to asking the caller to provide one explicitly. This is a simple
way to clearly link goroutines to a request and prevent context misuse.

Their [stopper](https://github.com/cockroachdb/cockroach/blob/master/pkg/cli/start.go)
package was written before the addition of the `context` package to the standard library
in Go 1.7. It fills a similar role, but uses channels to send shutdown signals to
GRs. It also adds tracing, panic recovery, deferals, and leak detection.

The `signal` package's core `Context` type essentially modernizes `stopper` by
merging it with `ctxgroup.Group.`

## Grouping Routines

The `Context` type provides a simple interface for forking a new routine.

```go
type Go interface {
    Go(f func (ctx Context) error, opts ...GoOption)
}
```

It's the responsibility of the caller to ensure that the routine exits when the
injected context is canceled. If routine exits before the context is canceled with
a non-nil error, the context will be canceled. This behavior matches `errgroup.Group`
and is a useful feature for managing goroutines that depend on each other.

The `Go` method can also receive a list of options. These include parameters for adding
conditional deferals, tracing, panic recovery etc. The goal of these options is to allow
the caller to modify the behavior of the routine without having to modify the
definition of f itself. This is particularly useful for handling functions that can
operate both within an operation and request scope.

## Waiting for Routines to Exit

The `Context` type provides an interface that extends the methods from `sync.WaitGroup`:

```go
type WaitGroup interface {
    Wait() error
    Stopped() <-chan struct{}
}
```

Wait implements the same semantics as `sync.WaitGroup.Wait`. `Stopped` returns a channel
that is closed when all routines have exited.

## Transient Errors

The `Context` type provides an interface called `Errors` that can be used to send
transient errors back to the caller:

```go
type Errors interface {
    Transient() chan error
}
```

This is a rudimentary implementation, and essentially provides the same functionality
as passing an error channel around. The important element here is to define a standard
interface for handling transient errors, that way we don't end up with a bunch of 
strange implementations that are difficult to read and link.