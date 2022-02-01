module tests

go 1.16

replace github.com/openconfig/ondatra => ./ondatra

require (
	github.com/golang/protobuf v1.5.2
	github.com/open-traffic-generator/snappi/gosnappi v0.6.21
	github.com/openconfig/gnmi v0.0.0-20210707145734-c69a5df04b53
	github.com/openconfig/ondatra v0.0.0-00010101000000-000000000000
	github.com/openconfig/ygot v0.12.0
	google.golang.org/grpc v1.42.0 // indirect
	google.golang.org/protobuf v1.27.1
)
