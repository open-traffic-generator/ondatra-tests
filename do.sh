#!/bin/sh

GO_VERSION=1.17.3
PROTOC_VERSION=3.17.3

KNE_COMMIT=2d0821b
MESHNET_COMMIT=4bf3db7

OPERATOR_RELEASE=0.0.70


set -e

if [ "$(id -u)" -eq 0 ] && [ -n "$SUDO_USER" ]
then
    echo "This script should not be run as sudo"
    exit 1
fi

# get installers based on host architecture
if [ "$(arch)" = "aarch64" ] || [ "$(arch)" = "arm64" ]
then
    echo "Host architecture is ARM64"
    GO_TARGZ=go${GO_VERSION}.linux-arm64.tar.gz
    PROTOC_ZIP=protoc-${PROTOC_VERSION}-linux-aarch_64.zip
elif [ "$(arch)" = "x86_64" ]
then
    echo "Host architecture is x86_64"
    GO_TARGZ=go${GO_VERSION}.linux-amd64.tar.gz
    PROTOC_ZIP=protoc-${PROTOC_VERSION}-linux-x86_64.zip
else
    echo "Host architecture $(arch) is not supported"
    exit 1
fi

# Avoid warnings for non-interactive apt-get install
export DEBIAN_FRONTEND=noninteractive

get_cluster_deps() {
    sudo apt-get update
    sudo apt-get install -y --no-install-recommends curl git vim apt-transport-https ca-certificates gnupg lsb-release
}

get_test_deps() {
    sudo apt-get update
    sudo apt-get install -y --no-install-recommends curl git vim wget unzip sudo
}

get_go() {
    go version 2> /dev/null && return
    echo "Installing Go ..."
    # install golang per https://golang.org/doc/install#tarball
    curl -kL https://dl.google.com/go/${GO_TARGZ} | sudo tar -C /usr/local/ -xzf -
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> $HOME/.profile
    # source path for current session
    . $HOME/.profile

    go version
}

get_go_test_deps() {
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
    go install golang.org/x/tools/cmd/goimports@v0.1.7
}

get_protoc() {
    curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
	unzip ${PROTOC_ZIP} -d /usr/local
	rm -f ${PROTOC_ZIP}
	protoc --version
}

get_docker() {
    sudo docker version 2> /dev/null && return
    echo "Installing docker ..."
    sudo apt-get remove docker docker-engine docker.io containerd runc 2> /dev/null || true

    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
        | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
        | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io

    echo "Adding $USER to group docker"
    # use docker without sudo
    sudo groupadd docker || true
    sudo usermod -aG docker $USER

    sudo docker version
}

get_kind() {
    go install sigs.k8s.io/kind@v0.11.1
}

get_kubectl() {
    echo "Copying kubectl from kind cluster to host ..."
    docker cp kind-control-plane:/usr/bin/kubectl $HOME/go/bin/
}

get_kne() {
    echo "Getting kne commit: $KNE_COMMIT ..."
    rm -rf kne
    git clone https://github.com/google/kne
    cd kne && git checkout $KNE_COMMIT && cd -
    cd kne/kne_cli && go install && cd -
    rm -rf kne
}

wait_for_all_pods_to_be_ready() {
    for n in $(kubectl get namespaces -o 'jsonpath={.items[*].metadata.name}')
    do
        for p in $(kubectl get pods -n ${n} -o 'jsonpath={.items[*].metadata.name}')
        do
            cecho "Waiting for pod/${p} in namespace ${n} (timeout=300s)..."
            kubectl wait -n ${n} pod/${p} --for condition=ready --timeout=300s
        done
    done
}

get_meshnet() {
    echo "Getting meshnet-cni commit: $MESHNET_COMMIT ..."
    rm -rf meshnet-cni && git clone https://github.com/networkop/meshnet-cni
    cd meshnet-cni && git checkout $MESHNET_COMMIT
    kubectl apply -k manifests/base
    wait_for_all_pods_to_be_ready

    cd -
    rm -rf meshnet-cni
}

get_ixia_c_operator() {
    echo "Getting ixia-c-operator ${OPERATOR_RELEASE} ..."
    kubectl apply -f https://github.com/open-traffic-generator/ixia-c-operator/releases/download/v${OPERATOR_RELEASE}/ixiatg-operator.yaml
    wait_for_all_pods_to_be_ready
}

get_metallb() {
    echo "Getting metallb ..."
    kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/namespace.yaml
    kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)" 
    kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/metallb.yaml
    
    wait_for_all_pods_to_be_ready

    prefix=$(docker network inspect -f '{{.IPAM.Config}}' kind | grep -Eo "[0-9]+\.[0-9]+\.[0-9]+" | tail -n 1)
    echo "Exposing servics on ${prefix}.100 - ${prefix}.250"
}

setup() {
    install_deps && get_protoc && get_go
}

setup_kind_cluster() {
    kind create cluster --wait 5m
    get_kubectl
    get_meshnet
    get_metallb
    get_ixia_c_operator
}

setup_cluster() {
    get_cluster_deps
    get_docker
    get_go
    get_kind
}

setup_test_client() {

}

setup_testbed() {
    setup_cluster
    setup_test_client
}

case $1 in
    *   )
    $1 || echo "usage: $0 [name of any function in script]"
    ;;
esac
