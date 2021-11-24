build:
	git submodule update --init --recursive
	ROOT=`pwd`
	cd kne/kne_cli
	go build -v ./...
	cd $(PWD)
	cd ondatra
	go mod tidy -e
	go generate -v ./...
	go build -v ./...

kne_init:
	./kne/kne_cli/kne_cli create resources/topology/topology-001.txt
	./kne/kne_cli/kne_cli topology push resources/topology/topology-001.txt arista1 resources/dutconfig/arista1.txt
	./kne/kne_cli/kne_cli topology push resources/topology/topology-001.txt arista2 resources/dutconfig/arista2.txt

kne_shut:
	./kne/kne_cli/kne_cli delete resources/topology/topology-001.txt

	