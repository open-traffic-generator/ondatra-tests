#!/bin/sh

GO_VERSION=1.17.3
PROTOC_VERSION=3.17.3

KNE_COMMIT=2d0821b
MESHNET_COMMIT=4bf3db7

OPERATOR_RELEASE=0.0.70
IXIA_C_RELEASE=0.0.1-2446

set -e
# source path for current session
. $HOME/.profile

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

cecho() {
    echo "\n\033[1;32m${1}\033[0m\n"
}

get_cluster_deps() {
    sudo apt-get update
    sudo apt-get install -y --no-install-recommends curl git vim apt-transport-https ca-certificates gnupg lsb-release
}

get_test_deps() {
    # these will be run inside container and hence do not use sudo
    # apt-get update
    # apt-get install -y --no-install-recommends curl git vim wget unzip sudo ca-certificates
    sudo apt-get install -y --no-install-recommends wget unzip
}

get_go() {
    go version 2> /dev/null && return
    cecho "Installing Go ..."
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
    go mod download
}

get_protoc() {
    curl -kLO https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
    rm -rf $HOME/.local
	unzip ${PROTOC_ZIP} -d $HOME/.local
	rm -f ${PROTOC_ZIP}
    echo 'export PATH=$PATH:$HOME/.local/bin' >> $HOME/.profile
    # source path for current session
    . $HOME/.profile
	protoc --version
}

get_docker() {
    sudo docker version 2> /dev/null && return
    cecho "Installing docker ..."
    sudo apt-get remove docker docker-engine docker.io containerd runc 2> /dev/null || true

    curl -kfsSL https://download.docker.com/linux/ubuntu/gpg \
        | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
        | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    # 1st hack to not detect MitM when a corporate proxy is sitting in between
    conf=/etc/apt/apt.conf.d/99docker-skip-cert-verify.conf
    echo "Acquire { https::Verify-Peer false }" | sudo tee -a "$conf"

    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io
    # undo 1st hack
    sudo rm -rf "$conf"

    cecho "Adding $USER to group docker"
    # use docker without sudo
    sudo groupadd docker || true
    sudo usermod -aG docker $USER

    # 2nd hack to skip verifying docker images while pulling
    reg=$(docker info | grep Registry | cut -d\  -f 3)
    echo "{\"insecure-registries\": [\"${reg}\"]}" | sudo tee -a /etc/docker/daemon.json
    sudo systemctl restart docker

    sudo docker version
    cecho "Please logout, login and execute previously entered command again !"
    exit 0
}

get_kind() {
    go install sigs.k8s.io/kind@v0.11.1
}

get_kubectl() {
    cecho "Copying kubectl from kind cluster to host ..."
    docker cp kind-control-plane:/usr/bin/kubectl $HOME/go/bin/
}

get_kne() {
    cecho "Getting kne commit: $KNE_COMMIT ..."
    rm -rf kne
    git clone https://github.com/google/kne
    cd kne && git checkout $KNE_COMMIT && cd -
    cd kne/kne_cli && go install && cd -
    rm -rf kne
}

gcloud_auth() {
    gcloud auth application-default login --no-launch-browser
    gcloud auth configure-docker --quiet us-central1-docker.pkg.dev
}

get_gcloud() {
    gcloud version 2>/dev/null && return
    cecho "Setting up gcloud"
    dl=google-cloud-sdk-349.0.0-linux-x86_64.tar.gz
    cd $HOME
    curl -kLO https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/${dl}
    tar xzvf $dl && rm -rf $dl
    cd -
    echo 'export PATH=$PATH:$HOME/google-cloud-sdk/bin' >> $HOME/.profile
    # source path for current session
    . $HOME/.profile

    gcloud init
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

    kubectl get pods -A
    kubectl get services -A
}

get_meshnet() {
    cecho "Getting meshnet-cni commit: $MESHNET_COMMIT ..."
    rm -rf meshnet-cni && git clone https://github.com/networkop/meshnet-cni
    cd meshnet-cni && git checkout $MESHNET_COMMIT
    kubectl apply -k manifests/base
    wait_for_all_pods_to_be_ready

    cd -
    rm -rf meshnet-cni
}

get_ixia_c_operator() {
    cecho "Getting ixia-c-operator ${OPERATOR_RELEASE} ..."
    kubectl apply -f https://github.com/open-traffic-generator/ixia-c-operator/releases/download/v${OPERATOR_RELEASE}/ixiatg-operator.yaml
    wait_for_all_pods_to_be_ready
}

get_metallb() {
    cecho "Getting metallb ..."
    kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/namespace.yaml
    kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)" 
    kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/metallb.yaml
    
    wait_for_all_pods_to_be_ready

    prefix=$(docker network inspect -f '{{.IPAM.Config}}' kind | grep -Eo "[0-9]+\.[0-9]+\.[0-9]+" | tail -n 1)
    yml=metallb-config.yaml
    echo "apiVersion: v1" > ${yml}
    echo "kind: ConfigMap" >> ${yml}
    echo "metadata:" >> ${yml}
    echo "  namespace: metallb-system" >> ${yml}
    echo "  name: config" >> ${yml}
    echo "data:" >> ${yml}
    echo "  config: |" >> ${yml}
    echo "   address-pools:" >> ${yml}
    echo "    - name: default" >> ${yml}
    echo "      protocol: layer2" >> ${yml}
    echo "      addresses:" >> ${yml}
    echo "      - ${prefix}.100 - ${prefix}.250" >> ${yml}

    cecho "Applying metallb config map for exposing internal services via public IP addresses ..."
    cat ${yml}
    kubectl apply -f ${yml}
    rm -rf ${yml}
}

