
---
title: "How to run the mock snmp device"
menus:
  main:
    parent: howto
    weight: 20
---

## The SNMP mock

The snmp mock is an UDP server mocking a device using SNMP with an mock MIB file (`./dev/TNO-PDU-MIB.txt`) and providing the following metrics `tnoPduEnergyValue` and `tnoPduPowerValue`.

## Running locally

### Requirements

- [net-snmp](https://www.net-snmp.org/)

### Running the server

Running the following command will start the mock snmp server locally.

```go
go run ./dev/mock_snmp.go
```

This will start a upd server on port `1161`.

### Running the latest version with docker

Running the following command will start the mock snmp server within docker and expose the port

```bash
docker run -p 1161:1161/udp ci.tno.nl/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/chantico-snmp-mock:latest
```

### Querying the server

Get the values of the server

```bash
# To get tnoPduEnergyValue
snmpget -v2c -c public -M +./dev -m +TNO-PDU-MIB localhost:1161 tnoPduEnergyValue
# Output: TNO-PDU-MIB::tnoPduPowerValue = INTEGER: 825

# To get tnoPduPowerValue
snmpget -v2c -c public -M +./dev -m +TNO-PDU-MIB localhost:1161 tnoPduPowerValue
# Output: TNO-PDU-MIB::tnoPduPowerValue = INTEGER: 68
```

## Querying the chantico-snmp-mock running in the development setup

If the development kind cluster is running the `chantico-snmp-mock` service is a Node Port that visible on port `31161`.
It can be queried as follow:

```bash
snmpget -v2c -c public -M +./dev -m +TNO-PDU-MIB localhost:31161 tnoPduEnergyValue
```

## Adding the snmp-mock in chantico (in custruction)

- Port forward `chantico-filebrowser`
```bash
    kubectl port-forward -n chantico svc/chantico-filebrowser 18888:80
```
- Login in the web UI, `localhost:18888` and Upload `./dev/TNO-PDU-MIB.txt` to `./snmp/mibs/TNO-PDU-MIB.txt`


(The following steps are temporary until the MeasurementDevice operator is rolled out)

- In the web UI, Upload the following yaml file at the following location `./snmp/yml/snmp.yml`
```yaml
auths:
  default:
    community: public
    version: 2
modules:
  init:
    get:
    - 1.3.6.1.4.1.99999.1.0
    metrics:
    - name: tnoPduEnergyValue
      oid: 1.3.6.1.4.1.99999.1
      type: gauge
      help: A random energy value (in J) - 1.3.6.1.4.1.99999.1
```
- Restart `chantico-snmp` service:
```bash
kubectl rollout restart -n chantico deployment/chantico-snmp
```
- Port forward the snmp_exporter
```bash
    kubectl port-forward -n chantico svc/chantico-snmp 9116:9116
```
- Curl the module (optionally this can be done via the web UI)
```bash
curl -X GET 'http://localhost:9116/snmp?target='$(kubectl get -n chantico svc/chantico-snmp-mock -o jsonpath='{.spec.clusterIP}')':1161&auth=default&module=init'
```
