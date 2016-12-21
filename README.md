# Blackfish: A minimalist membership and failure detector  

Blackfish is a Go implementation of the [SWIM](https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf) (Scalable Weakly-consistent Infection-style Membership) protocol developed at Cornell University by Motivala, et al. It is'nt a distributed data store in its own right, but rather a framework intended to facilitate the construction of such systems.

It was developed with a space-sensitive systems (mobile, IOT, containers) in mind, and as such as adopted a minimalist philosophy of doing a few things well. As such, its feature set it relatively small and limited to functionality around adding and removing nodes and detecting status changes on the cluster.

### Features

* Uses gossip (i.e., epidemic) protocol for dissemination, the latency of which grows logarithmically with the number of members.
* Low-bandwidth UDP-based failure detection and status dissemination.
* imposes a constant message load per group member, regardless of the number of members.
* Strongly complete: status changes are eventually detected by all non-faulty members of the cluster.
* Various knobs and levers to tweak ping and dissemination behavior, settable via the API or environment variables.

#### Coming soon!

* Support for multicast detection
* Re-try of lost nodes (with exponential backoff)
* Adaptive timeouts (defined as the 99th percentile of all recently seen responses; currently hard-coded at 150ms)

### Variations from [the SWIM paper](https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf)

* If a node has no status change updates to transmit, it will instead choose a random node from its "known nodes" list.

### How to use

### Examples