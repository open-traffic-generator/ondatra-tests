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
)

// func TestMain(m *testing.M) {
// 	ondatra.RunTests(m)
// }

const (
	trDuration     = 10 * time.Second
	stInterval     = 2 * time.Second
	trRate         = 100
	adRoutesv4CIDR = "203.0.113.1/32"
	adRoutesv6CIDR = "2001:db8::203:0:113:1/128"
	roCount        = 254
	p1AS           = 64500
	p2AS           = 64501
	plIPv4         = 30
	plIPv6         = 126
)

var (
	ateP1 = helpers.Attributes{
		Name:    "ateP1",
		IPv4:    "192.0.2.2",
		MAC:     "00:00:01:01:01:01",
		IPv6:    "2001:db8::192:0:2:2",
		IPv4Len: plIPv4,
		IPv6Len: plIPv6,
	}
	ateP2 = helpers.Attributes{
		Name:    "atedst",
		IPv4:    "192.0.2.6",
		MAC:     "00:00:02:01:01:01",
		IPv6:    "2001:db8::192:0:2:6",
		IPv4Len: plIPv4,
		IPv6Len: plIPv6,
	}
)

func configureOTG(t *testing.T, otg *ondatra.OTGAPI) (gosnappi.Config, helpers.ExpectedState) {

	config := otg.NewConfig(t)
	srcPort := config.Ports().Add().SetName("ixia-c-port1")
	srcDev := config.Devices().Add().SetName(ateP1.Name)
	srcEth := srcDev.Ethernets().Add().
		SetName(ateP1.Name + ".eth").
		SetPortName(srcPort.Name()).
		SetMac(ateP1.MAC)
	srcIpv4 := srcEth.Ipv4Addresses().Add().
		SetName(ateP1.Name + ".ipv4").
		SetAddress(ateP1.IPv4).
		SetGateway(ateP2.IPv4).
		SetPrefix(int32(ateP1.IPv4Len))
	srcIpv6 := srcEth.Ipv6Addresses().Add().
		SetName(ateP1.Name + ".ipv6").
		SetAddress(ateP1.IPv6).
		SetGateway(ateP2.IPv6).
		SetPrefix(int32(ateP1.IPv6Len))

	dstPort := config.Ports().Add().SetName("ixia-c-port2")
	dstDev := config.Devices().Add().SetName(ateP2.Name)
	dstEth := dstDev.Ethernets().Add().
		SetName(ateP2.Name + ".eth").
		SetPortName(dstPort.Name()).
		SetMac(ateP2.MAC)
	dstIpv4 := dstEth.Ipv4Addresses().Add().
		SetName(ateP2.Name + ".ipv4").
		SetAddress(ateP2.IPv4).
		SetGateway(ateP1.IPv4).
		SetPrefix(int32(ateP2.IPv4Len))
	dstIpv6 := dstEth.Ipv6Addresses().Add().
		SetName(ateP2.Name + ".ipv6").
		SetAddress(ateP2.IPv6).
		SetGateway(ateP1.IPv6).
		SetPrefix(int32(ateP2.IPv6Len))

	srcBgp4Name := ateP1.Name + ".bgp4.peer"
	srcBgp6Name := ateP1.Name + ".bgp6.peer"
	srcBgp := srcDev.Bgp().
		SetRouterId(srcIpv4.Address())
	srcBgp4Peer := srcBgp.Ipv4Interfaces().Add().
		SetIpv4Name(srcIpv4.Name()).
		Peers().Add().
		SetName(srcBgp4Name).
		SetPeerAddress(srcIpv4.Gateway()).
		SetAsNumber(p1AS).
		SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	srcBgp6Peer := srcBgp.Ipv6Interfaces().Add().
		SetIpv6Name(srcIpv6.Name()).
		Peers().Add().
		SetName(srcBgp6Name).
		SetPeerAddress(srcIpv6.Gateway()).
		SetAsNumber(p1AS).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP)

	dstBgp4Name := ateP2.Name + ".bgp4.peer"
	dstBgp6Name := ateP2.Name + ".bgp6.peer"
	dstBgp := dstDev.Bgp().
		SetRouterId(dstIpv4.Address())
	dstBgp4Peer := dstBgp.Ipv4Interfaces().Add().
		SetIpv4Name(dstIpv4.Name()).
		Peers().Add().
		SetName(dstBgp4Name).
		SetPeerAddress(dstIpv4.Gateway()).
		SetAsNumber(p2AS).
		SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	dstBgp6Peer := dstBgp.Ipv6Interfaces().Add().
		SetIpv6Name(dstIpv6.Name()).
		Peers().Add().
		SetName(dstBgp6Name).
		SetPeerAddress(dstIpv6.Gateway()).
		SetAsNumber(p2AS).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP)

	prefixInt4, _ := strconv.Atoi(strings.Split(adRoutesv4CIDR, "/")[1])
	prefixInt6, _ := strconv.Atoi(strings.Split(adRoutesv6CIDR, "/")[1])
	dstBgp4PeerRoutes := dstBgp4Peer.V4Routes().Add().
		SetName(dstBgp4Name + ".rr4").
		SetNextHopIpv4Address(dstIpv4.Address()).
		SetNextHopAddressType(gosnappi.BgpV4RouteRangeNextHopAddressType.IPV4).
		SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
	dstBgp4PeerRoutes.Addresses().Add().
		SetAddress(strings.Split(adRoutesv4CIDR, "/")[0]).
		SetPrefix(int32(prefixInt4)).
		SetCount(roCount)
	dstBgp6PeerRoutes := dstBgp6Peer.V6Routes().Add().
		SetName(dstBgp6Name + ".rr6").
		SetNextHopIpv6Address(dstIpv6.Address()).
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
	dstBgp6PeerRoutes.Addresses().Add().
		SetAddress(strings.Split(adRoutesv6CIDR, "/")[0]).
		SetPrefix(int32(prefixInt6)).
		SetCount(roCount)

	t.Logf("Pushing config to OTG and starting protocols...")
	flowipv4 := config.Flows().Add().SetName("bgpv4RoutesFlow")
	flowipv4.Metrics().SetEnable(true)
	flowipv4.TxRx().Device().
		SetTxNames([]string{srcIpv4.Name()}).
		SetRxNames([]string{dstBgp4PeerRoutes.Name()})
	flowipv4.Size().SetFixed(512)
	flowipv4.Rate().SetPps(trRate)
	// This should be used when Tx packets stat will become available
	flowipv4.Duration().SetChoice("continuous")
	e1 := flowipv4.Packet().Add().Ethernet()
	e1.Src().SetValue(srcEth.Mac())
	v4 := flowipv4.Packet().Add().Ipv4()
	v4.Src().SetValue(srcIpv4.Address())
	v4.Dst().Increment().SetStart(strings.Split(adRoutesv4CIDR, "/")[0]).SetCount(roCount)

	flowipv6 := config.Flows().Add().SetName("bgpv6RoutesFlow")
	flowipv6.Metrics().SetEnable(true)
	flowipv6.TxRx().Device().
		SetTxNames([]string{srcIpv6.Name()}).
		SetRxNames([]string{dstBgp6PeerRoutes.Name()})
	flowipv6.Size().SetFixed(512)
	flowipv6.Rate().SetPps(trRate)
	// This should be used when Tx packets stat will become available
	flowipv6.Duration().SetChoice("continuous")
	// flowipv6.Duration().SetChoice("fixed_packets")
	// flowipv6.Duration().FixedPackets().SetPackets(packetsToBeSent)
	e2 := flowipv6.Packet().Add().Ethernet()
	e2.Src().SetValue(srcEth.Mac())
	v6 := flowipv6.Packet().Add().Ipv6()
	v6.Src().SetValue(srcIpv6.Address())
	v6.Dst().Increment().SetStart(strings.Split(adRoutesv6CIDR, "/")[0]).SetCount(roCount)

	expected := helpers.ExpectedState{
		Bgp4: map[string]helpers.ExpectedBgpMetrics{
			srcBgp4Peer.Name(): {Advertised: 0, Received: roCount},
			dstBgp4Peer.Name(): {Advertised: roCount, Received: 0},
		},
		Bgp6: map[string]helpers.ExpectedBgpMetrics{
			srcBgp6Peer.Name(): {Advertised: 0, Received: roCount},
			dstBgp6Peer.Name(): {Advertised: roCount, Received: 0},
		},
		Flow: map[string]helpers.ExpectedFlowMetrics{
			flowipv4.Name(): {FramesRx: 0, FramesRxRate: 0},
			flowipv6.Name(): {FramesRx: 0, FramesRxRate: 0},
		},
	}

	return config, expected
}

