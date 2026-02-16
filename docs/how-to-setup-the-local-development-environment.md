---
title: "How to set-up the local development environment"
menus:
  main:
    parent: howto
    weight: 30
---

### Prerequisites

The development currently supports [WSL2](https://github.com/microsoft/WSL) and UNIX based environment.

It requires the following packages:

- go version v1.24.0+
- kind version v0.30.0+
- docker version v17.03+
- psql version v17.5+
- helm version 3.19+
- make version 4.3+
- kubectl version v0.30.0+

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
  export CHANTICO_PROMETHEUS_SERVICE_HOST="localhost"
  export CHANTICO_PROMETHEUS_SERVICE_PORT="19090"
  export CHANTICO_POSTGRES_DBSTRING="postgresql://chanticoUser:toulouse@localhost:15432/chantico"
  export CHANTICOVOLUMELOCATIONENV="$(kubectl get pv -o jsonpath='{range .items[?(@.spec.claimRef.name=="chantico-snmp-prometheus-volume-claim")]}{.spec.hostPath.path}{"\n"}{end}' | sed 's|/opt/local-path-provisioner|/tmp/chantico-local-path-data|')"
  export CHANTICOVOLUMECLAIMENV="chantico-snmp-prometheus-volume-claim"
  ```

  It might take a little while for the volume to show up, so redo the final 
  export or change the directory back and forth to reapply the direnv.

#### Checks

- Check that postgres is correctly set-up:

  ```bash
  psql "${CHANTICO_POSTGRES_DBSTRING}" -c '\d'
  ```

### Creating new resources

- In order to create new resources, you need to install `kubebuilder` as indicated in the [quick start](https://book.kubebuilder.io/quick-start):

  ```bash
  curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
  chmod +x kubebuilder && sudo mv kubebuilder /usr/local/bin/
  ```

- Then generate the resource scaffolding using:

  ```bash
  kubebuilder create api --group chantico --version v1alpha1 --kind <RESOURCE_TYPE>
  ```

- Remove the generated integration end-to-end tests with kubernetes client:

  ```bash
  rm internal/controller/suite_test.go internal/controller/<resource_type>_controller_test.go
  ```

- Make changes to the resource fields, then update the manifests and other generated files:

  ```bash
  make build
  ```

### Teardown

To teardown a local installation of the kind cluster, run the script:

```bash
./dev/teardown.sh
```
