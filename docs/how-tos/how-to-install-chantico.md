---
title: "How to install Chantico"
menus:
  main:
    parent: howto
    weight: 20
---

## Installation

### Deployment of Chantico on k8s cluster

To install chantico on a Kubernetes cluster:

1. Create a volume with at least 3Gi of storage. Example for our current cluster set-up:
    ```yaml
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: <PVC-NAME>
      namespace: chantico
    spec:
      storageClassName: csi-rbd
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 3Gi
      volumeMode: Filesystem
    ```
    Change the `storageClassName` if you have a different preference on your cluster. You may also need `accessModes` `ReadWriteMany` if you have a multi-node Ceph setup, for example. Then `kubectl apply -f pvc.yaml` or wherever you save this file.

1. Create registry credentials for pulling images from a private Docker registry:

    1. Go to GitLab -> Chantico project -> Settings -> Access tokens -> Add new token, with a descriptive/easy-to-copy "Token name" and "Scopes" have at least `read_registry` checked.
    1. Copy the access token, then:
      ```bash
      kubectl create -n chantico secret docker-registry regcred --docker-server=ci.tno.nl --docker-username=<TOKEN-NAME> --docker-password=<ACCESS-TOKEN> --docker-email=<YOUR-EMAIL>
      ```

1. Install snmp and prometheus via `helm` (change `chantico` if you want a different release name and/or namespace):
  ```bash
  helm install chantico config/initial-deployments/ --set persistentVolumeClaimName=<PVC-NAME> -n chantico --create-namespace
  ```

### Deployment of Chantico controller on k8s cluster

> If you want to run the Chantico controller locally (for testing purposes), please refer to [this guide](how-to-setup-the-local-development-environment.md).

1. First, verify the current context which is used by `kubectl config current-context` and if needed change current context with ` kubectl config set-context <HESI-MISD-CONTEXT> --current`.

    After confirming the current context links to the desired cluster, install the CRDs which are used by Chantico with the following command. This installs the CRDs of physical measurements, measurement devices and datacenter resources.

    ```bash
    make install
    ```

1. Now the Chantico controller can be deployed to the `chantico` namespace. This pulls the latest Chantico image from the Gitlab image container registry. If you would like to change to another version of Chantico please alter the `$IMG` variable in the `Makefile` accordingly.

    ```bash
    make deploy
    ```

1. Verify deployment


```bash
kubectl logs deployment/controller-manager -n chantico > /tmp/chantico.log
grep -n -A5 -B5 "forbidden\|Failed to watch" /tmp/chantico.log
```