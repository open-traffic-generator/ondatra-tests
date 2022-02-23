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

	oc "github.com/openconfig/ondatra/telemetry"
)

type flowMetricTestParam struct {
	routerID uint32
	srcAS    uint32
	dstAS    uint32
	dutSrc   helpers.Attributes
	dutDst   helpers.Attributes
	ateSrc   helpers.Attributes
	ateDst   helpers.Attributes
}

var (
	flowMetricTestParams = flowMetricTestParam{
		routerID: 3333,
		srcAS:    1111,
		dstAS:    2222,
		dutSrc: helpers.Attributes{
			IPv4:    "1.1.1.3",
			IPv6:    "0:1:1:1::3",
			IPv4Len: 24,
			IPv6Len: 64,
		},
		dutDst: helpers.Attributes{
			IPv4:    "2.2.2.3",
			IPv6:    "0:2:2:2::3",
			IPv4Len: 24,
			IPv6Len: 64,
		},
		ateSrc: helpers.Attributes{
			IPv4:    "1.1.1.1",
			IPv6:    "0:1:1:1::1",
			IPv4Len: 24,
			IPv6Len: 64,
		},
		ateDst: helpers.Attributes{
			IPv4:    "2.2.2.2",
			IPv6:    "0:2:2:2::2",
			IPv4Len: 24,
			IPv6Len: 64,
		},
	}
)

func flowMetricTestConfigureInterface(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Start DUT Interface Config")
	dc := dut.Config()

	i1 := flowMetricTestParams.dutSrc.NewInterface(helpers.InterfaceMap[dut.Port(t, "port1").Name()])
	dc.Interface(i1.GetName()).Replace(t, i1)

	i2 := flowMetricTestParams.dutDst.NewInterface(helpers.InterfaceMap[dut.Port(t, "port2").Name()])
	dc.Interface(i2.GetName()).Replace(t, i2)
}

func flowMetricTestBuildNbrList() []*helpers.BgpNeighbor {
	nbr1v4 := &helpers.BgpNeighbor{As: flowMetricTestParams.srcAS, NeighborIP: flowMetricTestParams.ateSrc.IPv4, IsV4: true}
	nbr1v6 := &helpers.BgpNeighbor{As: flowMetricTestParams.srcAS, NeighborIP: flowMetricTestParams.ateSrc.IPv6, IsV4: false}
	nbr2v4 := &helpers.BgpNeighbor{As: flowMetricTestParams.dstAS, NeighborIP: flowMetricTestParams.ateDst.IPv4, IsV4: true}
	nbr2v6 := &helpers.BgpNeighbor{As: flowMetricTestParams.dstAS, NeighborIP: flowMetricTestParams.ateDst.IPv6, IsV4: false}
	return []*helpers.BgpNeighbor{nbr1v4, nbr2v4, nbr1v6, nbr2v6}
}

func flowMetricTestConfigureBGP(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Start DUT BGP Config")
	dutConfPath := dut.Config().NetworkInstance("default").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	helpers.LogYgot(t, "DUT BGP Config before", dutConfPath, dutConfPath.Get(t))
	dutConfPath.Replace(t, nil)
	nbrList := flowMetricTestBuildNbrList()
	dutConf := helpers.BgpAppendNbr(flowMetricTestParams.routerID, nbrList)
	dutConfPath.Replace(t, dutConf)
}

func flowMetricTestConfigureDUT(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Start Setting DUT Config")

	flowMetricTestConfigureInterface(t, dut)
	helpers.ConfigDUTs(map[string]string{"arista1": "../resources/dutconfig/bgp_route_install/set_dut.txt"})
	flowMetricTestConfigureBGP(t, dut)
}

func flowMetricTestUnsetInterface(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Start Unsetting DUT Interface Config")
	dc := dut.Config()

	i1 := helpers.RemoveInterface(helpers.InterfaceMap[dut.Port(t, "port1").Name()])
	dc.Interface(i1.GetName()).Replace(t, i1)

	i2 := helpers.RemoveInterface(helpers.InterfaceMap[dut.Port(t, "port2").Name()])
	dc.Interface(i2.GetName()).Replace(t, i2)
}

