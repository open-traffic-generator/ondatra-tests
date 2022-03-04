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

// Package rt_1_3_route_propagation_test implements the rt-1.3 test plan
//
// For IPv4 and IPv6:
// - Advertise prefixes from ATE port-1, observe received prefixes at ATE port-2.
// - TODO(rdomigan): Specify default accept for received prefixes on DUT.
// - TODO(rdomigan): Specify table based neighbor configuration to cover - validating the supported capabilities
//   from the DUT.
//   - TODO(b/198193575): MRAI (minimum route advertisement interval), ensuring routes are advertised within specified
//     time.
//   - IPv4 routes with an IPv6 next-hop when negotiating RFC5549 - validating that
//     routes are accepted and advertised with the specified values.
//   - TODO(b/198193578): With ADD-PATH enabled, ensure that multiple routes are accepted from a neighbor when
//     advertised with individual path IDs, and that these routes are advertised to ATE port-2.
//
// To execute this test, run the following in your workstation:
// Substitute <path to conf> to the actual full path to the local conf file
// that contains the testbed config. See Readme for more details.
//
// blaze test --test_output=streamed --notest_loasd --test_arg=--use_kne_config=<path to conf>/testbed.conf rt_1_3_route_propagation_test
package tests

import (
	"fmt"
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

// This test topology connects the DUT and ATE over two ports.
//
// We are testing BGP route propagation across the DUT, so we are using the ATE to simulate two
// neighbors: AS 65536 on ATE port1, and AS 65538 on ATE port2, which both neighbor the DUT AS
// 65537 (both port1 and port2).

type otgPortDetails struct {
	mac, routerId string
}

var otgPort1Details otgPortDetails = otgPortDetails{
	mac:      "00:00:01:01:01:01",
	routerId: "1.1.1.1"}
var otgPort2Details otgPortDetails = otgPortDetails{
	mac:      "00:00:02:01:01:01",
	routerId: "2.2.2.2"}

var (
	// TODO(b/204349095): The DUT IP should be table driven as well, but assigning only an IPv6
	// address seems to prevent Ixia from bringing up the connection.
	dutPort1 = helpers.Attributes{
		Name:    "port1",
		Desc:    "To ATE",
		IPv4:    "192.0.2.1",
		IPv4Len: 30,
		IPv6:    "2001:db8::1",
		IPv6Len: 126,
	}
	dutPort2 = helpers.Attributes{
		Name:    "port2",
		Desc:    "To ATE",
		IPv4:    "192.0.2.5",
		IPv4Len: 30,
		IPv6:    "2001:db8::5",
		IPv6Len: 126,
	}
	dutAS1 = uint32(65537)

	ateAS1 = uint32(65536)
	ateAS2 = uint32(65538)
)

type ip struct {
	v4, v6 string
}

type ateData struct {
	Port1         ip
	Port2         ip
	Port1Neighbor string
	Port2Neighbor string
	prefixesStart ip
	prefixesCount uint32
}

func (ad *ateData) Configure(t *testing.T, otg *ondatra.OTGAPI, ateList []*ondatra.ATEDevice) gosnappi.Config {

	config := otg.NewConfig(t)
	bgp4ObjectMap := make(map[string]gosnappi.BgpV4Peer)
	bgp6ObjectMap := make(map[string]gosnappi.BgpV6Peer)
	ipv4ObjectMap := make(map[string]gosnappi.DeviceIpv4)
	ipv6ObjectMap := make(map[string]gosnappi.DeviceIpv6)
	ateIndex := 0
	for _, v := range []struct {
		iface    otgPortDetails
		ip       ip
		neighbor string
		as       uint32
	}{
		{otgPort1Details, ad.Port1, ad.Port1Neighbor, ateAS1},
		{otgPort2Details, ad.Port2, ad.Port2Neighbor, ateAS2},
	} {
		portName := ateList[ateIndex].Name()
		devName := ateList[ateIndex].Name() + ".dev"
		port := config.Ports().Add().SetName(portName)
		dev := config.Devices().Add().SetName(devName)
		ateIndex++
		eth := dev.Ethernets().Add().
			SetName(devName + ".eth").
			SetPortName(port.Name()).
			SetMac(v.iface.mac)
		bgp := dev.Bgp().
			SetRouterId(v.iface.routerId)
		if v.ip.v4 != "" {
			prefixInt4, _ := strconv.Atoi(strings.Split(v.ip.v4, "/")[1])
			ipv4 := eth.Ipv4Addresses().Add().
				SetName(devName + ".ipv4").
				SetAddress(strings.Split(v.ip.v4, "/")[0]).
				SetGateway(v.neighbor).
				SetPrefix(int32(prefixInt4))
			bgp4Name := devName + ".bgp4.peer"
			bgp4Peer := bgp.Ipv4Interfaces().Add().
				SetIpv4Name(ipv4.Name()).
				Peers().Add().
				SetName(bgp4Name).
				SetPeerAddress(ipv4.Gateway()).
				SetAsNumber(int32(v.as)).
				SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
			bgp4ObjectMap[bgp4Name] = bgp4Peer
			ipv4ObjectMap[devName+".ipv4"] = ipv4
		}
		if v.ip.v6 != "" {
			prefixInt6, _ := strconv.Atoi(strings.Split(v.ip.v6, "/")[1])
			ipv6 := eth.Ipv6Addresses().Add().
				SetName(devName + ".ipv6").
				SetAddress(v.ip.v6).
				SetGateway(v.neighbor).
				SetPrefix(int32(prefixInt6))
			bgp6Name := devName + ".bgp6.peer"
			bgp6Peer := bgp.Ipv6Interfaces().Add().
				SetIpv6Name(ipv6.Name()).
				Peers().Add().
				SetName(bgp6Name).
				SetPeerAddress(ipv6.Gateway()).
				SetAsNumber(int32(v.as)).
				SetAsType(gosnappi.BgpV6PeerAsType.EBGP)
			bgp6ObjectMap[bgp6Name] = bgp6Peer
			ipv6ObjectMap[devName+".ip6"] = ipv6
		}
	}
	if ad.prefixesStart.v4 != "" {
		prefixInt4, _ := strconv.Atoi(strings.Split(ad.prefixesStart.v4, "/")[1])
		bgp4Name := ateList[0].Name() + ".dev.bgp4.peer"
		bgp4Peer := bgp4ObjectMap[bgp4Name]
		ipv4 := ipv4ObjectMap[ateList[0].Name()+".dev.ipv4"]

		bgp4PeerRoutes := bgp4Peer.V4Routes().Add().
			SetName(bgp4Name + ".rr4").
			SetNextHopIpv4Address(ipv4.Address()).
			SetNextHopAddressType(gosnappi.BgpV4RouteRangeNextHopAddressType.IPV4).
			SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
		bgp4PeerRoutes.Addresses().Add().
			SetAddress(strings.Split(ad.prefixesStart.v4, "/")[0]).
			SetPrefix(int32(prefixInt4)).
			SetCount(int32(ad.prefixesCount))
	}
	if ad.prefixesStart.v6 != "" {
		prefixInt6, _ := strconv.Atoi(strings.Split(ad.prefixesStart.v6, "/")[1])
		bgp6Name := ateList[0].Name() + ".dev.bgp6.peer"
		bgp6Peer := bgp6ObjectMap[bgp6Name]
		ipv6 := ipv6ObjectMap[ateList[0].Name()+".dev.ipv6"]

		dstBgp6PeerRoutes := bgp6Peer.V6Routes().Add().
			SetName(bgp6Name + ".rr6").
			SetNextHopIpv6Address(ipv6.Address()).
			SetNextHopAddressType(gosnappi.BgpV6RouteRangeNextHopAddressType.IPV6).
			SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
		dstBgp6PeerRoutes.Addresses().Add().
			SetAddress(strings.Split(ad.prefixesStart.v6, "/")[0]).
			SetPrefix(int32(prefixInt6)).
			SetCount(int32(ad.prefixesCount))
	}
	return config
}

type dutData struct {
	bgpOC *oc.NetworkInstance_Protocol_Bgp
}

func (d *dutData) Configure(t *testing.T, dut *ondatra.DUTDevice) {
	for _, a := range []helpers.Attributes{dutPort1, dutPort2} {
		ocName := helpers.InterfaceMap[dut.Port(t, a.Name).Name()]
		dut.Config().Interface(ocName).Replace(t, a.NewInterface(ocName))
	}
	dutBGP := dut.Config().NetworkInstance("default").
		Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	dutBGP.Replace(t, d.bgpOC)
}

func (d *dutData) AwaitBGPEstablished(t *testing.T, dut *ondatra.DUTDevice) {
	for neighbor, _ := range d.bgpOC.Neighbor {
		dut.Telemetry().NetworkInstance("default").
			Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").
			Bgp().
			Neighbor(neighbor).
			SessionState().
			Await(t, time.Second*30, oc.Bgp_Neighbor_SessionState_ESTABLISHED)
	}
	t.Log("BGP sessions established")
}

func rt_1_3_UnsetDUT(t *testing.T, dut *ondatra.DUTDevice) {
	// t.Logf("Start Unsetting DUT Config")
	// helpers.ConfigDUTs(map[string]string{"arista1": "../resources/dutconfig/bgp_route_install/unset_dut.txt"})

	t.Logf("Start Unsetting DUT Interface Config")
	dc := dut.Config()

	i1 := helpers.RemoveInterface(helpers.InterfaceMap[dut.Port(t, "port1").Name()])
	dc.Interface(i1.GetName()).Replace(t, i1)

	i2 := helpers.RemoveInterface(helpers.InterfaceMap[dut.Port(t, "port2").Name()])
	dc.Interface(i2.GetName()).Replace(t, i2)

	t.Logf("Start Removing BGP config")
	dutConfPath := dut.Config().NetworkInstance("default").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "BGP").Bgp()
	helpers.LogYgot(t, "DUT BGP Config before", dutConfPath, dutConfPath.Get(t))
	dutConfPath.Replace(t, nil)

}

