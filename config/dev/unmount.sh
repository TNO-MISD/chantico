#!/bin/sh

sudo umount -f /tmp/chantico-volume-mount/
kill -9 "$(ps -aux | grep 8022 | grep sshfs | awk '{print $2}')"
kubectl delete -f ./config/dev/volume-mounter.yaml
