/*
Copyright 2021 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package rt_1_2_bgp_route_installation_test implements TE-1.2 from POPGate:
// Vendor Device Test Plan. go/wbb:vendor-testplan
//
// * Establish BGP sessions between:
//   * ATE port-1 and DUT port-1
// * For IPv4 and IPv6 routes
//   * Advertise prefixes from ATE port-1, observe received prefixes at ATE port-2.
// * TODO(b/184063580): Specify table based policy configuration to cover:
//   * Default accept for policies.
//   * TODO(aarunca): Default deny for policies.
//   * Explicitly specifying local preference.
//   * Explicitly specifying MED value.
//   * Explicitly prepending AS for advertisement with a specified AS number.
// * Validate that traffic can be forwarded to all installed routes between ATE port-1 and
//   ATE port-2, validate that flows between all denied routes cannot be forwarded.
// * Validate that traffic is not forwarded to withdrawn routes between ATE port-1 and ATE port-2.

package tests

import (
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"tests/tests/helpers"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra"
	oc "github.com/openconfig/ondatra/telemetry"
	"github.com/openconfig/ygot/ygot"
)

// func TestMain(m *testing.M) {
// 	ondatra.RunTests(m)
// }

// The testbed consists of ate:port1 -> dut:port1 and
// dut:port2 -> ate:port2.  The first pair is called the "source"
// pair, and the second the "destination" pair.
//
//   * Source: ate:port1 -> dut:port1 subnet 192.0.2.0/30 2001:db8::0/126
//   * Destination: dut:port2 -> ate:port2 subnet 192.0.2.4/30 2001:db8::4/126
//
// Note that the first (.0, .3) and last (.4, .7) IPv4 addresses are
// reserved from the subnet for broadcast, so a /30 leaves exactly 2
// usable addresses. This does not apply to IPv6 which allows /127
// for point to point links, but we use /126 so the numbering is
// consistent with IPv4.
//
// A traffic flow is configured from ate:port1 as the source interface
// and ate:port2 as the destination interface. Then 255 BGP routes 203.0.113.[1-254]/32
// are adverstised from port2 and traffic is sent originating from port1 to all
// these advertised routes. The traffic will pass only if the DUT installs the
// prefixes successfully in the routing table via BGP.Successful transmission of
// traffic will ensure BGP routes are properly installed on the DUT and programmed.
// Similarly, Traffic is sent for IPv6 destinations.

const (
	trafficDuration        = 10 * time.Second
	statsInterval          = 2 * time.Second
	trafficRate            = 100
	ipv4SrcTraffic         = "192.0.2.2"
	ipv6SrcTraffic         = "2001:db8::192:0:2:2"
	ipv4DstTrafficStart    = "203.0.113.1"
	ipv4DstTrafficEnd      = "203.0.113.254"
	ipv6DstTrafficStart    = "2001:db8::203:0:113:1"
	ipv6DstTrafficEnd      = "2001:db8::203:0:113:fe"
	advertisedRoutesv4CIDR = "203.0.113.1/32"
	advertisedRoutesv6CIDR = "2001:db8::203:0:113:1/128"
	routeCount             = 254
	dutAS                  = 64500
	ateAS                  = 64501
	plenIPv4               = 30
	plenIPv6               = 126
	tolerance              = 50
)

var (
	dutSrc = helpers.Attributes{
		Desc:    "DUT to ATE source",
		IPv4:    "192.0.2.1",
		IPv6:    "2001:db8::192:0:2:1",
		IPv4Len: plenIPv4,
		IPv6Len: plenIPv6,
	}
	ateSrc = helpers.Attributes{
		Name:    "ateSrc",
		IPv4:    "192.0.2.2",
		MAC:     "00:00:01:01:01:01",
		IPv6:    "2001:db8::192:0:2:2",
		IPv4Len: plenIPv4,
		IPv6Len: plenIPv6,
	}
	dutDst = helpers.Attributes{
		Desc:    "DUT to ATE destination",
		IPv4:    "192.0.2.5",
		IPv6:    "2001:db8::192:0:2:5",
		IPv4Len: plenIPv4,
		IPv6Len: plenIPv6,
	}
	ateDst = helpers.Attributes{
		Name:    "atedst",
		IPv4:    "192.0.2.6",
		MAC:     "00:00:02:01:01:01",
		IPv6:    "2001:db8::192:0:2:6",
		IPv4Len: plenIPv4,
		IPv6Len: plenIPv6,
	}
)

type bgpNeighbor struct {
	as         uint32
	neighborip string
	isV4       bool
}

func configureDUT(t *testing.T, dut *ondatra.DUTDevice) {
	// configureDUT configures all the interfaces on the DUT.
	dc := dut.Config()

	i1 := dutSrc.NewInterface(dut.Port(t, "port1").Name())
	dc.Interface(i1.GetName()).Replace(t, i1)

	i2 := dutDst.NewInterface(dut.Port(t, "port2").Name())
	dc.Interface(i2.GetName()).Replace(t, i2)
}

func verifyPortsUp(t *testing.T, dev *ondatra.Device) {
	t.Helper()
	for _, p := range dev.Ports() {
		status := dev.Telemetry().Interface(p.Name()).OperStatus().Get(t)
		if want := oc.Interface_OperStatus_UP; status != want {
			t.Errorf("%s Status: got %v, want %v", p, status, want)
		}
	}
}

func bgpAppendNbr(as uint32, nbrs []*bgpNeighbor) *oc.NetworkInstance_Protocol_Bgp {
	bgp := &oc.NetworkInstance_Protocol_Bgp{}
	g := bgp.GetOrCreateGlobal()
	g.As = ygot.Uint32(as)
	for _, nbr := range nbrs {
		if nbr.isV4 {
			nv4 := bgp.GetOrCreateNeighbor(nbr.neighborip)
			nv4.PeerAs = ygot.Uint32(nbr.as)
			nv4.Enabled = ygot.Bool(true)
			nv4.GetOrCreateAfiSafi(oc.BgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST).Enabled = ygot.Bool(true)
		} else {
			nv6 := bgp.GetOrCreateNeighbor(nbr.neighborip)
			nv6.PeerAs = ygot.Uint32(nbr.as)
			nv6.Enabled = ygot.Bool(true)
			nv6.GetOrCreateAfiSafi(oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST).Enabled = ygot.Bool(true)
		}
	}
	return bgp
}

func buildNbrList(sameAs bool) []*bgpNeighbor {
	var asN uint32
	if !sameAs {
		asN = ateAS
	} else {
		asN = dutAS
	}
	nbr1v4 := &bgpNeighbor{as: asN, neighborip: ateSrc.IPv4, isV4: true}
	nbr1v6 := &bgpNeighbor{as: asN, neighborip: ateSrc.IPv6, isV4: false}
	nbr2v4 := &bgpNeighbor{as: asN, neighborip: ateDst.IPv4, isV4: true}
	nbr2v6 := &bgpNeighbor{as: asN, neighborip: ateDst.IPv6, isV4: false}
	return []*bgpNeighbor{nbr1v4, nbr2v4, nbr1v6, nbr2v6}
}

func checkBgpParameters(t *testing.T, dut *ondatra.DUTDevice) {
	ifName := dut.Port(t, "port1").Name()
	lastFlapTime := dut.Telemetry().Interface(ifName).LastChange().Get(t)
	t.Logf("Verifying BGP state")
	statePath := dut.Telemetry().NetworkInstance("default").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	nbrPath := statePath.Neighbor(ateSrc.IPv4)
	nbrPathv6 := statePath.Neighbor(ateSrc.IPv6)
	nbr := statePath.Get(t).GetNeighbor(ateSrc.IPv4)

	// Get BGP adjacency state
	t.Logf("Waiting for BGP neighbor to establish...")
	_, ok := nbrPath.SessionState().Watch(t, time.Minute, func(val *oc.QualifiedE_Bgp_Neighbor_SessionState) bool {
		return val.IsPresent() && val.Val(t) == oc.Bgp_Neighbor_SessionState_ESTABLISHED
	}).Await(t)
	if !ok {
		helpers.LogYgot(t, "BGP reported state", nbrPath, nbrPath.Get(t))
		t.Fatal("No BGP neighbor formed...")
	}

	status := nbrPath.SessionState().Get(t)
	t.Logf("BGP adjacency for %s: %s", ateSrc.IPv4, status)
	if want := oc.Bgp_Neighbor_SessionState_ESTABLISHED; status != want {
		t.Errorf("Get(BGP peer %s status): got %d, want %d", ateSrc.IPv4, status, want)
	}
	// Check last established timestamp
	lastEstTime := nbrPath.Get(t).GetLastEstablished()
	lastEstTimev6 := nbrPathv6.Get(t).GetLastEstablished()
	if lastEstTime < lastFlapTime {
		t.Logf("Bad last-established BGPv4 timestamp: got %v, want <= %v", lastEstTime, lastFlapTime)
	}
	if lastEstTimev6 < lastFlapTime {
		t.Logf("Bad last-established BGPv6 timestamp: got %v, want <= %v", lastEstTimev6, lastFlapTime)
	}
	// Check BGP Transitions
	estTrans := nbr.GetEstablishedTransitions()
	t.Logf("Got established transitions: %d", estTrans)
	if estTrans != 1 {
		t.Errorf("Wrong established-transitions: got %v, want 1", estTrans)
	}
	// Check BGP neighbor address from telemetry
	addr := nbrPath.Get(t).GetNeighborAddress()
	addrv6 := nbrPathv6.Get(t).GetNeighborAddress()
	t.Logf("Got neighbor address: %s", addr)
	t.Logf("Got neighbor address: %s", addrv6)
	if addrv6 != ateSrc.IPv6 {
		t.Errorf("Bgp neighbor address: got %v, want %v", addrv6, ateSrc.IPv6)
	}
	// Check BGP neighbor address from telemetry
	peerAS := nbrPath.Get(t).GetPeerAs()
	t.Logf("Got neighbor AS: %d", peerAS)
	if peerAS != ateAS {
		t.Errorf("Bgp peerAs: got %v, want %v", peerAS, ateAS)
	}
	// Check BGP neighbor is enabled
	if !nbrPath.Get(t).GetEnabled() {
		t.Errorf("Expected neighbor %v to be enabled", ateSrc.IPv4)
	}
}

func configureATE(t *testing.T, otg *ondatra.OTG) (gosnappi.Config, helpers.ExpectedState) {

	config := otg.NewConfig()
	srcPort := config.Ports().Add().SetName("port1")
	dstPort := config.Ports().Add().SetName("port2")

	srcDev := config.Devices().Add().SetName(ateSrc.Name)
	srcEth := srcDev.Ethernets().Add().
		SetName(ateSrc.Name + ".eth").
		SetPortName(srcPort.Name()).
		SetMac(ateSrc.MAC)
	srcIpv4 := srcEth.Ipv4Addresses().Add().
		SetName(ateSrc.Name + ".ipv4").
		SetAddress(ateSrc.IPv4).
		SetGateway(dutSrc.IPv4).
		SetPrefix(int32(ateSrc.IPv4Len))
	srcIpv6 := srcEth.Ipv6Addresses().Add().
		SetName(ateSrc.Name + ".ipv6").
		SetAddress(ateSrc.IPv6).
		SetGateway(dutSrc.IPv6).
		SetPrefix(int32(ateSrc.IPv6Len))

	dstDev := config.Devices().Add().SetName(ateDst.Name)
	dstEth := dstDev.Ethernets().Add().
		SetName(ateDst.Name + ".eth").
		SetPortName(dstPort.Name()).
		SetMac(ateDst.MAC)
	dstIpv4 := dstEth.Ipv4Addresses().Add().
		SetName(ateDst.Name + ".ipv4").
		SetAddress(ateDst.IPv4).
		SetGateway(dutDst.IPv4).
		SetPrefix(int32(ateDst.IPv4Len))
	dstIpv6 := dstEth.Ipv6Addresses().Add().
		SetName(ateDst.Name + ".ipv6").
		SetAddress(ateDst.IPv6).
		SetGateway(dutDst.IPv6).
		SetPrefix(int32(ateDst.IPv6Len))

	srcBgp4Name := ateSrc.Name + ".bgp4.peer"
	srcBgp6Name := ateSrc.Name + ".bgp6.peer"
	srcBgp := srcDev.Bgp().
		SetRouterId(srcIpv4.Address())
	srcBgp4Peer := srcBgp.Ipv4Interfaces().Add().
		SetIpv4Name(srcIpv4.Name()).
		Peers().Add().
		SetName(srcBgp4Name).
		SetPeerAddress(srcIpv4.Gateway()).
		SetAsNumber(ateAS).
		SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	srcBgp6Peer := srcBgp.Ipv6Interfaces().Add().
		SetIpv6Name(srcIpv6.Name()).
		Peers().Add().
		SetName(srcBgp6Name).
		SetPeerAddress(srcIpv6.Gateway()).
		SetAsNumber(ateAS).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP)

	dstBgp4Name := ateDst.Name + ".bgp4.peer"
	dstBgp6Name := ateDst.Name + ".bgp6.peer"
	dstBgp := dstDev.Bgp().
		SetRouterId(dstIpv4.Address())
	dstBgp4Peer := dstBgp.Ipv4Interfaces().Add().
		SetIpv4Name(dstIpv4.Name()).
		Peers().Add().
		SetName(dstBgp4Name).
		SetPeerAddress(dstIpv4.Gateway()).
		SetAsNumber(ateAS).
		SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	dstBgp6Peer := dstBgp.Ipv6Interfaces().Add().
		SetIpv6Name(dstIpv6.Name()).
		Peers().Add().
		SetName(dstBgp6Name).
		SetPeerAddress(dstIpv6.Gateway()).
		SetAsNumber(ateAS).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP)

	prefixInt4, _ := strconv.Atoi(strings.Split(advertisedRoutesv4CIDR, "/")[1])
	prefixInt6, _ := strconv.Atoi(strings.Split(advertisedRoutesv6CIDR, "/")[1])
	dstBgp4PeerRoutes := dstBgp4Peer.V4Routes().Add().
		SetName(dstBgp4Name + ".rr4").
		SetNextHopIpv4Address(dstIpv4.Address()).
		SetNextHopAddressType(gosnappi.BgpV4RouteRangeNextHopAddressType.IPV4).
		SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
	dstBgp4PeerRoutes.Addresses().Add().
		SetAddress(strings.Split(advertisedRoutesv4CIDR, "/")[0]).
		SetPrefix(int32(prefixInt4)).
		SetCount(routeCount)
	dstBgp6PeerRoutes := dstBgp6Peer.V6Routes().Add().
		SetName(dstBgp6Name + ".rr6").
		SetNextHopIpv6Address(dstIpv6.Address()).
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
	dstBgp6PeerRoutes.Addresses().Add().
		SetAddress(strings.Split(advertisedRoutesv6CIDR, "/")[0]).
		SetPrefix(int32(prefixInt6)).
		SetCount(routeCount)

	flowipv4 := config.Flows().Add().SetName("bgpv4RoutesFlow")
	flowipv4.Metrics().SetEnable(true)
	flowipv4.TxRx().Device().
		SetTxNames([]string{srcIpv4.Name()}).
		SetRxNames([]string{dstBgp4PeerRoutes.Name()})
	flowipv4.Size().SetFixed(512)
	flowipv4.Rate().SetPps(trafficRate)
	flowipv4.Duration().SetChoice("continuous")
	e1 := flowipv4.Packet().Add().Ethernet()
	e1.Src().SetValue(srcEth.Mac())
	v4 := flowipv4.Packet().Add().Ipv4()
	v4.Src().SetValue(srcIpv4.Address())
	v4.Dst().Increment().SetStart(strings.Split(advertisedRoutesv4CIDR, "/")[0]).SetCount(routeCount)

	flowipv6 := config.Flows().Add().SetName("bgpv6RoutesFlow")
	flowipv6.Metrics().SetEnable(true)
	flowipv6.TxRx().Device().
		SetTxNames([]string{srcIpv6.Name()}).
		SetRxNames([]string{dstBgp6PeerRoutes.Name()})
	flowipv6.Size().SetFixed(512)
	flowipv6.Rate().SetPps(trafficRate)
	flowipv6.Duration().SetChoice("continuous")
	e2 := flowipv6.Packet().Add().Ethernet()
	e2.Src().SetValue(srcEth.Mac())
	v6 := flowipv6.Packet().Add().Ipv6()
	v6.Src().SetValue(srcIpv6.Address())
	v6.Dst().Increment().SetStart(strings.Split(advertisedRoutesv6CIDR, "/")[0]).SetCount(routeCount)

	expected := helpers.ExpectedState{
		Bgp4: map[string]helpers.ExpectedBgpMetrics{
			srcBgp4Peer.Name(): {Advertised: 0, Received: routeCount},
			dstBgp4Peer.Name(): {Advertised: routeCount, Received: 0},
		},
		Bgp6: map[string]helpers.ExpectedBgpMetrics{
			srcBgp6Peer.Name(): {Advertised: 0, Received: routeCount},
			dstBgp6Peer.Name(): {Advertised: routeCount, Received: 0},
		},
		Flow: map[string]helpers.ExpectedFlowMetrics{
			flowipv4.Name(): {FramesRx: 0, FramesRxRate: 0},
			flowipv6.Name(): {FramesRx: 0, FramesRxRate: 0},
		},
	}

	return config, expected
}

func verifyNoPacketLoss(t *testing.T, gnmiClient *helpers.GnmiClient) {
	fMetrics, err := gnmiClient.GetFlowMetrics([]string{})
	if err != nil {
		t.Fatal("Error while getting the flow metrics")
	}

	helpers.PrintMetricsTable(&helpers.MetricsTableOpts{
		ClearPrevious: false,
		FlowMetrics:   fMetrics,
	})

	pMetrics, err := gnmiClient.GetPortMetrics([]string{})
	if err != nil {
		t.Fatal("Error while getting the port metrics")
	}

	helpers.PrintMetricsTable(&helpers.MetricsTableOpts{
		ClearPrevious: false,
		PortMetrics:   pMetrics,
	})

	for _, f := range fMetrics.Items() {
		if f.FramesTx() != f.FramesRx() && f.FramesTx() > 0 {
			t.Errorf("Failed: Packet Loss detected")
		} else {
			t.Logf("Success: No packets loss on flow %s", f.Name())
		}

	}
}

func verifyPacketLoss(t *testing.T, gnmiClient *helpers.GnmiClient) {
	fMetrics, err := gnmiClient.GetFlowMetrics([]string{})
	if err != nil {
		t.Fatal("Error while getting the flow stats")
	}

	helpers.PrintMetricsTable(&helpers.MetricsTableOpts{
		ClearPrevious: false,
		FlowMetrics:   fMetrics,
	})

	// Once the Flow Tx Frames becomes available the next lines could be removed
	pMetrics, err := gnmiClient.GetPortMetrics([]string{})
	if err != nil {
		t.Fatal("Error while getting the port stats")
	}

	helpers.PrintMetricsTable(&helpers.MetricsTableOpts{
		ClearPrevious: true,
		PortMetrics:   pMetrics,
	})

	for _, f := range fMetrics.Items() {
		if f.FramesRx() > 0 && f.FramesTx() > 0 {
			t.Errorf("Failed: Flow packets unexpectedly received")
		} else {
			t.Logf("Success: No flow packets received on flow %s", f.Name())
		}
	}
}

func sendTraffic(t *testing.T, otg *ondatra.OTG, gnmiClient *helpers.GnmiClient) {
	t.Logf("Starting traffic")
	otg.StartTraffic(t)
	err := gnmiClient.WatchFlowMetrics(&helpers.WaitForOpts{Interval: statsInterval, Timeout: trafficDuration})
	if err != nil {
		log.Println(err)
	}
	t.Logf("Stop traffic")
	otg.StopTraffic(t)
}

func rt_1_2_UnsetDUT(t *testing.T, dut *ondatra.DUTDevice) {
	// t.Logf("Start Unsetting DUT Config")
	// helpers.ConfigDUTs(map[string]string{"arista1": "../resources/dutconfig/bgp_route_install/unset_dut.txt"})

	t.Logf("Start Unsetting DUT Interface Config")
	dc := dut.Config()

	i1 := helpers.RemoveInterface(dut.Port(t, "port1").Name())
	dc.Interface(i1.GetName()).Replace(t, i1)

	i2 := helpers.RemoveInterface(dut.Port(t, "port2").Name())
	dc.Interface(i2.GetName()).Replace(t, i2)

	t.Logf("Start Removing BGP config")
	dutConfPath := dut.Config().NetworkInstance("default").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	helpers.LogYgot(t, "DUT BGP Config before", dutConfPath, dutConfPath.Get(t))
	dutConfPath.Replace(t, nil)

}
func Test_rt_1_2(t *testing.T) {
	// DUT configurations.
	t.Logf("Start DUT config load:")
	dut := ondatra.DUT(t, "dut")

	// Configure interface on the DUT
	t.Logf("Start DUT interface Config")
	configureDUT(t, dut)
	helpers.ConfigDUTs(map[string]string{"arista": "../resources/dutconfig/rt_1_2_bgp_route_installation/set_dut_interface.txt"})

	// Configure BGP+Neighbors on the DUT
	t.Logf("Start DUT BGP Config")
	dutConfPath := dut.Config().NetworkInstance("default").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	helpers.LogYgot(t, "DUT BGP Config before", dutConfPath, dutConfPath.Get(t))
	dutConfPath.Replace(t, nil)
	nbrList := buildNbrList(false)
	dutConf := bgpAppendNbr(dutAS, nbrList)
	dutConfPath.Replace(t, dutConf)
	defer rt_1_2_UnsetDUT(t, dut)

	// ATE Configuration.
	t.Logf("Start ATE Config")
	ate := ondatra.ATE(t, "ate")
	otg := ate.OTG(t)

	defer helpers.CleanupTest(t, ate, otg, true, false)

	config, expected := configureATE(t, otg)

	otg.PushConfig(t, ate, config)
	t.Logf("Start ATE Protocols")
	otg.StartProtocols(t)

	gnmiClient, err := helpers.NewGnmiClient(otg.NewGnmiQuery(t), config)
	if err != nil {
		t.Fatal(err)
	}
	defer gnmiClient.Close()

	// Verify Port Status
	t.Logf("Verifying port status")
	verifyPortsUp(t, dut.Device)

	t.Logf("Check BGP parameters")
	checkBgpParameters(t, dut)
	// newCheckBgpParameters(t, dut)

	t.Logf("Check BGP sessions on OTG")
	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.AllBgp4SessionUp(expected) }, nil)
	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.AllBgp6SessionUp(expected) }, nil)

	// Sending ATE Traffic
	t.Logf("Sending traffic")
	sendTraffic(t, otg, gnmiClient)

	// Verify Traffic Flows for packet loss
	t.Logf("Verifying no packet loss")
	verifyNoPacketLoss(t, gnmiClient)

	//Configure BGP with mismatching AS number
	t.Logf("Start DUT BGP Config with mismatching AS")
	dutConfPath = dut.Config().NetworkInstance("default").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	dutConfPath.Replace(t, nil)
	nbrList = buildNbrList(true)
	dutConf = bgpAppendNbr(dutAS, nbrList)
	dutConfPath.Replace(t, dutConf)

	// Sending ATE Traffic
	t.Logf("Sending traffic")
	sendTraffic(t, otg, gnmiClient)

	// Verify traffic fails as routes are withdrawn and 100% packet loss is seen.
	verifyPacketLoss(t, gnmiClient)

}
