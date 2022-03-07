/* Test ISIS L2 P2P Adjacencies
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
	"tests/tests/helpers"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra"
)

func TestIsisL2P2pAdj(t *testing.T) {
	helpers.ConfigDUTs(map[string]string{"arista1": "../resources/dutconfig/isis_l2_p2p_adj/set_dut.txt"})
	defer helpers.ConfigDUTs(map[string]string{"arista1": "../resources/dutconfig/isis_l2_p2p_adj/unset_dut.txt"})

	ate := ondatra.ATE(t, "ate")
	otg := ate.OTG()
	defer helpers.CleanupTest(otg, t, true)

	config, expected := isisL2P2pAdjConfig(t, otg)
	otg.PushConfig(t, config)
	otg.StartProtocols(t)

	gnmiClient, err := helpers.NewGnmiClient(otg.NewGnmiQuery(t), config)
	if err != nil {
		t.Fatal(err)
	}

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.AllIsisSessionUp(expected) }, nil)

	otg.StartTraffic(t)

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.FlowMetricsOk(expected) }, nil)
}

func isisL2P2pAdjConfig(t *testing.T, otg *ondatra.OTGAPI) (gosnappi.Config, helpers.ExpectedState) {
	config := otg.NewConfig(t)
	port1 := config.Ports().Add().SetName("port1")
	port2 := config.Ports().Add().SetName("port2")
	dutPort1 := config.Devices().Add().SetName("dutPort1")
	dutPort1Eth := dutPort1.Ethernets().Add().
		SetName("dutPort1.eth").
		SetPortName(port1.Name()).
		SetMac("00:00:01:01:01:01")
	dutPort1Ipv4 := dutPort1Eth.Ipv4Addresses().Add().
		SetName("dutPort1.ipv4").
		SetAddress("1.1.1.1").
		SetGateway("1.1.1.3")
	dutPort1Eth.Ipv6Addresses().Add().
		SetName("dutPort1.ipv6").
		SetAddress("0:1:1:1::1").
		SetGateway("0:1:1:1::3")
	dutPort2 := config.Devices().Add().SetName("dutPort2")
	dutPort2Eth := dutPort2.Ethernets().Add().
		SetName("dutPort2.eth").
		SetPortName(port2.Name()).
		SetMac("00:00:02:01:01:01")
	dutPort2Ipv4 := dutPort2Eth.Ipv4Addresses().Add().
		SetName("dutPort2.ipv4").
		SetAddress("2.2.2.2").
		SetGateway("2.2.2.3")
	dutPort2Eth.Ipv6Addresses().Add().
		SetName("dutPort2.ipv6").
		SetAddress("0:2:2:2::2").
		SetGateway("0:2:2:2::3")
	// dut1 ISIS Router
	dutPort1Isis := dutPort1.Isis().
		SetName("dutPort1.isis.router").
		SetSystemId("640000000001")
	dutPort1Isis.Basic().SetIpv4TeRouterId(dutPort1Ipv4.Address())
	dutPort1Isis.Basic().SetHostname("ixia-c-port1")
	dutPort1Isis.Basic().SetEnableWideMetric(true)
	dutPort1Isis.Advanced().SetAreaAddresses([]string{"490002"})
	dutPort1Isis.Advanced().SetCsnpInterval(10000)
	dutPort1Isis.Advanced().SetEnableHelloPadding(true)
	dutPort1Isis.Advanced().SetLspLifetime(1200)
	dutPort1Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	dutPort1Isis.Advanced().SetLspRefreshRate(900)
	dutPort1Isis.Advanced().SetMaxAreaAddresses(3)
	dutPort1Isis.Advanced().SetMaxLspSize(1492)
	dutPort1Isis.Advanced().SetPsnpInterval(2000)
	dutPort1Isis.Advanced().SetEnableAttachedBit(false)
	//isis interface
	dutPort1IsisIntf := dutPort1Isis.Interfaces().
		Add().
		SetName("dutPort1.isis.interface").
		SetEthName(dutPort1Eth.Name()).
		SetNetworkType(gosnappi.IsisInterfaceNetworkType.POINT_TO_POINT).
		SetLevelType(gosnappi.IsisInterfaceLevelType.LEVEL_2).
		SetMetric(10)
	//isis L2 settings
	dutPort1IsisIntf.
		L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	//isis advanced settings
	dutPort1IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)
	//v4 routes
	dutPort1Isisv4routes := dutPort1Isis.
		V4Routes().
		Add().
		SetName("dutPort1.isis.rr4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	dutPort1Isisv4routes.Addresses().Add().
		SetAddress("40.40.40.0").
		SetPrefix(24).
		SetCount(5).
		SetStep(2)
	//v6 routes
	dutPort1Isisv6routes := dutPort1Isis.
		V6Routes().
		Add().
		SetName("dutPort1.isis.rr6").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV6RouteRangeOriginType.INTERNAL)
	dutPort1Isisv6routes.Addresses().Add().
		SetAddress("0:40:40:40::0").
		SetPrefix(64).
		SetCount(5).
		SetStep(2)
	// dut2 ISIS Router
	dutPort2Isis := dutPort2.Isis().
		SetName("dutPort2.isis.router").
		SetSystemId("650000000001")
	dutPort2Isis.Basic().SetIpv4TeRouterId(dutPort2Ipv4.Address())
	dutPort2Isis.Basic().SetHostname("ixia-c-port2")
	dutPort2Isis.Basic().SetEnableWideMetric(true)
	dutPort2Isis.Advanced().SetAreaAddresses([]string{"490002"})
	dutPort2Isis.Advanced().SetCsnpInterval(10000)
	dutPort2Isis.Advanced().SetEnableHelloPadding(true)
	dutPort2Isis.Advanced().SetLspLifetime(1200)
	dutPort2Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	dutPort2Isis.Advanced().SetLspRefreshRate(900)
	dutPort2Isis.Advanced().SetMaxAreaAddresses(3)
	dutPort2Isis.Advanced().SetMaxLspSize(1492)
	dutPort2Isis.Advanced().SetPsnpInterval(2000)
	dutPort2Isis.Advanced().SetEnableAttachedBit(false)
	//isis interface
	dutPort2IsisIntf := dutPort2Isis.Interfaces().
		Add().
		SetName("dutPort2.isis.interface").
		SetEthName(dutPort2Eth.Name()).
		SetNetworkType(gosnappi.IsisInterfaceNetworkType.POINT_TO_POINT).
		SetLevelType(gosnappi.IsisInterfaceLevelType.LEVEL_2).
		SetMetric(10)
	//isis L2 settings
	dutPort2IsisIntf.
		L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	//isis advanced settings
	dutPort2IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)
	dutPort2IsisV4RoutesPermit := dutPort2Isis.
		V4Routes().
		Add().
		SetName("dutPort2.isis.rr4.permit").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	dutPort2IsisV4RoutesPermit.Addresses().Add().
		SetAddress("50.50.50.0").
		SetPrefix(24).
		SetCount(5).
		SetStep(2)
	dutPort2IsisV6RoutesPermit := dutPort2Isis.V6Routes().Add().
		SetName("dutPort2.isis.rr6.permit").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV6RouteRangeOriginType.INTERNAL)
	dutPort2IsisV6RoutesPermit.Addresses().Add().
		SetAddress("0:50:50:50::0").
		SetPrefix(64).
		SetCount(5).
		SetStep(2)
	// OTG traffic configuration
	f1 := config.Flows().Add().SetName("p1.v4.p2.permit")
	f1.Metrics().SetEnable(true)
	f1.TxRx().Device().
		SetTxNames([]string{dutPort1Isisv4routes.Name()}).
		SetRxNames([]string{dutPort2IsisV4RoutesPermit.Name()})
	f1.Size().SetFixed(512)
	f1.Rate().SetPps(500)
	f1.Duration().FixedPackets().SetPackets(1000)
	e1 := f1.Packet().Add().Ethernet()
	e1.Src().SetValue(dutPort1Eth.Mac())
	e1.Dst().SetValue("00:00:00:00:00:00")
	v4 := f1.Packet().Add().Ipv4()
	v4.Src().SetValue("40.40.40.1")
	v4.Dst().Increment().SetStart("50.50.50.1").SetStep("0.0.0.1").SetCount(5)
	f1d := config.Flows().Add().SetName("p1.v4.p2.deny")
	f1d.Metrics().SetEnable(true)
	f1d.TxRx().Device().
		SetTxNames([]string{dutPort1Isisv4routes.Name()}).
		SetRxNames([]string{dutPort2IsisV4RoutesPermit.Name()})
	f1d.Size().SetFixed(512)
	f1d.Rate().SetPps(500)
	f1d.Duration().FixedPackets().SetPackets(1000)
	e1d := f1d.Packet().Add().Ethernet()
	e1d.Src().SetValue(dutPort1Eth.Mac())
	e1d.Dst().SetValue("00:00:00:00:00:00")
	v4d := f1d.Packet().Add().Ipv4()
	v4d.Src().SetValue("40.40.40.1")
	v4d.Dst().Increment().SetStart("60.60.60.1").SetStep("0.0.0.1").SetCount(5)
	f2 := config.Flows().Add().SetName("p1.v6.p2.permit")
	f2.Metrics().SetEnable(true)
	f2.TxRx().Device().
		SetTxNames([]string{dutPort1Isisv6routes.Name()}).
		SetRxNames([]string{dutPort2IsisV6RoutesPermit.Name()})
	f2.Size().SetFixed(512)
	f2.Rate().SetPps(500)
	f2.Duration().FixedPackets().SetPackets(1000)
	e2 := f2.Packet().Add().Ethernet()
	e2.Src().SetValue(dutPort1Eth.Mac())
	e2.Dst().SetValue("00:00:00:00:00:00")
	v6 := f2.Packet().Add().Ipv6()
	v6.Src().SetValue("0:40:40:40::1")
	v6.Dst().Increment().SetStart("0:50:50:50::1").SetStep("::1").SetCount(5)
	f2d := config.Flows().Add().SetName("p1.v6.p2.deny")
	f2d.Metrics().SetEnable(true)
	f2d.TxRx().Device().
		SetTxNames([]string{dutPort1Isisv6routes.Name()}).
		SetRxNames([]string{dutPort2IsisV6RoutesPermit.Name()})
	f2d.Size().SetFixed(512)
	f2d.Rate().SetPps(500)
	f2d.Duration().FixedPackets().SetPackets(1000)
	e2d := f2d.Packet().Add().Ethernet()
	e2d.Src().SetValue(dutPort1Eth.Mac())
	e2d.Dst().SetValue("00:00:00:00:00:00")
	v6d := f2d.Packet().Add().Ipv6()
	v6d.Src().SetValue("0:40:40:40::1")
	v6d.Dst().Increment().SetStart("0:60:60:60::1").SetStep("::1").SetCount(5)
	expected := helpers.ExpectedState{
		Isis: map[string]helpers.ExpectedIsisMetrics{
			dutPort1Isis.Name(): {L1SessionsUp: 0, L2SessionsUp: 1, L1DatabaseSize: 0, L2DatabaseSize: 3},
			dutPort2Isis.Name(): {L1SessionsUp: 0, L2SessionsUp: 1, L1DatabaseSize: 0, L2DatabaseSize: 3},
		},
		Flow: map[string]helpers.ExpectedFlowMetrics{
			f1.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f1d.Name(): {FramesRx: 0, FramesRxRate: 0},
			f2.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f2d.Name(): {FramesRx: 0, FramesRxRate: 0},
		},
	}
	return config, expected
}
