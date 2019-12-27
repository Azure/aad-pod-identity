package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	auth "github.com/Azure/aad-pod-identity/pkg/auth"
	k8s "github.com/Azure/aad-pod-identity/pkg/k8s"
	"github.com/Azure/aad-pod-identity/pkg/metrics"
	iptables "github.com/Azure/aad-pod-identity/pkg/nmi/iptables"
	"github.com/Azure/aad-pod-identity/pkg/pod"
	utils "github.com/Azure/aad-pod-identity/pkg/utils"
	"github.com/Azure/go-autorest/autorest/adal"
	"k8s.io/klog"
)

const (
	localhost = "127.0.0.1"
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
	IsNamespaced                       bool
	MICNamespace                       string
	Initialized                        bool
	BlockInstanceMetadata              bool

	ListPodIDsRetryAttemptsForCreated  int
	ListPodIDsRetryAttemptsForAssigned int
	ListPodIDsRetryIntervalInSeconds   int
	Reporter                           *metrics.Reporter
}

// NMIResponse is the response returned to caller
type NMIResponse struct {
	Token    msiResponse `json:"token"`
	ClientID string      `json:"clientid"`
}

// NewServer will create a new Server with default values.
func NewServer(isNamespaced bool, micNamespace string, blockInstanceMetadata bool) *Server {
	reporter, err := metrics.NewReporter()
	if err != nil {
		klog.Errorf("Error creating new reporter to emit metrics: %v", err)
	} else {
		// keeping this reference to be used in ServeHTTP, as server is not accessible in ServeHTTP
		appHandlerReporter = reporter
		auth.InitReporter(reporter)
	}
	return &Server{
		IsNamespaced:          isNamespaced,
		MICNamespace:          micNamespace,
		BlockInstanceMetadata: blockInstanceMetadata,
		Reporter:              reporter,
	}
}

// Run runs the specified Server.
func (s *Server) Run() error {
	go s.updateIPTableRules()

	mux := http.NewServeMux()
	mux.Handle("/metadata/identity/oauth2/token", appHandler(s.msiHandler))
	mux.Handle("/metadata/identity/oauth2/token/", appHandler(s.msiHandler))
	mux.Handle("/host/token", appHandler(s.hostHandler))
	mux.Handle("/host/token/", appHandler(s.hostHandler))
	if s.BlockInstanceMetadata {
		mux.Handle("/metadata/instance", http.HandlerFunc(forbiddenHandler))
	}
	mux.Handle("/", appHandler(s.defaultPathHandler))

	klog.Infof("Listening on port %s", s.NMIPort)
	if err := http.ListenAndServe(":"+s.NMIPort, mux); err != nil {
		klog.Fatalf("Error creating http server: %+v", err)
	}
	return nil
}

func (s *Server) updateIPTableRulesInternal() {
	klog.V(5).Infof("node(%s) hostip(%s) metadataaddress(%s:%s) nmiport(%s)", s.NodeName, s.HostIP, s.MetadataIP, s.MetadataPort, s.NMIPort)

	if err := iptables.AddCustomChain(s.MetadataIP, s.MetadataPort, s.HostIP, s.NMIPort); err != nil {
		klog.Fatalf("%s", err)
	}
	if err := iptables.LogCustomChain(); err != nil {
		klog.Fatalf("%s", err)
	}
}

// updateIPTableRules ensures the correct iptable rules are set
// such that metadata requests are received by nmi assigned port
// NOT originating from HostIP destined to metadata endpoint are
// routed to NMI endpoint
func (s *Server) updateIPTableRules() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(time.Second * time.Duration(s.IPTableUpdateTimeIntervalInSeconds))
	defer ticker.Stop()

	// Run once before the waiting on ticker for the rules to take effect
	// immediately.
	s.updateIPTableRulesInternal()
	s.Initialized = true

loop:
	for {
		select {
		case <-signalChan:
			handleTermination()
			break loop

		case <-ticker.C:
			s.updateIPTableRulesInternal()
		}
	}
}

type appHandler func(http.ResponseWriter, *http.Request) string

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

