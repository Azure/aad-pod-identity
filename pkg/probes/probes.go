package probes

import (
	"net/http"
	"time"

	"github.com/golang/glog"
)

// InitHealthProbe - sets up a health probe which responds with success (200 - OK) all the time once its called.
func InitHealthProbe() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
}

// InitReadyProbe - sets up a ready probe which returns success (200-OK) if the condition variable is set, else
// return an error (500). Also returned in the ready probe is the amount of time from the start of this container.
func InitReadyProbe(startTime time.Time, condition *bool) {
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if *condition {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(500)
		}
		data := (time.Since(startTime)).String()
		w.Write([]byte(data))
	})
}

//Start - Starts the required http server to start the probe to respond.
func Start(port string) {
	glog.Fatal(http.ListenAndServe(":"+port, nil))
}

// InitAndStart - Initialize the default probes and starts the http listening port.
func InitAndStart(port string, startTime time.Time, condition *bool) {
	InitHealthProbe()
	InitReadyProbe(startTime, condition)
	// start the probes.
	Start(port)
}
