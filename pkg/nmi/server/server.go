package server

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	aadpodidentity "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	k8s "github.com/Azure/aad-pod-identity/pkg/k8s"
	auth "github.com/Azure/aad-pod-identity/pkg/nmi/auth"
	log "github.com/sirupsen/logrus"
)

const (
	defaultMetadataAddress = "169.254.169.254"
	defaultNmiPort         = "2579"
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
	KubeClient      k8s.Client
	NMIPort         string
	MetadataAddress string
	Host            string
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

type msiTokenRequestBody struct {
	Resource string `json:"resource"`
}

func (s *Server) roleHandler(logger *log.Entry, w http.ResponseWriter, r *http.Request) {
	podIP := parseRemoteAddr(r.RemoteAddr)
	podns, podname, err := s.KubeClient.GetPodName(podIP)
	if err != nil {

	}
	azID, err := s.KubeClient.GetAzureAssignedIdentity(podns, podname)
	if err != nil {

	}

	validateRequestClientID := false
	rqClientID, err := parseRequestClientID(r.Body)
	if err != nil {
		validateRequestClientID = true
	}
	if validateRequestClientID {
		log.Infof("request clientid validation %s %v", rqClientID, validateRequestClientID)
	}

	switch azID.Spec.Type {
	case aadpodidentity.UserAssignedMSI:
		token, err = auth.GetServicePrincipalToken(rqClientID, "")
		response, err := json.Marshal(*token)
		if err != nil {
			return
		}
		w.Write(response)
		break
	default:
		break
	}
}

func parseRequestClientID(r io.Reader) (clientID string, err error) {
	dec := json.NewDecoder(r)
	clientID = ""
	for {
		var trb msiTokenRequestBody
		if err := dec.Decode(&trb); err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		split := strings.Split(trb.Resource, "client_id=")
		clientID = split[1]
	}

	return clientID, err
}

func (s *Server) reverseProxyHandler(logger *log.Entry, w http.ResponseWriter, r *http.Request) {
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: s.MetadataAddress})
	proxy.ServeHTTP(w, r)
	logger.WithField("metadata.url", s.MetadataAddress).Debug("proxy metadata request")
}

// Run runs the specified Server.
func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.Handle("/oauth2/token{role:.*}", appHandler(s.roleHandler))
	mux.Handle("/{path:.*}", appHandler(s.reverseProxyHandler))

	log.Infof("Listening on port %s", s.NMIPort)
	if err := http.ListenAndServe(":"+s.NMIPort, mux); err != nil {
		log.Fatalf("Error creating http server: %+v", err)
	}
	return nil
}

// NewServer will create a new Server with default values.
func NewServer() *Server {
	return &Server{
		MetadataAddress: defaultMetadataAddress,
		NMIPort:         defaultNmiPort,
	}
}
