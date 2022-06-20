# Delta - Segment Distribution

**Feature Name**: Delta - Segment Distribution \
**Status**: Proposed \
**Start Data**: 2022-06-04 \
**Jira Issue** [DA-154- [Delta & Cesium] - Segment Architecture](https://arya-analytics.atlassian.net/browse/DA-154)

# Table of Contents

# Summary

In this RFC I propose an architecture for exposing time-series segment storage as a monolithic data space. This proposal
focuses on serving simple range-based queries in an efficient manner, while also laying the foundations for data
replication and transfer across the cluster.

Defining a clear level of abstraction for the data space is challenging. The distribution layer must maintain adequate
low level control to support distributed aggregation, but must also minimize complexity for the layers above.

## Vocabulary

**Sample** - An arbitrary byte array recorded at a specific point in time. \
**Channel** - A collection of samples across a time range. \
**Segment** - A partitioned region of a channel's data. \
**Node** - A machine in the cluster.
**Cluster** - A group of nodes that can communicate with each other.
**Leaseholder** - A node that holds a lease on a particular piece of data. The leaseholder is the only node
that can modify the data.


## Motivation

## Design

