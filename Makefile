setup: protoc go build

go:
	./do.sh get_go

protoc:
	./do.sh get_protoc

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

ct:
	kne/kne_cli/kne_cli --kubecfg resources/kneconfig/config create resources/topology/ixia-arista-ixia.txt

dt:
	kne/kne_cli/kne_cli --kubecfg resources/kneconfig/config delete resources/topology/ixia-arista-ixia.txt

bgp_route_install:
	kne/kne_cli/kne_cli --kubecfg resources/kneconfig/config topology push resources/topology/ixia-arista-ixia.txt arista1 resources/dutconfig/bgp_route_install/set_dut.txt
	-CGO_ENALBED=0 go test -v -timeout 60s -run TestBGPRouteInstall tests/tests -config ../resources/kneconfig/kne-003.yaml -testbed ../resources/testbed/ixia-arista-ixia.txt
	kne/kne_cli/kne_cli --kubecfg resources/kneconfig/config topology push resources/topology/ixia-arista-ixia.txt arista1 resources/dutconfig/bgp_route_install/unset_dut.txt

bgp_route_policy:
	kne/kne_cli/kne_cli --kubecfg resources/kneconfig/config topology push resources/topology/ixia-arista-ixia.txt arista1 resources/dutconfig/bgp_route_policy/set_dut.txt
	-CGO_ENALBED=0 go test -v -timeout 60s -run TestBGPRoutePolicy tests/tests -config ../resources/kneconfig/kne-003.yaml -testbed ../resources/testbed/ixia-arista-ixia.txt
	kne/kne_cli/kne_cli --kubecfg resources/kneconfig/config topology push resources/topology/ixia-arista-ixia.txt arista1 resources/dutconfig/bgp_route_policy/unset_dut.txt

isis_route_install:
	kne/kne_cli/kne_cli --kubecfg resources/kneconfig/config topology push resources/topology/ixia-arista-ixia.txt arista1 resources/dutconfig/bgp_route_install/set_dut.txt
	-CGO_ENALBED=0 go test -v -timeout 60s -run TestISISRouteInstall tests/tests -config ../resources/kneconfig/kne-003.yaml -testbed ../resources/testbed/ixia-arista-ixia.txt
	kne/kne_cli/kne_cli --kubecfg resources/kneconfig/config topology push resources/topology/ixia-arista-ixia.txt arista1 resources/dutconfig/isis_route_install/unset_dut.txt

