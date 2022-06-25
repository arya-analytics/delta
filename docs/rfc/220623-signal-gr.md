# Signal - GoRoutine Manager

**Feature Name**: Signal - GoRoutine Manager \
**Status**: Implemented \
**Start Date**: 2020-06-23 \
**Jira
Issue**: [DA-171 - [Delta & X] - Signal RFC](https://arya-analytics.atlassian.net/browse/DA-171)

# Summary 

In this RFC I propose a framework for managing collections of goroutines in a CSP based
architecture. Virtually all of Delta's queries are served using a collection of
long-lived streams. Multiple goroutines inject values into, transform values through,
and extract values from these streams. Having clear visibility and control into the
lifecycle of the different goroutines involved in a stream is important for improving
performance and maintaining integrity across the Delta cluster.

Signal provides a framework for forking these goroutines, shutting them down safely, and
managing relationships between them. This framework takes heavy inspiration from the
Official Go
Blog's [Go Concurrency Patterns: Pipelines and Cancellation](https://go.dev/blog/pipelines).

# Vocabulary 

**GR** - goroutine

# Ruminations on Goroutine Managers

There are two main contexts in which a goroutine manager is used within Delta:

### Request Scoped

A request scoped manager controls goroutines that are scoped to a single request.
Imagine a client that opens a server-side iterator that consumes data across several
nodes in the cluster. We'd need a GR for each peer node we need to read data from,
a main GR to aggregate values returned by the peers, and additional routines to perform
aggregation, transformation, filtering, etc.

The important thing to note is that all GRs are involved in serving the *same* request.
This is particularly relevant for error handling. If we're hitting Node 1 for
segment data and it dies in the middle of request, we can no longer proceed. This is 
error is considered **fatal** to the request. In this case, we need to do two things:

1. Return the error to the main routine for the request. One option is to use
   an `errgroup.Group` forked from the main GR. The other option is to use a
   channel to communicate the error.
2. Acknowledge the error, and shutdown all other GRs involved in serving the request
   without panicking. This is easy to accomplish by cancelling a request context.

Handling this error is easy. Things get challenging when we start to handle 
**transient** errors. A transient error isn't a
showstopper, but still needs to be returned by the caller. A common transient error
we could encounter during iteration is attempting to read a range of segment data that
does not exist. We can continue iterating over the remaining segments without causing
integrity issues for the caller, but we still need to return the error to them somehow.
If we take the same approach as above, we start running into issues.

Let's say we use an error group in step one. We can't just return the error from the
node 1 client GR, because then we'll exit the routine and stop serving future iteration
requests for node 1. 

If we use a channel instead, we run into a another problem. If we pass an error through
a channel to the main GR, the main routine wil think the error is fatal the to
request, and will cancel the request context (shutting down all other routines).

We need a mechanism for distinguishing between transient and fatal errors.

### Application Scoped

An application scoped manager controls goroutines that are scoped to one or more 
requests. A common example is a worker pool that executes a set of operations on disk.
Because we can't concurrently access files, it's more efficient to pre-fork a set of
goroutines that execute operations for multiple requests.

Error handling becomes more complicated in this case. A fatal error for a particular 
request may not be fatal to other requests. In the same vein, a transient error for one
request shouldn't be returned to callers for other requests (meaning we can't use a 
single error channel). And, lastly, we may encounter fatal and transient failures for 
the application as a whole. For example, we may run out of disk space (fatal) or 
encounter a transient failure to a network file system (SMB, NFS, S3, etc).

### Impact

Request and application scoped managers have different semantics. Signal must provide
a way to control concurrent systems in a manner suitable for both of these use cases.

# Design

Signal tries to maintain an API similar to the standard library's concurrency helpers
`sync.WatiGroup` and `errgroup.Group` while adding support for two essential elements:
scheduled, graceful shutdowns and context cancellation.

The main entrypoint is the `Conductor` interface. Each conductor is responsible for
managing a set of goroutines involved in a particular stream. 

## Forking Goroutines

The conductor provides a single method for forking a new goroutine:

```
type Go interface {
    Go(f func(ctx Context) error, opts ...GoOption)
}
```

The provided `Context` struct implements the `context.Context` interface while also 
providing a channel that can be used to signal the routine to shut down gracefully.
The provider of the function should implement the following functionality:

1. If the given context completes (<-ctx.Done()), the goroutine should abruptly abort
operations and return the context error.
2. If the signal channel is closed (<-ctx.S), the goroutine should gracefully process
any remaining operations and return a nil error.

The caller can provide additional options to the `Go` method which can inject custom 
behavior and metadata into the goroutine (such as a name).

## Graceful Shutdown

The conductor provides a method for gracefully shutting down a 
