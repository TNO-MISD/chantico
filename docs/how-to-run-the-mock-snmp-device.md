
---
title: "How to run the mock snmp device"
menus:
  main:
    parent: howto
    weight: 20
---

## Running locally

### Requirements

- [net-snmp](https://www.net-snmp.org/)

### Running the server

Running the following command will start the mock snmp server locally.

```go
go run ./dev/mock_snmp.go
```

This will start a upd server on port `1161`.

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
