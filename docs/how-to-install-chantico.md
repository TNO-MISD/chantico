## Installation

To install chantico:

1. Create a volume with at least 3Gi of storage. In our current set-up:
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

1. Install snmp and prometheus via `helm`:

```bash
helm install config/initial-deployments/ --set persistentVolumeClaimName=<PVC-NAME>
```
