#!/bin/sh

GO_VERSION=1.17.3
PROTOC_VERSION=3.17.3

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

install_deps() {
	# Dependencies required by this project
    apt-get update \
	&& apt-get -y install --no-install-recommends apt-utils dialog 2>&1 \
    && apt-get install -y wget git make vim curl unzip
}

get_go() {
    curl -kLO https://dl.google.com/go/${GO_TARGZ} \
	&& rm -rf /usr/local/go \
    && tar -C /usr/local -xzf ${GO_TARGZ} \
	&& go version \
	&& go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26 \
	&& go install golang.org/x/tools/cmd/goimports@v0.1.7
}

get_protoc() {
    curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP} \
	&& unzip ${PROTOC_ZIP} -d /usr/local \
	&& rm -f ${PROTOC_ZIP} \
	&& protoc --version
}

setup() {
    install_deps && get_protoc && get_go
}

case $1 in
    *   )
    $1 || echo "usage: $0 [setup|install_deps|get_go|get_protoc]"
    ;;
esac
