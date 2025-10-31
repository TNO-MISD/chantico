#!/bin/bash -ex

SCRIPT_DIR=$(dirname -- "$( readlink -f -- "$0"; )")

# get kind
go install sigs.k8s.io/kind@v0.30.0

# If go is not yet added to $PATH:
#echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc

kind create cluster

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
kubectl create namespace monitoring --context kind-kind
helm install prometheus prometheus-community/prometheus --namespace monitoring
kubectl get pods -n monitoring --context kind-kind
# Create Postgres volume

kubectl create namespace chantico

# https://github.com/rancher/local-path-provisioner
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.32/deploy/local-path-storage.yaml
kubectl apply -f dev/k8s/pvc.yaml
kubectl create -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/examples/pod/pod.yaml


docker run -d -p 5000:5000 --restart always --name registry registry:2
sudo sh -c 'echo "{\"insecure-registries\": [\"localhost:5000\"]}" > /etc/docker/daemon.json'

# Make chantico docker image
pushd "$SCRIPT_DIR/../"
make docker-build docker-push IMG=localhost:5000/chantico:v0.1.0
make install
make deploy IMG=localhost:5000/chantico:v0.1.0
# docker tag localhost:5000/chantico:v0.1.0 chantico:v0.1.0
popd

# Make snmp-mock docker image
pushd "$SCRIPT_DIR"
docker build -t localhost:5000/snmp-mock:latest .
docker push localhost:5000/snmp-mock:latest
docker tag localhost:5000/snmp-mock:latest snmp-mock:latest

# Load into kind cluster
kind load docker-image snmp-mock:latest --name kind
kind load docker-image localhost:5000/chantico:v0.1.0 --name kind

# Apply to k8s
kubectl apply -f ../config/samples/chantico_v1alpha1_physicalmeasurement_mock.yaml
kubectl apply -f k8s/snmp-mock-deployment.yaml
kubectl apply -f k8s/snmp-mock-service.yaml
popd
