/* Test BGP Route Installation

Topology:
IXIA (40.40.40.0/24, 0:40:40:40::0/64) -----> ARISTA ------> IXIA (50.50.50.0/24, 0:50:50:50::0/64)

Flows:
- permit v4: 40.40.40.1 -> 50.50.50.1+
- deny v4: 40.40.40.1 -> 60.60.60.1+
- permit v4: 0:40:40:40::1 -> 0:50:50:50::1+
- deny v4: 0:40:40:40::1 -> 0:60:60:60::1+
*/
package tests

import (
	"testing"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra"

	"tests/tests/helpers"
)

func TestIpInIpB2b(t *testing.T) {

	ate := ondatra.ATE(t, "ate1")
	ondatra.ATE(t, "ate2")

	otg := ate.OTG()
	defer helpers.CleanupTest(otg, t, false)

	config, expected := ipInIpB2bConfig(t, otg)
	otg.PushConfig(t, config)
	gnmiClient, err := helpers.NewGnmiClient(helpers.NewGnmiQuery(ate), config)
	if err != nil {
		t.Fatal(err)
	}

	otg.StartTraffic(t)

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.FlowMetricsOk(expected) }, nil)
}

func ipInIpB2bConfig(t *testing.T, otg *ondatra.OTG) (gosnappi.Config, helpers.ExpectedState) {
	config := otg.NewConfig()

	port1 := config.Ports().Add().SetName("ixia-c-port1")
	port2 := config.Ports().Add().SetName("ixia-c-port2")

	// OTG traffic configuration
	f1 := config.Flows().Add().SetName("p1.v4.p2.permit")
	f1.Metrics().SetEnable(true)
	f1.TxRx().Port().
		SetTxName(port1.Name()).
		SetRxName(port2.Name())
	f1.Size().SetFixed(512)
	f1.Rate().SetPps(500)
	f1.Duration().FixedPackets().SetPackets(1000)
	e1 := f1.Packet().Add().Ethernet()
	e1.Src().SetValue("00:00:00:00:00:0A")
	e1.Dst().SetValue("00:00:00:00:00:0B")

	outerIp := f1.Packet().Add().Ipv4()
	outerIp.Src().SetValue("1.1.1.1")
	outerIp.Dst().SetValue("1.1.2.1")

	innerIp := f1.Packet().Add().Ipv4()
	innerIp.Src().Increment().SetStart("1.1.3.1").SetStep("0.0.0.1").SetCount(5)
	innerIp.Dst().Increment().SetStart("1.1.4.1").SetStep("0.0.0.1").SetCount(5)

	udp := f1.Packet().Add().Udp()
	udp.SrcPort().SetValue(6001)
	udp.DstPort().SetValue(6002)

	expected := helpers.ExpectedState{
		Flow: map[string]helpers.ExpectedFlowMetrics{
			f1.Name(): {FramesRx: 1000, FramesRxRate: 0},
		},
	}

	return config, expected
}
