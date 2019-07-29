package probes

import (
	"net/http"

	log "github.com/Azure/aad-pod-identity/pkg/logger"
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

func startAsync(port string, log log.Logger) {
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Errorf("Http listen and serve error: %+v", err)
		panic(err)
	} else {
		log.Info("Http listen and serve started !")
	}
}

//Start - Starts the required http server to start the probe to respond.
func Start(port string, log log.Logger) {
	go startAsync(port, log)
}

// InitAndStart - Initialize the default probes and starts the http listening port.
func InitAndStart(port string, condition *bool, log log.Logger) {
	InitHealthProbe(condition)
	log.Info("Initialized health probe")
	// start the probe.
	Start(port, log)
	log.Info("Started health probe")
}
