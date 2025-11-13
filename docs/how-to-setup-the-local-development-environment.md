---
title: "How to set-up the local development environment"
menus:
  main:
    parent: howto
    weight: 30
---

### Prerequisites

- go version v1.23.0+
- kind version v0.30.0+
- docker version v17.03+
- psql v17.5+

### Installation

- Login your docker client:

  ```bash
  docker login ci.tno.nl
  ```

- To install the kind docker cluster, run:

  ```bash
  ./dev/setup.sh
  ```

- In a separate terminal, setup the port forward:

  ```bash
  ./dev/port-forward.sh
  ```
  
  Redo this command whenever you end it to help developing.

- Set up the following environment variables (this can be automated using [direnv](https://direnv.net/))

  ```bash
  export CHANTICO_POSTGRES_SERVICE_HOST="localhost"
  export CHANTICO_POSTGRES_SERVICE_PORT="15432"
  export CHANTICO_POSTGRES_DBSTRING="postgresql://chanticoUser:toulouse@localhost:15432/chantico"
  export CHANTICOVOLUMELOCATIONENV="/tmp/chantico-local-path-data/$(ls -Art /tmp/chantico-local-path-data | tail -n1)"
  ```

  It might take a little while for the volume to show up, so redo the final 
  export or change the directory back and forth to reapply the direnv.

#### Checks

- Check that postgres is correctly set-up:

  ```bash
  psql "${CHANTICO_POSTGRES_DBSTRING}" -c '\d'
  ```

### Teardown

To teardown a local installation of the kind cluster, run the script:

```bash
./dev/teardown.sh
```