func flowMetricTestUnsetBGP(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Start Removing DUT BGP Neighbor")
	dutConfPath := dut.Config().NetworkInstance("default").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	helpers.LogYgot(t, "DUT BGP Config before", dutConfPath, dutConfPath.Get(t))
	dutConfPath.Replace(t, nil)
	nbrList := flowMetricTestBuildNbrList()
	dutConf := helpers.BgpDeleteNbr(nbrList)
	dutConfPath.Replace(t, dutConf)
}

func flowMetricTestUnsetDUT(t *testing.T, dut *ondatra.DUTDevice) {
	t.Logf("Start Un-Setting DUT Config")
	// helpers.ConfigDUTs(map[string]string{"arista1": "../resources/dutconfig/bgp_route_install/unset_dut.txt"})

	flowMetricTestUnsetInterface(t, dut)
	flowMetricTestUnsetBGP(t, dut)
}

func flowMetricTestCheckBgpParameters(t *testing.T, dut *ondatra.DUTDevice) {
	statePath := dut.Telemetry().NetworkInstance("default").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	nbrPath_1 := statePath.Neighbor(flowMetricTestParams.ateSrc.IPv4)
	nbrPathv6_1 := statePath.Neighbor(flowMetricTestParams.ateSrc.IPv6)
	nbrPath_2 := statePath.Neighbor(flowMetricTestParams.ateDst.IPv4)
	nbrPathv6_2 := statePath.Neighbor(flowMetricTestParams.ateDst.IPv6)

	// Get BGP adjacency state
	t.Logf("Verifying BGP Adjacency State")
	status := nbrPath_1.SessionState().Get(t)
	t.Logf("BGP adjacency for %s: %s", flowMetricTestParams.ateSrc.IPv4, status)
	if want := oc.Bgp_Neighbor_SessionState_ESTABLISHED; status != want {
		t.Errorf("Get(BGP peer %s status): got %d, want %d", flowMetricTestParams.ateSrc.IPv4, status, want)
	}

	status = nbrPathv6_1.SessionState().Get(t)
	t.Logf("BGP adjacency for %s: %s", flowMetricTestParams.ateSrc.IPv6, status)
	if want := oc.Bgp_Neighbor_SessionState_ESTABLISHED; status != want {
		t.Errorf("Get(BGP peer %s status): got %d, want %d", flowMetricTestParams.ateSrc.IPv6, status, want)
	}

	status = nbrPath_2.SessionState().Get(t)
	t.Logf("BGP adjacency for %s: %s", flowMetricTestParams.ateDst.IPv4, status)
	if want := oc.Bgp_Neighbor_SessionState_ESTABLISHED; status != want {
		t.Errorf("Get(BGP peer %s status): got %d, want %d", flowMetricTestParams.ateDst.IPv4, status, want)
	}

	status = nbrPathv6_2.SessionState().Get(t)
	t.Logf("BGP adjacency for %s: %s", flowMetricTestParams.ateDst.IPv6, status)
	if want := oc.Bgp_Neighbor_SessionState_ESTABLISHED; status != want {
		t.Errorf("Get(BGP peer %s status): got %d, want %d", flowMetricTestParams.ateDst.IPv6, status, want)
	}

	nbr_1 := statePath.Get(t).GetNeighbor(flowMetricTestParams.ateSrc.IPv4)
	nbrv6_1 := statePath.Get(t).GetNeighbor(flowMetricTestParams.ateSrc.IPv6)
	nbr_2 := statePath.Get(t).GetNeighbor(flowMetricTestParams.ateDst.IPv4)
	nbrv6_2 := statePath.Get(t).GetNeighbor(flowMetricTestParams.ateDst.IPv6)

	// Check BGP Transitions
	t.Logf("Verifying BGP Transitions")
	estTrans := nbr_1.GetEstablishedTransitions()
	t.Logf("Got established transitions for Neighbor %s : %d", flowMetricTestParams.ateSrc.IPv4, estTrans)
	if estTrans != 1 {
		t.Errorf("Wrong established-transitions: got %v, want 1", estTrans)
	}

	estTrans = nbrv6_1.GetEstablishedTransitions()
	t.Logf("Got established transitions for Neighbor %s : %d", flowMetricTestParams.ateSrc.IPv6, estTrans)
	if estTrans != 1 {
		t.Errorf("Wrong established-transitions: got %v, want 1", estTrans)
	}

	estTrans = nbr_2.GetEstablishedTransitions()
	t.Logf("Got established transitions for Neighbor %s : %d", flowMetricTestParams.ateDst.IPv4, estTrans)
	if estTrans != 1 {
		t.Errorf("Wrong established-transitions: got %v, want 1", estTrans)
	}

	estTrans = nbrv6_2.GetEstablishedTransitions()
	t.Logf("Got established transitions for Neighbor %s : %d", flowMetricTestParams.ateDst.IPv6, estTrans)
	if estTrans != 1 {
		t.Errorf("Wrong established-transitions: got %v, want 1", estTrans)
	}

	// Check BGP neighbor address from telemetry
	t.Logf("Verifying BGP Neighbor Addresses")
	addr_1 := nbrPath_1.Get(t).GetNeighborAddress()
	addrv6_1 := nbrPathv6_1.Get(t).GetNeighborAddress()
	addr_2 := nbrPath_2.Get(t).GetNeighborAddress()
	adddrv6_2 := nbrPathv6_2.Get(t).GetNeighborAddress()

	t.Logf("Got neighbor address: %s", addr_1)
	if addr_1 != flowMetricTestParams.ateSrc.IPv4 {
		t.Errorf("Bgp neighbor address: got %v, want %v", addr_1, flowMetricTestParams.ateSrc.IPv4)
	}
	t.Logf("Got neighbor address: %s", addrv6_1)
	if addrv6_1 != flowMetricTestParams.ateSrc.IPv6 {
		t.Errorf("Bgp neighbor address: got %v, want %v", addrv6_1, flowMetricTestParams.ateSrc.IPv6)
	}
	t.Logf("Got neighbor address: %s", addr_2)
	if addr_2 != flowMetricTestParams.ateDst.IPv4 {
		t.Errorf("Bgp neighbor address: got %v, want %v", addr_2, flowMetricTestParams.ateDst.IPv4)
	}
	t.Logf("Got neighbor address: %s", adddrv6_2)
	if adddrv6_2 != flowMetricTestParams.ateDst.IPv6 {
		t.Errorf("Bgp neighbor address: got %v, want %v", adddrv6_2, flowMetricTestParams.ateDst.IPv6)
	}

	// Check BGP neighbor AS number from telemetry
	t.Logf("Verifying BGP Neighbor AS Number")
	peerAS_1 := nbrPath_1.Get(t).GetPeerAs()
	peerAS_2 := nbrPath_2.Get(t).GetPeerAs()
	peerv6AS_1 := nbrPathv6_1.Get(t).GetPeerAs()
	peerv6AS_2 := nbrPathv6_2.Get(t).GetPeerAs()

	t.Logf("Got neighbor %s AS: %d", flowMetricTestParams.ateSrc.IPv4, peerAS_1)
	if peerAS_1 != flowMetricTestParams.srcAS {
		t.Errorf("Bgp peerAs: got %v, want %v", peerAS_1, flowMetricTestParams.srcAS)
	}

	t.Logf("Got neighbor %s AS: %d", flowMetricTestParams.ateSrc.IPv6, peerv6AS_1)
	if peerv6AS_1 != flowMetricTestParams.srcAS {
		t.Errorf("Bgp peerAs: got %v, want %v", peerv6AS_1, flowMetricTestParams.srcAS)
	}

	t.Logf("Got neighbor %s AS: %d", flowMetricTestParams.ateDst.IPv4, peerAS_2)
	if peerAS_2 != flowMetricTestParams.dstAS {
		t.Errorf("Bgp peerAs: got %v, want %v", peerAS_2, flowMetricTestParams.dstAS)
	}

	t.Logf("Got neighbor %s AS: %d", flowMetricTestParams.ateDst.IPv6, peerv6AS_2)
	if peerv6AS_2 != flowMetricTestParams.dstAS {
		t.Errorf("Bgp peerAs: got %v, want %v", peerv6AS_2, flowMetricTestParams.dstAS)
	}

	// Check BGP neighbor is enabled
	t.Logf("Verifying BGP Neighbors Are Enabled")
	if !nbrPath_1.Get(t).GetEnabled() {
		t.Errorf("Expected neighbor %v to be enabled", flowMetricTestParams.ateSrc.IPv4)
	} else {
		t.Logf("Neighbor %v is enabled", flowMetricTestParams.ateSrc.IPv4)
	}

	if !nbrPath_2.Get(t).GetEnabled() {
		t.Errorf("Expected neighbor %v to be enabled", flowMetricTestParams.ateDst.IPv6)
	} else {
		t.Logf("Neighbor %v is enabled", flowMetricTestParams.ateDst.IPv6)
	}

	if !nbrPathv6_1.Get(t).GetEnabled() {
		t.Errorf("Expected neighbor %v to be enabled", flowMetricTestParams.ateSrc.IPv6)
	} else {
		t.Logf("Neighbor %v is enabled", flowMetricTestParams.ateSrc.IPv6)
	}

	if !nbrPathv6_2.Get(t).GetEnabled() {
		t.Errorf("Expected neighbor %v to be enabled", flowMetricTestParams.ateDst.IPv4)
	} else {
		t.Logf("Neighbor %v is enabled", flowMetricTestParams.ateDst.IPv4)
	}
}

