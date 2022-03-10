package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/openconfig/ondatra"
	kinit "github.com/openconfig/ondatra/knebind/init"
)

// TestMain is the first thing that's executed upon running `go test ...`
func TestMain(m *testing.M) {
	fmt.Println(os.Args)
	os.Args = append(os.Args, "-config", "../resources/global/knebind-config.yaml", "-testbed", "../resources/testbed/ixia-arista-ixia.txt")
	ondatra.RunTests(m, kinit.Init)
}
