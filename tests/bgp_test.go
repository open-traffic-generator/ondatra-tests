package tests

import (
	"testing"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra"
)

func TestGoSnappiK8s_001(t *testing.T) {
	t.Log("TestGoSnappiK8s_001 - START ...")
	otg := ondatra.OTGs(t)

	defer otg.NewConfig(t)
	defer otg.StopProtocols(t)

	config := PacketForwardBgpv6Config(t, otg)
	otg.PushConfig(t, config)

	otg.StartProtocols(t)

	gnmiClient, err := NewGnmiClient(otg.NewGnmiQuery(t), config)
	if err != nil {
		t.Fatal(err)
	}

	// WaitFor(t,
	// 	func() (bool, error) { return gnmiClient.AllBgp6SessionUp(config) }, nil,
	// )

	otg.StartTraffic(t)

	WaitFor(t,
		func() (bool, error) { return gnmiClient.PortAndFlowMetricsOk(config) }, nil,
	)

	t.Log("TestGoSnappiK8s_001 - END ...")
}

func PacketForwardBgpv6Config(t *testing.T, otg *ondatra.OTG) gosnappi.Config {
	config := otg.NewConfig(t)

	// add ports
	p1 := config.Ports().Add().SetName("ixia-c-port1")
	p2 := config.Ports().Add().SetName("ixia-c-port2")
	p3 := config.Ports().Add().SetName("ixia-c-port3")

	// add devices
	d1 := config.Devices().Add().SetName("d1")
	d2 := config.Devices().Add().SetName("d2")
	d3 := config.Devices().Add().SetName("d3")

	// add flows and common properties
	for i := 1; i <= 4; i++ {
		flow := config.Flows().Add()
		flow.Metrics().SetEnable(true)
		flow.Duration().FixedPackets().SetPackets(1000)
		flow.Rate().SetPps(500)
	}

	// add protocol stacks for device d1
	d1Eth1 := d1.Ethernets().
		Add().
		SetName("d1Eth").
		SetPortName(p1.Name()).
		SetMac("00:00:01:01:01:01").
		SetMtu(1500)

	d1Eth1.
		Ipv6Addresses().
		Add().
		SetName("p1d1ipv6").
		SetAddress("0:1:1:1::1").
		SetGateway("0:1:1:1::2").
		SetPrefix(64)

	d1Bgp := d1.Bgp().
		SetRouterId("1.1.1.1")

	d1BgpIpv6Interface1 := d1Bgp.
		Ipv6Interfaces().Add().
		SetIpv6Name("p1d1ipv6")

	d1BgpIpv6Interface1Peer1 := d1BgpIpv6Interface1.
		Peers().
		Add().
		SetAsNumber(1111).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP).
		SetPeerAddress("0:1:1:1::2").
		SetName("BGPv6 Peer 1")

	d1BgpIpv6Interface1Peer1V6Route1 := d1BgpIpv6Interface1Peer1.
		V6Routes().
		Add().
		SetNextHopIpv6Address("0:1:1:1::1").
		SetName("p1d1peer1rrv6").
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)

	d1BgpIpv6Interface1Peer1V6Route1.Addresses().Add().
		SetAddress("0:10:10:10::0").
		SetPrefix(64).
		SetCount(2).
		SetStep(2)

	d1BgpIpv6Interface1Peer1V6Route1.Advanced().
		SetMultiExitDiscriminator(50).
		SetOrigin(gosnappi.BgpRouteAdvancedOrigin.EGP)

	d1BgpIpv6Interface1Peer1V6Route1.Communities().Add().
		SetAsNumber(1).
		SetAsCustom(2).
		SetType(gosnappi.BgpCommunityType.MANUAL_AS_NUMBER)

	d1BgpIpv6Interface1Peer1V6Route1AsPath := d1BgpIpv6Interface1Peer1V6Route1.AsPath().
		SetAsSetMode(gosnappi.BgpAsPathAsSetMode.INCLUDE_AS_SET)

	d1BgpIpv6Interface1Peer1V6Route1AsPath.Segments().Add().
		SetAsNumbers([]int64{1112, 1113}).
		SetType(gosnappi.BgpAsPathSegmentType.AS_SEQ)

	// add protocol stacks for device d2
	d2Eth1 := d2.Ethernets().
		Add().
		SetName("d2Eth").
		SetPortName(p2.Name()).
		SetMac("00:00:02:02:02:02").
		SetMtu(1500)

	d2Eth1.
		Ipv6Addresses().
		Add().
		SetName("p2d1ipv6").
		SetAddress("0:2:2:2::2").
		SetGateway("0:2:2:2::1").
		SetPrefix(64)

	d2Bgp := d2.Bgp().
		SetRouterId("2.2.2.2")

	d2BgpIpv6Interface1 := d2Bgp.
		Ipv6Interfaces().Add().
		SetIpv6Name("p2d1ipv6")

	d2BgpIpv6Interface1Peer1 := d2BgpIpv6Interface1.
		Peers().
		Add().
		SetAsNumber(2222).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP).
		SetPeerAddress("0:2:2:2::1").
		SetName("BGPv6 Peer 2")

	d2BgpIpv6Interface1Peer1V6Route1 := d2BgpIpv6Interface1Peer1.
		V6Routes().
		Add().
		SetNextHopIpv6Address("0:2:2:2::2").
		SetName("p2d1peer1rrv6").
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)

	d2BgpIpv6Interface1Peer1V6Route1.Addresses().Add().
		SetAddress("0:20:20:20::0").
		SetPrefix(64).
		SetCount(2).
		SetStep(2)

	d2BgpIpv6Interface1Peer1V6Route1.Advanced().
		SetMultiExitDiscriminator(40).
		SetOrigin(gosnappi.BgpRouteAdvancedOrigin.EGP)

	d2BgpIpv6Interface1Peer1V6Route1.Communities().Add().
		SetAsNumber(100).
		SetAsCustom(2).
		SetType(gosnappi.BgpCommunityType.MANUAL_AS_NUMBER)

	d2BgpIpv6Interface1Peer1V6Route1AsPath := d2BgpIpv6Interface1Peer1V6Route1.AsPath().
		SetAsSetMode(gosnappi.BgpAsPathAsSetMode.INCLUDE_AS_SET)

	d2BgpIpv6Interface1Peer1V6Route1AsPath.Segments().Add().
		SetAsNumbers([]int64{2223, 2224, 2225}).
		SetType(gosnappi.BgpAsPathSegmentType.AS_SEQ)

	// add protocol stacks for device d3

	d3Eth1 := d3.Ethernets().
		Add().
		SetName("d3Eth").
		SetPortName(p3.Name()).
		SetMac("00:00:03:03:03:02").
		SetMtu(1500)

	d3Eth1.
		Ipv6Addresses().
		Add().
		SetName("p3d1ipv6").
		SetAddress("0:3:3:3::2").
		SetGateway("0:3:3:3::1").
		SetPrefix(64)

	d3Bgp := d3.Bgp().
		SetRouterId("3.3.3.2")

	d3BgpIpv6Interface1 := d3Bgp.
		Ipv6Interfaces().Add().
		SetIpv6Name("p3d1ipv6")

	d3BgpIpv6Interface1Peer1 := d3BgpIpv6Interface1.
		Peers().
		Add().
		SetAsNumber(3332).
		SetAsType(gosnappi.BgpV6PeerAsType.EBGP).
		SetPeerAddress("0:3:3:3::1").
		SetName("BGPv6 Peer 3")

	d3BgpIpv6Interface1Peer1V6Route1 := d3BgpIpv6Interface1Peer1.
		V6Routes().
		Add().
		SetNextHopIpv6Address("0:3:3:3::2").
		SetName("p3d1peer1rrv6").
		SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
		SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)

	d3BgpIpv6Interface1Peer1V6Route1.Addresses().Add().
		SetAddress("0:30:30:30::0").
		SetPrefix(64).
		SetCount(2).
		SetStep(2)

	d3BgpIpv6Interface1Peer1V6Route1.Advanced().
		SetMultiExitDiscriminator(33).
		SetOrigin(gosnappi.BgpRouteAdvancedOrigin.EGP)

	d3BgpIpv6Interface1Peer1V6Route1.Communities().Add().
		SetAsNumber(1).
		SetAsCustom(2).
		SetType(gosnappi.BgpCommunityType.MANUAL_AS_NUMBER)

	d3BgpIpv6Interface1Peer1V6Route1AsPath := d3BgpIpv6Interface1Peer1V6Route1.AsPath().
		SetAsSetMode(gosnappi.BgpAsPathAsSetMode.INCLUDE_AS_SET)

	d3BgpIpv6Interface1Peer1V6Route1AsPath.Segments().Add().
		SetAsNumbers([]int64{3333, 3334}).
		SetType(gosnappi.BgpAsPathSegmentType.AS_SEQ)

	// add endpoints and packet description flow 1
	f1 := config.Flows().Items()[0]
	f1.SetName(p1.Name() + " -> " + p2.Name()).
		TxRx().Device().
		SetTxNames([]string{d1BgpIpv6Interface1Peer1V6Route1.Name()}).
		SetRxNames([]string{d2BgpIpv6Interface1Peer1V6Route1.Name()})

	f1Eth := f1.Packet().Add().Ethernet()
	f1Eth.Src().SetValue(d1Eth1.Mac())
	f1Eth.Dst().SetValue("00:00:00:00:00:00")

	f1Ip := f1.Packet().Add().Ipv6()
	f1Ip.Src().SetValue("0:10:10:10::1")
	f1Ip.Dst().SetValue("0:20:20:20::1")

	// add endpoints and packet description flow 2
	f2 := config.Flows().Items()[1]
	f2.SetName(p1.Name() + " -> " + p3.Name()).
		TxRx().Device().
		SetTxNames([]string{d1BgpIpv6Interface1Peer1V6Route1.Name()}).
		SetRxNames([]string{d3BgpIpv6Interface1Peer1V6Route1.Name()})

	f2Eth := f2.Packet().Add().Ethernet()
	f2Eth.Src().SetValue(d1Eth1.Mac())
	f2Eth.Dst().SetValue("00:00:00:00:00:00")

	f2Ip := f2.Packet().Add().Ipv6()
	f2Ip.Src().SetValue("0:10:10:10::1")
	f2Ip.Dst().SetValue("0:30:30:30::1")

	// add endpoints and packet description flow 3
	f3 := config.Flows().Items()[2]
	f3.SetName(p2.Name() + " -> " + p1.Name()).
		TxRx().Device().
		SetTxNames([]string{d2BgpIpv6Interface1Peer1V6Route1.Name()}).
		SetRxNames([]string{d1BgpIpv6Interface1Peer1V6Route1.Name()})

	f3Eth := f3.Packet().Add().Ethernet()
	f3Eth.Src().SetValue(d2Eth1.Mac())
	f3Eth.Dst().SetValue("00:00:00:00:00:00")

	f3Ip := f3.Packet().Add().Ipv6()
	f3Ip.Src().SetValue("0:20:20:20::1")
	f3Ip.Dst().SetValue("0:10:10:10::1")

	// add endpoints and packet description flow 4
	f4 := config.Flows().Items()[3]
	f4.SetName(p3.Name() + " -> " + p1.Name()).
		TxRx().Device().
		SetTxNames([]string{d3BgpIpv6Interface1Peer1V6Route1.Name()}).
		SetRxNames([]string{d1BgpIpv6Interface1Peer1V6Route1.Name()})

	f4Eth := f4.Packet().Add().Ethernet()
	f4Eth.Src().SetValue(d3Eth1.Mac())
	f4Eth.Dst().SetValue("00:00:00:00:00:00")

	f4Ip := f4.Packet().Add().Ipv6()
	f4Ip.Src().SetValue("0:30:30:30::1")
	f4Ip.Dst().SetValue("0:10:10:10::1")

	return config

}
