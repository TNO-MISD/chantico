---
title: "Use-cases"
menus:
  main:
    weight: 10
---

## First use-case

### Use case

![](puml/use-case-1.png)

The first considered use case of chantico is a server plugged on two PDU outlets from two different PDUs with a baremetal offering (with IPMI interface access to the consumer).

### Sequence diagram

The sequence diagram are the interactions between chantico, the workflow orchestrator and the engineer.
![](puml/workflows-edc-interactions.png)

### Architecture

The components, features and outputs of chantico related to the first use case and later uses cases as context are demonstrated in the following flow diagram.
![](puml/high-level-architecture.png)
