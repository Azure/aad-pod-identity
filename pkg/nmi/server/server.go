package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	aadpodidentity "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	auth "github.com/Azure/aad-pod-identity/pkg/auth"
	k8s "github.com/Azure/aad-pod-identity/pkg/k8s"
	log "github.com/sirupsen/logrus"
)

const (
	defaultMetadataIP   = "169.254.169.254"
	defaultMetadataPort = "80"
	defaultNmiPort      = "2579"
)

// Server encapsulates all of the parameters necessary for starting up
// the server. These can either be set via command line or directly.
type Server struct {
	KubeClient   k8s.Client
	NMIPort      string
	MetadataIP   string
	MetadataPort string
	Host         string
}

// NewServer will create a new Server with default values.
func NewServer() *Server {
	return &Server{
		MetadataIP:   defaultMetadataIP,
		MetadataPort: defaultMetadataPort,
		NMIPort:      defaultNmiPort,
	}
}

// Run runs the specified Server.
func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.Handle("/metadata/identity/oauth2/token", appHandler(s.roleHandler))
	mux.Handle("/{path:.*}", appHandler(s.reverseProxyHandler))

	log.Infof("Listening on port %s", s.NMIPort)
	if err := http.ListenAndServe(":"+s.NMIPort, mux); err != nil {
		log.Fatalf("Error creating http server: %+v", err)
	}
	return nil
}

type appHandler func(*log.Entry, http.ResponseWriter, *http.Request)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// ServeHTTP implements the net/http server Handler interface
// and recovers from panics.
func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{
		"req.method": r.Method,
		"req.path":   r.URL.Path,
		"req.remote": parseRemoteAddr(r.RemoteAddr),
	})
	start := time.Now()
	defer func() {
		var err error
		if rec := recover(); rec != nil {
			switch t := rec.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = errors.New("Unknown error")
			}
			logger.WithField("res.status", http.StatusInternalServerError).
				Errorf("PANIC error processing request: %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	rw := newResponseWriter(w)
	fn(logger, rw, r)
	if r.URL.Path != "/healthz" {
		latency := time.Since(start)
		logger.WithFields(log.Fields{"res.duration": latency.Nanoseconds(), "res.status": rw.statusCode}).
			Infof("%s %s (%d) took %d ns", r.Method, r.URL.Path, rw.statusCode, latency.Nanoseconds())
	}
}

func (s *Server) roleHandler(logger *log.Entry, w http.ResponseWriter, r *http.Request) {
	podIP := parseRemoteAddr(r.RemoteAddr)
	log.Infof("received request %s", podIP)

	podns, podname, err := s.KubeClient.GetPodName(podIP)
	if err != nil {
		logger.Errorf("Error getting podname for podip:%s, %+v", podIP, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	azID, err := s.KubeClient.GetAzureAssignedIdentity(podns, podname)
	if err != nil {
		logger.Errorf("No AzureAssignedIdentity found for pod:%s/%s, %+v", podns, podname, err)
		msg := fmt.Sprintf("No AzureAssignedIdentity found for pod:%s/%s", podns, podname)
		http.Error(w, msg, http.StatusForbidden)
		return
	}
	logger.Infof("found matching azID %s", azID.Name)
	rqClientID := parseRequestClientID(r)
	logger.Infof("request clientid %s", rqClientID)

	switch azID.Spec.Type {
	case aadpodidentity.UserAssignedMSI:
		token, err := auth.GetServicePrincipalToken(rqClientID, "")
		if err != nil {
			logger.Errorf("failed to get service pricipal token, %+v", err)
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		response, err := json.Marshal(*token)
		if err != nil {
			logger.Errorf("failed to Marshal service pricipal token, %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(response)
		break
	default:
		break
	}
}

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

func parseRequestClientID(r *http.Request) (clientID string) {
	vals := r.URL.Query()
	if vals != nil {
		clientID = vals.Get("client_id")
	}
	return clientID
}

func (s *Server) reverseProxyHandler(logger *log.Entry, w http.ResponseWriter, r *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: s.MetadataIP})
	proxy.ServeHTTP(w, r)
	logger.WithField("metadata.url", s.MetadataIP).Debug("proxy metadata request")
}
