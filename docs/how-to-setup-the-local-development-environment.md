---
title: "How to set an SNMP device type"
menu:
  main:
    weight: 30
---

### Prerequisite

- go version v1.23.0+
- kind version v0.30.0+
- docker version v17.03+
- psql v17.5+

### Installation

- Login you docker client:

```bash
docker login ci.tno.nl
```

- To install the kind docker run

```bash
./dev/setup.sh
```

- In a separate terminal setup the port forward

```bash
./dev/port-forward.sh
```

- Set up the following environment variables (this can be automated using [direnv](https://direnv.net/))

```bash
export CHANTICO_POSTGRES_SERVICE_HOST="localhost"
export CHANTICO_POSTGRES_SERVICE_PORT="15432"
export CHANTICO_POSTGRES_DBSTRING="postgresql://chanticoUser:toulouse@localhost:15432/chantico"
export CHANTICOVOLUMELOCATIONENV="/tmp/chantico-local-path-data/$(ls -Art /tmp/chantico-local-path-data | tail -n1)"
```

#### Checks

- Checks that postgres is correctly set-up

```bash
psql "${CHANTICO_POSTGRES_DBSTRING}" -c '\d'
```

### Teardown

To teardown a local installation of the kind cluster run the script

```bash
./dev/teardown.sh
```
