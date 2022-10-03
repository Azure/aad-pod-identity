package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

var (
	rtr       *mux.Router
	server    *httptest.Server
	tokenPath = "/metadata/identity/oauth2/token/"
)

func setup() {
	rtr = mux.NewRouter()
	server = httptest.NewServer(rtr)
}

func teardown() {
	server.Close()
}

func TestMsiHandler_NoMetadataHeader(t *testing.T) {
	setup()
	defer teardown()

	s := &Server{
		MetadataHeaderRequired: true,
	}
	rtr.PathPrefix("/{type:(?i:metadata)}/identity/oauth2/token/").Handler(appHandler(s.msiHandler))

	req, err := http.NewRequest(http.MethodGet, tokenPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	rtr.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", recorder.Code)
	}

	resp := &MetadataResponse{
		Error:            "invalid_request",
		ErrorDescription: "Required metadata header not specified",
	}
	expected, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	if string(expected) != strings.TrimSpace(recorder.Body.String()) {
		t.Errorf("Unexpected response body %s", recorder.Body.String())
	}
}

func TestMsiHandler_NoRemoteAddress(t *testing.T) {
	setup()
	defer teardown()

	s := &Server{
		MetadataHeaderRequired: false,
	}
	rtr.PathPrefix("/{type:(?i:metadata)}/identity/oauth2/token/").Handler(appHandler(s.msiHandler))

	req, err := http.NewRequest(http.MethodGet, tokenPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	rtr.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code %d", recorder.Code)
	}

	expected := "request remote address is empty"
	if expected != strings.TrimSpace(recorder.Body.String()) {
		t.Errorf("Unexpected response body %s", recorder.Body.String())
	}
}

func TestParseTokenRequest(t *testing.T) {
	const endpoint = "http://127.0.0.1/metadata/identity/oauth2/token"

	t.Run("query present", func(t *testing.T) {
		const resource = "https://vault.azure.net"
		const clientID = "77788899-f67e-42e1-9a78-89985f6bff3e"
		const resourceID = "/subscriptions/9f2be85c-f8ae-4569-9353-38e5e8b459ef/resourcegroups/test/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test"

		var r http.Request
		r.URL, _ = url.Parse(fmt.Sprintf("%s?client_id=%s&msi_res_id=%s&resource=%s", endpoint, clientID, resourceID, resource))

		result := parseTokenRequest(&r)

		if result.ClientID != clientID {
			t.Errorf("invalid ClientID - expected: %q, actual: %q", clientID, result.ClientID)
		}

		if result.ResourceID != resourceID {
			t.Errorf("invalid ResourceID - expected: %q, actual: %q", resourceID, result.ResourceID)
		}

		if result.Resource != resource {
			t.Errorf("invalid Resource - expected: %q, actual: %q", resource, result.Resource)
		}
	})

	t.Run("bare endpoint", func(t *testing.T) {
		var r http.Request
		r.URL, _ = url.Parse(endpoint)

		result := parseTokenRequest(&r)

		if result.ClientID != "" {
			t.Errorf("invalid ClientID - expected: %q, actual: %q", "", result.ClientID)
		}

		if result.ResourceID != "" {
			t.Errorf("invalid ResourceID - expected: %q, actual: %q", "", result.ResourceID)
		}

		if result.Resource != "" {
			t.Errorf("invalid Resource - expected: %q, actual: %q", "", result.Resource)
		}
	})
}

func TestTokenRequest_ValidateResourceParamExists(t *testing.T) {
	tr := TokenRequest{
		Resource: "https://vault.azure.net",
	}

	if !tr.ValidateResourceParamExists() {
		t.Error("ValidateResourceParamExists should have returned true when the resource is set")
	}

	tr.Resource = ""
	if tr.ValidateResourceParamExists() {
		t.Error("ValidateResourceParamExists should have returned false when the resource is unset")
	}
}

func TestRouterPathPrefix(t *testing.T) {
	tests := []struct {
		name               string
		url                string
		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "token request",
			url:                "/metadata/identity/oauth2/token/",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "token_request_handler",
		},
		{
			name:               "token request without / suffix",
			url:                "/metadata/identity/oauth2/token",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "token_request_handler",
		},
		{
			name:               "token request with upper case metadata",
			url:                "/Metadata/identity/oauth2/token/",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "token_request_handler",
		},
		{
			name:               "token request with upper case identity",
			url:                "/metadata/Identity/oauth2/token/",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "default_handler",
		},
		{
			name:               "host token request",
			url:                "/host/token/",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "host_token_request_handler",
		},
		{
			name:               "host token request without / suffix",
			url:                "/host/token",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "host_token_request_handler",
		},
		{
			name:               "instance metadata request",
			url:                "/metadata/instance",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "instance_request_handler",
		},
		{
			name:               "instance metadata request with upper case metadata",
			url:                "/Metadata/instance",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "instance_request_handler",
		},
		{
			name:               "instance metadata request / suffix",
			url:                "/Metadata/instance/",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "instance_request_handler",
		},
		{
			name:               "default metadata request",
			url:                "/metadata/",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "default_handler",
		},
		{
			name:               "invalid token request with \\oauth2",
			url:                `/metadata/identity\oauth2/token/`,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "invalid_request_handler",
		},
		{
			name:               "invalid token request with \\token",
			url:                `/metadata/identity/oauth2\token/`,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "invalid_request_handler",
		},
		{
			name:               "invalid token request with \\oauth2\\token",
			url:                `/metadata/identity\oauth2\token/`,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "invalid_request_handler",
		},
		{
			name:               "invalid token request with mix of / and \\",
			url:                `/metadata/identity/\oauth2\token/`,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "invalid_request_handler",
		},
		{
			name:               "invalid token request with multiple \\",
			url:                `/metadata/identity\\\oauth2\\token/`,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "invalid_request_handler",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setup()
			defer teardown()

			rtr.PathPrefix(tokenPathPrefix).HandlerFunc(testTokenHandler)
			rtr.MatcherFunc(invalidTokenPathMatcher).HandlerFunc(testInvalidRequestHandler)
			rtr.PathPrefix(hostTokenPathPrefix).HandlerFunc(testHostTokenHandler)
			rtr.PathPrefix(instancePathPrefix).HandlerFunc(testInstanceHandler)
			rtr.PathPrefix("/").HandlerFunc(testDefaultHandler)

			req, err := http.NewRequest(http.MethodGet, test.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			rtr.ServeHTTP(recorder, req)

			if recorder.Code != test.expectedStatusCode {
				t.Errorf("unexpected status code %d", recorder.Code)
			}

			if test.expectedBody != strings.TrimSpace(recorder.Body.String()) {
				t.Errorf("unexpected response body %s", recorder.Body.String())
			}
		})
	}
}

func testTokenHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "token_request_handler\n")
}

func testHostTokenHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "host_token_request_handler\n")
}

func testInstanceHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "instance_request_handler\n")
}

func testDefaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "default_handler\n")
}

func testInvalidRequestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "invalid_request_handler\n")
}
