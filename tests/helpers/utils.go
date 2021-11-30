package helpers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// using protojson to marshal will emit property names with lowerCamelCase
// instead of snake_case
var protoMarshaller = protojson.MarshalOptions{UseProtoNames: true}
var prettyProtoMarshaller = protojson.MarshalOptions{UseProtoNames: true, Multiline: true}

type WaitForOpts struct {
	Condition string
	Interval  time.Duration
	Timeout   time.Duration
}

type MetricsTableOpts struct {
	ClearPrevious bool
	FlowMetrics   gosnappi.MetricsResponseFlowMetricIter
	PortMetrics   gosnappi.MetricsResponsePortMetricIter
	Bgpv4Metrics  gosnappi.MetricsResponseBgpv4MetricIter
	Bgpv6Metrics  gosnappi.MetricsResponseBgpv6MetricIter
}

func Timer(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %d ms", name, elapsed.Milliseconds())
}

func LogWarnings(warnings []string) {
	for _, w := range warnings {
		log.Printf("WARNING: %v", w)
	}
}

func LogErrors(errors *[]string) error {
	if errors == nil {
		return fmt.Errorf("")
	}
	for _, e := range *errors {
		log.Printf("ERROR: %v", e)
	}

	return fmt.Errorf("%v", errors)
}

func PrettyStructString(v interface{}) string {
	var bytes []byte
	var err error

	switch v := v.(type) {
	case protoreflect.ProtoMessage:
		bytes, err = prettyProtoMarshaller.Marshal(v)
		if err != nil {
			log.Println(err)
			return ""
		}
	default:
		bytes, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			log.Println(err)
			return ""
		}
	}

	return string(bytes)
}

func ProtoToJsonStruct(in protoreflect.ProtoMessage, out interface{}) error {
	log.Println("Marshalling from proto to json struct ...")

	bytes, err := protoMarshaller.Marshal(in)
	if err != nil {
		return fmt.Errorf("could not marshal from proto to json: %v", err)
	}
	if err := json.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("could not unmarshal from json to struct: %v", err)
	}
	return nil
}

func JsonStructToProto(in interface{}, out protoreflect.ProtoMessage) error {
	log.Println("Marshalling from struct to json ... ")

	bytes, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("could not marshal from struct to json: %v", err)
	}
	if err := protojson.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("could not unmarshal from json to proto: %v", err)
	}
	return nil
}

