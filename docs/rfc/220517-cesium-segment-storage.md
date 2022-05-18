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

The proposed design is named after [Cesium](https://en.wikipedia.org/wiki/Caesium), the stage most commonly used in
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
cesiun.NewCreateChannel().
   WithRate(100 * cesium.Hz).
   WithType(8 * cesium.Byte).
   Exec(ctx)

// Using a pre-defined type alias.
cesium.NewCreateChannel().
   WithRate(100 * cesium.Hz).
   WithType(cesium.Float64).
   Exec(ctx)
```

### Segments

The implications of these restrictions becomes apparent when designing  **segments**. A **segment** is a contiguous run
of a channel's data. A segment stores the following information:

```go
type Segment struct {
// Start stores a nanosecond precision timestamp of the first sample 
// in the segment.
Start int64
// Data stores a set of regular, contiguous, binary encoded samples.
Data []byte
}
```

Because samples are regularly spaced, we only need to store the start time of the segment. The timestamp of any sample
can be calculated with the following equation:

<p align="middle">
<br>
<img src="https://render.githubusercontent.com/render/math?math=t_{n} = t_{0} * \frac{n*D}{S}" height="30px" alt="latex eq" >
</p> 

Where `D` is the density of the channel in bytes, `S` is the sample rate in Hz, and the independent variable `n`
represents
the nth sample in the segment (the first sample has index 0).

A segment places no restrictions on the amount of samples it can store. This has important implications for both
durability and write throughput. Larger segments are less durable (written less frequently) but can achieve a higher
throughput for both reads and writes, as segment data is written contiguously on disk. See [Data Layout](#data-layout)
and [Providing Elastic Throughput](#providing-elastic-throughput) for more details.

## Handling Arbitrary Data Types

Cesium places no restrictions on the data types can be stored, and instead represents a type using a **density**.
This is atypical for a time-series database, but provides flexibility for the caller to define custom data types such
as images, audio, etc. Creating a custom data type is as simple as defining a constant:

```go
// TenByTenImage is a custom data type where each sample is 10 * 10 * 3 bytes in size.
const TenByTenImage cesium.DataType = 10 * 10 * 3

// Create a new channel that accepts samples of type TenByTenImage.
cesium.NewCreateChannel().
WithRate(100 * cesium.Hz).
WithType(TenByTenImage).
Exec(ctx)
```

It's important to note that Cesium does not plan to validate the data type. It's the caller's responsibility to ensure
that partial
samples are not added to a segment. This is mainly for simplicity and separation of concern, as the caller typically
has more information about the data being written than the storage engine itself does. This decision is definitely
not hard and fast, as adding simple validation is relatively easy (we can assert `len(data) % DataType == 0` for
example).

## Extending an Existing Key-Value Store

Cesium's data can be separated into two categories: **metadata** and **segment data**. Metadata is context that can be
used to fulfill a particular request for segment data. Segment data is the actual time-series samples to be stored,
retrieved,
or removed from disk.

Instead of writing a storage engine that can handle both metadata and segment data, Cesium proposes an alternative
architecture that *extends* an existing key-value store. This store handles all metadata, and Cesium uses to index the
location of Segments on disk.

This approach drastically simplifies Cesium's implementation, allowing it to make use of well-written iteration APIs
to execute queries in an efficient manner. Although the actual key-value store used is of relative unimportance, I
chose CockroachDB's [Pebble](https://github.com/cockroachdb/pebble) as it provides a RocksDB compatible API with
well written prefix iteration utilities (very useful for range based lookups).

There are a number of alternatives such as Dgraph's [Badger](https://github.com/dgraph-io/badger). I haven't done any
significant research into the pros and cons of each, as the performance across most of these stores seems comparable.

## Designing for Streaming and Iteration

Optimizing IO is an essential factor in building data intensive distributed systems. Running network and disk IO
concurrently can lead to significant performance improvements for large data sets. Cesium aims to provide simple
streaming interfaces that lend themselves to concurrent access. Cesium is built in what I'm calling a 'pipe' based
model as it bears a resemblance to Unix pipes.

Core vocabulary for the following technical specification:

**Stage**: An interface that receives samples from one or more streams, does some operation on those samples, and
pipes the results to one or more output streams. In a [Sawzall](https://research.google/pubs/pub61/) style processing
engine, an stage would be comparable to an aggregator.

**Individual Stage** - An stage that is involved in serving only one request.

**Shared Stage** - An stage that is involved in serving multiple requests (i.e. several input streams from different
processes)

**Pipe**: A pipe is an ordered sequence of stages, where the output stream(s) of each stage is the input stream(s)
for the next stage. In Cesium's case, the ends of the pipe are the caller and disk reader respectively (the order
reverses for different query variants).

**Assembly**: The processing of selecting and initializing segments for a particular pipe. Assembly is a process that
typically parses a query, builds a plan, and assembles the pipe.

**Execution**: The transfer/processing of samples from one end of the pipe to the other i.e. the streaming process.
Often times, the Assembly process doesn't provide enough information to fully execute the query, so the execution
process
can parse context within the samples to order additional transformations/alternate routing.

**Query** - The process of writing a structured request for pipe assembly and execution. A query is often assembled
using some sort of ORM styled method chaining API, packed into a serializable structure, and passed to a processing
engine that can execute it.

**Operation**  - Cesium Specific - A set of instructions to perform on a particular location on the disk. This can be
reading, writing, etc.

Cesium's query execution model involves a set of individual stages that perform high-level query specific tasks,
connected to low level batching, debouncing, queueing, and ultimately disk IO stages.

### Retrieve Query Execution

A query with the following syntax:

```go
// res is a channel that returns read segments along with errors encountered 
// during execution. err is an error encountered during query assembly.
res, err := cesium.NewRetrieve().
WhereChannels(1).
WhereTimeRange(telem.NewTimeRange(0, 15)).
Stream(ctx)
```

We're looking for all data from a channel with key 1 from time range 0 to 15 (the units are unimportant). We can use
the following pipe:

**Stage 0** - Individual - Assembly - Parses a query and does KV operations to generate a set of disk operations. Passes
these operations to Stage 1.

**Stage 1** - Individual - Interface - Queues a set of disk operations and waits for their execution to complete. Closes
the response channel when all ops are completed.

**Stage 2** - Shared - Debounced Queue - Debounces disk operations from an input stream and flushes them to the next
stage after either reaching a pre-configured maximum batch size or a ticker with a pre-configured interval has elapsed.
This is used to modulate disk IO and improve the quality of batching in the next stage.

**Stage 3** - Shared - Batcher - Receives a set of disk operations and batches them into more efficient groups. This
stage first groups together disk operations that are related to the same file, and then sorts the operations by the
offset
in the file. This maximizes sequential IO.

**Stage 4** - Shared - Persist - Receives a set of disk operations and distributes them over a set of workers to perform
concurrent access on different files. This stage also manages a set of locks on top of a file system to ensure multiple
workers don't access the same file in parallel. This stage is also shared with the create query pipe.

<p align="middle">
<img src="images/220517-cesium-segment-storage/retrieve-pipe.png" width="50%">
<h6 align="middle">Retrieve Query Pipe</h6>
</p>

### Create Query Execution

A query with the following syntax:

```go
// req is a channel that sends segments for persistence to disk.
// res is a channel that returns any errors encountered during execution.
// err is an error encountered during query assembly.
req, res, err := cesium.NewCreate().WhereChannels(1).Stream(ctx)
```

We're writing a stream of sequential segments for a channel with key 1 to disk. We can use the following pipe:

**Stage 0** - Individual - Assembly - Acquires a lock on the channel and does KV operations for metadata context. Forks
stage 1.

**Stage 1** - Interface/Parser - Receives a stream of create requests from the caller, validates them, does KV
operations
for metadata context, and passes a set of parsed operations to the next stage.

**Stage 2** - Debounced Queue - Same behavior as for [Retrieve](#retrieve-query-execution).

**Stage 3** - Shared - Batcher - Receives a set of disk operations and batches them into more efficient groups. It first
groups disk operations belonging to the same file, then groups them by channel, and finally sorts them in time-order.

**Stage 4** - Shared - Persist - Same behavior as for [Retrieve](#retrieve-query-execution). This stage is shared
with the retrieve query pipe.

<p align="middle">
<img src="images/220517-cesium-segment-storage/create-pipe.png" width="50%">
<h6 align="middle">Create Query Pipe</h6>
</p>

### Combined Pipe Architecture

<p align="middle">
<img src="images/220517-cesium-segment-storage/pipe.png" width="100%">
<h6 align="middle">Combined Cesium Pipe Architecture</h6>
</p>

Future iterations may involve inserting stages into the simplex stream between the Operation and Interface stage
to perform aggregations on the data before returning it to the caller.

## Data Layout + Operations

When writing data to disk, cesium holds the following principle in mind: *sequential IO is better than random IO*.

At the lowest level, there are two ways to structure time-series data on disk: in rows vs. in columns. In rows, the
first column is a timestamp for the sample, and subsequent columns are samples for a particular channel. The following
table
is a simple representation:

| Timestamp | Channel 1 | Channel 2 | Channel 3 |
|-----------|-----------|-----------|-----------|
| 15:00:00  | 1         | 2         | 3         |
| 15:00:01  | 4         | 5         | 6         |
| 15:00:02  | 7         | 8         | 9         |

A row can be represented as a tuple of values: `(15:00:00, 1, 2, 3)`. Each row is serialized and saved to disk
sequentially.
This storage format is ideal for irregular samples where channels are queried in groups i.e. the caller requests
data for Channels 1, 2, and 3 at the same time.

Columnar storage, on the other hand, writes samples for an individual channel sequentially. This is ideal for Delta's
use
case, as the timestamps of regular samples can be compacted, and a caller often requests data for a small number of
channels at once. The following represents the layout of a columnar segment on disk:

| 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 |
|---|---|---|---|---|---|---|---|---|

This representation omits the following metadata:

1. Defining the timestamp of the segment,
2. The key of the channel, and
3. The spacing between samples.

An option is to include this metadata along with the segment:

| Key 1 | 15:00:00 | 25Hz | 1   | 2   | 3   | 4   | 5   | 6   | 7   | 8   | 9   |
|-------|----------|------|-----|-----|-----|-----|-----|-----|-----|-----|-----|

Adding this 'header' is the most intuitive way to represent the data, but has implications for retrieving it.
When searching for the start of a time range, Cesium must jump from header to header until it finds a matching
timestamp. For larger files, this can be a costly operation. Instead, Cesium stores the segment header in key-value 
storage along with its file and offset. When retrieving a set of segments, Cesium first does a kv lookup to find the
the location on disk, then proceeds to read it from the file. 

## Providing Elastic Throughput

## Batching

## Debouncing

## Iteration

## Deletes

## Aggregation, Downsampling, and Rudimentary Transformations

