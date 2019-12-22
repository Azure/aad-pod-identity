package metrics

import (
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"

	"k8s.io/klog"
)

// newPrometheusExporter creates prometheus exporter and run the same on given port
func newPrometheusExporter(namespace string, portNumber string) (*prometheus.Exporter, error) {

	prometheusExporter, err := prometheus.NewExporter(prometheus.Options{
		Namespace: namespace,
	})

	if err != nil {
		klog.Errorf("Failed to create the Prometheus exporter. error: %+v", err)
		return nil, err
	}
	klog.Info("Starting Prometheus exporter")
	// Run the Prometheus exporter as a scrape endpoint.
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", prometheusExporter)
		address := fmt.Sprintf(":%v", portNumber)
		if err := http.ListenAndServe(address, mux); err != nil {
			klog.Errorf("Failed to run Prometheus scrape endpoint: %v", err)
		}
	}()
	return prometheusExporter, nil
}
