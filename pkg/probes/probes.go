package probes

import (
	"encoding/json"
	"net/http"

	msg "github.com/Microsoft/hcnproxy/pkg/types"
	"k8s.io/klog/v2"

	hcnclient "github.com/Microsoft/hcnproxy/pkg/client"
)

// InitHealthProbe - sets up a health probe which responds with success (200 - OK) once its initialized.
// The contents of the healthz endpoint will be the string "Active" if the condition is satisfied.
// The condition is set to true when the sync cycle has become active in case of MIC and the iptables
// rules set in case of NMI.
func InitHealthProbe(condition *bool) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if *condition {
			_, _ = w.Write([]byte("Active"))
		} else {
			_, _ = w.Write([]byte("Not Active"))
		}
	})
}

func startAsync(port string) {
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		klog.Fatalf("http listen and serve error: %+v", err)
	}

	klog.Info("http listen and serve started !")
}

// Start starts the required http server to start the probe to respond.
func Start(port string) {
	go startAsync(port)
}

// InitAndStart initializes the default probes and starts the http listening port.
func InitAndStart(port string, condition *bool) {
	InitHealthProbe(condition)
	klog.Infof("Initialized health probe on port %s", port)

	// Start the probe.
	Start(port)
	klog.Info("started health probe")
}

// InitAndStartNMIWindowsProbe - Initialize the nmi windows probes and starts the http listening port.
func InitAndStartNMIWindowsProbe(port string, condition *bool, node string) {
	initNMIWindowsHealthProbe(condition, node)
	klog.Infof("Initialized nmi Windows health probe on port %s", port)

	// Start the nmi windows probe.
	Start(port)
	klog.Info("Started NMI Windows health probe")
}

func initNMIWindowsHealthProbe(condition *bool, nodeName string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {

		klog.Info("Started to handle healthz: %s", nodeName)

		request := msg.HNSRequest{
			Entity:    msg.EndpointV1,
			Operation: msg.Enumerate,
			Request:   nil,
		}

		klog.Info("Started to call hcn agent.")

		res := hcnclient.InvokeHNSRequest(request)
		if res.Error != nil {
			klog.Info("Call hcn agent failed with error: %+v", res.Error)
			w.WriteHeader(500)
		} else {
			klog.Info("Call hcn agent Successfully.")

			b, _ := json.Marshal(res)
			klog.Infof("Server response: %s", string(b))
			w.WriteHeader(200)
		}

		if *condition {
			w.Write([]byte("Active"))
		} else {
			w.Write([]byte("Not Active"))
		}
	})
}
