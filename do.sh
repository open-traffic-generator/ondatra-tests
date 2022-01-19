#!/bin/sh

GO_VERSION=1.17.3
PROTOC_VERSION=3.17.3

KNE_COMMIT=2d0821b
MESHNET_COMMIT=4bf3db7

OPERATOR_RELEASE=0.0.70

KNEBIND_CONFIG="../resources/global/knebind-config.yaml"

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


    # hack to not detect MitM when a corporate proxy is sitting in between
    conf=/etc/apt/apt.conf.d/99docker-skip-cert-verify.conf
    curl -fsL https://download.docker.com/linux/ubuntu/gpg 2>&1 > /dev/null \
        || echo "Acquire { https::Verify-Peer false }" | sudo tee -a "$conf" \
        && sudo mkdir -p /etc/docker \
        && echo '{ "registry-mirrors": ["https://docker-remote.artifactorylbj.it.keysight.com"] }' | sudo tee -a /etc/docker/daemon.json
    
    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io
    # partially undo hack
    sudo rm -rf "$conf"
    # remove docker.list from apt-get if hack is applied (otherwise apt-get update will fail)
    curl -fsL https://download.docker.com/linux/ubuntu/gpg 2>&1 > /dev/null \
        || sudo rm -rf /etc/apt/sources.list.d/docker.list

    cecho "Adding $USER to group docker"
    # use docker without sudo
    sudo groupadd docker || true
    sudo usermod -aG docker $USER

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
        if [ "${1}" = "-ns" ] && [ "${2}" != "${n}" ]
        then
            continue
        fi
        for p in $(kubectl get pods -n ${n} -o 'jsonpath={.items[*].metadata.name}')
        do
            cecho "Waiting for pod/${p} in namespace ${n} (timeout=300s)..."
            kubectl wait -n ${n} pod/${p} --for condition=ready --timeout=300s
        done
    done

    cecho "Pods:"
    kubectl get pods -A
    cecho "Services:"
    kubectl get services -A
}

wait_for_pod_counts() {
    namespace=${1}
    count=${2}
    start=$SECONDS
    while true
    do
        echo "Waiting for all pods to be expected under namespace ${1}..."
        
        echo "Expected Pods ${2}"
        pod_count=$(kubectl get pods -n ${1} --no-headers 2> /dev/null | wc -l)
        echo "Actual Pods ${pod_count}"
        # if expected pod count is 0, then check that actual count is 0 as well
        if [ "${2}" = 0 ] && [ "${pod_count}" = 0 ]
        then
            break
        else if [ "${2}" -gt 0 ]
        then
            # if expected pod count is more than 0, then ensure actual count is more than 0 as well
            break
        fi
        fi

        elapsed=$(( SECONDS - start ))
        if [ $elapsed -gt 300 ]
        then
            echo "All pods are not as expected under namespace ${1} with 300 seconds"
            exit 1
        fi
        sleep 0.5
    done

    cecho "Pods:"
    kubectl get pods -A
}

get_meshnet() {
    cecho "Getting meshnet-cni commit: $MESHNET_COMMIT ..."
    rm -rf meshnet-cni && git clone https://github.com/networkop/meshnet-cni
    cd meshnet-cni && git checkout $MESHNET_COMMIT
    kubectl apply -k manifests/base
    wait_for_pod_counts meshnet 1
    wait_for_all_pods_to_be_ready -ns meshnet

    cd -
    rm -rf meshnet-cni
}

get_ixia_c_operator() {
    cecho "Getting ixia-c-operator ${OPERATOR_RELEASE} ..."
    kubectl apply -f https://github.com/open-traffic-generator/ixia-c-operator/releases/download/v${OPERATOR_RELEASE}/ixiatg-operator.yaml
    wait_for_pod_counts ixiatg-op-system 1
    wait_for_all_pods_to_be_ready -ns ixiatg-op-system
}

rm_ixia_c_operator() {
    cecho "Removing ixia-c-operator ${OPERATOR_RELEASE} ..."
    kubectl delete -f https://github.com/open-traffic-generator/ixia-c-operator/releases/download/v${OPERATOR_RELEASE}/ixiatg-operator.yaml
}

