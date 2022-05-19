# Aspen - Gossip Based Peer to Peer Network

**Feature Name**: Aspen, a Gossip Based Peer to Peer Network \
**Status**: Proposed \
**Start Data**: 2020-05-18 \
**Authors**: emilbon99 \
**Jira Issue** - [DA-153 - [Aspen] - RFC](https://arya-analytics.atlassian.net/browse/DA-153)

# Table of Contents

# Summary

In this RFC I propose an architecture for a gossip based network that can meet Delta's distributed storage and cluster
membership
requirements. Gossip based dissemination is an efficient method for sharing cluster wide state in an eventually
consistent
fashion. Delta requires a relatively small distributed store that should ideally be available even on loss of connection
to the rest of the cluster. A Gossip based network lays the foundations for dynamic cluster membership, failure
detection,
and the eventual construction of a strongly consistent store.

This proposal focuses on extreme simplicity to achieve a minimum viable implementation. It aims to provide only
functionality
that contributes towards meeting the requirements laid out in
the [Delta specification](https://arya-analytics.atlassian.net/wiki/spaces/AA/pages/9601025/01+-+Delta).

# Vocabulary

**Node** - A machine in the cluster. \
**Cluster** - A group of nodes that can communicate with each other.

# Motivation

This RFC is largely driven by the lack of a distributed key-value store that meets Delta's needs. OLTP databases would
typically rather fail than risk data loss. Delta requires the opposite: a distributed data store capable of providing
services even when the rest of the cluster is unreachable.

Consider a set of redlines that are executed when a node loses connection. If the redline is kept in an integrity first
database, the node will be unable to retrieve the proper information to shut down the system safely. In the case of a
launch control system, this can result in the loss of a vehicle.



