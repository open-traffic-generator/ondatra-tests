/* Test ISIS L2 P2P Multiple VLAN
Topology:
IXIA  ---------------------> ARISTA ---------------------> IXIA
(10.10.10.1/24, VLAN: 100)                                 (20.20.20.1/24, VLAN: 200)
(30.30.30.1/24, VLAN: 300)                                 (40.40.40.1/24, VLAN: 400)
(50.50.50.1/24, VLAN: 500)                                 (60.60.60.1/24, VLAN: 600)
(70.70.70.1/24, VLAN: 700)                                 (80.80.80.1/24, VLAN: 800)
(90.90.90.1/24, VLAN: 900)                                 (100.100.100.1/24, VLAN: 1000)

Flows:
- f1: 10.10.10.1 -> 20.20.20.1+, vlan: 100
- f2: 20.20.20.1 -> 10.10.10.1+, vlan: 200
- f3: 30.30.30.1 -> 40.40.40.1+, vlan: 300
- f4: 40.40.40.1 -> 30.30.30.1+, vlan: 400
- f5: 50.50.50.1 -> 60.60.60.1+, vlan: 500
- f6: 60.60.60.1 -> 50.50.50.1+, vlan: 600
- f7: 70.70.70.1 -> 80.80.80.1+, vlan: 700
- f8: 80.80.80.1 -> 70.70.70.1+, vlan: 800
- f9: 90.90.90.1 -> 100.100.100.1+, vlan: 900
- f10: 100.100.100.1 -> 90.90.90.1+, vlan: 1000
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
	for i := range config.Flows().Items() {
		mac := dutMacs[dut.Port(t, "port2").Name()]
		if i%2 == 0 {
			mac = dutMacs[dut.Port(t, "port1").Name()]
		}
		config.Flows().Items()[i].Packet().Items()[0].Ethernet().Dst().SetValue(mac)
	}

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

	// port 1 device 4
	p1d4 := config.Devices().Add().SetName("p1d4")
	// port 1 device 4 ethernet
	p1d4Eth := p1d4.Ethernets().Add().
		SetName("p1d4Eth").
		SetMac("00:00:07:07:07:07").
		SetMtu(1500).
		SetPortName(port1.Name())

	// port 1 device 4 ipv4
	p1d4Ipv4 := p1d4Eth.Ipv4Addresses().
		Add().
		SetAddress("7.7.7.2").
		SetGateway("7.7.7.1").
		SetName("p1d4Ipv4").
		SetPrefix(24)

	// port 1 device 4 vlan
	p1d4Vlan := p1d4Eth.Vlans().Add().
		SetId(700).
		SetName("p1d4vlan")

	// port 1 device 4 isis
	p1d4Isis := p1d4.Isis().SetName("p1d4Isis").SetSystemId("700000000001")

	// port 1 device 4 isis basic
	p1d4Isis.Basic().SetIpv4TeRouterId(p1d4Ipv4.Address())
	p1d4Isis.Basic().SetHostname("ixia-c-port1")
	p1d4Isis.Basic().SetEnableWideMetric(true)

	// port 1 device 4 isis advance
	p1d4Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p1d4Isis.Advanced().SetCsnpInterval(10000)
	p1d4Isis.Advanced().SetEnableHelloPadding(true)
	p1d4Isis.Advanced().SetLspLifetime(1200)
	p1d4Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p1d4Isis.Advanced().SetLspRefreshRate(900)
	p1d4Isis.Advanced().SetMaxAreaAddresses(3)
	p1d4Isis.Advanced().SetMaxLspSize(1492)
	p1d4Isis.Advanced().SetPsnpInterval(2000)
	p1d4Isis.Advanced().SetEnableAttachedBit(false)

	// port 1 device 4 isis interface
	p1d4IsisIntf := p1d4Isis.Interfaces().Add().
		SetEthName(p1d4Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p1d4IsisIntf")
	p1d4IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p1d4IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 1 device 4 isis v4 routes
	p1d4Isisv4routes := p1d4Isis.
		V4Routes().
		Add().
		SetName("p1d4IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p1d4Isisv4routes.Addresses().Add().
		SetAddress("70.70.70.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// port 1 device 5
	p1d5 := config.Devices().Add().SetName("p1d5")
	// port 1 device 5 ethernet
	p1d5Eth := p1d5.Ethernets().Add().
		SetName("p1d5Eth").
		SetMac("00:00:09:09:09:09").
		SetMtu(1500).
		SetPortName(port1.Name())

	// port 1 device 5 ipv4
	p1d5Ipv4 := p1d5Eth.Ipv4Addresses().
		Add().
		SetAddress("9.9.9.2").
		SetGateway("9.9.9.1").
		SetName("p1d5Ipv4").
		SetPrefix(24)

	// port 1 device 5 vlan
	p1d5Vlan := p1d5Eth.Vlans().Add().
		SetId(900).
		SetName("p1d5vlan")

	// port 1 device 5 isis
	p1d5Isis := p1d5.Isis().SetName("p1d5Isis").SetSystemId("720000000001")

	// port 1 device 5 isis basic
	p1d5Isis.Basic().SetIpv4TeRouterId(p1d5Ipv4.Address())
	p1d5Isis.Basic().SetHostname("ixia-c-port1")
	p1d5Isis.Basic().SetEnableWideMetric(true)

	// port 1 device 5 isis advance
	p1d5Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p1d5Isis.Advanced().SetCsnpInterval(10000)
	p1d5Isis.Advanced().SetEnableHelloPadding(true)
	p1d5Isis.Advanced().SetLspLifetime(1200)
	p1d5Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p1d5Isis.Advanced().SetLspRefreshRate(900)
	p1d5Isis.Advanced().SetMaxAreaAddresses(3)
	p1d5Isis.Advanced().SetMaxLspSize(1492)
	p1d5Isis.Advanced().SetPsnpInterval(2000)
	p1d5Isis.Advanced().SetEnableAttachedBit(false)

	// port 1 device 5 isis interface
	p1d5IsisIntf := p1d5Isis.Interfaces().Add().
		SetEthName(p1d5Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p1d5IsisIntf")
	p1d5IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p1d5IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 1 device 5 isis v4 routes
	p1d5Isisv4routes := p1d5Isis.
		V4Routes().
		Add().
		SetName("p1d5IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p1d5Isisv4routes.Addresses().Add().
		SetAddress("90.90.90.1").
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

	// port 2 device 4
	p2d4 := config.Devices().Add().SetName("p2d4")
	// port 2 device 4 ethernet
	p2d4Eth := p2d4.Ethernets().Add().
		SetName("p2d4Eth").
		SetMac("00:00:08:08:08:08").
		SetMtu(1500).
		SetPortName(port2.Name())

	// port 2 device 4 ipv4
	p2d4Ipv4 := p2d4Eth.Ipv4Addresses().
		Add().
		SetAddress("8.8.8.2").
		SetGateway("8.8.8.1").
		SetName("p2d4Ipv4").
		SetPrefix(24)

	// port 2 device 4 vlan
	p2d4Vlan := p2d4Eth.Vlans().Add().
		SetId(800).
		SetName("p2d4vlan")

	// port 2 device 4 isis
	p2d4Isis := p2d4.Isis().SetName("p2d4Isis").SetSystemId("710000000001")

	// port 2 device 4 isis basic
	p2d4Isis.Basic().SetIpv4TeRouterId(p2d4Ipv4.Address())
	p2d4Isis.Basic().SetHostname("ixia-c-port2")
	p2d4Isis.Basic().SetEnableWideMetric(true)

	// port 2 device 3 isis advance
	p2d4Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p2d4Isis.Advanced().SetCsnpInterval(10000)
	p2d4Isis.Advanced().SetEnableHelloPadding(true)
	p2d4Isis.Advanced().SetLspLifetime(1200)
	p2d4Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p2d4Isis.Advanced().SetLspRefreshRate(900)
	p2d4Isis.Advanced().SetMaxAreaAddresses(3)
	p2d4Isis.Advanced().SetMaxLspSize(1492)
	p2d4Isis.Advanced().SetPsnpInterval(2000)
	p2d4Isis.Advanced().SetEnableAttachedBit(false)

	// port 2 device 4 isis interface
	p2d4IsisIntf := p2d4Isis.Interfaces().Add().
		SetEthName(p2d4Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p2d4IsisIntf")
	p2d4IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p2d4IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 2 device 4 isis v4 routes
	p2d4Isisv4routes := p2d4Isis.
		V4Routes().
		Add().
		SetName("p2d4IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p2d4Isisv4routes.Addresses().Add().
		SetAddress("80.80.80.1").
		SetPrefix(32).
		SetCount(2).
		SetStep(1)

	// port 2 device 5
	p2d5 := config.Devices().Add().SetName("p2d5")
	// port 2 device 5 ethernet
	p2d5Eth := p2d5.Ethernets().Add().
		SetName("p2d5Eth").
		SetMac("00:00:11:11:11:11").
		SetMtu(1500).
		SetPortName(port2.Name())

	// port 2 device 5 ipv4
	p2d5Ipv4 := p2d5Eth.Ipv4Addresses().
		Add().
		SetAddress("11.11.11.2").
		SetGateway("11.11.11.1").
		SetName("p2d5Ipv4").
		SetPrefix(24)

	// port 2 device 5 vlan
	p2d5Vlan := p2d5Eth.Vlans().Add().
		SetId(1000).
		SetName("p2d5vlan")

	// port 2 device 5 isis
	p2d5Isis := p2d5.Isis().SetName("p2d5Isis").SetSystemId("730000000001")

	// port 2 device 5 isis basic
	p2d5Isis.Basic().SetIpv4TeRouterId(p2d5Ipv4.Address())
	p2d5Isis.Basic().SetHostname("ixia-c-port2")
	p2d5Isis.Basic().SetEnableWideMetric(true)

	// port 2 device 5 isis advance
	p2d5Isis.Advanced().SetAreaAddresses([]string{"490001"})
	p2d5Isis.Advanced().SetCsnpInterval(10000)
	p2d5Isis.Advanced().SetEnableHelloPadding(true)
	p2d5Isis.Advanced().SetLspLifetime(1200)
	p2d5Isis.Advanced().SetLspMgroupMinTransInterval(5000)
	p2d5Isis.Advanced().SetLspRefreshRate(900)
	p2d5Isis.Advanced().SetMaxAreaAddresses(3)
	p2d5Isis.Advanced().SetMaxLspSize(1492)
	p2d5Isis.Advanced().SetPsnpInterval(2000)
	p2d5Isis.Advanced().SetEnableAttachedBit(false)

	// port 2 device 5 isis interface
	p2d5IsisIntf := p2d5Isis.Interfaces().Add().
		SetEthName(p2d5Eth.Name()).
		SetNetworkType("point_to_point").
		SetLevelType("level_2").
		SetMetric(10).
		SetName("p2d5IsisIntf")
	p2d5IsisIntf.L2Settings().
		SetDeadInterval(30).
		SetHelloInterval(10).
		SetPriority(0)
	p2d5IsisIntf.
		Advanced().SetAutoAdjustSupportedProtocols(true)

	// port 2 device 5 isis v4 routes
	p2d5Isisv4routes := p2d5Isis.
		V4Routes().
		Add().
		SetName("p2d5IsisIpv4").
		SetLinkMetric(10).
		SetOriginType(gosnappi.IsisV4RouteRangeOriginType.INTERNAL)
	p2d5Isisv4routes.Addresses().Add().
		SetAddress("100.100.100.1").
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

	f7 := config.Flows().Add().SetName("p1.v4.p2.vlan.700")
	f7.Metrics().SetEnable(true)
	f7.TxRx().Device().
		SetTxNames([]string{p1d4Isisv4routes.Name()}).
		SetRxNames([]string{p2d4Isisv4routes.Name()})
	f7.Size().SetFixed(512)
	f7.Rate().SetPps(500)
	f7.Duration().FixedPackets().SetPackets(1000)
	e7 := f7.Packet().Add().Ethernet()
	e7.Src().SetValue(p1d4Eth.Mac())

	vlan7 := f7.Packet().Add().Vlan()
	vlan7.Id().SetValue(p1d4Vlan.Id())
	vlan7.Tpid().SetValue(33024)

	v4 = f7.Packet().Add().Ipv4()
	v4.Src().SetValue("70.70.70.1")
	v4.Dst().SetValue("80.80.80.1")

	f8 := config.Flows().Add().SetName("p2.v4.p1.vlan.800")
	f8.Metrics().SetEnable(true)
	f8.TxRx().Device().
		SetTxNames([]string{p2d4Isisv4routes.Name()}).
		SetRxNames([]string{p1d4Isisv4routes.Name()})
	f8.Size().SetFixed(512)
	f8.Rate().SetPps(500)
	f8.Duration().FixedPackets().SetPackets(1000)
	e8 := f8.Packet().Add().Ethernet()
	e8.Src().SetValue(p2d4Eth.Mac())

	vlan8 := f8.Packet().Add().Vlan()
	vlan8.Id().SetValue(p2d4Vlan.Id())
	vlan8.Tpid().SetValue(33024)

	v4 = f8.Packet().Add().Ipv4()
	v4.Src().SetValue("80.80.80.1")
	v4.Dst().SetValue("70.70.70.1")

	f9 := config.Flows().Add().SetName("p1.v4.p2.vlan.900")
	f9.Metrics().SetEnable(true)
	f9.TxRx().Device().
		SetTxNames([]string{p1d5Isisv4routes.Name()}).
		SetRxNames([]string{p2d5Isisv4routes.Name()})
	f9.Size().SetFixed(512)
	f9.Rate().SetPps(500)
	f9.Duration().FixedPackets().SetPackets(1000)
	e9 := f9.Packet().Add().Ethernet()
	e9.Src().SetValue(p1d5Eth.Mac())

	vlan9 := f9.Packet().Add().Vlan()
	vlan9.Id().SetValue(p1d5Vlan.Id())
	vlan9.Tpid().SetValue(33024)

	v4 = f9.Packet().Add().Ipv4()
	v4.Src().SetValue("90.90.90.1")
	v4.Dst().SetValue("100.100.100.1")

	f10 := config.Flows().Add().SetName("p2.v4.p1.vlan.1000")
	f10.Metrics().SetEnable(true)
	f10.TxRx().Device().
		SetTxNames([]string{p2d5Isisv4routes.Name()}).
		SetRxNames([]string{p1d5Isisv4routes.Name()})
	f10.Size().SetFixed(512)
	f10.Rate().SetPps(500)
	f10.Duration().FixedPackets().SetPackets(1000)
	e10 := f10.Packet().Add().Ethernet()
	e10.Src().SetValue(p2d5Eth.Mac())

	vlan10 := f10.Packet().Add().Vlan()
	vlan10.Id().SetValue(p2d5Vlan.Id())
	vlan10.Tpid().SetValue(33024)

	v4 = f10.Packet().Add().Ipv4()
	v4.Src().SetValue("100.100.100.1")
	v4.Dst().SetValue("90.90.90.1")

	expected := helpers.ExpectedState{
		Isis: map[string]helpers.ExpectedIsisMetrics{
			p1d1Isis.Name(): {L1SessionsUp: 0, L2SessionsUp: 1, L1DatabaseSize: 0, L2DatabaseSize: 11},
			p2d1Isis.Name(): {L1SessionsUp: 0, L2SessionsUp: 1, L1DatabaseSize: 0, L2DatabaseSize: 11},
		},
		Flow: map[string]helpers.ExpectedFlowMetrics{
			f1.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f2.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f3.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f4.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f5.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f6.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f7.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f8.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f9.Name():  {FramesRx: 1000, FramesRxRate: 0},
			f10.Name(): {FramesRx: 1000, FramesRxRate: 0},
		},
	}
	return config, expected
}