rm_kind_cluster() {
    kind delete cluster 2> /dev/null
    rm -rf $HOME/.kube
    rm -rf $HOME/go/bin/kubectl
}

setup_kind_cluster() {
    rm_kind_cluster
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
    setup_kind_cluster
}

build_ondatra() {
    cd ondatra
    go mod tidy -compat=1.17
    go generate -v ./...
    CGO_ENABLED=0 go build -v ./...
    cd -
}

setup_ondatra_tests() {
    get_test_deps
    # get_go
    get_go_test_deps
    get_protoc
    get_kne
    build_ondatra
}

rm_test_client() {
    docker stop ondatra-tests 2> /dev/null
    docker rm ondatra-tests 2> /dev/null
    docker rmi -f ondatra-tests:client 2> /dev/null
}

setup_test_client() {
    # rm_test_client
    cecho "Building test client ..."
    # docker build -t ondatra-tests:client .
    # docker run -td --network host --name ondatra-tests ondatra-tests:client
    # docker cp $HOME/.kube/config ondatra-tests:/home/ondatra-tests/resources/kneconfig/
    setup_ondatra_tests
    cp $HOME/.kube/config resources/global/kubecfg
}

# TODO: this is currently not exercised anywhere in the script
setup_gcp_secret() {
    cecho "Setting up K8S pull secret for GCP ..."
    echo -n "Enter GCP Email: "
    read email
    echo -n "Enter GCP Password: "
    stty -echo
    read password
    stty echo
    echo

    kubectl delete secret -n ixiatg-op-system ixia-pull-secret 2> /dev/null
    kubectl create secret \
        -n ixiatg-op-system docker-registry ixia-pull-secret \
        --docker-server=us-central1-docker.pkg.dev \
        --docker-username="${email}" \
        --docker-password="${password}" \
        --docker-email="${email}"
    kubectl annotate secret ixia-pull-secret -n ixiatg-op-system secretsync.ixiatg.com/replicate='true'
}

load_ceos() {
    ceos="us-central1-docker.pkg.dev/kt-nts-athena-dev/keysight/ceos:4.26.1F"
    docker pull ${ceos}
    kind load docker-image ${ceos}
}

load_images() {
    IMG=""
    TAG=""
    yml=ixia-configmap.yaml

    rm -rf ${yml}
    cecho "Loading docker images for Ixia-c release ${IXIA_C_RELEASE} ..."
    curl -kLO https://github.com/open-traffic-generator/ixia-c/releases/download/v${IXIA_C_RELEASE}/${yml}

    while read line
    do
        if [ -z "${IMG}" ]
        then
            IMG=$(echo "$line" | grep path | cut -d\" -f4)
        elif [ -z "${TAG}" ]
        then
            TAG=$(echo "$line" | grep tag | cut -d\" -f4)
        else
            PTH="$IMG:$TAG"
            IMG=""
            TAG=""

            cecho "Loading $PTH"
            docker pull $PTH
            kind load docker-image $PTH
        fi
    done <${yml}

    rm -rf ${yml}
    load_ceos
}

setup_repo() {
    get_gcloud
    gcloud_auth
    load_images
}

setup_testbed() {
    setup_cluster
    setup_repo
    setup_test_client
    cecho "Please logout and login again !"
}

newtop() {
    kne_cli -v trace --kubecfg resources/global/kubecfg create resources/topology/ixia-arista-ixia.txt
    wait_for_all_pods_to_be_ready
}

rmtop() {
    kne_cli -v trace --kubecfg resources/global/kubecfg delete resources/topology/ixia-arista-ixia.txt
}

run() {
    name=$(grep -Eo "Test[0-9a-zA-Z]+" ${2})
    prefix=$(basename ${2} | sed 's/_test.go//g')
    topo=resources/topology/ixia-arista-ixia.txt
    tb=resources/testbed/ixia-arista-ixia.txt

    mkdir -p logs
    kne_cli -v trace --kubecfg resources/global/kubecfg topology push ${topo} arista1 resources/dutconfig/${prefix}/set_dut.txt || exit 1
    cecho "Staring tests, output will be stored in logs/${prefix}.log"
    CGO_ENABLED=0 go test -v -timeout 60s -run ${name} tests/tests \
        -config ../resources/global/kneconfig.yaml \
        -testbed ../${tb} | tee logs/${prefix}.log \
    || true
    kne_cli -v trace --kubecfg resources/global/kubecfg topology push ${topo} arista1 resources/dutconfig/${prefix}/unset_dut.txt
}

case $1 in
    *   )
        $1 ${@} || cecho "usage: $0 [name of any function in script]"
    ;;
esac
