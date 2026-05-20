package rhsm2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// TestCreateCorrelationID test the createCorrelationId()
func TestCreateCorrelationID(t *testing.T) {
	xCorrelationId := createCorrelationId()
	err := uuid.Validate(xCorrelationId)
	if err != nil {
		t.Fatalf("%s is not valid correlation ID", xCorrelationId)
	}
}

// TestGetServerStatusMetadata test the case, when we try to
// call some REST API call with not empty RequestMetadata structure
func TestGetServerStatusMetadata(t *testing.T) {
	handlerCounter := 0
	expectedLocale := "de-DE"
	expectedIPCSender := "foo-varlink-client"
	expectedCorrelationId := "test-correlation-id"

	server := httptest.NewTLSServer(
		// It is expected that GetServerStatus() will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("expected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/status"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Test that request contains Connection header
			connectionHeader := req.Header.Get("Connection")
			if connectionHeader == "" {
				t.Fatalf("Connection header is missing")
			}
			if connectionHeader != "keep-alive" {
				t.Fatalf("expected Connection header: %s, got: %s", "keep-alive", connectionHeader)
			}

			// Test that request contains Keep-Alive header
			keepAliveTimeout := req.Header.Get("Keep-Alive")
			if keepAliveTimeout == "" {
				t.Fatalf("Keep-Alive header is missing")
			}
			if keepAliveTimeout != "timeout=60" {
				t.Fatalf("expected Keep-Alive timeout: %s, got: %s", "timeout=60", keepAliveTimeout)
			}

			// Test that HTTP headers are correct
			CorrelationId := req.Header.Get("Correlation-ID")
			if CorrelationId == "" {
				t.Fatalf("Correlation-ID is empty string")
			}
			RequestId := req.Header.Get("Request-ID")
			if RequestId == "" {
				t.Fatalf("Request-ID is empty string")
			}
			locale := req.Header.Get("Accept-Language")
			if locale != expectedLocale {
				t.Fatalf("expected Accept-Language HTTP header: %s, got: %s", expectedLocale, locale)
			}
			userAgent := req.Header.Get("User-Agent")
			expectedUserAgent := fmt.Sprintf(
				"unit-tester/0.1 (trigger-by: %s) foo-linux/10.0",
				expectedIPCSender,
			)
			if userAgent != expectedUserAgent {
				t.Fatalf("expected User-Agent HTTP header: %s, got: %s", expectedUserAgent, userAgent)
			}

			// Return code 200
			rw.WriteHeader(200)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(serverStatusResponse))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered,
	// but no entitlement cert/key has been installed yet
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	metadata := RequestMetadata{&expectedLocale, &expectedIPCSender, &expectedCorrelationId}
	serverStatus, err := rhsmClient.GetServerStatus(&metadata)
	if err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for getting server status REST API pointed not called once, but called: %d",
			handlerCounter)
	}

	expectedServerVer := "4.3.8"
	if serverStatus.Version != expectedServerVer {
		t.Fatalf("expected server version: %s, got: %s", expectedServerVer, serverStatus.Version)
	}
}

// TestCreateHTTPsClientProxyFromConf test the case, when proxy server
// is used. The /status endpoint is used for testing
func TestCreateHTTPsClientProxyFromConf(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that GetServerStatus() will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/status"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 200
			rw.WriteHeader(200)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(serverStatusResponse))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, false)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Add extra configuration related to proxy server to conf
	// structure
	rhsmClient.RHSMConf.Server.ProxyHostname = "proxy.server.org"
	rhsmClient.RHSMConf.Server.ProxyPort = "3129"
	rhsmClient.RHSMConf.Server.ProxyUser = "user"
	rhsmClient.RHSMConf.Server.ProxyPassword = "secret"

	correlationID := "66bf0b7a-aaae-4b31-a7bf-bc22052afebf"
	metadata := RequestMetadata{nil, nil, &correlationID}
	serverStatus, err := rhsmClient.GetServerStatus(&metadata)
	if err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for getting server status REST API pointed not called once, but called: %d",
			handlerCounter)
	}

	expectedServerVer := "4.3.8"
	if serverStatus.Version != expectedServerVer {
		t.Fatalf("expected server version: %s, got: %s", expectedServerVer, serverStatus.Version)
	}
}

// TestGetCertAuthConnectionRegistered test the case when we try to get connection
// using consumer certificate authentication for registered system
func TestGetCertAuthConnectionRegistered(t *testing.T) {
	t.Parallel()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem with consumer certificates
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath,
		true,
		true,
		false,
		false,
		true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Verify that client certificate was provided
			if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
				t.Fatalf("expected client certificate, but none was provided")
			}

			// Return code 200
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("OK"))
		}))
	defer server.Close()

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Get cert auth connection
	connection, err := rhsmClient.getCertAuthConnection()
	if err != nil {
		t.Fatalf("failed to create cert auth connection: %s", err)
	}

	if connection == nil {
		t.Fatalf("connection should not be nil")
	}

	// Verify connection has transport configured
	if connection.Client == nil {
		t.Fatalf("connection http client should not be nil")
	}
}

