package main

import (
	"flag"
	"github.com/pallasscat/redfish_exporter/collector"
	"github.com/pallasscat/redfish_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stmcginnis/gofish"
	"log"
	"net/http"
	"net/url"
	"os"
)

func handlerFunc(w http.ResponseWriter, r *http.Request, c *config.Config) {
	params := r.URL.Query()
	target := params.Get("target")
	if target == "" {
		http.Error(w, "target parameter is required", http.StatusBadRequest)
		return
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "target parameter is malformed", http.StatusBadRequest)
		return
	}

	tgt := url.URL{
		Scheme: targetURL.Scheme,
		Host:   targetURL.Host,
	}

	cfg, err := c.GetEndpointConfig(tgt.String())
	if err != nil {
		http.Error(w, "target not found in config file", http.StatusBadRequest)
		return
	}

	rc := collector.NewRedfishCollector(gofish.ClientConfig{
		Endpoint: target,
		Username: cfg.Username,
		Password: cfg.Password,
		Insecure: cfg.Insecure,
	})

	registry := prometheus.NewRegistry()
	registry.MustRegister(rc)

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func main() {
	var (
		listenAddress = flag.String("listen-address", "0.0.0.0:10015", "address for Prometheus requests")
		configPath    = flag.String("config-path", "./config.yml", "path to config file")
	)
	flag.Parse()

	log.SetFlags(log.Lmsgprefix)
	log.SetOutput(os.Stdout)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting redfish_exporter on %s", *listenAddress)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/redfish", func(w http.ResponseWriter, r *http.Request) {
		handlerFunc(w, r, cfg)
	})

	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatal(err)
	}
}