func b2bVerifyNoPacketLoss(t *testing.T, gnmiClient *helpers.GnmiClient) {
	fMetrics, err := gnmiClient.GetFlowMetrics([]string{})
	if err != nil {
		t.Fatal("Error while getting the flow metrics")
	}

	helpers.PrintMetricsTable(&helpers.MetricsTableOpts{
		ClearPrevious: false,
		FlowMetrics:   fMetrics,
	})

	// Once the Flow Tx Frames becomes available the next lines could be removed and the original condition could be restored
	pMetrics, err := gnmiClient.GetPortMetrics([]string{})
	if err != nil {
		t.Fatal("Error while getting the port metrics")
	}

	helpers.PrintMetricsTable(&helpers.MetricsTableOpts{
		ClearPrevious: false,
		PortMetrics:   pMetrics,
	})

	portFramesTx := pMetrics.Items()[0].FramesTx()
	conditionalFrames := float32(portFramesTx/2) - 0.001*float32(portFramesTx)
	for _, f := range fMetrics.Items() {
		if float32(f.FramesRx()) < conditionalFrames && f.FramesRx() > 0 {
			t.Errorf("Packet Loss detected")
		}
		// if f.FramesTx() != f.FramesRx() && f.FramesTx() > 0 {
		// 	t.Errorf("Packet Loss detected")
		// }
	}
}

