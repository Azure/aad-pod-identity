package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	auth "github.com/Azure/aad-pod-identity/pkg/auth"
	k8s "github.com/Azure/aad-pod-identity/pkg/k8s"
	iptables "github.com/Azure/aad-pod-identity/pkg/nmi/iptables"
	log "github.com/sirupsen/logrus"
)

const (
	iptableUpdateTimeIntervalInSeconds = 10
	localhost                          = "127.0.0.1"
)

// Server encapsulates all of the parameters necessary for starting up
// the server. These can be set via command line.
type Server struct {
	KubeClient                         k8s.Client
	NMIPort                            string
	MetadataIP                         string
	MetadataPort                       string
	HostIP                             string
	NodeName                           string
	IPTableUpdateTimeIntervalInSeconds int
}

type NMIResponse struct {
    Token adal.Token `json:"token"`
    ClientID string `json:"clientid"`
}

// NewServer will create a new Server with default values.
func NewServer() *Server {
	return &Server{}
}

// Run runs the specified Server.
func (s *Server) Run() error {
	go s.updateIPTableRules()

	mux := http.NewServeMux()
	mux.Handle("/metadata/identity/oauth2/token", appHandler(s.msiHandler))
	mux.Handle("/metadata/identity/oauth2/token/", appHandler(s.msiHandler))
	mux.Handle("/host/token", appHandler(s.hostHandler))
	mux.Handle("/host/token/", appHandler(s.hostHandler))
	mux.Handle("/", appHandler(s.defaultPathHandler))

	log.Infof("Listening on port %s", s.NMIPort)
	if err := http.ListenAndServe(":"+s.NMIPort, mux); err != nil {
		log.Fatalf("Error creating http server: %+v", err)
	}
	return nil
}

// updateIPTableRules ensures the correct iptable rules are set
// such that metadata requests are received by nmi assigned port
func (s *Server) updateIPTableRules() {
	log.Infof("node: %s", s.NodeName)
	podcidr, err := s.KubeClient.GetPodCidr(s.NodeName)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	for range time.Tick(time.Second * time.Duration(s.IPTableUpdateTimeIntervalInSeconds)) {
		log.Infof("node(%s) hostip(%s) podcidr(%s) metadataaddress(%s:%s) nmiport(%s)", s.NodeName, s.HostIP, podcidr, s.MetadataIP, s.MetadataPort, s.NMIPort)
		if err := iptables.AddCustomChain(podcidr, s.MetadataIP, s.MetadataPort, s.HostIP, s.NMIPort); err != nil {
			log.Fatalf("%s", err)
		}
		if err := iptables.LogCustomChain(); err != nil {
			log.Fatalf("%s", err)
		}
	}
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

// ServeHTTP implements the net/http server handler interface
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
			_, file, line, _ := runtime.Caller(3)
			stack := string(debug.Stack())
			switch t := rec.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = errors.New("Unknown error")
			}
			logger.WithField("res.status", http.StatusInternalServerError).
				Errorf("Panic processing request: %+v, file: %s, line: %d, stacktrace: '%s'", r, file, line, stack)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	rw := newResponseWriter(w)
	fn(logger, rw, r)
	latency := time.Since(start)
	logger.Infof("Status (%d) took %d ns", rw.statusCode, latency.Nanoseconds())
}

