module tests

go 1.18

replace github.com/openconfig/ondatra => ./ondatra

require (
	github.com/open-traffic-generator/snappi/gosnappi v0.7.18
	github.com/openconfig/ondatra v0.0.0-00010101000000-000000000000
	github.com/openconfig/ygot v0.16.3
	google.golang.org/protobuf v1.28.0
)