func TestFlowMetric(t *testing.T) {
	dut := ondatra.DUT(t, "dut")
	// Set DUT Config over gNMI
	flowMetricTestConfigureDUT(t, dut)

	// Unset DUT Config over gNMI
	defer flowMetricTestUnsetDUT(t, dut)

	ate := ondatra.ATE(t, "ate1")
	ondatra.ATE(t, "ate2")

	otg := ate.OTG()
	defer helpers.CleanupTest(otg, t, true)

	config, expected := flowMetricTestConfig(t, otg)

	otg.PushConfig(t, config)
	otg.StartProtocols(t)

	gnmiClient, err := helpers.NewGnmiClient(otg.NewGnmiQuery(t), config)
	if err != nil {
		t.Fatal(err)
	}

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.AllBgp4SessionUp(expected) }, nil)
	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.AllBgp6SessionUp(expected) }, nil)

	t.Logf("Verifying Port Status")
	helpers.VerifyPortsUp(t, dut.Device)

	t.Logf("Check BGP Parameters")
	flowMetricTestCheckBgpParameters(t, dut)

	otg.StartTraffic(t)

	ExpectedFlowMetrics := helpers.ExpectedState{
		Flow: map[string]helpers.ExpectedFlowMetrics{
			"f1": {FramesTx: 1000, FramesTxRate: 0, FramesRx: 1000, FramesRxRate: 0, Transmit: "stopped"},
			"f2": {FramesTx: -1, FramesTxRate: -1, FramesRx: -1, FramesRxRate: -1, Transmit: "started"},
			"f3": {FramesTx: -1, FramesTxRate: -1, FramesRx: -1, FramesRxRate: -1, Transmit: "started"},
		},
	}

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.FlowMetricsOk(ExpectedFlowMetrics) }, nil)

	ExpectedFlowMetrics = helpers.ExpectedState{
		Flow: map[string]helpers.ExpectedFlowMetrics{
			"f1": {FramesTx: 1000, FramesTxRate: 0, FramesRx: 1000, FramesRxRate: 0, Transmit: "stopped"},
			"f2": {FramesTx: 2000, FramesTxRate: 0, FramesRx: 2000, FramesRxRate: 0, Transmit: "stopped"},
			"f3": {FramesTx: -1, FramesTxRate: -1, FramesRx: -1, FramesRxRate: -1, Transmit: "started"},
		},
	}

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.FlowMetricsOk(ExpectedFlowMetrics) }, nil)

	ExpectedFlowMetrics = helpers.ExpectedState{
		Flow: map[string]helpers.ExpectedFlowMetrics{
			"f1": {FramesTx: 1000, FramesTxRate: 0, FramesRx: 1000, FramesRxRate: 0, Transmit: "stopped"},
			"f2": {FramesTx: 2000, FramesTxRate: 0, FramesRx: 2000, FramesRxRate: 0, Transmit: "stopped"},
			"f3": {FramesTx: 3000, FramesTxRate: 0, FramesRx: 3000, FramesRxRate: 0, Transmit: "stopped"},
		},
	}

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.FlowMetricsOk(ExpectedFlowMetrics) }, nil)
}

