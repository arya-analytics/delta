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
**Regular** - (in relation to time-series) - A 'regular' Channel is one who's samples are recorded at regular intervals
(1Hz, 5Hz, 10Hz, etc.)

This RFC expands on these definitions by defining specific properties of a Channel, Segment, and Sample.
These properties are omitted from the above definitions as they may fluctuate and affect storage engine implementation
details. 

# Motivation

The product pivot from [Arya Core](https://arya-analytics.atlassian.net/wiki/spaces/AA/pages/819257/00+-+Arya+Core) to 
[Delta](https://arya-analytics.atlassian.net/wiki/spaces/AA/pages/9601025/01+-+Delta) is the main driver behind this RFC.
Moving from a 'database proxy' a single binary 'database' architecture means we must make the following decision:

1. Find an existing embedded storage engine written in Go.
2. Write a new storage engine tailored towards Delta's specific use case.

## Existing Solutions

### Key-Value Stores

There are a number of popular key-value stores implemented in Go. Most of these are inspired by earlier alternatives
written in C or C++ such as [RocksDB](http://rocksdb.org/) or [LevelDB](https://github.com/google/leveldb]). The most popular
are [badger](https://github.com/dgraph-io/badger), [bolt](https://github.com/boltdb/bolt), and [pebble](https://github.com/cockroachdb/pebble).

These all use an LSM + WAL style architecture, which is a good fit for frequent reads and writes on small amounts 
of data. However, when it comes to high rate append only workloads, they do not scale as well as one might hope. Pebbles own 
[benchmarks](https://cockroachdb.github.io/pebble/) show a maximum write throughput of 60,000 values per second, far below
Arya Core's throughput of 6 million values per second. An elastic throughput in the range several hundreds of millions of 
values per second is reasonable for append only writes to an SSD. 

It's naive to think we can reach comparable performance to slamming random bytes into a disk, but it's not unreasonable to 
assume we can drastically improve on the speed of a key-value store for a time-series only workload.


### Time-Series Stores

The embedded time-series storage options available in Go are limited. The most popular I've found it [tstorage](https://github.com/nakabonne/tstorage) 
which tailored towards irregular time-series data. It's benchmarks shows a maximum write throughput of about 3,000,000 samples per second.

Delta is unique in that almost all of its use cases involve storing regular time-series data. This is a huge advantage in 
terms of database simplicity and performance. `tstorage` doesn't seem optimized towards our particular use case.



