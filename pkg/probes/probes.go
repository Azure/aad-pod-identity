package probes

import (
	"net/http"

	"github.com/golang/glog"
)

// InitHealthProbe - sets up a health probe which responds with success (200 - OK) once its initialized.
// The contents of the healthz endpoint will be the string "Active" if the condition is satisfied.
// The condition is set to true when the sync cycle has become active in case of MIC and the iptables
// rules set in case of NMI.
func InitHealthProbe(condition *bool) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if *condition {
			w.Write([]byte("Active"))
		} else {
			w.Write([]byte("Not Active"))
		}
	})
}

func startAsync(port string) {
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		glog.Fatalf("Http listen and serve error: %+v", err)
	} else {
		glog.V(1).Infof("Http listen and serve started !")
	}
}

//Start - Starts the required http server to start the probe to respond.
func Start(port string) {
	go startAsync(port)
}

// InitAndStart - Initialize the default probes and starts the http listening port.
func InitAndStart(port string, condition *bool) {
	InitHealthProbe(condition)
	glog.V(1).Infof("Initialized health probe")
	// start the probe.
	Start(port)
	glog.Infof("Initialized and started health probe")
}
