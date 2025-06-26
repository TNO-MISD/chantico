# Chantico

This document presents the technical proposal of chantico.

## Naming

```text
In Aztec religion, Chantico ("she who dwells in the house") is the deity reigning over the fires
```

As the aforecited extract of the Wikipedia page of [Chantico](https://en.wikipedia.org/wiki/Chantico), Chantico is reigning.
It therefore felt natural to call the energy domain controller developped within the MISD project according to that deity.

## Installation

[Please refer to the following document](docs/how-to-install-chantico.md)

## Technical proposal

The idea behind chantico is to use the kubernetes control plane as a basis to have a fully declarative approach to the energy domain control.
To make this happen Chantico is built as a [kubernetes controller](https://kubernetes.io/docs/concepts/architecture/controller/) operating over a set [custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

### Components

#### chantico-filebrowser

The `chantico-filebrowser` is a kubernetes deployment living in the `chantico` namespace.
It allows to add configuration files via drag and drop (e.g. uploading MIB files for the registration of a PDU).

#### chantico-postgres

The `chantico-postgres` is a kubernetes service living in the `chantico` namespace.
It acts as long term storage database for chantico.

#### chantico-snmp

The `chantico-snmp` is a kubernetes deployment living in the `chantico` namespace.
It hosts an `snmp_exporter` instance that query devices using the SNMP format and exposes a prometheus compatible format.

#### chantico-prometheus

The `chantico-snmp` is a kubernetes service living in the `chantico` namespace.
It hosts an `prometheus` that scrapes devices via `chantico-snmp`.


## Technical choices

The backbone of chantico was implemented using [operator-sdk](https://sdk.operatorframework.io/).

### Programming language

To seamlessly interoperate with kubernetes the [go](https://go.dev/) programming language was chosen.

### Repository

### Interface with postgres


#### Migrations

The SQL migrations are handled by [goose](https://pressly.github.io/goose/).

#### Go code

To avoid the [short-comings](https://en.wikipedia.org/wiki/Object%E2%80%93relational_impedance_mismatch) of ORMs an approach based on generating idiomatic directly from SQL queries have been prefered.
To do this we use the [sqlc](https://sqlc.dev/) library.
