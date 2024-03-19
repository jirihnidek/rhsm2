package rhsm2

import (
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreateXCorrelationID test the createCorrelationId()
func TestCreateXCorrelationID(t *testing.T) {
	xCorrelationId := createCorrelationId()
	err := uuid.Validate(xCorrelationId)
	if err != nil {
		t.Fatalf("%s is not valid correlation ID", xCorrelationId)
	}
}

// TestUserAgentString test the case, when client has not set command of UserAgent
// String() method should return default value
func TestUserAgentString(t *testing.T) {
	userAgentStr := UserAgent.String()
	expectedUserAgent := "RHSM/2.0"
	if userAgentStr != expectedUserAgent {
		t.Fatalf("expected UserAgent: \"%s\", got: \"%s\"", expectedUserAgent, userAgentStr)
	}
}

// TestSetUserAgentCmd test the case, when client set command of UserAgent
func TestSetUserAgentCmd(t *testing.T) {
	SetUserAgentCmd("foo-cmd")
	userAgentStr := UserAgent.String()
	expectedUserAgent := "RHSM/2.0 (cmd=foo-cmd)"
	if userAgentStr != expectedUserAgent {
		t.Fatalf("expected UserAgent: \"%s\", got: \"%s\"", expectedUserAgent, userAgentStr)
	}
}

// TestGetServerStatusClientInfo test the case, when we try to
// call some REST API call with not empty ClientInfo structure
func TestGetServerStatusClientInfo(t *testing.T) {
	handlerCounter := 0
	expectedLocale := "de-DE"
	expectedDBusSender := "foo-dbus-client"
	expectedUserAgentCmd := "tester"

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

			// Test that HTTP headers are correct
			xCorrelationId := req.Header.Get("X-Correlation-ID")
			if xCorrelationId == "" {
				t.Fatalf("X-Correlation-ID is empty string")
			}
			locale := req.Header.Get("Accept-Language")
			if locale != expectedLocale {
				t.Fatalf("expected Accept-Language HTTP header: %s, got: %s", expectedLocale, locale)
			}
			userAgent := req.Header.Get("User-Agent")
			expectedUserAgent := fmt.Sprintf(
				"RHSM/2.0 (cmd=%s) (dbus_sender=%s)",
				expectedUserAgentCmd,
				expectedDBusSender,
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

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	SetUserAgentCmd(expectedUserAgentCmd)

	clientInfo := ClientInfo{expectedLocale, expectedDBusSender, ""}
	serverStatus, err := rhsmClient.GetServerStatus(&clientInfo)
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

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Add extra configuration related to proxy server to conf
	// structure
	rhsmClient.RHSMConf.Server.ProxyHostname = "proxy.server.org"
	rhsmClient.RHSMConf.Server.ProxyPort = "3129"
	rhsmClient.RHSMConf.Server.ProxyUser = "user"
	rhsmClient.RHSMConf.Server.ProxyPassword = "secret"

	clientInfo := ClientInfo{"", "", "66bf0b7a-aaae-4b31-a7bf-bc22052afebf"}
	serverStatus, err := rhsmClient.GetServerStatus(&clientInfo)
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
