# Cesium - Channel Segment Storage Engine

**Feature Name:** Channel Segment Storage Engine \
**Status**: In Review \
**Start Date**: 2022-05-17 \
**Authors**: emilbon99 \
**Jira Issue**:  [DA-149 - [Cesium] - RFC](https://arya-analytics.atlassian.net/browse/DA-149)

# Table of Contents

# Summary

In this RFC I propose an architecture for a time-series storage engine that can serve as Delta's primary means of data
persistence. This proposal places a heavy focus on interface as opposed to implementation details; a primary goal is
to define a sustainable interface that can serve as a clear boundary between Delta and an underlying storage engine.

This design is by no means complete, and is intended to be a starting point for continuous iteration as Delta's demands
evolve.

## Vocabulary

**Sample** - An arbitrary byte array recorded at a specific point in time.  \
**Channel** - A collection of samples across a time range. \
**Segment** - A partitioned region of a channel's data.  \
**Regular** - (in relation to time-series) - A 'regular' Channel is one whose samples are recorded at regular intervals
(1Hz, 5Hz, 10Hz, etc.) \
**Samples/Second** - A basic measure of write throughput. The size of a regular sample should be assumed as 8 bytes (
i.e. a
float64 value) unless otherwise specified, whereas an irregular sample is assumed to contain an additional 64 bit
timestamp.
Write throughput can also be expressed in terms of a frequency (1Hz, 5Hz, 25 KHz, 1 MHz, etc.) \
**DAQ** - Data Acquisition Computer.

This RFC expands on these definitions by defining specific properties of a Channel, Segment, and Sample.
These properties are omitted from the above definitions as they may fluctuate and affect storage engine implementation
details.

# Motivation

The product pivot from [Arya Core](https://arya-analytics.atlassian.net/wiki/spaces/AA/pages/819257/00+-+Arya+Core) to
[Delta](https://arya-analytics.atlassian.net/wiki/spaces/AA/pages/9601025/01+-+Delta) is the main driver behind this
RFC.
Moving from a 'database proxy' a single binary 'database' architecture means we must either:

1. Find an existing embedded storage engine written in Go.
2. Write a new storage engine tailored towards Delta's specific use case.

Writing a database storage engine is quite an endeavour, taking years and many development cycles, so we'd ideally use
an existing storage engine or at least extend its functionality.

## Existing Solutions

### Key-Value Stores

There are a number of popular key-value stores implemented in Go. Most of these are inspired by earlier alternatives
written in C or C++ such as [RocksDB](http://rocksdb.org/) or [LevelDB](https://github.com/google/leveldb]). The most
popular
are [badger](https://github.com/dgraph-io/badger), [bolt](https://github.com/boltdb/bolt),
and [pebble](https://github.com/cockroachdb/pebble).

These all use an LSM + WAL style architecture, which is a good fit for frequent reads and writes on small amounts
of data. However, when it comes to high rate append only workloads, they do not scale as well as one might hope. Pebbles
own
[benchmarks](https://cockroachdb.github.io/pebble/) show a maximum write throughput of (approximately) 60,000 samples
per second, far
below Arya Core's throughput of 6 million values per second. An elastic throughput in the range several hundreds of
millions
of values per second is reasonable for append only writes to an SSD.

It's naive to think we can reach comparable performance to slamming random bytes into a disk, but it's not unreasonable
to assume we can drastically improve on the speed of a key-value store for a time-series only workload.

### Time-Series Specific Stores

The embedded time-series storage options available in Go are limited. The most popular I've found
is [tstorage](https://github.com/nakabonne/tstorage), which is tailored towards irregular time-series data.
It's benchmarks shows a maximum write throughput of about 3,000,000 samples per second.

Delta is unique in that almost all of its uses involve storing regular time-series data
(see [Restrictions on Time-Series](#restrictions-on-time-series). This is a huge advantage in terms of database
simplicity
and performance. `tstorage` doesn't take advantage of data regularity, and is missing out on the benefits it provides.

### Distributed Key-Value Stores

Using a distributed key-value store is theoretically a great fit for Delta, as it meets requirements for both cluster
wide metadata in addition to segmented telemetry.

[etcd](https://etcd.io/) is the most popular choice in this category, and can be run in a pseudo-embedded mode using
[embed](https://pkg.go.dev/go.etcd.io/etcd/embed). This package allows for embedded bootstrapping of a cluster.
Unfortunately,
but API calls to the key-value interfaces must still be done using a client over a network.

etcd uses Raft to achieve consensus, and replicates writes to all nodes in the cluster. This means that write
amplification
scales along with the number of cluster members. This is ok for small deployments, but quickly becomes a problem for
larger
clusters (the authors of etcd advise against running a cluster with greater than seven nodes).

etcd's write amplification over the network is also problematic for large quantities of data. It's unreasonable to
expect
a write throughput in the tens of millions of samples/s for a cluster of seven nodes, even over a very performant
network.

# Design

The proposed design is named after [Cesium](https://en.wikipedia.org/wiki/Caesium), the element most commonly used in
atomic clocks. It focuses on simplicity by restricting:

1. The attributes of the time-series data that can be stored.
2. The available query patterns to a simple range based lookup (while still allowing future implementations
   to support more complex patterns).

Cesium expects certain queries to request 100+ GB of data, and uses a pipe based architecture to serve
long-running queries as streams that return data to the caller as it is read and parsed from disk. This functionality
is extended to provide support for client side iterators. This is ideal for maximizing IO throughput by allowing the
client to transform or transfer data over the network as more segments are read.

## Restrictions on Time-Series

Delta is designed to work with data acquisition hardware, and as such, must be optimized for time-series data that
arrives
at predictable, high sample rates (25 Khz+). This is very different to the typical IOT use case that involves
edge devices streaming low rate data at unpredictable intervals.

This is also very different from a software infrastructure monitoring system that can frequently discard old data. Delta
stores data that must be kept for long periods of time.

### Channels

A channel's **sample rate** must be predefined. This is by far the largest restriction and optimization that Cesium
makes.
When creating a new channel, the caller must provide a fixed sample rate:

```go
cesium.NewCreateChannel().WithRate(100 * cesium.Hz).Exec(ctx)
```

Samples written to this channel are assumed to have a constant separation of 10ms between them. Actual separations
between samples are not validated or stored. Even the most precise sensors and devices have minor irregularities in
their
sample rates (+/- a few nanoseconds in the case of most data acquisition computers (DAQ)). Cesium leaves it to the
caller to
decide whether fluctuations in the sample rate are acceptable.

This decision was made with an assumption that the precision of data recorded by a DAQ is high enough that the
consumer doesn't really care about the exact timestamp of a particular sample. This assumption can be extended beyond
the high rate hardware DAQ use case to IOT or infrastructure monitoring workloads. For example, a DevOps engineer wants
to monitor the number of requests to a particular API endpoint. The web server pushes this data to a Cesium backed
monitoring service at intervals of 5 seconds +/- 1 second. Cesium would assume these values are written to the channel
at even, five second intervals e.g. 0s, 5s, 10s, 15s as opposed to 0s, 6s, 9s, 15s, etc. The DevOps engineer probably
doesn't care about the exact
regularity of the data.

Of course there are cases where precise spacing is critical. In this case, Cesium is probably not the best choice.

A channel's **data type** must also be predefined. This is typical for a time-series database, but Cesium places no
restrictions on the data types that can be stored. A **data type** in Cesium is essentially an alias for its **density**
i.e. the number of bytes per sample. For example, a caller could create a new channel that accepts `float64` samples
by setting the byte density to 8:

```go 
// Setting the byte density manually.
cesiun.NewCreateChannel().WithRate(100 * cesium.Hz).WithType(8 * cesium.Byte).Exec(ctx)

// Using a pre-defined type alias.
cesium.NewCreateChannel().WithRate(100 * cesium.Hz).WithType(cesium.Float64).Exec(ctx)
```

### Segments

The implications of these restrictions becomes apparent when designing  **segments**. A **segment** is a contiguous run
of a channel's data. A segment stores the following information:

```go
package main

type Segment struct {
    // Start stores a nanosecond precision timestamp of the first sample in the segment.
    Start int64
    // Data stores a set of regular, contiguous, binary encoded samples.
    Data []byte
}
```

Because samples are regularly spaces, we only need to store the start time of the segment. The timestamp of any sample
can be calculated with the following equation:

$$t_{n} = t_{0} * \frac{n*D}{S}$$

### Constant Sample Size

## Handling Arbitrary Data Types

## Designing for Streams

## Providing Elastic Throughput

## Data Layout

### Segment KV

### Segment Meta Data

## Batching

## Debouncing

## Iteration

## Deletes

## Aggregation, Downsampling, and Rudimentary Transformations

