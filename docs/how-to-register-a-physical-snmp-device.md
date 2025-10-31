---
title: "How to register a physical snmp device"
---

In the current setting, a type of device using SNMP can be configured uploading the MIBS and defining a `PhysicalMeasurement` custom resource.
In our First use-case (see `goal.md`) this corresponds to the `createPDU1` and `createPDU2` phases.

1. Create the PhysicalMeasurement matching the required type of PhysicalMeasurement
  1. Create a `physical_measurement.yaml` file

  ```yaml
  apiVersion: chantico.ci.tno.nl/v1alpha1
  kind: PhysicalMeasurement
  metadata:
    labels:
      app.kubernetes.io/name: chantico
      app.kubernetes.io/managed-by: kustomize
    name: physicalmeasurement-pdu1-out
    namespace: chantico
  spec:
    ip: 10.5.1.1
    serviceId: dee263f8-50e0-11f0-8cb5-00155d8a81e1 # This can be any type of UUID
    measurementDevice:  schleifenbauer-out # This has to be a currently valid MeasurementDevice name
  ```
  1. Apply the yaml file
  ```sh
  kubectl apply -f physical_measurement.yaml
  ```
1. Verify the new device setting
  1. Wait that `chantico-prometheus` is correctly redeployed
  1. Port-forward the SNMP exporter
  ```sh
  kubectl port-forward -n chantico deployment/chantico-prometheus 9090
  ```
  1. Check that the config (http://localhost:9090/targets)

