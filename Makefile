gofile = go1.17.3.linux-amd64.tar.gz
go:
	wget https://golang.org/dl/$(gofile)
	sudo tar -C /usr/local -xzf $(gofile)
	rm -f $(gofile)
	go version

ver = 3.17.3
protoc_zip = protoc-$(ver)-linux-x86_64.zip
protoc:
	curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v$(ver)/$(protoc_zip)
	sudo unzip -o $(protoc_zip) -d /usr/local bin/protoc
	sudo unzip -o $(protoc_zip) -d /usr/local 'include/*'
	rm -f $(protoc_zip)
	protoc --version

build:
	git submodule update --init --recursive
	cd ./kne/kne_cli && go build -v ./...
	cd ./ondatra &&	go mod tidy -e && go generate -v ./... && go build -v ./...

kne_init:
	./kne/kne_cli/kne_cli create resources/topology/topology-001.txt
	./kne/kne_cli/kne_cli topology push resources/topology/topology-001.txt arista1 resources/dutconfig/arista1.txt
	./kne/kne_cli/kne_cli topology push resources/topology/topology-001.txt arista2 resources/dutconfig/arista2.txt

kne_shut:
	./kne/kne_cli/kne_cli delete resources/topology/topology-001.txt

	