package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const entCertKeyList = `
[ {
  "created" : "2023-10-04T07:42:10+0000",
  "updated" : "2023-10-04T07:42:10+0000",
  "id" : "4028fcc68aef65d7018af9a319270b3b",
  "key" : "-----BEGIN PRIVATE KEY-----\n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\n8nmeg9dC0e3G42ddx57Led2N/9eK2M0=\n-----END PRIVATE KEY-----\n",
  "cert" : "-----BEGIN CERTIFICATE-----\n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\nT6c93eXCser5vMQ8azrL7wEkevM=\n-----END CERTIFICATE-----\n-----BEGIN ENTITLEMENT DATA-----\n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\nNGUzvr69/Q9tPMxs\n-----END ENTITLEMENT DATA-----\n-----BEGIN RSA SIGNATURE-----\nuAb5tkHl0Jqn3EC7Zqv5AL8Pb9OrTz4+FLAx5SA9+HgBnKhRmLmeBENLoqeyejPf\nEM9jUqYw0cVI3oc89kb0rNREaiIKGcKSmAoBRdxqu77vljJyspjjQ8bKfE+8ZHlP\nWVLK/KSqC8Xa+gha/NeaAdVlvf8KHZIe3XuNwlC1Eb4rksF3uQvyWG/NW9FEGu3d\np14i4M4Q52F+JmH4zWct6WIiMLf51gYYVUa3qtfSDpRfWFqplmHoJFO5yaTzGtyV\nn3hENy+5gCIEqNkSQfucoEjb9zF2wKHGe8g7CXnom/W/WTp9ji/mIrehNJcGRibv\nT3sY7qCOGKEAdrMUapGfU53kRRKls2axpkLv5yGOYuEbFjf90c/AOka1RVkW+Veu\n2It6AUAgbAFzDkltkRTKP1n1ao6jPdx/mDlSFzuN/YmQbdCmaSWEmeOZkpvui3jN\n/lx0scI9IfsJYUr940fHk2BIgtW5JGPTMV8CALwegeR0iJ4uHB5nnMCXtQfpOSkz\n+Tm6RizFt5dShOskaKtA0A+v/yRjmHtnmefegvVUTodZP3+0aORriT1fQfzAE9JZ\nyywwHMlT0HxFYKkxviSssxQNNfTRyxEpKwmRuPR5jy4/mRYvkyTmNWK/Sts0U9r6\nvL5fU8ynofD2e6eWafrXmeNVb2KqOXCj1IncoZtGjsg=\n-----END RSA SIGNATURE-----\n",
  "serial" : {
    "created" : "2023-10-04T07:42:10+0000",
    "updated" : "2023-10-04T07:42:10+0000",
    "id" : 2150990815908364188,
    "serial" : 2150990815908364188,
    "expiration" : "2024-10-04T06:42:10+0000",
    "revoked" : false
  }
} ]
`

// TestGetEntitlementCertificate test the case, when it is possible to get SCA
// certificate and key from the server
func TestGetEntitlementCertificate(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that getSCAEntitlementCertificate() will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID + "/certificates"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 200
			rw.WriteHeader(200)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(entCertKeyList))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered,
	// but no entitlement cert/key has been installed yet
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	entCertKeys, err := rhsmClient.getSCAEntitlementCertificates()
	if err != nil {
		t.Fatalf("failed to get SCA entitlement cert and key: %s", err)
	}

	// It is SCA entitlement cert/key. There should be only one record.
	expectedNumberOfCertAndKey := 1
	if len(entCertKeys) != expectedNumberOfCertAndKey {
		t.Fatalf("expected number of SCA entitlement certs/keys returned: %d, got: %d",
			expectedNumberOfCertAndKey, len(entCertKeys))
	}

	// Handler function should be called only once
	if handlerCounter != 1 {
		t.Fatalf("handler for getting SCA entitlement cert REST API pointed not called once, but called: %d",
			handlerCounter)
	}

	// Entitlement cert & key should be installed
	isEmpty, err := isDirEmpty(&testingFiles.EntitlementDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.EntitlementDirPath, err)
	}
	if isEmpty == true {
		t.Fatalf("no entitlement cert or key has been installed to: %s",
			testingFiles.EntitlementDirPath)
	}

	// Test that entitlement cert is installed
	expectedEntCertFilePath := filepath.Join(testingFiles.EntitlementDirPath, "2150990815908364188.pem")
	if _, err := os.Stat(expectedEntCertFilePath); err != nil {
		t.Fatalf("expected entitlement cert: %s is not installed: %s", expectedEntCertFilePath, err)
	}

	// Test that entitlement key is installed
	expectedEntKeyFilePath := filepath.Join(testingFiles.EntitlementDirPath, "2150990815908364188-key.pem")
	if _, err := os.Stat(expectedEntKeyFilePath); err != nil {
		t.Fatalf("expected entitlement key: %s is not installed: %s", expectedEntKeyFilePath, err)
	}
}

// TestGetEntitlementCertificateWrongConsumerUUID test the case, when wrong Consumer UUID
// is used.
func TestGetEntitlementCertificateWrongConsumerUUID(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID + "/certificates"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 404
			rw.WriteHeader(404)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return JSON document with error
			_, _ = rw.Write([]byte(response404))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered,
	// but no entitlement cert/key has been installed yet
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	_, err = rhsmClient.getSCAEntitlementCertificates()
	if err == nil {
		t.Fatalf("no error raised, when server responses with 404 status code")
	}
}

// TestGetEntitlementCertificateDeletedConsumerUUID test the case, when deleted Consumer UUID
// is used. It is related to the case, when consumer has been deleted on the server
func TestGetEntitlementCertificateDeletedConsumerUUID(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID + "/certificates"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 410
			rw.WriteHeader(410)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return JSON document with error
			_, _ = rw.Write([]byte(response410))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered,
	// but no entitlement cert/key has been installed yet
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	_, err = rhsmClient.getSCAEntitlementCertificates()
	if err == nil {
		t.Fatalf("no error raised, when server responses with 410 status code")
	}
}

// TestGetEntitlementCertificateInternalServerError test the case, when there is
// some internal server error
func TestGetEntitlementCertificateInternalServerError(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID + "/certificates"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 500
			rw.WriteHeader(500)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return JSON document with error
			_, _ = rw.Write([]byte(response500))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered,
	// but no entitlement cert/key has been installed yet
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	_, err = rhsmClient.getSCAEntitlementCertificates()
	if err == nil {
		t.Fatalf("no error raised, when server responses with 410 status code")
	}
}
