#!/bin/sh

kubectl apply -f ./config/dev/volume-mounter.yaml || exit 1

sleep 3s

kubectl port-forward -n chantico svc/sshfs-service 8022:22 &2> /dev/null &1> /dev/null &

sleep 2s

mkdir -p /tmp/chantico-volume-mount/
sshfs root@localhost:/data /tmp/chantico-volume-mount/ -p 8022

