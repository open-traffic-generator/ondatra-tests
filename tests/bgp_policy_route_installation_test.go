/* Test BGP Policy Route Installation

Topology:
INFO[0000] Adding Link: ixia-c-port1:eth1 arista1:eth1
INFO[0000] Adding Link: arista1:eth2 arista2:eth1
INFO[0000] Adding Link: arista2:eth2 ixia-c-port2:eth1
INFO[0000] Adding Link: arista2:eth3 ixia-c-port3:eth1

Configuration:
Establish two BGP sessions:
(1) between ATE port-1 and DUT port-1, and
(2) between ATE port-2 and DUT port-2.

Advertise prefixes from ATE port-1, observe received prefixes at ATE port-2.
Send traffic flow in both IPv4 and IPv6 of various SrcNet and DstNet pairs
between ATE port-1 and ATE port-2.

Validation:
- Traffic is forwarded to all installed routes.
- Traffic is not forwarded for denied or withdrawn routes.
*/
package tests

import (
	"testing"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra"
)

func TestBGPPolicyRouteInstallation(t *testing.T) {
	otg := ondatra.OTGs(t)
	defer otg.NewConfig(t)
	defer otg.StopProtocols(t)
	defer otg.StopTraffic(t)

	config, expected := configureOTG(t, otg)
	otg.PushConfig(t, config)
	otg.StartProtocols(t)

	gnmiClient, err := NewGnmiClient(otg.NewGnmiQuery(t), config)
	if err != nil {
		t.Fatal(err)
	}

	WaitFor(t, func() (bool, error) { return gnmiClient.AllBgp4SessionUp(expected) }, nil)

	WaitFor(t, func() (bool, error) { return gnmiClient.AllBgp6SessionUp(expected) }, nil)

	otg.StartTraffic(t)

	WaitFor(t, func() (bool, error) { return gnmiClient.FlowMetricsOk(expected) }, nil)
}

