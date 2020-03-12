package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	mux       *http.ServeMux
	server    *httptest.Server
	tokenPath = "/metadata/identity/oauth2/token"
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
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
	mux.Handle(tokenPath, appHandler(s.msiHandler))

	req, err := http.NewRequest(http.MethodGet, tokenPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

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
	mux.Handle(tokenPath, appHandler(s.msiHandler))

	req, err := http.NewRequest(http.MethodGet, tokenPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code %d", recorder.Code)
	}

	expected := "request remote address is empty"
	if expected != strings.TrimSpace(recorder.Body.String()) {
		t.Errorf("Unexpected response body %s", recorder.Body.String())
	}
}