get_metallb() {
    cecho "Getting metallb ..."
    kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/namespace.yaml
    kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)" 
    kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/metallb.yaml
    
    wait_for_pod_counts metallb-system 1
    wait_for_all_pods_to_be_ready -ns metallb-system

    prefix=$(docker network inspect -f '{{.IPAM.Config}}' kind | grep -Eo "[0-9]+\.[0-9]+\.[0-9]+" | tail -n 1)

    yml=resources/global/metallb-config
    sed -e "s/\${prefix}/${prefix}/g" ${yml}.template.yaml > ${yml}.yaml
    cecho "Applying metallb config map for exposing internal services via public IP addresses ..."
    cat ${yml}.yaml
    kubectl apply -f ${yml}.yaml
}

rm_kind_cluster() {
    kind delete cluster 2> /dev/null
    rm -rf $HOME/.kube
    rm -rf $HOME/go/bin/kubectl
}

setup_kind_cluster() {
    rm_kind_cluster
    kind create cluster --config=resources/global/kind-config.yaml --wait 5m
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

load_dut_img() {
    img=$(cat resources/global/${1}-${2}.yaml | cut -d\  -f2)
    cecho "Loading container image for DUT ${2}: ${img} ..."
    
    docker pull ${img}
    kind load docker-image ${img}
}

load_images() {
    IMG=""
    TAG=""

    yml=resources/global/${1}-ixia-configmap.yaml
    cecho "Loading container images for Ixia-c from ${yml} ..."

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

    load_dut_img ${1} ${2}
}

ghcr_login() {
    cecho "Enter Github username and personal-access-token (https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)"
    echo -n "username: "
    read username
    echo -n "token: "
    stty -echo
    read pat
    stty echo
    echo

    echo ${pat} | docker login ghcr.io -u ${username} --password-stdin
}

generate_topology_configs() {
    ixia_version=$(grep '"release":' resources/global/${1}-ixia-configmap.yaml | cut -d\" -f4)
    dut_image=$(cat resources/global/${1}-${2}.yaml | cut -d\  -f2)

    for f in $(ls resources/topology/*.template.txt)
    do
        txt=$(echo "${f}" | sed -e "s/.template.txt//g")
        cat ${txt}.template.txt | \
            sed -e "s/\${ixia_version}/${ixia_version}/g" | \
            sed -e "s#\${dut_image}#${dut_image}#" | \
            tee ${txt}.txt > /dev/null
    done
}

setup_repo() {
    if [ "${1}" = "ghcr" ]
    then
        ghcr_login
        load_images ${1} ${2}
    else
        get_gcloud
        gcloud_auth
        load_images ${1} ${2}
    fi
    
    generate_topology_configs ${1} ${2}
    kubectl apply -f resources/global/${1}-ixia-configmap.yaml
}

setup_testbed() {
    setup_cluster
    setup_test_client
    cecho "Please logout and login again !"
}

get_knebind_conf() {
    cd tests
    KCLI=$(grep "cli:" ${KNEBIND_CONFIG} | cut -d: -f2 | sed -e "s/ //g")
    KCFG=$(grep "kubecfg:" ${KNEBIND_CONFIG} | cut -d: -f2 | sed -e "s/ //g")
    KTOP=$(grep "topology:" ${KNEBIND_CONFIG} | cut -d: -f2 | sed -e "s/ //g")
    KTBD=$(echo ${KTOP} | sed -e "s#/topology/#/testbed/#g")
    echo ${KTBD}
    cd -
}

newtop() {
    get_knebind_conf

    cd tests
    ${KCLI} -v trace --kubecfg ${KCFG} create ${KTOP}
    wait_for_pod_counts ixia-c 1
    wait_for_all_pods_to_be_ready -ns ixia-c
    cd -
}

rmtop() {
    get_knebind_conf

    cd tests
    ${KCLI} -v trace --kubecfg ${KCFG} delete ${KTOP}
    wait_for_pod_counts ixia-c 0
    cd -
}

run() {
    name=$(grep -Eo "Test[0-9a-zA-Z]+" ${1})
    prefix=$(basename ${1} | sed 's/_test.go//g')

    get_knebind_conf

    mkdir -p logs
    cecho "Staring tests, output will be stored in logs/${prefix}.log"
    CGO_ENABLED=0 go test -v -timeout 60s -run ${name} tests/tests \
        -config ${KNEBIND_CONFIG} \
        -testbed ${KTBD} | tee logs/${prefix}.log \
    || true
}

case $1 in
    *   )
        # shift positional arguments so that arg 2 becomes arg 1, etc.
        cmd=${1}
        shift 1
        ${cmd} ${@} || cecho "usage: $0 [name of any function in script]"
    ;;
esac