var appHandlerReporter *metrics.Reporter

// ServeHTTP implements the net/http server handler interface
// and recovers from panics.
func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tracker := fmt.Sprintf("req.method=%s reg.path=%s req.remote=%s", r.Method, r.URL.Path, parseRemoteAddr(r.RemoteAddr))

	// Set the header in advance so that both success as well
	// as error paths have it set as application/json content type.
	w.Header().Set("Content-Type", "application/json")
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
			klog.Errorf("Panic processing request: %+v, file: %s, line: %d, stacktrace: '%s' %s res.status=%d", r, file, line, stack, tracker, http.StatusInternalServerError)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	rw := newResponseWriter(w)
	ns := fn(rw, r)
	latency := time.Since(start)
	klog.Infof("Status (%d) took %d ns", rw.statusCode, latency.Nanoseconds())

	_, resource := parseRequestClientIDAndResource(r)

	if appHandlerReporter != nil {
		appHandlerReporter.ReportOperationAndStatus(
			r.URL.Path,
			strconv.Itoa(rw.statusCode),
			ns,
			resource,
			metrics.NMIOperationsDurationM.M(metrics.SinceInSeconds(start)))
	}
}

func (s *Server) hostHandler(w http.ResponseWriter, r *http.Request) (ns string) {
	hostIP := parseRemoteAddr(r.RemoteAddr)
	rqClientID, rqResource := parseRequestClientIDAndResource(r)

	podns, podname := parseRequestHeader(r)
	if podns == "" || podname == "" {
		klog.Error("missing podname and podns from request")
		http.Error(w, "missing 'podname' and 'podns' from request header", http.StatusBadRequest)
		return
	}
	// set the ns so it can be used for metrics
	ns = podns
	if hostIP != localhost {
		klog.Errorf("request remote address is not from a host")
		http.Error(w, "request remote address is not from a host", http.StatusInternalServerError)
		return
	}
	if !validateResourceParamExists(rqResource) {
		klog.Warning("parameter resource cannot be empty")
		http.Error(w, "parameter resource cannot be empty", http.StatusBadRequest)
		return
	}
	podIDs, identityInCreatedStateFound, err := s.listPodIDsWithRetry(r.Context(), s.KubeClient, podns, podname, rqClientID)
	if err != nil {
		msg := fmt.Sprintf("no AzureAssignedIdentity found for pod:%s/%s in desired state", podns, podname)
		klog.Errorf("%s, %+v", msg, err)
		http.Error(w, msg, getErrorResponseStatusCode(identityInCreatedStateFound))
		return
	}

	// filter out if we are in namespaced mode
	filterPodIdentities := []aadpodid.AzureIdentity{}
	for _, val := range podIDs {
		if s.IsNamespaced || aadpodid.IsNamespacedIdentity(&val) {
			// namespaced mode
			if val.Namespace == podns {
				// matched namespace
				filterPodIdentities = append(filterPodIdentities, val)
			} else {
				// unmatched namespaced
				klog.Errorf("pod:%s/%s has identity %s/%s but identity is namespaced will be ignored", podns, podname, val.Name, val.Namespace)
			}
		} else {
			// not in namespaced mode
			filterPodIdentities = append(filterPodIdentities, val)
		}
	}
	podIDs = filterPodIdentities
	token, clientID, err := getTokenForMatchingID(s.KubeClient, rqClientID, rqResource, podIDs)
	if err != nil {
		klog.Errorf("failed to get service principal token for pod:%s/%s, err: %+v", podns, podname, err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	nmiResp := NMIResponse{
		Token:    newMSIResponse(*token),
		ClientID: clientID,
	}
	response, err := json.Marshal(nmiResp)
	if err != nil {
		klog.Errorf("failed to marshal service principal token and clientid for pod:%s/%s, err: %+v", podns, podname, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(response)
	return
}

// msiResponse marshals in a format that matches the underlying
// metadata endpoint more closely. This increases compatibility
// with callers built on older versions of adal client libraries.
type msiResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	ExpiresIn string `json:"expires_in"`
	ExpiresOn string `json:"expires_on"`
	NotBefore string `json:"not_before"`

	Resource string `json:"resource"`
	Type     string `json:"token_type"`
}

func newMSIResponse(token adal.Token) msiResponse {
	return msiResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.ExpiresIn.String(),
		ExpiresOn:    token.ExpiresOn.String(),
		NotBefore:    token.NotBefore.String(),
		Resource:     token.Resource,
		Type:         token.Type,
	}
}

func (s *Server) isMIC(podNS, rsName string) bool {
	micRegEx := regexp.MustCompile(`^mic-*`)
	if strings.EqualFold(podNS, s.MICNamespace) && micRegEx.MatchString(rsName) {
		return true
	}
	return false
}

func (s *Server) getTokenForExceptedPod(rqClientID, rqResource string) ([]byte, int, error) {
	var token *adal.Token
	var err error
	// ClientID is empty, so we are going to use System assigned MSI
	if rqClientID == "" {
		klog.Infof("Fetching token for system assigned MSI")
		token, err = auth.GetServicePrincipalTokenFromMSI(rqResource)
	} else { // User assigned identity usage.
		klog.Infof("Fetching token for user assigned MSI for resource: %s", rqResource)
		token, err = auth.GetServicePrincipalTokenFromMSIWithUserAssignedID(rqClientID, rqResource)
	}
	if err != nil {
		klog.Errorf("Failed to get service principal token, err: %+v", err)
		// TODO: return the right status code based on the error we got from adal.
		return nil, http.StatusForbidden, err
	}
	response, err := json.Marshal(newMSIResponse(*token))
	if err != nil {
		klog.Errorf("Failed to marshal service principal token, err: %+v", err)
		return nil, http.StatusInternalServerError, err
	}
	return response, http.StatusOK, nil
}

// msiHandler uses the remote address to identify the pod ip and uses it
// to find a matching client id, and then returns the token sourced through
// AAD using adal
// if the requests contains client id it validates it against the admin
// configured id.
func (s *Server) msiHandler(w http.ResponseWriter, r *http.Request) (ns string) {
	podIP := parseRemoteAddr(r.RemoteAddr)
	rqClientID, rqResource := parseRequestClientIDAndResource(r)

	if podIP == "" {
		klog.Error("request remote address is empty")
		http.Error(w, "request remote address is empty", http.StatusInternalServerError)
		return
	}
	if !validateResourceParamExists(rqResource) {
		klog.Warning("parameter resource cannot be empty")
		http.Error(w, "parameter resource cannot be empty", http.StatusBadRequest)
		return
	}
	podns, podname, rsName, selectors, err := s.KubeClient.GetPodInfo(podIP)
	if err != nil {
		klog.Errorf("missing podname for podip:%s, %+v", podIP, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// set ns for using in metrics
	ns = podns
	exceptionList, err := s.KubeClient.ListPodIdentityExceptions(podns)
	if err != nil {
		klog.Errorf("getting list of azurepodidentityexceptions in %s namespace failed with error: %+v", podns, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If its mic, then just directly get the token and pass back.
	if pod.IsPodExcepted(selectors.MatchLabels, *exceptionList) || s.isMIC(podns, rsName) {
		klog.Infof("Exception pod %s/%s token handling", podns, podname)
		response, errorCode, err := s.getTokenForExceptedPod(rqClientID, rqResource)
		if err != nil {
			klog.Errorf("failed to get service principal token for pod:%s/%s.  Error code: %d. Error: %+v", podns, podname, errorCode, err)
			http.Error(w, err.Error(), errorCode)
			return
		}
		w.Write(response)
		return
	}

	podIDs, identityInCreatedStateFound, err := s.listPodIDsWithRetry(r.Context(), s.KubeClient, podns, podname, rqClientID)
	if err != nil {
		msg := fmt.Sprintf("no AzureAssignedIdentity found for pod:%s/%s in assigned state", podns, podname)
		klog.Errorf("%s, %+v", msg, err)
		http.Error(w, msg, getErrorResponseStatusCode(identityInCreatedStateFound))
		return
	}

	token, _, err := getTokenForMatchingID(s.KubeClient, rqClientID, rqResource, podIDs)
	if err != nil {
		klog.Errorf("failed to get service principal token for pod:%s/%s, %+v", podns, podname, err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	response, err := json.Marshal(newMSIResponse(*token))
	if err != nil {
		klog.Errorf("failed to marshal service principal token for pod:%s/%s, %+v", podns, podname, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(response)
	return
}

func getTokenForMatchingID(kubeClient k8s.Client, rqClientID, rqResource string, podIDs []aadpodid.AzureIdentity) (token *adal.Token, clientID string, err error) {
	rqHasClientID := len(rqClientID) != 0
	for _, v := range podIDs {
		clientID := v.Spec.ClientID
		if rqHasClientID && !strings.EqualFold(rqClientID, clientID) {
			klog.Warningf("clientid mismatch, requested:%s available:%s", rqClientID, clientID)
			continue
		}

		idType := v.Spec.Type
		switch idType {
		case aadpodid.UserAssignedMSI:
			klog.Infof("matched identityType:%v clientid:%s resource:%s", idType, utils.RedactClientID(clientID), rqResource)
			token, err := auth.GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, rqResource)
			return token, clientID, err
		case aadpodid.ServicePrincipal:
			tenantid := v.Spec.TenantID
			klog.Infof("matched identityType:%v tenantid:%s clientid:%s resource:%s", idType, tenantid, utils.RedactClientID(clientID), rqResource)
			secret, err := kubeClient.GetSecret(&v.Spec.ClientPassword)
			if err != nil {
				return nil, clientID, err
			}
			clientSecret := ""
			for _, v := range secret.Data {
				clientSecret = string(v)
				break
			}
			token, err := auth.GetServicePrincipalToken(tenantid, clientID, clientSecret, rqResource)
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
func (s *Server) defaultPathHandler(w http.ResponseWriter, r *http.Request) (ns string) {
	client := &http.Client{}
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil || req == nil {
		klog.Errorf("failed creating a new request for %s, err: %+v", r.URL.String(), err)
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
		klog.Errorf("failed executing request for %s, err: %+v", req.URL.String(), err)
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
		klog.Errorf("failed io operation of reading response body for %s, %+v", req.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(body)
	return
}

// forbiddenHandler responds to any request with HTTP 403 Forbidden
func forbiddenHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Request blocked by AAD Pod Identity NMI", http.StatusForbidden)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func handleTermination() {
	klog.Info("Received SIGTERM, shutting down")

	exitCode := 0
	// clean up iptables
	if err := iptables.DeleteCustomChain(); err != nil {
		klog.Errorf("Error cleaning up during shutdown: %v", err)
		exitCode = 1
	}

	// wait for pod to delete
	klog.Info("Handled termination, awaiting pod deletion")
	time.Sleep(10 * time.Second)

	klog.Infof("Exiting with %v", exitCode)
	os.Exit(exitCode)
}

// listPodIDsWithRetry returns a list of matched identities in Assigned state, boolean indicating if at least an identity was found in Created state and error if any
func (s *Server) listPodIDsWithRetry(ctx context.Context, kubeClient k8s.Client, podns, podname, rqClientID string) ([]aadpodid.AzureIdentity, bool, error) {
	attempt := 0
	var err error
	var idStateMap map[string][]aadpodid.AzureIdentity

	// this loop will run to ensure we have assigned identities before we return. If there are no assigned identities in created state within 80s (16 retries * 5s wait) then we return an error.
	// If we get an assigned identity in created state within 80s, then loop will continue until 100s to find assigned identity in assigned state.
	// Retry interval for CREATED state is set to 80s because avg time for identity to be assigned to the node is 35-37s.
	for attempt < s.ListPodIDsRetryAttemptsForCreated+s.ListPodIDsRetryAttemptsForAssigned {
		idStateMap, err = kubeClient.ListPodIds(podns, podname)
		if err == nil {
			if len(rqClientID) == 0 {
				// check to ensure backward compatability with assignedIDs that have no state
				// assigned identites created with old version of mic will not contain a state. So first we check to see if an assigned identity with
				// no state exists that matches req client id.
				if len(idStateMap[""]) != 0 {
					klog.Warningf("found assignedIDs with no state for pod:%s/%s. AssignedIDs created with old version of mic.", podns, podname)
					return idStateMap[""], true, nil
				}
				if len(idStateMap[aadpodid.AssignedIDAssigned]) != 0 {
					return idStateMap[aadpodid.AssignedIDAssigned], true, nil
				}
				if len(idStateMap[aadpodid.AssignedIDCreated]) == 0 && attempt >= s.ListPodIDsRetryAttemptsForCreated {
					return nil, false, fmt.Errorf("getting assigned identities for pod %s/%s in CREATED state failed after %d attempts, retry duration [%d]s. Error: %v",
						podns, podname, s.ListPodIDsRetryAttemptsForCreated, s.ListPodIDsRetryIntervalInSeconds, err)
				}
			} else {
				// if client id exists in request, we need to ensure the identity with this client
				// exists and is in Assigned state
				// check to ensure backward compatability with assignedIDs that have no state
				for _, podID := range idStateMap[""] {
					if strings.EqualFold(rqClientID, podID.Spec.ClientID) {
						klog.Warningf("found assignedIDs with no state for pod:%s/%s. AssignedIDs created with old version of mic.", podns, podname)
						return idStateMap[""], true, nil
					}
				}
				for _, podID := range idStateMap[aadpodid.AssignedIDAssigned] {
					if strings.EqualFold(rqClientID, podID.Spec.ClientID) {
						return idStateMap[aadpodid.AssignedIDAssigned], true, nil
					}
				}
				var foundMatch bool
				for _, podID := range idStateMap[aadpodid.AssignedIDCreated] {
					if strings.EqualFold(rqClientID, podID.Spec.ClientID) {
						foundMatch = true
						break
					}
				}
				if !foundMatch && attempt >= s.ListPodIDsRetryAttemptsForCreated {
					return nil, false, fmt.Errorf("getting assigned identities for pod %s/%s in CREATED state failed after %d attempts, retry duration [%d]s. Error: %v",
						podns, podname, s.ListPodIDsRetryAttemptsForCreated, s.ListPodIDsRetryIntervalInSeconds, err)
				}
			}
		}
		attempt++

		select {
		case <-time.After(time.Duration(s.ListPodIDsRetryIntervalInSeconds) * time.Second):
		case <-ctx.Done():
			err = ctx.Err()
			return nil, true, err
		}
		klog.V(4).Infof("failed to get assigned ids for pod:%s/%s in ASSIGNED state, retrying attempt: %d", podns, podname, attempt)
	}
	return nil, true, fmt.Errorf("getting assigned identities for pod %s/%s in ASSIGNED state failed after %d attempts, retry duration [%d]s. Error: %v",
		podns, podname, s.ListPodIDsRetryAttemptsForCreated+s.ListPodIDsRetryAttemptsForAssigned, s.ListPodIDsRetryIntervalInSeconds, err)
}

func getErrorResponseStatusCode(identityFound bool) int {
	// if at least an identity was found in created state then we return 404 which is a retriable error code
	// in the go-autorest library. If the identity is in CREATED state then the identity is being processed in
	// this sync cycle and should move to ASSIGNED state soon.
	if identityFound {
		return http.StatusNotFound
	}
	// if no identity in at least CREATED state was found, then it means the identity creation is not part of the
	// current ongoing sync cycle. So we return 403 which a non-retriable error code so we give mic enough time to
	// finish current sync cycle and process identity in the next sync cycle.
	return http.StatusForbidden
}

func validateResourceParamExists(resource string) bool {
	// check if resource exists in the request
	// if resource doesn't exist in the request, then adal libraries will return the same error
	// IMDS also returns an error with 400 response code if resource parameter is empty
	// this is done to emulate same behavior observed while requesting token from IMDS
	if len(resource) == 0 {
		return false
	}
	return true
}
