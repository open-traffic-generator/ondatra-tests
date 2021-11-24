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
