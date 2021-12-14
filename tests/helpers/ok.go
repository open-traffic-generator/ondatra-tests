package helpers

import (
	"github.com/open-traffic-generator/snappi/gosnappi"
)

type ExpectedBgpMetrics struct {
	Advertised int32
	Received   int32
}

type ExpectedIsisMetrics struct {
	L1SessionsUp   int32
	L2SessionsUp   int32
	L1DatabaseSize int32
	L2DatabaseSize int32
}
type ExpectedPortMetrics struct {
	FramesRx int32
}

type ExpectedFlowMetrics struct {
	FramesRx     int64
	FramesRxRate float32
}

type ExpectedState struct {
	Port map[string]ExpectedPortMetrics
	Flow map[string]ExpectedFlowMetrics
	Bgp4 map[string]ExpectedBgpMetrics
	Bgp6 map[string]ExpectedBgpMetrics
	Isis map[string]ExpectedIsisMetrics
}

func NewExpectedState() ExpectedState {
	e := ExpectedState{
		Port: map[string]ExpectedPortMetrics{},
		Flow: map[string]ExpectedFlowMetrics{},
		Bgp4: map[string]ExpectedBgpMetrics{},
		Bgp6: map[string]ExpectedBgpMetrics{},
		Isis: map[string]ExpectedIsisMetrics{},
	}
	return e
}

func (client *GnmiClient) FlowMetricsOk(expectedState ExpectedState) (bool, error) {
	dNames := []string{}
	for name := range expectedState.Flow {
		dNames = append(dNames, name)
	}

	fMetrics, err := client.GetFlowMetrics(dNames)
	if err != nil {
		return false, err
	}

	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		FlowMetrics:   fMetrics,
	})

	expected := true
	for _, f := range fMetrics.Items() {
		expectedMetrics := expectedState.Flow[f.Name()]
		if f.FramesRx() != expectedMetrics.FramesRx || f.FramesRxRate() != expectedMetrics.FramesRxRate {
			expected = false
		}
	}

	return expected, nil
}

func (client *GnmiClient) AllBgp4SessionUp(expectedState ExpectedState) (bool, error) {
	dNames := []string{}
	for name := range expectedState.Bgp4 {
		dNames = append(dNames, name)
	}

	dMetrics, err := client.GetBgpv4Metrics(dNames)
	if err != nil {
		return false, err
	}

	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		Bgpv4Metrics:  dMetrics,
	})

	expected := true
	for _, d := range dMetrics.Items() {
		expectedMetrics := expectedState.Bgp4[d.Name()]
		if d.SessionState() != gosnappi.Bgpv4MetricSessionState.UP || d.RoutesAdvertised() != expectedMetrics.Advertised || d.RoutesReceived() != expectedMetrics.Received {
			expected = false
		}
	}

	return expected, nil
}

func (client *GnmiClient) AllBgp6SessionUp(expectedState ExpectedState) (bool, error) {
	dNames := []string{}
	for name := range expectedState.Bgp6 {
		dNames = append(dNames, name)
	}

	dMetrics, err := client.GetBgpv6Metrics(dNames)
	if err != nil {
		return false, err
	}

	PrintMetricsTable(&MetricsTableOpts{
		ClearPrevious: false,
		Bgpv6Metrics:  dMetrics,
	})

	expected := true
	for _, d := range dMetrics.Items() {
		expectedMetrics := expectedState.Bgp6[d.Name()]
		if d.SessionState() != gosnappi.Bgpv6MetricSessionState.UP || d.RoutesAdvertised() != expectedMetrics.Advertised || d.RoutesReceived() != expectedMetrics.Received {
			expected = false
		}
	}

	return expected, nil
}

func (client *GnmiClient) AllIsisSessionUp(expectedState ExpectedState, isisInterfaceLevelType gosnappi.IsisInterfaceLevelTypeEnum, expDatabaseSize int32) (bool, error) {
	// rNames := []string{}

	// dNames := []string{}
	// for name := range expectedState.Bgp6 {
	// 	dNames = append(dNames, name)
	// }

	// dMetrics, err := client.GetIsisMetrics(dNames)
	// if err != nil {
	// 	return false, err
	// }

	// PrintMetricsTable(&MetricsTableOpts{
	// 	ClearPrevious: false,
	// 	IsisMetrics:   dMetrics,
	// })

	// for _, router := range rNames {
	// 	routerFound := false
	// 	for _, d := range dMetrics.Items() {
	// 		name := d.Name()
	// 		if name == router {
	// 			routerFound = true
	// 			l1SessionUpCount := d.L1SessionsUp()
	// 			l2SessionUpCount := d.L2SessionsUp()
	// 			l1DatabaseSize := d.L1DatabaseSize()
	// 			l2DatabaseSize := d.L2DatabaseSize()

	// 			switch isisInterfaceLevelType {
	// 			case gosnappi.IsisInterfaceLevelType.LEVEL_1:
	// 				if l1SessionUpCount != 1 || l2SessionUpCount != 0 || l1DatabaseSize != expDatabaseSize {
	// 					return false, nil
	// 				}
	// 			case gosnappi.IsisInterfaceLevelType.LEVEL_2:
	// 				if l1SessionUpCount != 0 || l2SessionUpCount != 1 || l2DatabaseSize != expDatabaseSize {
	// 					return false, nil
	// 				}
	// 			case gosnappi.IsisInterfaceLevelType.LEVEL_1_2:
	// 				if l1SessionUpCount != 1 || l2SessionUpCount != 1 || l1DatabaseSize != expDatabaseSize || l2DatabaseSize != expDatabaseSize {
	// 					return false, nil
	// 				}
	// 			default:
	// 				return false, fmt.Errorf("invalid IS-IS interface level type : %v", isisInterfaceLevelType)
	// 			}
	// 		}
	// 	}
	// 	if !routerFound {
	// 		return false, nil
	// 	}
	// }

	return true, nil
}