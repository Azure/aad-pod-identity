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
	azureResourceName                  = "https://management.azure.com/"
	iptableUpdateTimeIntervalInSeconds = 10
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
		logger.Errorf("Error getting podname for podip:%s, %+v", podIP, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	azIDs, err := s.KubeClient.GetUserAssignedIdentities(podns, podname)
	if err != nil || len(*azIDs) == 0 {
		msg := fmt.Sprintf("no AzureAssignedIdentity found for pod:%s/%s", podns, podname)
		logger.Errorf("%s, %+v", msg, err)
		http.Error(w, msg, http.StatusForbidden)
		return
	}
	rqClientID := parseRequestClientID(r)
	token, err := getTokenForMatchingID(logger, rqClientID, azIDs)
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

func getTokenForMatchingID(logger *log.Entry, rqClientID string, azIDs *[]aadpodid.AzureAssignedIdentity) (token *adal.Token, err error) {
	rqHasClientID := len(rqClientID) != 0
	for _, v := range *azIDs {
		clientID := v.Spec.AzureIdentityRef.Spec.ClientID
		if rqHasClientID && !strings.EqualFold(rqClientID, clientID) {
			logger.Warningf("clientid mismatch, requested:%s available:%s", rqClientID, clientID)
			continue
		}
		idType := v.Spec.AzureIdentityRef.Spec.Type
		switch idType {
		case aadpodid.UserAssignedMSI:
			logger.Infof("matched identityType:%v clientid:%s resource:%s", idType, clientID, azureResourceName)
			return auth.GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, azureResourceName)
		case aadpodid.ServicePrincipal:
			tenantid := ""
			logger.Infof("matched identityType:%v tenantid:%s clientid:%s", idType, tenantid, clientID)
			secret := v.Spec.AzureIdentityRef.Spec.Password.String()
			return auth.GetServicePrincipalToken(tenantid, clientID, secret)
		default:
			return nil, fmt.Errorf("unsupported identity type %+v", idType)
		}
	}

	// We have not yet returned, so pass up an error
	return nil, fmt.Errorf("azureidentity is not configured for the pod")
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

// defaultPathHandler creates a new request and returns the response body and code
func (s *Server) defaultPathHandler(logger *log.Entry, w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	r.URL.Host = s.MetadataIP
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	req.Host = s.MetadataIP
	copyHeader(req.Header, r.Header)
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), resp.StatusCode)
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
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
