package nmi

import (
	"net"
	"net/http"
	"strings"
)

func parseRemoteAddr(addr string) string {
	n := strings.IndexByte(addr, ':')
	if n <= 1 {
		return ""
	}
	hostname := addr[0:n]
	if net.ParseIP(hostname) == nil {
		return ""
	}
	return hostname
}

// Server encapsulates all of the parameters necessary for starting up
// the server. These can either be set via command line or directly.
type Server struct {
	KubeClient Client,
	NMIPort string
}

// Run runs the specified Server.
func (s *Server) Run(host, token string, insecure bool) error {
	kubeClient, err := NewKubeClient()
	if err != nil {
		return err
	}
	s.KubeClient = kubeClient
	r := mux.NewRouter()
	r.Handle("/{version}/meta-data/msi/security-credentials/{role:.*}", appHandler(s.msiHandler))
	r.Handle("/healthz", appHandler(s.healthHandler))
	r.Handle("/{path:.*}", appHandler(s.reverseProxyHandler))

	log.Infof("Listening on port %s", s.NMIPort)
	if err := http.ListenAndServe(":"+s.NMIPort, r); err != nil {
		log.Fatalf("Error creating http server: %+v", err)
	}
	return nil
}
