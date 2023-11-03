package main

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/hired-varied/stupid-proxy/utils"
)

func TestIsAuthenticated(t *testing.T) {
	// Initialize a defaultHandler with a sample config
	handler := &defaultHandler{
		config: Config{
			Auth: map[string]string{
				"testuser": "testpass",
			},
		},
	}

	// Test an authenticated request
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	authorized, username := handler.isAuthenticated(authHeader)
	if !authorized {
		t.Errorf("Expected request to be authorized")
	}
	if username != "testuser" {
		t.Errorf("Expected username to be 'testuser', got %s", username)
	}

	// Test an invalid request
	authHeader = "InvalidHeader"
	authorized, _ = handler.isAuthenticated(authHeader)
	if authorized {
		t.Errorf("Expected request to be unauthorized")
	}
}

func TestCopyHeader(t *testing.T) {
	src := make(http.Header)
	src.Add("Key1", "Value1")
	src.Add("Key2", "Value2")

	dst := make(http.Header)
	copyHeader(dst, src)

	if len(dst) != 2 {
		t.Errorf("Expected destination header to have 2 entries")
	}
	if dst.Get("Key1") != "Value1" {
		t.Errorf("Expected Key1 to have Value1")
	}
	if dst.Get("Key2") != "Value2" {
		t.Errorf("Expected Key2 to have Value2")
	}
}

func TestHandleTunneling(t *testing.T) {
	// Create a test server for handling the CONNECT request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the behavior of the upstream server
		// In this example, we just send a success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a fake client request to test the handleTunneling function
	clientRequest, err := http.NewRequest(http.MethodConnect, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create client request: %v", err)
	}

	// Create a fake ResponseWriter for the test
	responseRecorder := httptest.NewRecorder()

	// Call the handleTunneling function
	handleTunneling(responseRecorder, clientRequest, "testuser")

	// Check if the response has the expected status code
	if responseRecorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseRecorder.Code)
	}
}

func TestHandleHTTP(t *testing.T) {
	// Create a test server for handling HTTP requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the behavior of the upstream server
		// In this example, we just send a simple response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Response Body"))
	}))
	defer server.Close()

	// Create a fake client request to test the handleHTTP function
	clientRequest, err := http.NewRequest(http.MethodGet, server.URL, bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Failed to create client request: %v", err)
	}

	// Create a fake ResponseWriter for the test
	responseRecorder := httptest.NewRecorder()

	// Call the handleHTTP function
	handleHTTP(responseRecorder, clientRequest, "testuser")

	// Check if the response has the expected status code and body
	if responseRecorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseRecorder.Code)
	}

	expectedBody := "Response Body"
	if responseRecorder.Body.String() != expectedBody {
		t.Errorf("Expected response body %q, got %q", expectedBody, responseRecorder.Body.String())
	}
}

func TestLoadConfigFile(t *testing.T) {
	// Initialize a Config struct and load the test config
	config := &Config{}
	utils.LoadConfigFile("./config.sample.yaml", config)

	if config.UpstreamAddr != "http://example.com" || config.ListenAddr != "0.0.0.0:3000" || config.AuthTriggerPath != "/auth" || config.Auth["username"] != "password" {
		t.Errorf("Loaded config does not match the expected config")
	}
}

func TestServeHTTPWithAuthenticatedUser(t *testing.T) {
	// Create a test server for handling HTTP requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the behavior of the upstream server
		// In this example, we just send a simple response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("proxixed request"))
	}))
	defer server.Close()

	// Create a sample config for the defaultHandler
	config := Config{
		UpstreamAddr:    "https://www.google.com",
		ListenAddr:      "127.0.0.1:8080",
		AuthTriggerPath: "/auth",
		Auth: map[string]string{
			"testuser": "testpass",
		},
	}

	// Create a fake client request with an authenticated user
	clientRequest, err := http.NewRequest(http.MethodGet, server.URL, bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Failed to create client request: %v", err)
	}
	clientRequest.Header.Set("Proxy-Authorization", "Basic dGVzdHVzZXI6dGVzdHBhc3M=") // Base64("testuser:testpass")

	// Create a fake ResponseWriter for the test
	responseRecorder := httptest.NewRecorder()

	// Initialize the defaultHandler with the sample config
	reverseProxyURL, _ := url.Parse(config.UpstreamAddr)
	handler := &defaultHandler{
		config:       config,
		reverseProxy: httputil.NewSingleHostReverseProxy(reverseProxyURL),
	}

	// Call the ServeHTTP method with the client request and response writer
	handler.ServeHTTP(responseRecorder, clientRequest)

	// Check if the response has the expected status code and body
	if responseRecorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseRecorder.Code)
	}

	expectedBody := "proxixed request"
	if responseRecorder.Body.String() != expectedBody {
		t.Errorf("Expected response body %q, got %q", expectedBody, responseRecorder.Body.String())
	}
}