func (s *Server) hostHandler(logger *log.Entry, w http.ResponseWriter, r *http.Request) {
	hostIP := parseRemoteAddr(r.RemoteAddr)
	if hostIP != localhost {
		msg := "request remote address is not from a host"
		logger.Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	podns, podname := parseRequestHeader(r)
	if podns == "" || podname == "" {
		logger.Errorf("missing podname and podns from request")
		http.Error(w, "missing 'podname' and 'podns' from request header", http.StatusBadRequest)
		return
	}

	podIDs, err := s.KubeClient.ListPodIds(podns, podname)
	if err != nil || len(*podIDs) == 0 {
		msg := fmt.Sprintf("no AzureAssignedIdentity found for pod:%s/%s", podns, podname)
		logger.Errorf("%s, %+v", msg, err)
		http.Error(w, msg, http.StatusForbidden)
		return
	}
	rqClientID, rqResource := parseRequestClientIDAndResource(r)
	token, clientID, err := getTokenForMatchingID(logger, rqClientID, rqResource, podIDs)
	if err != nil {
		logger.Errorf("failed to get service principal token for pod:%s/%s, %+v", podns, podname, err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	nmiResp := NMIResponse{
		Token: *token,
		ClientID: clientID,
	}
	response, err := json.Marshal(nmiResp)
	if err != nil {
		logger.Errorf("failed to marshal service principal token and clientid for pod:%s/%s, %+v", podns, podname, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(response)
}

// msiHandler uses the remote address to identify the pod ip and uses it
// to find maching client id, and then returns the token sourced through
// AAD using adal
// if the requests contains client id it validates it againsts the admin
// configured id.
func (s *Server) msiHandler(logger *log.Entry, w http.ResponseWriter, r *http.Request) {
	podIP := parseRemoteAddr(r.RemoteAddr)
	if podIP == "" {
		msg := "request remote address is empty"
		logger.Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	podns, podname, err := s.KubeClient.GetPodName(podIP)
	if err != nil {
		logger.Errorf("missing podname for podip:%s, %+v", podIP, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	podIDs, err := s.KubeClient.ListPodIds(podns, podname)
	if err != nil || len(*podIDs) == 0 {
		msg := fmt.Sprintf("no AzureAssignedIdentity found for pod:%s/%s", podns, podname)
		logger.Errorf("%s, %+v", msg, err)
		http.Error(w, msg, http.StatusForbidden)
		return
	}
	rqClientID, rqResource := parseRequestClientIDAndResource(r)
	token, _, err := getTokenForMatchingID(logger, rqClientID, rqResource, podIDs)
	if err != nil {
		logger.Errorf("failed to get service principal token for pod:%s/%s, %+v", podns, podname, err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	response, err := json.Marshal(*token)
	if err != nil {
		logger.Errorf("failed to marshal service principal token for pod:%s/%s, %+v", podns, podname, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(response)
}

func getTokenForMatchingID(logger *log.Entry, rqClientID string, rqResource string, podIDs *[]aadpodid.AzureIdentity) (token *adal.Token, clientID string, err error) {
	rqHasClientID := len(rqClientID) != 0
	for _, v := range *podIDs {
		clientID := v.Spec.ClientID
		if rqHasClientID && !strings.EqualFold(rqClientID, clientID) {
			logger.Warningf("clientid mismatch, requested:%s available:%s", rqClientID, clientID)
			continue
		}
		idType := v.Spec.Type
		switch idType {
		case aadpodid.UserAssignedMSI:
			logger.Infof("matched identityType:%v clientid:%s resource:%s", idType, clientID, rqResource)
			token, err := auth.GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, rqResource)
			return token, clientID, err
		case aadpodid.ServicePrincipal:
			tenantid := v.Spec.TenantID
			logger.Infof("matched identityType:%v tenantid:%s clientid:%s", idType, tenantid, clientID)
			secret := v.Spec.ClientPassword.String()
			token, err := auth.GetServicePrincipalToken(tenantid, clientID, secret)
			return token, clientID, err
		default:
			return nil, clientID, fmt.Errorf("unsupported identity type %+v", idType)
		}
	}

	// We have not yet returned, so pass up an error
	return nil, "", fmt.Errorf("azureidentity is not configured for the pod")
}

func parseRequestHeader(r *http.Request) (podns string, podname string) {
	podns = r.Header.Get("podns")
	podname = r.Header.Get("podname")

	return podns, podname
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

func parseRequestClientIDAndResource(r *http.Request) (clientID string, resource string) {
	vals := r.URL.Query()
	if vals != nil {
		clientID = vals.Get("client_id")
		resource = vals.Get("resource")
	}
	return clientID, resource
}

// defaultPathHandler creates a new request and returns the response body and code
func (s *Server) defaultPathHandler(logger *log.Entry, w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil || req == nil {
		logger.Errorf("failed creating a new request, %s %+v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	host := fmt.Sprintf("%s:%s", s.MetadataIP, s.MetadataPort)
	req.Host = host
	req.URL.Host = host
	req.URL.Scheme = "http"
	if r.Header != nil {
		copyHeader(req.Header, r.Header)
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("failed executing request, %s %+v", req.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("failed io operation of reading response body, %+v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
