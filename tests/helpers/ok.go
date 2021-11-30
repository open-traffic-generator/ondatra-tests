package helpers

import (
	"github.com/open-traffic-generator/snappi/gosnappi"
)

type ExpectedBgpMetrics struct {
	Advertised int32
	Received   int32
}

type ExpectedPortMetrics struct {
	FramesRx int32
}

type ExpectedFlowMetrics struct {
	FramesRx     int32
	FramesRxRate float32
}

type ExpectedState struct {
	Port map[string]ExpectedPortMetrics
	Flow map[string]ExpectedFlowMetrics
	Bgp4 map[string]ExpectedBgpMetrics
	Bgp6 map[string]ExpectedBgpMetrics
}

func NewExpectedState() ExpectedState {
	e := ExpectedState{
		Port: map[string]ExpectedPortMetrics{},
		Flow: map[string]ExpectedFlowMetrics{},
		Bgp4: map[string]ExpectedBgpMetrics{},
		Bgp6: map[string]ExpectedBgpMetrics{},
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
