build:
	ROOT=`pwd`
    cd ./ondatra
	go generate ./...
	go mod tidy -e
	cd $(PWD)
	go build -v ./...