// TestGetCertAuthConnectionNotRegistered test the case when we try to get connection
// using consumer certificate authentication, but system is not registered
func TestGetCertAuthConnectionNotRegistered(t *testing.T) {
	t.Parallel()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem without consumer certificates
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath,
		false,
		false,
		false,
		false,
		false)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Return code 200
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("OK"))
		}))
	defer server.Close()

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}
	// Set the "consumer" connection to nil
	rhsmClient.consumerCertAuthConnection = nil

	// Try to get cert auth connection
	connection, err := rhsmClient.getCertAuthConnection()
	if err == nil || connection != nil {
		t.Fatalf("it should not be possible to get connection without consumer certificate and key")
	}
}

// TestGetEntitlementCertAuthConnection test the case when we try to get the existing connection
// using entitlement certificate authentication
func TestGetEntitlementCertAuthConnection(t *testing.T) {
	t.Parallel()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem with entitlement certificates
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath,
		true,
		true,
		true,
		true,
		true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Verify that client certificate was provided
			if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
				t.Fatalf("expected client certificate, but none was provided")
			}

			// Return code 200
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("OK"))
		}))
	defer server.Close()

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Get entitlement cert auth connection
	connection, err := rhsmClient.getEntitlementCertAuthConnection()
	if err != nil {
		t.Fatalf("failed to create entitlement cert auth connection: %s", err)
	}

	if connection == nil {
		t.Fatalf("connection should not be nil")
	}

	// Verify connection has transport configured
	if connection.Client == nil {
		t.Fatalf("connection http client should not be nil")
	}
}

// TestGetEntitlementCertAuthConnectionWithoutCertAndKey test the case when it is not possible
// to get connection using entitlement certificate authentication, because there is no entitlement
// certificate and key
func TestGetEntitlementCertAuthConnectionWithoutCertAndKey(t *testing.T) {
	t.Parallel()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem with entitlement certificates
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath,
		true,
		true,
		false,
		false,
		true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Verify that client certificate was provided
			if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
				t.Fatalf("expected client certificate, but none was provided")
			}

			// Return code 200
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("OK"))
		}))
	defer server.Close()

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Try to get entitlement cert auth connection
	connection, err := rhsmClient.getEntitlementCertAuthConnection()
	if err == nil || connection != nil {
		t.Fatalf("it should not be possible to get connection without entitlement certificate and key")
	}

	// Test the case when some process installs entitlement certificate meanwhile
	err = testingFiles.setupEntitlementCertKey(nil)
	if err != nil {
		t.Fatalf("unable to setup testing entitlement certificate and key: %s", err)
	}

	// Try to get entitlement cert auth connection. The getEntitlementCertAuthConnection
	// should create a new connection
	connection, err = rhsmClient.getEntitlementCertAuthConnection()
	if err != nil {
		t.Fatalf("failed to create entitlement cert auth connection: %s", err)
	}

	if connection == nil {
		t.Fatalf("connection should not be nil")
	}

	// Verify connection has transport configured
	if connection.Client == nil {
		t.Fatalf("connection http client should not be nil")
	}
}

// TestGetEntitlementCertAuthConnectionWithProxy test the case when we create connection
// using entitlement certificate authentication with proxy configuration
func TestGetEntitlementCertAuthConnectionWithProxy(t *testing.T) {
	t.Parallel()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem with entitlement certificates
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath,
		true,
		true,
		true,
		true,
		true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Verify that client certificate was provided
			if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
				t.Fatalf("expected client certificate, but none was provided")
			}

			// Return code 200
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("OK"))
		}))
	defer server.Close()

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Add proxy configuration
	rhsmClient.RHSMConf.Server.ProxyHostname = "proxy.example.com"
	rhsmClient.RHSMConf.Server.ProxyPort = "8080"
	rhsmClient.RHSMConf.Server.ProxyUser = "proxyuser"
	rhsmClient.RHSMConf.Server.ProxyPassword = "proxypass"

	// Create entitlement cert auth connection with proxy
	connection, err := rhsmClient.getEntitlementCertAuthConnection()
	if err != nil {
		t.Fatalf("failed to create entitlement cert auth connection with proxy: %s", err)
	}

	if connection == nil {
		t.Fatalf("connection should not be nil")
	}

	// Verify connection has transport configured
	if connection.Client == nil {
		t.Fatalf("connection http client should not be nil")
	}
}

// TestGetNoAuthConnection test the case when we try to get the no-auth connection
func TestGetNoAuthConnection(t *testing.T) {
	t.Parallel()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem without consumer certificates
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath,
		false,
		false,
		false,
		false,
		false)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Verify that no client certificate was provided
			if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
				t.Fatalf("expected no client certificate, but one was provided")
			}

			// Return code 200
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("OK"))
		}))
	defer server.Close()

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Get no-auth connection
	connection, err := rhsmClient.getNoAuthConnection()
	if err != nil {
		t.Fatalf("failed to get no-auth connection: %s", err)
	}

	if connection == nil {
		t.Fatalf("connection should not be nil")
	}

	// Verify connection has transport configured
	if connection.Client == nil {
		t.Fatalf("connection http client should not be nil")
	}
}
