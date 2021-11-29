#!/bin/bash

# create the cluster
kind create cluster --config  ../resources/kind_cluster.yaml
kubectl cluster-info --context kind-kind

# add the network load balancer
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/namespace.yaml
kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)"
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/metallb.yaml
kubectl apply -f ../resources/metallb_configmap.yaml

# load the arista docker image into the kind-control-plane docker instance
# to confirm loaded images use the following commands: 
# docker exec -it kind-control-plane bash
# crictl images
kind load docker-image gcr.io/kt-nts-athena-dev/athena/ceosimage:4.26.0F --name=kind

# initialize meshnet
kubectl apply -k ../meshnet-cni/manifests/base

# deploy ixia-c operator
VERSION=0.0.70
curl -kLO https://github.com/open-traffic-generator/ixia-c-operator/releases/download/v$VERSION/ixiatg-operator.yaml
docker pull ixiacom/ixia-c-operator:$VERSION
kubectl apply -f ixiatg-operator.yaml

# create ixia secret
kubectl create secret -n ixiatg-op-system docker-registry ixia-pull-secret \
        --docker-server=us-central1-docker.pkg.dev \
        --docker-username=_json_key \
        --docker-password="$(cat kt-nts-athena-dev-3ba2488cc69b.json)" \
        --docker-email=himanshu.ashwini@keysight.com 
kubectl annotate secret ixia-pull-secret -n ixiatg-op-system secretsync.ixiatg.com/replicate='true'

echo "############### DEPLOY KNE TOPOLOGY ###############"
./kne_init.sh
echo "############### DONE KNE TOPOLOGY ###############"
sleep $SLEEP_TIME

echo "############### DISPLAY TOPOLOGY ###############"
kubectl get all -A
kubectl get Topology -A
kubectl get all -n ixia
echo "############### DONE DISPLAY TOPOLOGY ###############"