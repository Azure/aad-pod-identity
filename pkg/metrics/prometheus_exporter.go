package metrics

import (
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
	log "github.com/Azure/aad-pod-identity/pkg/logger"
)

// newPrometheusExporter creates prometheus exporter and run the same on given port
func newPrometheusExporter(namespace string, portNumber string, log log.Logger) (*prometheus.Exporter, error) {

	prometheusExporter, err := prometheus.NewExporter(prometheus.Options{
		Namespace: namespace,
	})

	if err != nil {
		log.Errorf("Failed to create the Prometheus exporter. error: %+v", err)
		return nil, err
	}
	log.Info("Starting Prometheus exporter")
	// Run the Prometheus exporter as a scrape endpoint.
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", prometheusExporter)
		address := fmt.Sprintf(":%v", portNumber)
		if err := http.ListenAndServe(address, mux); err != nil {
			log.Errorf("Failed to run Prometheus scrape endpoint: %v", err)
		}
	}()
	return prometheusExporter, nil
}
