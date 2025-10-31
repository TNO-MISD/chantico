---
title: "How to install Chantico"
---

## Installation

> Note this installation does not explain how to deploy the controller and custom resources yet

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
