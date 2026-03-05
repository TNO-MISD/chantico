
---
title: "How to run the mock snmp device"
menus:
  main:
    parent: howto
    weight: 20
---

## The SNMP mock

The snmp mock is an UDP server mocking a device using SNMP with an mock MIB file (`./dev/TNO-PDU-MIB.txt`) and providing the following metrics `tnoPduEnergyValue` and `tnoPduPowerValue`. This file details how to set up the mock device, and how to subsequently run a demo with it including both the PhysicalMeasurement and MeasurementDevice custom resources.

### requirements

Ensure you have followed the instructions in [How to set up the local development environment](how-to-setup-the-local-development-environment.md) to set up a local development environment. After this:

- The development cluster is running.
- The controller is running (locally via `make run` or in-cluster).
- Port-forwarding is active (`./dev/port-forward.sh`).


## Build and load the snmp-mock image into the kind environment

```bash
export CI_REGISTRY="ci.tno.nl/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico"
export SNMP_MOCK_TAG="${SNMP_MOCK_TAG:-latest}"
export SNMP_MOCK_IMAGE="$CI_REGISTRY/chantico-snmp-mock:$SNMP_MOCK_TAG"
docker pull "$SNMP_MOCK_IMAGE"
docker tag "$SNMP_MOCK_IMAGE" chantico-snmp-mock:latest
kind load docker-image chantico-snmp-mock:latest --name kind
```

## Apply the mock to Kubernetes

```bash
kubectl apply -f dev/k8s/snmp-mock-deployment.yaml
kubectl apply -f dev/k8s/snmp-mock-service.yaml

```

kubectl apply -f config/samples/chantico_v1alpha1_physicalmeasurement_mock.yaml

## Querying the chantico-snmp-mock running in the development setup

If the development kind cluster is running the `chantico-snmp-mock` service, there is a Node Port that is visible on port `31161`.

It can be queried as follow:

```bash
snmpget -v2c -c public -M +./dev -m +TNO-PDU-MIB localhost:31161 tnoPduEnergyValue
```

## Chantico workflow with the snmp-mock as snmp device (full demo)

For an overview of the workflow that is run in the background in this demo please see the below image.

![](puml/PhysicalMeasurement-and-MeasurementDevice-sequence.png)

This section demonstrates a full flow: MIB upload → `MeasurementDevice` → `PhysicalMeasurement` → Prometheus targets.

1. Login in the web UI of `chantico-filebrowser`, `localhost:18888` and Upload `./dev/TNO-PDU-MIB.txt` to `./snmp/mibs/TNO-PDU-MIB.txt`

1. Create a `MeasurementDevice` for the mock MIB:

```bash
kubectl apply -f ./config/samples/chantico_v1alpha1_measurementdevice_mock.yaml
```

1. Wait for the SNMP generator job:
```bash
kubectl get jobs -n chantico | grep update-snmp
```

1. Create a `PhysicalMeasurement` pointing at the mock target:
```bash
kubectl apply -f ./config/samples/chantico_v1alpha1_physicalmeasurement_mock.yaml
```

1. Port-forward Prometheus and verify targets:
```bash
kubectl port-forward -n chantico deployment/chantico-prometheus 9090:9090
```
Open http://localhost:9090/targets and verify the target is `UP`.
