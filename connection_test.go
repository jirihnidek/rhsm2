package rhsm2

import (
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

	serverStatus, err := rhsmClient.GetServerStatus()
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
