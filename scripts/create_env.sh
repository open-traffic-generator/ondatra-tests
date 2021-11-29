#!/bin/bash

# one time step to authenticate and pull resources locally
# the .keys/<service key> needs to be downloaded from the gcloud iam&admin service accounts
# sudo snap install google-cloud-sdk --classic
# cat ~/.keys/kt-nts-athena-dev-b52945713b45.json | docker login -u _json_key --password-stdin https://gcr.io
# docker pull gcr.io/kt-nts-athena-dev/athena/ceosimage:4.26.0F

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

# reference: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry
# -n is the namespace
# docker-registry is the type of secret
# ixia-pull-secret is the name of the secret
kubectl create secret -n ixiatg-op-system docker-registry ixia-pull-secret \
        --docker-server=us-central1-docker.pkg.dev \
        --docker-username=_json_key \
        --docker-password="$(cat ~/.keys/kt-nts-athena-dev-b52945713b45.json)" \
        --docker-email=himanshu.ashwini@keysight.com 
kubectl annotate secret ixia-pull-secret -n ixiatg-op-system secretsync.ixiatg.com/replicate='true'