func b2bVerifyPacketLoss(t *testing.T, gnmiClient *helpers.GnmiClient) {
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

	portFramesTx := pMetrics.Items()[0].FramesTx()
	for _, f := range fMetrics.Items() {
		if f.FramesRx() > 0 && portFramesTx > 0 {
			t.Errorf("Flow packets were unexpectedly received")
		}
	}
}

func b2bSendTraffic(t *testing.T, otg *ondatra.OTGAPI, gnmiClient *helpers.GnmiClient) {
	t.Logf("Starting traffic")
	otg.StartTraffic(t)
	err := gnmiClient.WatchFlowMetrics(&helpers.WaitForOpts{Interval: stInterval, Timeout: trDuration})
	if err != nil {
		log.Println(err)
	}
	t.Logf("Stop traffic")
	otg.StopTraffic(t)
}

func Test_b2b(t *testing.T) {

	// OTG Configuration.
	t.Logf("Start ATE Config")
	ate1 := ondatra.ATE(t, "ate1")
	_ = ondatra.ATE(t, "ate2")
	otg := ate1.OTGAPI
	defer helpers.CleanupTest(otg, t, true)

	config, expected := configureOTG(t, otg)

	otg.PushConfig(t, config)
	t.Logf("Start OTG Protocols")
	otg.StartProtocols(t)

	gnmiClient, err := helpers.NewGnmiClient(otg.NewGnmiQuery(t), config)
	if err != nil {
		t.Fatal(err)
	}
	defer gnmiClient.Close()

	t.Logf("Check BGP sessions on OTG")
	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.AllBgp4SessionUp(expected) }, nil)
	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.AllBgp6SessionUp(expected) }, nil)

	// Sending ATE Traffic
	t.Logf("Sending traffic")
	b2bSendTraffic(t, otg, gnmiClient)

	// Verify Traffic Flows for packet loss
	t.Logf("Verifying no packet loss")
	b2bVerifyNoPacketLoss(t, gnmiClient)

}
