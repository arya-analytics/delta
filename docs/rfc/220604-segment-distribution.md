# Delta - Segment Distribution

**Feature Name**: Delta - Segment Distribution \
**Status**: Proposed \
**Start Data**: 2022-06-04 \
**Jira Issue** [DA-154- [Delta & Cesium] - Segment Architecture](https://arya-analytics.atlassian.net/browse/DA-154)

# Table of Contents

# Summary

In this RFC I propose an architecture for exposing time-series segment storage as a
monolithic data space. This proposal focuses on serving simple, range-based queries in
an efficient manner while laying the foundations for data replication and transfer
across the cluster.

Defining a clear level of abstraction for the data space is challenging. The distribution
layer must maintain adequate low level control to support distributed aggregation,
but must also minimize complexity (in terms of locality and networking) for the layers
above.

This RFC lays out a domain-oriented, locality abstracted interface that allows callers
to read and write data to the cluster as if it was a single machine. This interface
does not require the user to be aware of the underlying storage location, but provides
additional context if the caller wants to perform network optimization themselves (this
is a similar approach to the one taken by CockroachDB between their transaction and SQl
layers).

## Vocabulary

**Sample** - An arbitrary byte array recorded at a specific point in time. \
**Channel** - A collection of samples across a time range. \
**Segment** - A partitioned region of a channel's data. \
**Node** - A machine in the cluster. \
**Cluster** - A group of nodes that can communicate with each other. \
**Leaseholder** - A node that holds a lease on a particular piece of data. The leaseholder is the only node
that can modify the data. \
**Data Warehouse (DWH)** - A system used for storing and reporting data for analysis
and business intelligence purposes. Data warehouses typically involve long-running
queries on much larger data sets than typical OLTP systems. Data warehouse queries
fall into the OLAP category of workloads.
**OTN** - Over the Network.

## Motivation

Separating storage and compute has become a popular technique for scaling data
intensive systems (see [The Firebolt Cloud Data Warehouse Whitepaper](https://www.firebolt.io/resources/firebolt-cloud-data-warehouse-whitepaper)).
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
billion rows, a warehouse would need to retrieve massive amounts of data over the
network from storage, compute the count, and then return the value. To reduce network 
traffic, a DWH can pre-compute a set of materialized indexes and aggregations (i.e.
pre-calculate a generalized count for common query patterns). A network trip that 
once required hundreds of gigabytes of data may now require only a few hundred bytes.
Pre-aggregation is expensive, and the challenge comes in determining the most *useful* 
aggregations for a particular data set (constantly computing an un-queried average on 
billions of rows is a massive waste of cpu time).

The above example is extreme, but still outlines the value of performing aggregations 
closer to the data source in order to reduce the amount of information transferred OTN.


## Design

