#!/bin/bash

# create the cluster
kind create cluster --config  ../resources/kind_cluster.yaml

# add the network load balancer
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/namespace.yaml
kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)"
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/metallb.yaml
kubectl apply -f resources/metallb-configmap.yaml

# load the arista docker image into the kind-control-plane docker instance
# confirm loaded images using: 
# docker exec -it kind-control-plane bash
# crictl images
kind load docker-image gcr.io/kt-nts-athena-dev/athena/ceosimage:4.26.0F --name=kind

# initialize meshnet
kubectl apply -k ../meshnet-cni/manifests/base

echo "############### INITIALIZING MESHNET ###############"
./meshnet_init.sh
echo "############### DONE INITIALIZING MESHNET ###############"
sleep $SLEEP_TIME

echo "############### DEPLOY IXIA OPERATOR  ###############"
#./operator_init.sh
./operator_init_yaml.sh
echo "############### DONE DEPLOY IXIA OPERATOR  ###############"
sleep $SLEEP_TIME

echo "############### DEPLOY IXIA SECRET  ###############"
./secret_init.sh
echo "############### DONE DEPLOY IXIA SECRET  ###############"
sleep $SLEEP_TIME

echo "############### DEPLOY KNE TOPOLOGY ###############"
./kne_init.sh
echo "############### DONE KNE TOPOLOGY ###############"
sleep $SLEEP_TIME

echo "############### DISPLAY TOPOLOGY ###############"
kubectl get all -A
kubectl get Topology -A
kubectl get all -n ixia
echo "############### DONE DISPLAY TOPOLOGY ###############"