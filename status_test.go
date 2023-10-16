package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const serverStatusResponse = `{
  "mode" : "NORMAL",
  "modeReason" : null,
  "modeChangeTime" : null,
  "result" : true,
  "version" : "4.3.8",
  "release" : "1",
  "standalone" : false,
  "timeUTC" : "2023-10-10T07:39:11+0000",
  "rulesSource" : "default",
  "rulesVersion" : "5.44",
  "managerCapabilities" : [ "instance_multiplier", "derived_product", "vcpu", "cert_v3", "hypervisors_heartbeat", "remove_by_pool_id", "syspurpose", "storage_band", "cores", "multi_environment", "hypervisors_async", "org_level_content_access", "guest_limit", "ram", "batch_bind" ],
  "keycloakRealm" : null,
  "keycloakAuthUrl" : null,
  "keycloakResource" : null,
  "deviceAuthRealm" : null,
  "deviceAuthUrl" : null,
  "deviceAuthClientId" : null,
  "deviceAuthScope" : null
}`

// TestGetServerStatus test the case, when it is possible
// to get server status
func TestGetServerStatus(t *testing.T) {
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

// TestGetServerStatus test the case, when it is possible
// to get server status
func TestGetServerStatusInternalServerError(t *testing.T) {
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

			// Return code 500
			rw.WriteHeader(500)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(internalServerError))
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

	_, err = rhsmClient.GetServerStatus()
	if err == nil {
		t.Fatalf("no error raised, when there was internal server error")
	}
}
