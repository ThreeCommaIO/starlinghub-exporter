package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	prom_version "github.com/prometheus/common/version"
	"github.com/threecommaio/starlinghub"
	"github.com/threecommaio/starlinghub_exporter/pkg/version"
)

const (
	namespace = "starlinghub"
)

var (
	deviceContactState = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "device_contact_state"),
		"Device contact state from Starling Hub",
		[]string{"where", "name"}, nil,
	)
)

// Exporter collects stats and exports them using
// the prometheus metrics package.
type Exporter struct {
	Key string
	URL string
}

// NewExporter returns an initialized Exporter.
func NewExporter(key, url string) (*Exporter, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}
	return &Exporter{key, url}, nil
}

// Describe describes all the metrics ever exported by the fast exporter.
// It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- deviceContactState
}

// Collect fetches the stats from starling hub connect api and delivers them
// as Prometheus metrics.
// It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	log.Infof("starling hub connect exporter running")

	client := starlinghub.New(e.URL, e.Key, nil)
	devices, _, err := client.ListDevices()
	if err != nil {
		log.Error(err)
		return
	}
	for _, device := range devices {
		if device.Type == "detect" {
			prop, _, err := client.GetDevice(device.ID)
			if err != nil {
				log.Error(err)
				return
			}
			state := float64(0)
			switch prop.ContactState {
			case "closed":
				state = 0
			case "open":
				state = 1
			default:
				state = -1
			}
			ch <- prometheus.MustNewConstMetric(deviceContactState, prometheus.GaugeValue, state, device.Where, device.Name)
		}
	}

	log.Infof("starling hub connect exporter finished")
}

func init() {
	prometheus.MustRegister(prom_version.NewCollector("starlinghub_exporter"))
}

func main() {
	var (
		enableVersion = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9112", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		key           = flag.String("key", "", "Starling Hub API key")
		url           = flag.String("url", "", "Starling Hub API url")
	)
	flag.Parse()

	if *enableVersion {
		fmt.Print(version.Version)
		os.Exit(0)
	}
	exporter, err := NewExporter(*key, *url)
	if err != nil {
		log.Errorf("can't create exporter : %s", err)
		os.Exit(1)
	}
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())

	log.Infoln("listening on", *listenAddress)
	log.Infof("serving metrics at path: %s", *metricsPath)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