func configureOTG(t *testing.T, otg *ondatra.OTG) (gosnappi.Config, ExpectedState) {
	config := otg.NewConfig(t)
	expected := NewExpectedState()

	port1 := config.Ports().Add().SetName("ixia-c-port1")
	port2 := config.Ports().Add().SetName("ixia-c-port2")

	iDut1 := config.Devices().Add().SetName("iDut1")
	iDut1Eth := iDut1.Ethernets().Add().
		SetName("iDut1.eth").
		SetPortName(port1.Name()).
		SetMac("00:00:01:01:01:01")
	iDut1Ipv4 := iDut1Eth.Ipv4Addresses().Add().
		SetName("iDut1.ipv4").
		SetAddress("1.1.1.1").
		SetGateway("1.1.1.2")
	iDut1Ipv6 := iDut1Eth.Ipv6Addresses().Add().
		SetName("iDut1.ipv6").
		SetAddress("0:1:1:1::1").
		SetGateway("0:1:1:1::2")
	iDut2 := config.Devices().Add().SetName("bgpNeti1")
	iDut2Eth := iDut2.Ethernets().Add().
		SetName("iDut2.eth").
		SetPortName(port2.Name()).
		SetMac("00:00:02:01:01:01")
	iDut2Ipv4 := iDut2Eth.Ipv4Addresses().Add().
		SetName("iDut2.ipv4").
		SetAddress("2.2.2.2").
		SetGateway("2.2.2.1")
	iDut2Ipv6 := iDut2Eth.Ipv6Addresses().Add().
		SetName("iDut2.ipv6").
		SetAddress("0:2:2:2::2").
		SetGateway("0:2:2:2::1")

	// dut1 peers
	dut1Bgp := iDut1.Bgp().
		SetRouterId(iDut1Ipv4.Address())
	dut1Bgp4Peer := dut1Bgp.Ipv4Interfaces().Add().
		SetIpv4Name(iDut1Ipv4.Name()).
		Peers().Add().
		SetName("iDut1.bgp4.peer").
		SetPeerAddress(iDut1Ipv4.Gateway()).
		SetAsNumber(1111).
		SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	dut1Bgp6Peer := dut1Bgp.Ipv6Interfaces().Add().
		SetIpv6Name(iDut1Ipv6.Name()).
		Peers().Add().
		SetName("iDut1.bgp6.peer").
		SetPeerAddress(iDut1Ipv6.Gateway()).
		SetAsNumber(1111).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP)
	// dut2 routes
	dut1Bgp4PeerRoutes := dut1Bgp4Peer.V4Routes().Add().
		SetName("iDut1.bgp4.peer.rr4").
		SetNextHopIpv4Address(iDut1Ipv4.Address()).
		SetNextHopAddressType(gosnappi.BgpV4RouteRangeNextHopAddressType.IPV4).
		SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
	dut1Bgp4PeerRoutes.Addresses().Add().
		SetAddress("30.30.30.0").
		SetPrefix(24).
		SetCount(5).
		SetStep(2)
	dut1Bgp6PeerRoutes := dut1Bgp6Peer.V6Routes().Add().
		SetName("iDut1.bgp4.peer.rr6").
		SetNextHopIpv6Address(iDut1Ipv6.Address()).
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
	dut1Bgp6PeerRoutes.Addresses().Add().
		SetAddress("0:40:40:40::0").
		SetPrefix(64).
		SetCount(5).
		SetStep(2)

	// dut2 peers
	dut2Bgp := iDut2.Bgp().
		SetRouterId(iDut2Ipv4.Address())
	dut2BgpIf4 := dut2Bgp.Ipv4Interfaces().Add().
		SetIpv4Name(iDut2Ipv4.Name())
	dut2Bgp4Peer := dut2BgpIf4.Peers().Add().
		SetName("iDut2.bgp4.peer").
		SetPeerAddress(iDut2Ipv4.Gateway()).
		SetAsNumber(2222).
		SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	dut2Bgp6Peer := dut2Bgp.Ipv6Interfaces().Add().
		SetIpv6Name(iDut2Ipv6.Name()).
		Peers().Add().
		SetName("iDut2.bgp6.peer").
		SetPeerAddress(iDut2Ipv6.Gateway()).
		SetAsNumber(2222).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP)
	// dut2 routes
	dut2Bgp4PeerRoutes := dut2Bgp4Peer.V4Routes().Add().
		SetName("iDut2.bgp4.peer.rr4").
		SetNextHopIpv4Address(iDut2Ipv4.Address()).
		SetNextHopAddressType(gosnappi.BgpV4RouteRangeNextHopAddressType.IPV4).
		SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
	dut2Bgp4PeerRoutes.Addresses().Add().
		SetAddress("50.50.50.0").
		SetPrefix(24).
		SetCount(5).
		SetStep(2)
	dut2Bgp6PeerRoutes := dut2Bgp6Peer.V6Routes().Add().
		SetName("iDut2.bgp4.peer.rr6").
		SetNextHopIpv6Address(iDut2Ipv6.Address()).
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
	dut2Bgp6PeerRoutes.Addresses().Add().
		SetAddress("0:60:60:60::0").
		SetPrefix(64).
		SetCount(5).
		SetStep(2)

	// OTG traffic configuration
	f1 := config.Flows().Add().SetName("v4.ok")
	f1.Metrics().SetEnable(true)
	f1.TxRx().Device().
		SetTxNames([]string{dut2Bgp4PeerRoutes.Name()}).
		SetRxNames([]string{dut1Bgp4PeerRoutes.Name()})
	f1.Size().SetFixed(512)
	f1.Rate().SetPps(500)
	f1.Duration().FixedPackets().SetPackets(1000)
	e1 := f1.Packet().Add().Ethernet()
	e1.Src().SetValue(iDut1Eth.Mac())
	e1.Dst().SetValue("00:00:00:00:00:00")
	v4 := f1.Packet().Add().Ipv4()
	v4.Src().Increment().SetStart("50.50.50.1").SetStep("0.0.0.1").SetCount(5)
	v4.Dst().Increment().SetStart("30.30.30.1").SetStep("0.0.0.1").SetCount(5)

	f1d := config.Flows().Add().SetName("v4.denied")
	f1d.Metrics().SetEnable(true)
	f1d.TxRx().Device().
		SetTxNames([]string{dut2Bgp4PeerRoutes.Name()}).
		SetRxNames([]string{dut1Bgp4PeerRoutes.Name()})
	f1d.Size().SetFixed(512)
	f1d.Rate().SetPps(500)
	f1d.Duration().FixedPackets().SetPackets(1000)
	e1d := f1d.Packet().Add().Ethernet()
	e1d.Src().SetValue(iDut1Eth.Mac())
	e1d.Dst().SetValue("00:00:00:00:00:00")
	v4d := f1d.Packet().Add().Ipv4()
	v4d.Src().SetValues([]string{"50.50.50.1", "90.90.90.1"})
	v4d.Dst().SetValues([]string{"4.4.4.1", "31.30.30.1"})

	f2 := config.Flows().Add().SetName("v6.ok")
	f2.Metrics().SetEnable(true)
	f2.TxRx().Device().
		SetTxNames([]string{dut2Bgp6PeerRoutes.Name()}).
		SetRxNames([]string{dut1Bgp4PeerRoutes.Name()})
	f2.Size().SetFixed(512)
	f2.Rate().SetPps(500)
	f2.Duration().FixedPackets().SetPackets(1000)
	e2 := f2.Packet().Add().Ethernet()
	e2.Src().SetValue(iDut2Eth.Mac())
	e2.Dst().SetValue("00:00:00:00:00:00")
	v6 := f2.Packet().Add().Ipv6()
	v6.Src().Increment().SetStart("0:60:60:60::1").SetStep("::1").SetCount(5)
	v6.Dst().Increment().SetStart("0:40:40:40::1").SetStep("::1").SetCount(5)

	expected.Bgp4[dut1Bgp4Peer.Name()] = ExpectedBgpMetrics{Advertised: 5, Received: 5}
	expected.Bgp4[dut2Bgp4Peer.Name()] = ExpectedBgpMetrics{Advertised: 5, Received: 5}
	expected.Bgp6[dut1Bgp6Peer.Name()] = ExpectedBgpMetrics{Advertised: 5, Received: 5}
	expected.Bgp6[dut2Bgp6Peer.Name()] = ExpectedBgpMetrics{Advertised: 5, Received: 5}
	expected.Flow[f1.Name()] = ExpectedFlowMetrics{FramesRx: 1000, FramesRxRate: 0}
	expected.Flow[f1d.Name()] = ExpectedFlowMetrics{FramesRx: 0, FramesRxRate: 0}
	expected.Flow[f2.Name()] = ExpectedFlowMetrics{FramesRx: 1000, FramesRxRate: 0}

	return config, expected
}
