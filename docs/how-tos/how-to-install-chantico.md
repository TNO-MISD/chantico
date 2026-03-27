---
title: "How to install Chantico"
menus:
  main:
    parent: howto
    weight: 20
---

## Installation

### Getting the Chantico image

#### Option A: Build it yourself

```bash
make docker-build IMG=<your-registry>/chantico:<tag>
```

#### Option B: Pull from GitLab

> The Chantico repository on GitHub does not host images yet in the container registry there. This is still work in progress. The following steps only work if you have access to the GitLab repository of Chantico. 

The GitLab repository of Chantico hosts several relevant images, including the one of the Chantico controller itself. First, create registry credentials for pulling images from a private Docker registry:

1. Go to GitLab -> Chantico project -> Settings -> Access tokens -> Add new token, with a descriptive/easy-to-copy "Token name" and "Scopes" have at least `read_registry` checked.
1. Copy the access token, then:
  
  ```bash
  kubectl create namespace chantico
  kubectl create -n chantico secret docker-registry regcred \
    --docker-server=ci.tno.nl \
    --docker-username=<TOKEN-NAME> \
    --docker-password=<ACCESS-TOKEN> \
    --docker-email=<YOUR-EMAIL>
  ```

### Deployment of Chantico on k8s cluster

1. Install CRDs

The CRDs used by Chantico are typically already in place under `config/deployment/crd`. If you want to (re)install them there, do so with the following make command:
```
make install
```

2. Deploy Chantico and dependencies with Helm

```bash
# With selfmade Chantico image:
helm install chantico config/deployment/ \
  --set controller.image=<your-registry>/chantico:<tag> \
  -n chantico --create-namespace

# Or with Chantico image from GitLab:
# Latest image of Chantico
helm install chantico config/deployment/ \
  --set controller.image=ci.tno.nl/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/chantico:latest \
  -n chantico --create-namespace
```
