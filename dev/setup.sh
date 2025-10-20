# get podman
sudo apt-get -y install podman

# get kind
go install sigs.k8s.io/kind@v0.30.0

# nerdctl (not available from apt)
brew install nerdctl

# If go is not yet added to $PATH:
#echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc

# Create Kind cluster
cat <<EOF | kind create cluster --name chantico-cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
EOF

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
kubectl create namespace monitoring --context kind-chantico-cluster
helm install prometheus prometheus-community/prometheus --namespace monitoring
kubectl get pods -n monitoring --context kind-chantico-cluster
# Create Postgres volume

# https://github.com/rancher/local-path-provisioner
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.32/deploy/local-path-storage.yaml
kubectl apply -f dev/pvc.yaml
kubectl create -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/examples/pod/pod.yaml

# kubectl apply -k config/samples/