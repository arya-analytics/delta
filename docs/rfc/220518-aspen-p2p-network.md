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
**Redline** - A threshold for a particular channel value. Most often represents a safety limit that triggers an action.
**Initiator** - A node that initiates gossip with another node.
**Peer** - A node that engages in gossip with an initiating node.

# Motivation

This RFC is largely driven by the lack of a distributed key-value store that meets Delta's needs. The ACID demands of
OLTP databases would typically require that they fail rather than risk data loss. This is generally a good idea for
use cases such as finance, but can be potentially disastrous in hardware control systems.

Consider a set of redlines that execute when a node loses connection. Upon losing communication with the rest of the
cluster,
an ACID compliant distributed database would stop serving reads and writes in order to preserve data integrity. This
could hinder a nodes' ability to shut down the system safely. In extreme scenarios, such as launch control systems, this
can result in the loss of a vehicle or even a life.

In short, Delta requires a distributed data store capable of servicing queries even when the rest of the cluster is
unreachable.

# Design

Aspens design consists of two gossip layers:

1. Layer 1 - Uses a Susceptible-Infected (SI) model to spread cluster state in a fashion
   resembling [Apache Cassandra(https://cassandra.apache.org/_/index.html)]. All nodes gossip their version of state at
   a regular interval. This is used to disseminate information about cluster membership and node health. This
   includes reporting information about failed or suspected nodes

2. Layer 2 - Uses a Susceptible-Infected-Recovered (SIR) model to propagate key-value sets and deletes in an eventually
   consistent manner. After receiving a set operation, the node will gossip the key-value pair to all other nodes until
   a certain number of redundant conversations (i.e. the node already received the update) have occurred.

## Membership and Cluster State Synchronization

Delta aims to provide dynamic cluster membership. This is more difficult to accomplish if each node is required to
know about *all* other nodes in the cluster before being initialized. This is the approach taken
by [etcd](https://etcd.io/).
By using a gossip based network, Delta can provide a cluster membership system that is dynamic and resilient to failure.

This cluster membership and state gossip is considered Layer 1. Layer 1 is implemented using a Susceptible-Infected (SI)
model. In an SI gossip model, nodes never stop spreading a message. This means quite a bit of network message
amplification
but is useful when it comes to failure detection and membership.

### Cluster State Data Structure

Layer 1 holds cluster state in a node map `map[NodeID]Node`. `NodeID` is a unique `int16` identifier for each node.
`Node` holds various pieces of identifying information about the node along with its current state.

```go
package irrelivant

// NodeID is a unique identifier for a node. 
type NodeID int16

type Node struct {
	ID NodeID
	// Address is a reachable address for the node.
	Address address.Address
	// Version is software version of the node.
	Version version.MajorMinor
	// Heartbeat is the gossip heartbeat of the node. See [Heartbeat] for more.
	Heartbeat Heartbeat
	// Various additions such as failure detection state, etc. will go here.
}

type NodeMap map[NodeID]Node
```

A node's Heartbeat tracks two values:

```go
package irrelivant

type Heartbeat struct {
	// Version is incremented every time the node gossips information about its state. This is used to merge differing
	// versions of cluster state during gossip.
	Version uint32
	// Generation is incremented every time the node is restarted. This is useful for bringing a node
	// back up to speed after a long period of absence.
	Generation uint16
}
```

### Anatomy of a Conversation

#### Sync Message

A node initiates conversation with another node by sending a 'sync' message to another node. This message contains
a list of node digests.

```go
package irrelivant

type Digest struct {
	// NodeID is the node's unique identifier
	ID NodeID
	// Heartbeat is the gossip heartbeat. 
	Heartbeat Heartbeat
}

type SyncMessage struct {
	Digests []Digest
}
```

A digest is added to the message for every node in the initiator's state.

#### Ack Message

After receiving a sync message from the initiator node, the peer node will respond with an ack message:

```go
package irrelivant

type AckMessage struct {
	// A list of digests for nodes in the peer's state that:
	//    1. The peer node has not seen.
	//    2. Have a younger heartbeat than in the sender's Digest.
	Digests []Digest
	// A NodeMap of nodes in the peer's state that:
	//
	//    1. The initiating node has not seen.
	//    2. Have an older heartbeat than in the sender's Digest.
	NodeMap NodeMap
}

```

The peer node makes no updates during this period.

#### Ack2 Message

After receiving an ack message from the peer, the initiator updates its own state and responds with a final ack2
message.
The initiator compares the heartbeat of every node in the `AckMessage.NodeMap` with its on state. If the peer sent a
new node or a node with an older heartbeat, the initiator's will replace the node in its state with the node from the
peer.
It will them compose a new message:

```go
package irrelivant

type Ack2Message struct {
	// A NodeMap of nodes in the initiator's state that:
	//  1. Are in the peer's ack digests.
	NodeMap NodeMap
}
```

## Key-Value Store

### Recovery Constant

## Failure Detection

### Layer 1 Piggyback

## Node ID Assignment and Distributed Counting

## Cluster Topology and Routing





