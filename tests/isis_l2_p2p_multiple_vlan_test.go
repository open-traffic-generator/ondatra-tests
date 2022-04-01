/* Test ISIS L2 P2P Multiple VLAN
Topology:
IXIA  ---------------------> ARISTA ---------------------> IXIA
(10.10.10.1/24, VLAN: 100)                                 (20.20.20.1/24, VLAN: 200)
(30.30.30.1/24, VLAN: 300)                                 (40.40.40.1/24, VLAN: 400)

Flows:
- f1: 10.10.10.1 -> 20.20.20.1+, vlan: 100
- f2: 20.20.20.1 -> 10.10.10.1+, vlan: 200
- f3: 30.30.30.1 -> 40.40.40.1+, vlan: 300
- f4: 40.40.40.1 -> 30.30.30.1+, vlan: 400
*/
package tests

import (
	"testing"
	"tests/tests/helpers"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/ondatra"
)

func GetInterfaceMacs(t *testing.T, dev *ondatra.Device) map[string]string {
	t.Helper()
	dutMacDetails := make(map[string]string)
	for _, p := range dev.Ports() {
		eth := dev.Telemetry().Interface(p.Name()).Ethernet().Get(t)
		t.Logf("Mac address of Interface %s in DUT: %s", p.Name(), eth.GetMacAddress())
		dutMacDetails[p.Name()] = eth.GetMacAddress()
	}
	return dutMacDetails
}

func TestIsisL2P2pMultiVLAN(t *testing.T) {
	ate := ondatra.ATE(t, "ate")
	dut := ondatra.DUT(t, "dut")

	otg := ate.OTG(t)
	defer helpers.CleanupTest(t, ate, otg, true)

	config, expected := isisL2P2pMultiVlanConfig(t, otg)
	dutMacs := GetInterfaceMacs(t, dut.Device)
	config.Flows().Items()[0].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacs[dut.Port(t, "port1").Name()])
	config.Flows().Items()[1].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacs[dut.Port(t, "port2").Name()])
	config.Flows().Items()[2].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacs[dut.Port(t, "port1").Name()])
	config.Flows().Items()[3].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacs[dut.Port(t, "port2").Name()])
	config.Flows().Items()[4].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacs[dut.Port(t, "port1").Name()])
	config.Flows().Items()[5].Packet().Items()[0].Ethernet().Dst().SetValue(dutMacs[dut.Port(t, "port2").Name()])

	if ate.Port(t, "port1").Name() == "eth1" {
		dut.Config().New().WithAristaFile("../resources/dutconfig/isis_l2_p2p_multi_vlan/set_dut.txt").Push(t)
	} else {
		dut.Config().New().WithAristaFile("../resources/dutconfig/isis_l2_p2p_multi_vlan/set_dut_alternative.txt").Push(t)
	}
	defer dut.Config().New().WithAristaFile("../resources/dutconfig/isis_l2_p2p_multi_vlan/unset_dut.txt").Push(t)

	otg.PushConfig(t, ate, config)
	otg.StartProtocols(t)

	gnmiClient, err := helpers.NewGnmiClient(otg.NewGnmiQuery(t), config)
	if err != nil {
		t.Fatal(err)
	}

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.AllIsisSessionUp(expected) }, nil)

	otg.StartTraffic(t)

	helpers.WaitFor(t, func() (bool, error) { return gnmiClient.FlowMetricsOk(expected) }, nil)
}

