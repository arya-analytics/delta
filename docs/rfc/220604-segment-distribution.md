# Delta - Segment Distribution

**Feature Name**: Delta - Segment Distribution \
**Status**: Proposed \
**Start Data**: 2022-06-04 \
**Jira
Issue** [DA-154- [Delta & Cesium] - Segment Architecture](https://arya-analytics.atlassian.net/browse/DA-154)

# Table of Contents

# Summary

In this RFC I propose an architecture for exposing time-series segment storage as a
monolithic data space. This proposal focuses on serving simple, range-based queries in
an efficient manner while laying the foundations for data replication and transfer
across the cluster.

Defining a clear level of abstraction for the data space is challenging. The
distribution
layer must maintain adequate low level control to support distributed aggregation,
but must also minimize complexity (in terms of locality and networking) for the layers
above.

This RFC lays out a domain-oriented, locality abstracted interface that allows callers
to read and write data to the cluster as if it was a single machine. This interface
does not require the user to be aware of the underlying storage location, but provides
additional context if the caller wants to perform network optimization themselves (this
is a similar approach to the one taken by CockroachDB between their transaction and SQl
layers).

# Vocabulary

**Sample** - An arbitrary byte array recorded at a specific point in time. \
**Channel** - A collection of samples across a time range. \
**Segment** - A partitioned region of a channel's data. \
**Node** - A machine in the cluster. \
**Cluster** - A group of nodes that can communicate with each other. \
**Leaseholder** - A node that holds a lease on a particular piece of data. The
leaseholder is the only node
that can modify the data. \
**Data Warehouse (DWH)** - A system used for storing and reporting data for analysis
and business intelligence purposes. Data warehouses typically involve long-running
queries on much larger data sets than typical OLTP systems. Data warehouse queries
fall into the OLAP category of workloads.
**OTN** - Over the Network.

# Motivation

Separating storage and compute has become a popular technique for scaling data
intensive systems (
see [The Firebolt Cloud Data Warehouse Whitepaper](https://www.firebolt.io/resources/firebolt-cloud-data-warehouse-whitepaper))
.
This decoupling is a double-edged sword. Processing engines and storage layers can scale
independently, allowing the data warehouse to flexibly scale to meet the needs of its
users. However, processing engines must now retrieve data from storage OTN, which is
a costly operation that can cause problems when retrieving
large datasets.

The simplest way to solve this problem is by reducing the amount of data a processing
engine must retrieve OTN from the storage layer. This idea, while obvious, is
challenging to implement.

DWH queries typically perform aggregations on large spans of data, returning a small
value (such as an average, sum, or count to the caller). To serve a count over one
billion rows, a warehouse would need to retrieve massive amounts of data from storage,
compute the count, and then return the value. To reduce network traffic, a DWH can
pre-compute a set of materialized indexes and aggregations (i.e. pre-calculate a
generalized
count for common query patterns). A network trip that
once required hundreds of gigabytes of data may now require only a few hundred bytes.
Pre-aggregation is expensive, and the challenge comes in determining the most *useful*
aggregations for a particular data set (constantly computing an un-queried average on
billions of rows is a massive waste of CPU time).

The above example is extreme, but still outlines the value of performing aggregations
closer to the data source in order to reduce the amount of information transferred OTN.

Delta falls into a category that blends the lines between a data warehouse and a
traditional OLTP database. On the one hand, aggregations are very common (i.e. maximum
value for a sensor over a particular time-range). On the other hand, it's typical for a
user to retrieve massive amounts of raw time-series data for advanced computing
(such as signal processing). The first pattern lends itself well to a decoupled
architecture, while the second benefits greatly from reducing the amount of network
hops.

This RFC attempts to reconcile these two workloads by providing an architecture
that separates the algorithms/components for storing data from those who perform
aggregations/computations on it. Defining clear requirements and interfaces for the
distribution layer is essential to the success of this reconciliation. What are the
algorithms in the distribution layer responsible for? Should we provide rudimentary
support for aggregations? Should we make the caller aware of the underlying network
topology to enable optimization? Or should we make it a completely black box? The
following sections reason about and propose an architecture that answers these
questions.

# Design

The proposed distribution layer (DL) architecture will expose cluster storage as a
monolithic data space that provides *optional* locality context to caller. A user can
read and write data from the DL as a black box without any knowledge of the underlying
cluster topology, but can also ask for additional context to perform optimizations
within its own layer/domain.

This is a similar approach to CockroachDB's separation between their
[Distributed SQL](https://github.com/cockroachdb/cockroach/blob/master/docs/RFCS/20160421_distributed_sql.md)
and key-value layers. When executing a query, the SQL layer can turn a logical plan
into a physical plan that executes on a single machine, performing unaware reads and
writes from the distributed kv layer below. It can, however, also construct a physical
plan that moves aggregation logic to the SQL layer's of *other* machines in the
cluster. This distributed physical plan can perform aggregations on nodes where the
data is stored, and then return a much smaller result OTN back to the SQL layer of the
responsible node.

Delta's distribution layer plays a similar role to the key-value layer in CRDB. Its
main focus, however, will be to serve time-series segments instead of key-value pairs.
Layers above the DL will do the heavy lifting of generating and executing a physical
plan for a particular query. Parsing a physical plan that can be distributed
across multiple nodes is by no means an easy task. CockroachDB was already several
years old before the development team began to implement these optimizations.
By providing topology abstraction in the distribution layer, we enable a simple path
forward to a Delta MVP while laying the groundwork for distributed optimizations.

## Principles

**Computation and Aggregation** - DL contains no computation or
aggregation logic. Its focus is completely on serving raw segments reads and writes
efficiently.

**Network Awareness** - DL's interface does *not* require the
caller to be aware of data locality or underlying network topology. The distribution
layer provides optional context to the caller if they want to implement optimizations
themselves.

**Layer Boundary** - Services/domains that do *not* require custom distribution logic
do not have any components within DL.

**Domain Oriented** - DL does not expose a single facade as its interface. Instead,
it composes a set of domain-separated services that rely on common distribution logic.

**Generic** - DL only supports rudimentary, low-level queries in a similar fashion to
a key-value store. It should not provide any support for specific data types or
specialty queries.

**Transport Abstraction** - DL is not partial to a particular network transport
implementation (GRPC, WS, HTTP, etc.). It's core logic does not interact with any
specific networking APIs.

## Storage Engine Integration

Delta's distribution layer directly interacts with two storage engines: Cesium and 
Aspen. DL uses [Aspen](https://github.com/arya-analytics/delta/blob/main/docs/rfc/220518-aspen-distributed-storage.md)
for querying cluster topology as well as storing distributed key-value data. 
It uses one or more [Cesium](https://github.com/arya-analytics/delta/blob/main/docs/rfc/220517-cesium-segment-storage.md) 
database(s) for reading and writing time-series data from disk. Because the 
distribution layer uses multiple storage engines, there's a certain amount of overlap
and data reconciliation that must be performed in order to ensure that information 
stays consistent (this is particularly relevant for [channels](#Channels)).


### Aspen

Aspen implements two critical pieces of functionality that the distribution layer 
depends on. The first is the ability to query the address of a node in the cluster:

```go
addr, err := aspenDB.Resolve(1)
```

This query returns the address of the node with an ID of `1`. The DL uses this to 
determine the location of a channel's lease and its corresponding segment data.

The second piece of functionality is an eventually consistent distributed key-value 
store. The DL uses aspen to propagate two important pieces of metadata across the 
cluster:

1. Channels in the cluster (name, key, data rate, leaseholder node, etc.)
2. Segments for a channel (i.e. what ranges of a channel's data exist on which node).

### Cesium

Cesium is the main time-series storage engine for Delta. Each Cesium database occupies a
single directory on disk. The distribution layer interacts with Cesium via four APIs:

```go
// Create a new channel.
db.CreateChannel()
// Retrieve a channel.
db.RetrieveChannel()
// Write segments.
db.NewCreate().Exec(ctx)
// Read segments.
db.NewRetrieve().Iterate(ctx)
```

Besides these four interfaces, Delta treats Cesium as a black box.

### Integrity/Reconciliation

## Channels

### Keys

### Query Patterns

### Multi-Data Store Reconciliation

## Segment Reads

### Query Patterns

### Iteration

### Networking Details

## Segment Writes

### Query Patterns

### Networking Details

## Distributed Physical Plans