func WaitFor(t *testing.T, fn func() (bool, error), opts *WaitForOpts) error {
	if opts == nil {
		opts = &WaitForOpts{
			Condition: "condition to be true",
		}
	}
	defer Timer(time.Now(), fmt.Sprintf("Waiting for %s", opts.Condition))

	if opts.Interval == 0 {
		opts.Interval = 500 * time.Millisecond
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	start := time.Now()
	log.Printf("Waiting for %s ...\n", opts.Condition)

	for {
		done, err := fn()
		if err != nil {
			t.Fatal(fmt.Errorf("error waiting for %s: %v", opts.Condition, err))
		}
		if done {
			log.Printf("Done waiting for %s\n", opts.Condition)
			return nil
		}

		if time.Since(start) > opts.Timeout {
			t.Fatal(fmt.Errorf("timeout occurred while waiting for %s", opts.Condition))
		}
		time.Sleep(opts.Interval)
	}
}

func ClearScreen() {
	switch runtime.GOOS {
	case "darwin":
		fallthrough
	case "linux":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		return
	}
}

func PrintMetricsTable(opts *MetricsTableOpts) {
	if opts == nil {
		return
	}
	out := "\n"

	if opts.Bgpv4Metrics != nil {
		border := strings.Repeat("-", 20*9+5)
		out += "\nBgpv4 Metrics\n" + border + "\n"
		out += fmt.Sprintf(
			"%-20s%-20s%-20s%-20s%-20s%-20s%-20s%-20s%-20s\n",
			"Name",
			"Session State",
			"Session Flaps",
			"Routes Advertised",
			"Routes Received",
			"Route Withdraws Tx",
			"Route Withdraws Rx",
			"Keepalives Tx",
			"Keepalives Rx",
		)
		for _, d := range opts.Bgpv4Metrics.Items() {
			if d != nil {
				name := d.Name()
				sessionState := d.SessionState()
				sessionFlapCount := d.SessionFlapCount()
				routesAdvertised := d.RoutesAdvertised()
				routesReceived := d.RoutesReceived()
				keepalivesSent := d.KeepalivesSent()
				keepalivesReceived := d.KeepalivesReceived()
				routeWithdrawsSent := d.RouteWithdrawsSent()
				routeWithdrawsReceived := d.RouteWithdrawsReceived()
				out += fmt.Sprintf(
					"%-20v%-20v%-20v%-20v%-20v%-20v%-20v%-20v%-20v\n",
					name,
					sessionState,
					sessionFlapCount,
					routesAdvertised,
					routesReceived,
					routeWithdrawsSent,
					routeWithdrawsReceived,
					keepalivesSent,
					keepalivesReceived,
				)
			}
		}
		out += border + "\n\n"
	}

	if opts.Bgpv6Metrics != nil {
		border := strings.Repeat("-", 20*9+5)
		out += "\nBgpv6 Metrics\n" + border + "\n"
		out += fmt.Sprintf(
			"%-20s%-20s%-20s%-20s%-20s%-20s%-20s%-20s%-20s\n",
			"Name",
			"Session State",
			"Session Flaps",
			"Routes Advertised",
			"Routes Received",
			"Route Withdraws Tx",
			"Route Withdraws Rx",
			"Keepalives Tx",
			"Keepalives Rx",
		)
		for _, d := range opts.Bgpv6Metrics.Items() {
			if d != nil {
				name := d.Name()
				sessionState := d.SessionState()
				sessionFlapCount := d.SessionFlapCount()
				routesAdvertised := d.RoutesAdvertised()
				routesReceived := d.RoutesReceived()
				keepalivesSent := d.KeepalivesSent()
				keepalivesReceived := d.KeepalivesReceived()
				routeWithdrawsSent := d.RouteWithdrawsSent()
				routeWithdrawsReceived := d.RouteWithdrawsReceived()
				out += fmt.Sprintf(
					"%-20v%-20v%-20v%-20v%-20v%-20v%-20v%-20v%-20v\n",
					name,
					sessionState,
					sessionFlapCount,
					routesAdvertised,
					routesReceived,
					routeWithdrawsSent,
					routeWithdrawsReceived,
					keepalivesSent,
					keepalivesReceived,
				)
			}
		}
		out += border + "\n\n"
	}

	if opts.PortMetrics != nil {
		border := strings.Repeat("-", 15*4+5)
		out += "\nPort Metrics\n" + border + "\n"
		out += fmt.Sprintf(
			"%-15s%-15s%-15s%-15s\n",
			"Name", "Frames Tx", "Frames Rx", "FPS Tx",
		)
		for _, m := range opts.PortMetrics.Items() {
			if m != nil {
				name := m.Name()
				tx := m.FramesTx()
				rx := m.FramesRx()
				txRate := m.FramesTxRate()

				out += fmt.Sprintf(
					"%-15v%-15v%-15v%-15v\n",
					name, tx, rx, txRate,
				)
			}
		}
		out += border + "\n\n"
	}

	if opts.FlowMetrics != nil {
		border := strings.Repeat("-", 15*3+5)
		out += "\nFlow Metrics\n" + border + "\n"
		out += fmt.Sprintf("%-15s%-15s%-15s\n", "Name", "Frames Rx", "FPS Rx")
		for _, m := range opts.FlowMetrics.Items() {
			if m != nil {
				name := m.Name()
				rx := m.FramesRx()
				rxRate := m.FramesRxRate()
				out += fmt.Sprintf("%-15v%-15v%-15v\n", name, rx, rxRate)
			}
		}
		out += border + "\n\n"
	}

	if opts.ClearPrevious {
		ClearScreen()
	}
	log.Println(out)
}

func GetCapturePorts(c gosnappi.Config) []string {
	capturePorts := []string{}
	if c == nil {
		return capturePorts
	}

	for _, capture := range c.Captures().Items() {
		capturePorts = append(capturePorts, capture.PortNames()...)
	}
	return capturePorts
}
