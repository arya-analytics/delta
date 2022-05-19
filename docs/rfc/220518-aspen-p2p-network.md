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
**Cluster** - A group of nodes that can communicate with each other. \
**Redline** - A threshold for a particular channel value. Most often represents a safety limit that triggers an
action. \
**Initiator** - A node that initiates gossip with another node. \
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
   resembling [Apache Cassandra](https://cassandra.apache.org/_/index.html). All nodes gossip their version of state at
   a regular interval. This is used to disseminate information about cluster membership and node health. This
   includes reporting information about failed or suspected nodes

2. Layer 2 - Uses a Susceptible-Infected-Recovered (SIR) model to propagate key-value sets and deletes in an eventually
   consistent manner. After receiving a set operation, the node will gossip the key-value pair to all other nodes until
   a certain number of redundant conversations (i.e. the node already received the update) have occurred.

## Cluster State Synchronization

Delta aims to provide dynamic cluster membership. This is more difficult to accomplish if each node is required to
know about *all* other nodes in the cluster before being initialized. This is the approach taken
by [etcd](https://etcd.io/).
By using a gossip based network, Delta can provide a cluster membership system that is dynamic and resilient to failure.

This cluster membership and state gossip is considered Layer 1. Layer 1 is implemented using a Susceptible-Infected (SI)
model. In an SI gossip model, nodes never stop spreading a message. This means quite a bit of network message
amplification but is useful when it comes to failure detection and membership.

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
	// Version is incremented every time the node gossips information about its state. This is 
	//used to merge differing versions of cluster state during gossip.
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
message. The initiator compares the heartbeat of every node in the `AckMessage.NodeMap` with its on state. If the peer
sent a new node or a node with an older heartbeat, the initiator's will replace the node in its state with the node
from the peer. It will them compose a new message:

```go
package irrelivant

type Ack2Message struct {
	// A NodeMap of nodes in the initiator's state that:
	//  1. Are in the peer's ack digests.
	NodeMap NodeMap
}
```

#### Closing the Conversation

After receiving the final ack2 message, the initiator will update its own state using the same policy as the peer
in the section. It will then close the conversation.

### Propagation Rate

The propagation rate of cluster state is tuned by the interval at which a node gossips. Higher propagation rates
will result in heavier network traffic, so it's up to the application to determine the appropriate balance.

## Adding a Member

Aspen employs a relatively complex process for joining a node to a cluster. This is due to a desire to identify nodes
using a unique `int16` value. The ID of a node is propagated with almost every message. By using an `int16` vs. `UUID`,
we can reduce overall network traffic by a significant amount. Node IDs are also used far and wide across the rest of
Delta, such as in the key for a channel `<NodeID><ChannelID>`. This results in a sample that is 40 percent smaller than
with a `UUID`.

The downside of using `int16` id's for nodes is that we need to design a distributed counter. Fortunately, this is a
solved problem. The join process is as follows:

### Step 1 - Request a Peer to Join

When joining a new node to a cluster, the joining node (known as the **pledge**) receives a set of one or more peer
addresses
of other nodes in the cluster. The **pledge** node will choose a peer at random and send a join request to it. If the
peer
acknowledges the request, the joining node will then wait for a second message assigning it an ID. If the peer rejects
the
request or doesn't respond, it attempts to send the request to another peer. This cycle continues until a peer
acknowledges
the request or a preset threshold is reached.

The peer that accepts the **pledge** join request is known as the **responsible**. This node is responsible for safely
initiating
the **pledge**.

### Step 2 - Propose an ID

The **responsible** node will begin the initiation process by finding the highest id of the nodes within its state.
It will then select a quorum (>50%) of its peers and send a proposed id with a value one higher. It will then wait
for all peers to approve the proposed value (these peers are called **jurors**). A juror will approve the value if it
does not have a node in its state with the given ID. A **juror** tracks the results of all accepted proposals until the
state of the accepted **pledge** has been disseminated. The approval process is serialized by a mutex.

If any node rejects the id, the **responsible** node will reissue the proposal with an incremented value.
This process continues until an ID is accepted. If the **responsible** node tries to contact an unresponsive peer, it
will
reselect a quorum of peers and try again. Once an ID is selected, the **responsible** node will send it to the **
pledge**.

### Step 3 - Disseminate New Node

Once the **pledge** receives an ID assignment from the **responsible** node, it will begin to gossip its state to the
rest of the cluster. As information about the new node spreads, **jurors** will remove processed approvals from
their state.

### The First Node

The first node to join the cluster is provided with no peer addresses. It will automatically assign itself an ID of 1.

### Implications of Algorithm

Using a quorum based approach to ID assignment means that we get a strong guarantee that a node will be assigned a
unique identifier. It also means that a cluster with less than half of its nodes available will not be able to add new members.
This is an important property to consider in scenarios with extremely dynamic cluster membership.

## Key-Value Store

Aspen implements a leased driven key-value store on top of layer 1. The gossip protocol that disseminates kv updates
and tombstones is known as layer 2.

### Vocabulary

**Host** - The node that is responsible for serving the kv operation to the caller
(i.e. the node where `Get` or `Set` is called). \
**Leaseholder** - The only node that can accept writes for a particular key. \

### Interface

At the simplest level, the key-value store implements the following interface.

```go
package irrelivant

type NodeID int16

type KV interface {
	// Set sets a key to a value. nodeID specifies the node that holds the lease on the key. 
	// If nodeID is 0, the lease is assigned to the host node.
	Set(key []byte, leaseholder NodeID, value []byte) error
	// Rest of interface is the same as github.com/arya-analytics/x/kv.KV.
}
```

## Life of a Set/Delete

A kv set is processed by the database as follows. It's important to note that deletes and sets are both propagated
using the same steps.

### Step 1 - Forward Request to Leaseholder

If the node ID is non-zero, perform a layer 1 lookup for the leaseholder's address. Forward the request to the
leaseholder.
If the node ID is zero, allocate the least to the host node.

### Step 2 - Process the Forwarded Set

Add the key-value pair to an update propagation list. This list has the following structure:

```go
package irrelivant

type UpdateState byte

const (
	// StateInfected means the node is actively gossiping the update to other nodes in the cluster.
	StateInfected UpdateState = iota
	// StateRecovered means the node is no longer gossiping the update. 
	StateRecovered
)

type Operation byte

const (
	// OperationSet represents a kv set operation.
	OperationSet Operation = iota
	// OperationDelete represents a kv delete operation.
	OperationDelete
)

type Update struct {
	// Key is the key for the key-value pair.
	Key []byte
	// Value is the value for the key-value pair.
	Value []byte
	// Leaseholder is the ID of the leaseholder node.
	Leaseholder NodeID
	// State is the SIR state of the update.
	State UpdateState
	// Version is incremented every time an existing key is updated.
	Version int32
}

type UpdatePropagationList map[interface{}]Update
```

After adding the update to the propagation list, we persist the set to an underlying kv store, and send a durability
acknowledgement to the host node.

### Step 3 - Propagate the Update

A node will initiate layer 2 gossip at a set interval (default is 1 second). The gossip process is as follows:

#### Step A - Initiator Propagates Update (Sync)

The initiating node selects a random peer from layer 1, and set

1. Select a random peer from layer 1, and send a sync message:

```go
package irrelivant

type SyncMessage struct {
	// Updates contains a list of all updates in the nodes current state where:
	// 1. Update.State == StateInfected
	Updates UpdatePropagationList
}
```

#### Step B - Peer Processes Update and Response (Ack)

After receiving a sync message, the peer node processes the updates by merging its own state based on the version of
each message. The node also persists the updates to state. The peer node then sends the following ack message back to
the initiator:

```go
package irrelivant

// Feedback is a struct representing an update that has already been processed by a node.
type Feedback struct {
	Key     []byte
	Version int32
}

type AckMessage struct {
	// Updates contains a list of all updates in the nodes current state that:
	//   1. Update.State == StateInfected
	//   2. Are not already in the peer node's update list. 
	//   3. Have a higher version than the peer node's update.
	Updates UpdatePropagationList
	// Feedback is a list of Feedback for the updates a node already has (versions must be identical). 
	Feedback []Feedback
}
```

#### Step C - Initiator Processes Update

After receiving an ack message, the initiator node processes the updates in the same manner as step B. Then it processes
each feedback entry in the following manner:

1. Sets the state of the update with the matching key to StateRemoved based on a recovery probability `R`
   and persists the change to KV.

End of gossip.

### Life of a Get

Aspen does not support remote get requests. If a key cannot be found in underlying KV, returns a ErrNotFound error. This
decision was made for two reasons:

1. We maintain a consistent view of storage even when other cluster members cannot be reached.
2. We can simply extend the kv interface of an existing store, providing functionality such as prefix iteration.

This means that the only difference between a read to a local KV store is that we check for deletion tombstones before
returning a value.

Providing consistent remote reads is an undertaking for future iterations.

### Merging Updates

### Garbage Collecting Tombstones

### Recovery Constant

### Range Replication and Lease Transfer

## Failure Detection

## Failure Recovery

## Cluster Topology and Routing