func flowMetricTestConfig(t *testing.T, otg *ondatra.OTGAPI) (gosnappi.Config, helpers.ExpectedState) {
	config := otg.NewConfig(t)

	port1 := config.Ports().Add().SetName("ixia-c-port1")
	port2 := config.Ports().Add().SetName("ixia-c-port2")

	dutPort1 := config.Devices().Add().SetName("dutPort1")
	dutPort1Eth := dutPort1.Ethernets().Add().
		SetName("dutPort1.eth").
		SetPortName(port1.Name()).
		SetMac("00:00:01:01:01:01")
	dutPort1Ipv4 := dutPort1Eth.Ipv4Addresses().Add().
		SetName("dutPort1.ipv4").
		SetAddress(flowMetricTestParams.ateSrc.IPv4).
		SetGateway(flowMetricTestParams.dutSrc.IPv4).
		SetPrefix(int32(flowMetricTestParams.ateSrc.IPv4Len))
	dutPort1Ipv6 := dutPort1Eth.Ipv6Addresses().Add().
		SetName("dutPort1.ipv6").
		SetAddress(flowMetricTestParams.ateSrc.IPv6).
		SetGateway(flowMetricTestParams.dutSrc.IPv6).
		SetPrefix(int32(flowMetricTestParams.ateSrc.IPv6Len))
	dutPort2 := config.Devices().Add().SetName("dutPort2")
	dutPort2Eth := dutPort2.Ethernets().Add().
		SetName("dutPort2.eth").
		SetPortName(port2.Name()).
		SetMac("00:00:02:01:01:01")
	dutPort2Ipv4 := dutPort2Eth.Ipv4Addresses().Add().
		SetName("dutPort2.ipv4").
		SetAddress(flowMetricTestParams.ateDst.IPv4).
		SetGateway(flowMetricTestParams.dutDst.IPv4).
		SetPrefix(int32(flowMetricTestParams.ateDst.IPv4Len))
	dutPort2Ipv6 := dutPort2Eth.Ipv6Addresses().Add().
		SetName("dutPort2.ipv6").
		SetAddress(flowMetricTestParams.ateDst.IPv6).
		SetGateway(flowMetricTestParams.dutDst.IPv6).
		SetPrefix(int32(flowMetricTestParams.ateDst.IPv6Len))

	dutPort1Bgp := dutPort1.Bgp().
		SetRouterId(dutPort1Ipv4.Address())
	dutPort1Bgp4Peer := dutPort1Bgp.Ipv4Interfaces().Add().
		SetIpv4Name(dutPort1Ipv4.Name()).
		Peers().Add().
		SetName("dutPort1.bgp4.peer").
		SetPeerAddress(dutPort1Ipv4.Gateway()).
		SetAsNumber(int32(flowMetricTestParams.srcAS)).
		SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	dutPort1Bgp6Peer := dutPort1Bgp.Ipv6Interfaces().Add().
		SetIpv6Name(dutPort1Ipv6.Name()).
		Peers().Add().
		SetName("dutPort1.bgp6.peer").
		SetPeerAddress(dutPort1Ipv6.Gateway()).
		SetAsNumber(int32(flowMetricTestParams.srcAS)).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP)

	dutPort1Bgp4PeerRoutes := dutPort1Bgp4Peer.V4Routes().Add().
		SetName("dutPort1.bgp4.peer.rr4").
		SetNextHopIpv4Address(dutPort1Ipv4.Address()).
		SetNextHopAddressType(gosnappi.BgpV4RouteRangeNextHopAddressType.IPV4).
		SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
	dutPort1Bgp4PeerRoutes.Addresses().Add().
		SetAddress("40.40.40.0").
		SetPrefix(24).
		SetCount(5).
		SetStep(2)
	dutPort1Bgp6PeerRoutes := dutPort1Bgp6Peer.V6Routes().Add().
		SetName("dutPort1.bgp4.peer.rr6").
		SetNextHopIpv6Address(dutPort1Ipv6.Address()).
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
	dutPort1Bgp6PeerRoutes.Addresses().Add().
		SetAddress("0:40:40:40::0").
		SetPrefix(64).
		SetCount(5).
		SetStep(2)

	dutPort2Bgp := dutPort2.Bgp().
		SetRouterId(dutPort2Ipv4.Address())
	dutPort2BgpIf4 := dutPort2Bgp.Ipv4Interfaces().Add().
		SetIpv4Name(dutPort2Ipv4.Name())
	dutPort2Bgp4Peer := dutPort2BgpIf4.Peers().Add().
		SetName("dutPort2.bgp4.peer").
		SetPeerAddress(dutPort2Ipv4.Gateway()).
		SetAsNumber(int32(flowMetricTestParams.dstAS)).
		SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	dutPort2Bgp6Peer := dutPort2Bgp.Ipv6Interfaces().Add().
		SetIpv6Name(dutPort2Ipv6.Name()).
		Peers().Add().
		SetName("dutPort2.bgp6.peer").
		SetPeerAddress(dutPort2Ipv6.Gateway()).
		SetAsNumber(int32(flowMetricTestParams.dstAS)).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP)

	dutPort2Bgp4PeerRoutes := dutPort2Bgp4Peer.V4Routes().Add().
		SetName("dutPort2.bgp4.peer.rr4").
		SetNextHopIpv4Address(dutPort2Ipv4.Address()).
		SetNextHopAddressType(gosnappi.BgpV4RouteRangeNextHopAddressType.IPV4).
		SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
	dutPort2Bgp4PeerRoutes.Addresses().Add().
		SetAddress("50.50.50.0").
		SetPrefix(24).
		SetCount(5).
		SetStep(2)
	dutPort2Bgp6PeerRoutes := dutPort2Bgp6Peer.V6Routes().Add().
		SetName("dutPort2.bgp4.peer.rr6").
		SetNextHopIpv6Address(dutPort2Ipv6.Address()).
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
	dutPort2Bgp6PeerRoutes.Addresses().Add().
		SetAddress("0:50:50:50::0").
		SetPrefix(64).
		SetCount(5).
		SetStep(2)

	// OTG traffic configuration
	f1 := config.Flows().Add().SetName("f1")
	f1.Metrics().SetEnable(true)
	f1.TxRx().Device().
		SetTxNames([]string{dutPort1Bgp4PeerRoutes.Name()}).
		SetRxNames([]string{dutPort2Bgp4PeerRoutes.Name()})
	f1.Size().SetFixed(512)
	f1.Rate().SetPps(200)
	f1.Duration().FixedPackets().SetPackets(1000)
	e1 := f1.Packet().Add().Ethernet()
	e1.Src().SetValue(dutPort1Eth.Mac())
	e1.Dst().SetValue("00:00:00:00:00:00")
	v4 := f1.Packet().Add().Ipv4()
	v4.Src().SetValue("40.40.40.1")
	v4.Dst().Increment().SetStart("50.50.50.1").SetStep("0.0.0.1").SetCount(5)

	f2 := config.Flows().Add().SetName("f2")
	f2.Metrics().SetEnable(true)
	f2.TxRx().Device().
		SetTxNames([]string{dutPort1Bgp6PeerRoutes.Name()}).
		SetRxNames([]string{dutPort2Bgp6PeerRoutes.Name()})
	f2.Size().SetFixed(512)
	f2.Rate().SetPps(200)
	f2.Duration().FixedPackets().SetPackets(2000)
	e2 := f2.Packet().Add().Ethernet()
	e2.Src().SetValue(dutPort1Eth.Mac())
	e2.Dst().SetValue("00:00:00:00:00:00")
	v6 := f2.Packet().Add().Ipv6()
	v6.Src().SetValue("0:40:40:40::1")
	v6.Dst().Increment().SetStart("0:50:50:50::1").SetStep("::1").SetCount(5)

	f3 := config.Flows().Add().SetName("f3")
	f3.Metrics().SetEnable(true)
	f3.TxRx().Device().
		SetTxNames([]string{dutPort1Bgp6PeerRoutes.Name()}).
		SetRxNames([]string{dutPort2Bgp6PeerRoutes.Name()})
	f3.Size().SetFixed(512)
	f3.Rate().SetPps(200)
	f3.Duration().FixedPackets().SetPackets(3000)
	e3 := f2.Packet().Add().Ethernet()
	e3.Src().SetValue(dutPort1Eth.Mac())
	e3.Dst().SetValue("00:00:00:00:00:00")
	v62 := f2.Packet().Add().Ipv6()
	v62.Src().SetValue("0:40:40:40::1")
	v62.Dst().Increment().SetStart("0:50:50:50::1").SetStep("::1").SetCount(5)

	expected := helpers.ExpectedState{
		Bgp4: map[string]helpers.ExpectedBgpMetrics{
			dutPort1Bgp4Peer.Name(): {Advertised: 5, Received: 5},
			dutPort2Bgp4Peer.Name(): {Advertised: 5, Received: 5},
		},
		Bgp6: map[string]helpers.ExpectedBgpMetrics{
			dutPort1Bgp6Peer.Name(): {Advertised: 5, Received: 5},
			dutPort2Bgp6Peer.Name(): {Advertised: 5, Received: 5},
		},
	}

	return config, expected
}
