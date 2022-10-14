package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	"log"
	"sync"
	"time"
)

const namespace = "redfish"

type Collector interface {
	Collect(chan<- prometheus.Metric) error
}

type RedfishCollector struct {
	config gofish.ClientConfig
	upDesc *prometheus.Desc
}

func NewRedfishCollector(config gofish.ClientConfig) *RedfishCollector {
	return &RedfishCollector{
		config: config,
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Redfish service status; 0: Down, 1: Up",
			nil, nil,
		),
	}
}

func (c *RedfishCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upDesc
}

func (c *RedfishCollector) Collect(ch chan<- prometheus.Metric) {
	log.SetPrefix(fmt.Sprintf("endpoint %s: ", c.config.Endpoint))

	client, err := gofish.Connect(c.config)
	if err != nil {
		log.Printf("error connecting to Redfish server: %s", err)
		ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 0)
		return
	}
	defer client.Logout()

	collectors := map[string]Collector{
		"chassis": &ChassisCollector{client},
		"system":  &SystemCollector{client},
		"manager": &ManagerCollector{client},
	}

	wg := sync.WaitGroup{}
	wg.Add(len(collectors))
	for name, collector := range collectors {
		go func(name string, collector Collector) {
			execute(name, collector, ch)
			wg.Done()
		}(name, collector)
	}
	wg.Wait()

	ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 1)
}

func execute(name string, collector Collector, ch chan<- prometheus.Metric) {
	durationDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "duration_seconds"),
		"Scrape duration, s",
		nil, prometheus.Labels{"collector": name},
	)
	successDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "success"),
		"Scrape success; 0: Fail, 1: Success",
		nil, prometheus.Labels{"collector": name},
	)

	start := time.Now()
	err := collector.Collect(ch)
	duration := time.Since(start).Seconds()

	var success float64 = 1
	if err != nil {
		log.Printf("collector %s failed: %s", name, err)
		success = 0
	}

	ch <- prometheus.MustNewConstMetric(durationDesc, prometheus.GaugeValue, duration)
	ch <- prometheus.MustNewConstMetric(successDesc, prometheus.GaugeValue, success)
}