func isisL2P2pMultiVlanConfig(t *testing.T, otg *ondatra.OTG) (gosnappi.Config, helpers.ExpectedState) {
	config := otg.NewConfig()
	port1 := config.Ports().Add().SetName("port1")
	port2 := config.Ports().Add().SetName("port2")

	// port 1 device 1
	p1d1 := config.Devices().Add().SetName("p1d1")
	// port 1 device 1 ethernet
	p1d1Eth := p1d1.Ethernets().Add().
		SetName("p1d1Eth").
		SetMac("00:00:01:01:01:01").
		SetMtu(1500).
		SetPortName(port1.Name())

	// port 1 device 1 ipv4
	p1d1Ipv4 := p1d1Eth.Ipv4Addresses().
		Add().
		SetAddress("1.1.1.2").
		SetGateway("1.1.1.1").
		SetName("p1d1Ipv4").
		SetPrefix(24)

	// port 1 device 1 vlan
	p1d1Vlan := p1d1Eth.Vlans().Add().
		SetId(100).
		SetName("p1d1vlan")

	// port 1 device 1 isis
	p1d1Isis := p1d1.Isis().SetName("p1d1Isis").SetSystemId("640000000001")

	// port 1 device 1 isis basic
	p1d1Isis.Basic().SetIpv4TeRouterId(p1d1Ipv4.Address())
	p1d1Isis.Basic().SetHostname("ixia-c-port1")
	p1d1Isis.Basic().SetEnableWideMetric(true)

	// port 1 device 1 isis advance
	p1d1Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p1d1Isis.Advanced().SetCsnpInterval(10000)
	p1d1Isis.Advanced().SetEnableHelloPadding(true)
	p1d1Isis.Advanced().SetLspLifetime(1200)
	p1d1Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p1d1Isis.Advanced().SetLspRefreshRate(900)
	p1d1Isis.Advanced().SetMaxAreaAddresses(3)
	p1d1Isis.Advanced().SetMaxLspSize(1492)
	p1d1Isis.Advanced().SetPsnpInterval(2000)
	p1d1Isis.Advanced().SetEnableAttachedBit(false)

	// port 1 device 1 isis interface
	p1d1IsisIntf := p1d1Isis.Interfaces().Add().
		SetEthName(p1d1Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p1d1IsisIntf")
	p1d1IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p1d1IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 1 device 1 isis v4 routes
	p1d1Isisv4routes := p1d1Isis.
		V4Routes().
		Add().
		SetName("p1d1IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p1d1Isisv4routes.Addresses().Add().
		SetAddress("10.10.10.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// port 1 device 2
	p1d2 := config.Devices().Add().SetName("p1d2")
	// port 1 device 2 ethernet
	p1d2Eth := p1d2.Ethernets().Add().
		SetName("p1d2Eth").
		SetMac("00:00:03:03:03:03").
		SetMtu(1500).
		SetPortName(port1.Name())

	// port 1 device 2 ipv4
	p1d2Ipv4 := p1d2Eth.Ipv4Addresses().
		Add().
		SetAddress("3.3.3.2").
		SetGateway("3.3.3.1").
		SetName("p1d2Ipv4").
		SetPrefix(24)

	// port 1 device 2 vlan
	p1d2Vlan := p1d2Eth.Vlans().Add().
		SetId(300).
		SetName("p1d2vlan")

	// port 1 device 2 isis
	p1d2Isis := p1d2.Isis().SetName("p1d2Isis").SetSystemId("660000000001")

	// port 1 device 2 isis basic
	p1d2Isis.Basic().SetIpv4TeRouterId(p1d2Ipv4.Address())
	p1d2Isis.Basic().SetHostname("ixia-c-port1")
	p1d2Isis.Basic().SetEnableWideMetric(true)

	// port 1 device 2 isis advance
	p1d2Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p1d2Isis.Advanced().SetCsnpInterval(10000)
	p1d2Isis.Advanced().SetEnableHelloPadding(true)
	p1d2Isis.Advanced().SetLspLifetime(1200)
	p1d2Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p1d2Isis.Advanced().SetLspRefreshRate(900)
	p1d2Isis.Advanced().SetMaxAreaAddresses(3)
	p1d2Isis.Advanced().SetMaxLspSize(1492)
	p1d2Isis.Advanced().SetPsnpInterval(2000)
	p1d2Isis.Advanced().SetEnableAttachedBit(false)

	// port 1 device 2 isis interface
	p1d2IsisIntf := p1d2Isis.Interfaces().Add().
		SetEthName(p1d2Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p1d2IsisIntf")
	p1d2IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p1d2IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 1 device 2 isis v4 routes
	p1d2Isisv4routes := p1d2Isis.
		V4Routes().
		Add().
		SetName("p1d2IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p1d2Isisv4routes.Addresses().Add().
		SetAddress("30.30.30.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// port 1 device 3
	p1d3 := config.Devices().Add().SetName("p1d3")
	// port 1 device 3 ethernet
	p1d3Eth := p1d3.Ethernets().Add().
		SetName("p1d3Eth").
		SetMac("00:00:05:05:05:05").
		SetMtu(1500).
		SetPortName(port1.Name())

	// port 1 device 3 ipv4
	p1d3Ipv4 := p1d3Eth.Ipv4Addresses().
		Add().
		SetAddress("5.5.5.2").
		SetGateway("5.5.5.1").
		SetName("p1d3Ipv4").
		SetPrefix(24)

	// port 1 device 3 vlan
	p1d3Vlan := p1d3Eth.Vlans().Add().
		SetId(500).
		SetName("p1d3vlan")

	// port 1 device 3 isis
	p1d3Isis := p1d3.Isis().SetName("p1d3Isis").SetSystemId("680000000001")

	// port 1 device 3 isis basic
	p1d3Isis.Basic().SetIpv4TeRouterId(p1d3Ipv4.Address())
	p1d3Isis.Basic().SetHostname("ixia-c-port1")
	p1d3Isis.Basic().SetEnableWideMetric(true)

	// port 1 device 3 isis advance
	p1d3Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p1d3Isis.Advanced().SetCsnpInterval(10000)
	p1d3Isis.Advanced().SetEnableHelloPadding(true)
	p1d3Isis.Advanced().SetLspLifetime(1200)
	p1d3Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p1d3Isis.Advanced().SetLspRefreshRate(900)
	p1d3Isis.Advanced().SetMaxAreaAddresses(3)
	p1d3Isis.Advanced().SetMaxLspSize(1492)
	p1d3Isis.Advanced().SetPsnpInterval(2000)
	p1d3Isis.Advanced().SetEnableAttachedBit(false)

	// port 1 device 3 isis interface
	p1d3IsisIntf := p1d3Isis.Interfaces().Add().
		SetEthName(p1d3Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p1d3IsisIntf")
	p1d3IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p1d3IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 1 device 3 isis v4 routes
	p1d3Isisv4routes := p1d3Isis.
		V4Routes().
		Add().
		SetName("p1d3IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p1d3Isisv4routes.Addresses().Add().
		SetAddress("50.50.50.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// port 2 device 1
	p2d1 := config.Devices().Add().SetName("p2d1")
	// port 2 device 1 ethernet
	p2d1Eth := p2d1.Ethernets().Add().
		SetName("p2d1Eth").
		SetMac("00:00:02:02:02:02").
		SetMtu(1500).
		SetPortName(port2.Name())

	// port 2 device 1 ipv4
	p2d1Ipv4 := p2d1Eth.Ipv4Addresses().
		Add().
		SetAddress("2.2.2.2").
		SetGateway("2.2.2.1").
		SetName("p2d1Ipv4").
		SetPrefix(24)

	// port 2 device 1 vlan
	p2d1Vlan := p2d1Eth.Vlans().Add().
		SetId(200).
		SetName("p2d1vlan")

	// port 2 device 1 isis
	p2d1Isis := p2d1.Isis().SetName("p2d1Isis").SetSystemId("650000000001")

	// port 2 device 1 isis basic
	p2d1Isis.Basic().SetIpv4TeRouterId(p2d1Ipv4.Address())
	p2d1Isis.Basic().SetHostname("ixia-c-port2")
	p2d1Isis.Basic().SetEnableWideMetric(true)

	// port 2 device 1 isis advance
	p2d1Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p2d1Isis.Advanced().SetCsnpInterval(10000)
	p2d1Isis.Advanced().SetEnableHelloPadding(true)
	p2d1Isis.Advanced().SetLspLifetime(1200)
	p2d1Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p2d1Isis.Advanced().SetLspRefreshRate(900)
	p2d1Isis.Advanced().SetMaxAreaAddresses(3)
	p2d1Isis.Advanced().SetMaxLspSize(1492)
	p2d1Isis.Advanced().SetPsnpInterval(2000)
	p2d1Isis.Advanced().SetEnableAttachedBit(false)

	// port 2 device 1 isis interface
	p2d1IsisIntf := p2d1Isis.Interfaces().Add().
		SetEthName(p2d1Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p2d1IsisIntf")
	p2d1IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p2d1IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 2 device 1 isis v4 routes
	p2d1Isisv4routes := p2d1Isis.
		V4Routes().
		Add().
		SetName("p2d1IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p2d1Isisv4routes.Addresses().Add().
		SetAddress("20.20.20.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// port 2 device 2
	p2d2 := config.Devices().Add().SetName("p2d2")
	// port 2 device 2 ethernet
	p2d2Eth := p2d2.Ethernets().Add().
		SetName("p2d2Eth").
		SetMac("00:00:04:04:04:04").
		SetMtu(1500).
		SetPortName(port2.Name())

	// port 2 device 2 ipv4
	p2d2Ipv4 := p2d2Eth.Ipv4Addresses().
		Add().
		SetAddress("4.4.4.2").
		SetGateway("4.4.4.1").
		SetName("p2d2Ipv4").
		SetPrefix(24)

	// port 2 device 2 vlan
	p2d2Vlan := p2d2Eth.Vlans().Add().
		SetId(400).
		SetName("p2d2vlan")

	// port 2 device 2 isis
	p2d2Isis := p2d2.Isis().SetName("p2d2Isis").SetSystemId("670000000001")

	// port 2 device 2 isis basic
	p2d2Isis.Basic().SetIpv4TeRouterId(p2d2Ipv4.Address())
	p2d2Isis.Basic().SetHostname("ixia-c-port2")
	p2d2Isis.Basic().SetEnableWideMetric(true)

	// port 2 device 2 isis advance
	p2d2Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p2d2Isis.Advanced().SetCsnpInterval(10000)
	p2d2Isis.Advanced().SetEnableHelloPadding(true)
	p2d2Isis.Advanced().SetLspLifetime(1200)
	p2d2Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p2d2Isis.Advanced().SetLspRefreshRate(900)
	p2d2Isis.Advanced().SetMaxAreaAddresses(3)
	p2d2Isis.Advanced().SetMaxLspSize(1492)
	p2d2Isis.Advanced().SetPsnpInterval(2000)
	p2d2Isis.Advanced().SetEnableAttachedBit(false)

	// port 2 device 2 isis interface
	p2d2IsisIntf := p2d2Isis.Interfaces().Add().
		SetEthName(p2d2Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p2d2IsisIntf")
	p2d2IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p2d2IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 2 device 2 isis v4 routes
	p2d2Isisv4routes := p2d2Isis.
		V4Routes().
		Add().
		SetName("p2d2IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p2d2Isisv4routes.Addresses().Add().
		SetAddress("40.40.40.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// port 2 device 3
	p2d3 := config.Devices().Add().SetName("p2d3")
	// port 2 device 3 ethernet
	p2d3Eth := p2d3.Ethernets().Add().
		SetName("p2d3Eth").
		SetMac("00:00:06:06:06:06").
		SetMtu(1500).
		SetPortName(port2.Name())

	// port 2 device 3 ipv4
	p2d3Ipv4 := p2d3Eth.Ipv4Addresses().
		Add().
		SetAddress("6.6.6.2").
		SetGateway("6.6.6.1").
		SetName("p2d3Ipv4").
		SetPrefix(24)

	// port 2 device 3 vlan
	p2d3Vlan := p2d3Eth.Vlans().Add().
		SetId(600).
		SetName("p2d3vlan")

	// port 2 device 3 isis
	p2d3Isis := p2d3.Isis().SetName("p2d3Isis").SetSystemId("690000000001")

	// port 2 device 3 isis basic
	p2d3Isis.Basic().SetIpv4TeRouterId(p2d3Ipv4.Address())
	p2d3Isis.Basic().SetHostname("ixia-c-port2")
	p2d3Isis.Basic().SetEnableWideMetric(true)

	// port 2 device 3 isis advance
	p2d3Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p2d3Isis.Advanced().SetCsnpInterval(10000)
	p2d3Isis.Advanced().SetEnableHelloPadding(true)
	p2d3Isis.Advanced().SetLspLifetime(1200)
	p2d3Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p2d3Isis.Advanced().SetLspRefreshRate(900)
	p2d3Isis.Advanced().SetMaxAreaAddresses(3)
	p2d3Isis.Advanced().SetMaxLspSize(1492)
	p2d3Isis.Advanced().SetPsnpInterval(2000)
	p2d3Isis.Advanced().SetEnableAttachedBit(false)

	// port 2 device 3 isis interface
	p2d3IsisIntf := p2d3Isis.Interfaces().Add().
		SetEthName(p2d3Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p2d3IsisIntf")
	p2d3IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p2d3IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 2 device 3 isis v4 routes
	p2d3Isisv4routes := p2d3Isis.
		V4Routes().
		Add().
		SetName("p2d3IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p2d3Isisv4routes.Addresses().Add().
		SetAddress("60.60.60.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// OTG traffic configuration
	f1 := config.Flows().Add().SetName("p1.v4.p2.vlan.100")
	f1.Metrics().SetEnable(true)
	f1.TxRx().Device().
		SetTxNames([]string{p1d1Isisv4routes.Name()}).
		SetRxNames([]string{p2d1Isisv4routes.Name()})
	f1.Size().SetFixed(512)
	f1.Rate().SetPps(500)
	f1.Duration().FixedPackets().SetPackets(1000)
	e1 := f1.Packet().Add().Ethernet()
	e1.Src().SetValue(p1d1Eth.Mac())

	vlan1 := f1.Packet().Add().Vlan()
	vlan1.Id().SetValue(p1d1Vlan.Id())
	vlan1.Tpid().SetValue(33024)

	v4 := f1.Packet().Add().Ipv4()
	v4.Src().SetValue("10.10.10.1")
	v4.Dst().SetValue("20.20.20.1")

	f2 := config.Flows().Add().SetName("p2.v4.p1.vlan.200")
	f2.Metrics().SetEnable(true)
	f2.TxRx().Device().
		SetTxNames([]string{p2d1Isisv4routes.Name()}).
		SetRxNames([]string{p1d1Isisv4routes.Name()})
	f2.Size().SetFixed(512)
	f2.Rate().SetPps(500)
	f2.Duration().FixedPackets().SetPackets(1000)
	e2 := f2.Packet().Add().Ethernet()
	e2.Src().SetValue(p2d1Eth.Mac())

	vlan2 := f2.Packet().Add().Vlan()
	vlan2.Id().SetValue(p2d1Vlan.Id())
	vlan2.Tpid().SetValue(33024)

	v4 = f2.Packet().Add().Ipv4()
	v4.Src().SetValue("20.20.20.1")
	v4.Dst().SetValue("10.10.10.1")

	f3 := config.Flows().Add().SetName("p1.v4.p2.vlan.300")
	f3.Metrics().SetEnable(true)
	f3.TxRx().Device().
		SetTxNames([]string{p1d2Isisv4routes.Name()}).
		SetRxNames([]string{p2d2Isisv4routes.Name()})
	f3.Size().SetFixed(512)
	f3.Rate().SetPps(500)
	f3.Duration().FixedPackets().SetPackets(1000)
	e3 := f3.Packet().Add().Ethernet()
	e3.Src().SetValue(p1d2Eth.Mac())

	vlan3 := f3.Packet().Add().Vlan()
	vlan3.Id().SetValue(p1d2Vlan.Id())
	vlan3.Tpid().SetValue(33024)

	v4 = f3.Packet().Add().Ipv4()
	v4.Src().SetValue("30.30.30.1")
	v4.Dst().SetValue("40.40.40.1")

	f4 := config.Flows().Add().SetName("p2.v4.p1.vlan.400")
	f4.Metrics().SetEnable(true)
	f4.TxRx().Device().
		SetTxNames([]string{p2d2Isisv4routes.Name()}).
		SetRxNames([]string{p1d2Isisv4routes.Name()})
	f4.Size().SetFixed(512)
	f4.Rate().SetPps(500)
	f4.Duration().FixedPackets().SetPackets(1000)
	e4 := f4.Packet().Add().Ethernet()
	e4.Src().SetValue(p2d2Eth.Mac())

	vlan4 := f4.Packet().Add().Vlan()
	vlan4.Id().SetValue(p2d2Vlan.Id())
	vlan4.Tpid().SetValue(33024)

	v4 = f4.Packet().Add().Ipv4()
	v4.Src().SetValue("40.40.40.1")
	v4.Dst().SetValue("30.30.30.1")

	f5 := config.Flows().Add().SetName("p1.v4.p2.vlan.500")
	f5.Metrics().SetEnable(true)
	f5.TxRx().Device().
		SetTxNames([]string{p1d3Isisv4routes.Name()}).
		SetRxNames([]string{p2d3Isisv4routes.Name()})
	f5.Size().SetFixed(512)
	f5.Rate().SetPps(500)
	f5.Duration().FixedPackets().SetPackets(1000)
	e5 := f5.Packet().Add().Ethernet()
	e5.Src().SetValue(p1d3Eth.Mac())

	vlan5 := f5.Packet().Add().Vlan()
	vlan5.Id().SetValue(p1d3Vlan.Id())
	vlan5.Tpid().SetValue(33024)

	v4 = f5.Packet().Add().Ipv4()
	v4.Src().SetValue("50.50.50.1")
	v4.Dst().SetValue("60.60.60.1")

	f6 := config.Flows().Add().SetName("p2.v4.p1.vlan.600")
	f6.Metrics().SetEnable(true)
	f6.TxRx().Device().
		SetTxNames([]string{p2d3Isisv4routes.Name()}).
		SetRxNames([]string{p1d3Isisv4routes.Name()})
	f6.Size().SetFixed(512)
	f6.Rate().SetPps(500)
	f6.Duration().FixedPackets().SetPackets(1000)
	e6 := f6.Packet().Add().Ethernet()
	e6.Src().SetValue(p2d3Eth.Mac())

	vlan6 := f6.Packet().Add().Vlan()
	vlan6.Id().SetValue(p2d3Vlan.Id())
	vlan6.Tpid().SetValue(33024)

	v4 = f6.Packet().Add().Ipv4()
	v4.Src().SetValue("60.60.60.1")
	v4.Dst().SetValue("50.50.50.1")

	expected := helpers.ExpectedState{
		Isis: map[string]helpers.ExpectedIsisMetrics{
			p1d1Isis.Name(): {L1SessionsUp: 0, L2SessionsUp: 1, L1DatabaseSize: 0, L2DatabaseSize: 7},
			p2d1Isis.Name(): {L1SessionsUp: 0, L2SessionsUp: 1, L1DatabaseSize: 0, L2DatabaseSize: 7},
		},
		Flow: map[string]helpers.ExpectedFlowMetrics{
			f1.Name(): {FramesRx: 1000, FramesRxRate: 0},
			f2.Name(): {FramesRx: 1000, FramesRxRate: 0},
			f3.Name(): {FramesRx: 1000, FramesRxRate: 0},
			f4.Name(): {FramesRx: 1000, FramesRxRate: 0},
			f5.Name(): {FramesRx: 1000, FramesRxRate: 0},
			f6.Name(): {FramesRx: 1000, FramesRxRate: 0},
		},
	}
	return config, expected
}
