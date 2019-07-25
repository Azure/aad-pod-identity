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
		// Readiness probe being 500 causes too many events to be produced.
		// Hence we post the real status as result of the readiness probe instead of
		// marking the state as not ready.
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
func InitAndStart(port string, startTime time.Time, condition *bool) {
	InitHealthProbe()
	glog.V(1).Infof("Initialized health probe")
	InitReadyProbe(startTime, condition)
	glog.V(1).Infof("Initialized readiness probe")
	// start the probes.
	Start(port)
	glog.Infof("Initialized and started health and readiness probe")
}
