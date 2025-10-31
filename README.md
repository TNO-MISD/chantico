# Chantico - energy controller

![](assets/logo/chantico.png)

## Description
In Aztec religion, [Chantico](https://en.wikipedia.org/wiki/Chantico) is the deity who reigns over the fires of hearths and fire stoves. If you would subsitute hearths with datacenter bare metals, Chantico would have similar ruling power in our context. Chantico is a [K8s SDK operator](https://sdk.operatorframework.io) project handling the monitoring of power usage of SNMP devices.

## Getting Started

How-to guides can be found in the `/docs` folder.

### Prerequisites
- go version v1.23.0+
- docker version 17.03+.

If not using local development using kind:
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

## Local development (Setup using Kind)

### Installation & usage

Kind is used for testing with a local K8s cluster. Kind requires go and docker.
Podman and nerdctl are alternatives for kind but we use the docker backend.

Install Kind and setup Chantico cluster by using `./dev/setup.sh` (you need `sudo` rights).
This mocks a SNMP device and exposes this on port `:1000`.
Verify the pods run correctly after setting up the cluster using the script.
The mocking is done in `mock_snmp.go` and is a simple TCP server with a fake SNMP signal.

Also set in an `.envrc`:

```bash
export CHANTICOVOLUMELOCATIONENV="/tmp/chantico-local-path-data"
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

