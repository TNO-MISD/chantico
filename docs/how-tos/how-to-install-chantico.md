---
title: "How to install Chantico"
menus:
  main:
    parent: howto
    weight: 20
---

## Installation

### Getting the Chantico image

#### Option A: Build and host image yourself

```bash
make docker-build IMG=<your-registry>/chantico:<tag>
```

Then host this image on a container registry of choice and make sure to synchronize credentials as listed in option (B).

#### Option B: Pull from Chantico GitLab repository

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

Install the CRDs within `config/deployment/crd` to the cluster:
```
make install
```

2. Deploy Chantico and dependencies with Helm

> This project makes use of Helm templating. If desired default parameters can be changed in `config/deployment/values.yaml`. 

```bash
# Or with Chantico image hosted somewhere else:
helm install chantico config/deployment/ \
  --set controller.image=ci.tno.nl/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/chantico:latest \
  -n chantico --create-namespace
```

### Getting started with the deployed Chantico

After Chantico is succesfully deployed on your cluster, you can start making use of it for measuring your datacenter hardware of interest. Currently this can only be done with manual configuration, until a more automated approach has been implemented. Chantico inherently configures SNMP walks for endpoints by means of `MIB` and `.yaml` files. The steps of configurating this typically follows the following how-to guides:

1. [How to register an SNMP device type](how-to-register-an-snmp-device-type.md) - Upload the MIB files to use and make `.yaml` files for measurement devices. Also see the example at `config/samples/chantico_v1alpha1_measurementdevice.yaml`.
1. [How to register a physical snmp device](how-to-register-a-physical-snmp-device.md) - Define IP address(es) of interest in physical measurement `.yaml` file. Example at `config/samples/chantico_v1alpha1_physicalmeasurement.yaml`.
1. With the MIB files, measurement devices and physical measurements in place, the targets should be accessable and scrapable in Prometheus. Perform port forwarding on the Prometheus deployment to validate the result of this setup. If done successful one should see a timeseries of the requested value(s).
1. [How to register data center resources](how-to-register-data-center-resources.md) When desired, encapsulate data center structure usin data center resources.

### Remove deployed Chantico on K8s cluster

1. Remove custom resources and uninstall the CRDs:

Start by uninstalling CRDs, this will delete all custom resource definitions (PhysicalMeasurements, MeasurementDevices, DataCenterResources) and remove any instances from the cluster.
```bash
make uninstall
```

2. Uninstall the Helm release:

Upon completion of the uninstallment of the CRDs, Chantico can then be removed from the cluster using helm uninstall.
```bash
helm uninstall chantico -n chantico
```

This removes all namespaced resources (Deployments, Services, ServiceAccount, Roles, RoleBindings, PVC) as well as the cluster-scoped resources (ClusterRole, ClusterRoleBinding) managed by the release.


3. Optionally, delete the namespace:

```bash
kubectl delete namespace chantico
```
