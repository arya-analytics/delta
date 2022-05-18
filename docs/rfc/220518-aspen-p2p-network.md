# Aspen - Gossip Based Peer to Peer Network

**Feature Name**: Aspen, a Gossip Based Peer to Peer Network \
**Status**: Proposed \
**Start Data**: 2020-05-18 \
**Authors**: emilbon99 \
**Jira Issue** - [DA-153 - [Aspen] - RFC](https://arya-analytics.atlassian.net/browse/DA-153)

# Table of Contents

# Summary

In this RFC I propose an architecture for a gossip based network that can meet Delta's distributed storage and cluster membership 
requirements. Gossip based dissemination is an efficient method for sharing cluster wide state in an eventually consistent
fashion. Delta requires a relatively small distributed store that should ideally be available even on loss of connection
to the rest of the cluster. A Gossip based network also lays the foundations for building strongly consistent stores
when required, as well as allowing for dynamic cluster membership and failure detection.

This proposal focuses on extreme simplicity to achieve a minimum viable implementation. It aims to provide only functionality
that contributes towards meeting the requirements laid out in the [Delta specification](https://arya-analytics.atlassian.net/wiki/spaces/AA/pages/9601025/01+-+Delta).

# Vocabulary

**Node** - A machine in the cluster.
