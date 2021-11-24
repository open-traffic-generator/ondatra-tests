build:
	git submodule update --init --recursive
	ROOT=`pwd`
	cd ondatra
	go generate -v ./...
	go mod tidy -e
	cd $(PWD)
	go build -v ./...