func Test_rt_1_3(t *testing.T) {
	tests := []struct {
		desc, fullDesc string
		skipReason     string
		dut            dutData
		ate            ateData
		wantPrefixes   []ip
	}{{
		desc:     "propagate IPv4 over IPv4",
		fullDesc: "Advertise prefixes from ATE port1, observe received prefixes at ATE port2",
		dut: dutData{&oc.NetworkInstance_Protocol_Bgp{
			Global: &oc.NetworkInstance_Protocol_Bgp_Global{
				As: ygot.Uint32(dutAS1),
			},
			Neighbor: map[string]*oc.NetworkInstance_Protocol_Bgp_Neighbor{
				"192.0.2.2": {
					PeerAs:          ygot.Uint32(ateAS1),
					NeighborAddress: ygot.String("192.0.2.2"),
					AfiSafi: map[oc.E_BgpTypes_AFI_SAFI_TYPE]*oc.NetworkInstance_Protocol_Bgp_Neighbor_AfiSafi{
						oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST: {
							AfiSafiName: oc.BgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST,
							Enabled:     ygot.Bool(true),
						},
					},
				},
				"192.0.2.6": {
					PeerAs:          ygot.Uint32(ateAS2),
					NeighborAddress: ygot.String("192.0.2.6"),
					AfiSafi: map[oc.E_BgpTypes_AFI_SAFI_TYPE]*oc.NetworkInstance_Protocol_Bgp_Neighbor_AfiSafi{
						oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST: {
							AfiSafiName: oc.BgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST,
							Enabled:     ygot.Bool(true),
						},
					},
				},
			},
		}},
		ate: ateData{
			Port1:         ip{v4: "192.0.2.2/30"},
			Port1Neighbor: dutPort1.IPv4,
			Port2:         ip{v4: "192.0.2.6/30"},
			Port2Neighbor: dutPort2.IPv4,
			prefixesStart: ip{v4: "198.51.100.0/32"},
			prefixesCount: 4,
		},
		wantPrefixes: []ip{
			{v4: "198.51.100.0/32"},
			{v4: "198.51.100.1/32"},
			{v4: "198.51.100.2/32"},
			{v4: "198.51.100.3/32"},
		},
	}, {
		desc:     "propagate IPv6 over IPv6",
		fullDesc: "Advertise IPv6 prefixes from ATE port1, observe received prefixes at ATE port2",
		dut: dutData{&oc.NetworkInstance_Protocol_Bgp{
			Global: &oc.NetworkInstance_Protocol_Bgp_Global{
				As: ygot.Uint32(dutAS1),
			},
			Neighbor: map[string]*oc.NetworkInstance_Protocol_Bgp_Neighbor{
				"2001:db8::2": {
					PeerAs:          ygot.Uint32(ateAS1),
					NeighborAddress: ygot.String("2001:db8::2"),
					AfiSafi: map[oc.E_BgpTypes_AFI_SAFI_TYPE]*oc.NetworkInstance_Protocol_Bgp_Neighbor_AfiSafi{
						oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST: {
							AfiSafiName: oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST,
							Enabled:     ygot.Bool(true),
						},
					},
				},
				"2001:db8::6": {
					PeerAs:          ygot.Uint32(ateAS2),
					NeighborAddress: ygot.String("2001:db8::6"),
					AfiSafi: map[oc.E_BgpTypes_AFI_SAFI_TYPE]*oc.NetworkInstance_Protocol_Bgp_Neighbor_AfiSafi{
						oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST: {
							AfiSafiName: oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST,
							Enabled:     ygot.Bool(true),
						},
					},
				},
			},
		}},
		ate: ateData{
			Port1:         ip{v6: "2001:db8::2/126"},
			Port1Neighbor: dutPort1.IPv6,
			Port2:         ip{v6: "2001:db8::6/126"},
			Port2Neighbor: dutPort2.IPv6,
			prefixesStart: ip{v6: "2001:db8:1::1/128"},
			prefixesCount: 4,
		},
		wantPrefixes: []ip{
			{v6: "2001:db8:1::1/128"},
			{v6: "2001:db8:1::2/128"},
			{v6: "2001:db8:1::3/128"},
			{v6: "2001:db8:1::4/128"},
		},
	}, {
		desc:       "propagate IPv4 over IPv6",
		skipReason: "TODO(b/203683090): Prefixes do not propagate as Arista currently requires RFC5549 to be enabled explicitly and OpenConfig does not currently provide a signal.",
		fullDesc:   "IPv4 routes with an IPv6 next-hop when negotiating RFC5549 - validating that routes are accepted and advertised with the specified values.",
		dut: dutData{&oc.NetworkInstance_Protocol_Bgp{
			Global: &oc.NetworkInstance_Protocol_Bgp_Global{
				As: ygot.Uint32(dutAS1),
			},
			Neighbor: map[string]*oc.NetworkInstance_Protocol_Bgp_Neighbor{
				"2001:db8::2": {
					PeerAs:          ygot.Uint32(ateAS1),
					NeighborAddress: ygot.String("2001:db8::2"),
					AfiSafi: map[oc.E_BgpTypes_AFI_SAFI_TYPE]*oc.NetworkInstance_Protocol_Bgp_Neighbor_AfiSafi{
						oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST: {
							AfiSafiName: oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST,
							Enabled:     ygot.Bool(true),
						},
					},
				},
				"192.0.2.6": {
					PeerAs:          ygot.Uint32(ateAS2),
					NeighborAddress: ygot.String("192.0.2.6"),
					AfiSafi: map[oc.E_BgpTypes_AFI_SAFI_TYPE]*oc.NetworkInstance_Protocol_Bgp_Neighbor_AfiSafi{
						oc.BgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST: {
							AfiSafiName: oc.BgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST,
							Enabled:     ygot.Bool(true),
						},
					},
				},
			},
		}},
		ate: ateData{
			Port1:         ip{v6: "2001:db8::2/126"},
			Port1Neighbor: dutPort1.IPv6,
			Port2:         ip{v4: "192.0.2.6/30"},
			Port2Neighbor: dutPort2.IPv4,
			prefixesStart: ip{v4: "198.51.100.0/32"},
			prefixesCount: 4,
		},
		wantPrefixes: []ip{
			{v4: "198.51.100.0/32"},
			{v4: "198.51.100.1/32"},
			{v4: "198.51.100.2/32"},
			{v4: "198.51.100.3/32"},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Log(tc.fullDesc)

			if tc.skipReason != "" {
				t.Skip(tc.skipReason)
			}

			t.Logf("Start DUT Config")
			dut := ondatra.DUT(t, "dut")
			defer rt_1_3_UnsetDUT(t, dut)
			tc.dut.Configure(t, dut)
			helpers.ConfigDUTs(map[string]string{"arista1": "../resources/dutconfig/rt_1_3_bgp_route_propagation/set_dut_interface.txt"})
			t.Logf("DUT Configured")

			ate1 := ondatra.ATE(t, "ate1")
			ate2 := ondatra.ATE(t, "ate2")
			ateList := []*ondatra.ATEDevice{
				ate1,
				ate2,
			}
			otg := ate1.OTG()
			defer helpers.CleanupTest(otg, t, true)
			t.Logf("Start OTG Config")
			config := tc.ate.Configure(t, otg, ateList)
			otg.PushConfig(t, config)
			otg.StartProtocols(t)
			t.Logf("OTG Configured")

			tc.dut.AwaitBGPEstablished(t, dut)

			for _, prefix := range tc.wantPrefixes {
				rib := ate2.Telemetry().NetworkInstance("port1").
					Protocol(
						oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
						fmt.Sprintf("%d", ateAS2),
					).Bgp().
					Rib()
				// Don't care about the value, but I can only fetch leaves from ATE telemetry. This
				// should fail in the Get(t) method if the Route is missing.
				if prefix.v4 != "" {
					_ = rib.AfiSafi(oc.BgpTypes_AFI_SAFI_TYPE_IPV4_UNICAST).Ipv4Unicast().
						Neighbor(tc.ate.Port2Neighbor).
						AdjRibInPre().
						Route(prefix.v4, 0).
						AttrIndex().Get(t)
				}
				if prefix.v6 != "" {
					_ = rib.AfiSafi(oc.BgpTypes_AFI_SAFI_TYPE_IPV6_UNICAST).Ipv6Unicast().
						Neighbor(tc.ate.Port2Neighbor).
						AdjRibInPre().
						Route(prefix.v6, 0).
						AttrIndex().Get(t)
				}
			}
		})
	}
}
