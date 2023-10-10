package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetContentFromEntCert tests the case, when content is successfully loaded
// from entitlement certificate
func TestGetContentFromEntCert(t *testing.T) {
	filePath := "test/etc/pki/entitlement/6490061114713729830.pem"
	engineeringProducts, err := getContentFromEntCertFile(&filePath)
	if err != nil {
		t.Fatalf("unable to get engineering products from file: %s: %s", filePath, err)
	}

	if len(engineeringProducts) != 1 {
		t.Fatalf("file: %s does not contain one engineering product", filePath)
	}

	engProduct := engineeringProducts[0]

	// Yes, it is " Content Access" and not "Content Access". The entitlement
	// certificate just contain this.
	if engProduct.Name != " Content Access" {
		t.Fatalf("expected product name: ' Content Access', got: '%s'", engProduct.Name)
	}
	if engProduct.Id != "content_access" {
		t.Fatalf("expected product id: 'content_access', got: '%s'", engProduct.Id)
	}
	// Version is empty for some reason in entitlement certificate
	if engProduct.Version != "" {
		t.Fatalf("expected product version: '', got: '%s'", engProduct.Version)
	}
	// No architectures provided in entitlement certificate
	if len(engProduct.Architectures) != 0 {
		t.Fatalf("expected product architectures: '[]', got: '%v'", engProduct.Architectures)
	}
	if len(engProduct.Content) != 92 {
		t.Fatalf("expected product content definitions: 92, got: '%v'", len(engProduct.Content))
	}

	// Test content definition only for the first content
	content := engProduct.Content[0]
	// Id
	expectedVal := "10000000000006011121"
	if content.Id != expectedVal {
		t.Fatalf("expected content id: '%s', got: '%s'", expectedVal, content.Id)
	}
	// Name
	expectedVal = "awesomeos-s390x-100000000000060"
	if content.Name != expectedVal {
		t.Fatalf("expected content id: '%s', got: '%s'", expectedVal, content.Id)
	}
	// Label
	expectedVal = "awesomeos-s390x-100000000000060"
	if content.Label != expectedVal {
		t.Fatalf("expected content label: '%s', got: '%s'", expectedVal, content.Label)
	}
	// Path
	expectedVal = "/path/to/awesomeos/s390x/100000000000060-11121"
	if content.Path != expectedVal {
		t.Fatalf("expected content path: '%s', got: '%s'", expectedVal, content.Path)
	}

	// Note: if you are bored, then add tests for remaining fields of 'content' struct
}

// TestWriteRepoFile test the case, when content definition from
// entitlement certificate is successfully written to repo file
func TestWriteRepoFile(t *testing.T) {
	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.generateRepoFileFromInstalledEntitlementCerts()

	if err != nil {
		t.Fatalf("unable to generate '%s': %s", testingFiles.YumRepoFilePath, err)
	}
}

// TestWriteRepoFileNoEntCert test the case, when content definition from
// entitlement certificate is not possible to read, because there is no
// entitlement certificate installed
func TestWriteRepoFileNoEntCert(t *testing.T) {
	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.generateRepoFileFromInstalledEntitlementCerts()

	if err != nil {
		t.Fatalf("when no entitlement certificate installed, error returned: %s", err)
	}
}
