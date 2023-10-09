package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestReadAllProductCertificates tries to test the case, when product
// certificates are successfully read from directory
func TestReadAllProductCertificates(t *testing.T) {
	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for reading product certs, %s %s called",
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

	installedProducts, err := rhsmClient.readAllProductCertificates()
	if err != nil {
		t.Fatalf("reading of product certs failed: %s", err)
	}

	if len(installedProducts) != 2 {
		t.Fatalf("no product certs read from: %s or %s",
			rhsmClient.RHSMConf.RHSM.ProductCertDir, rhsmClient.RHSMConf.RHSM.DefaultProductCertDir)
	}
}

// TestReadAllProductCertificatesNoInstalled tries to test the case, when product
// certificates are successfully read from directory, but there are not
// installed product certificates (there are only default product certs)
func TestReadAllProductCertificatesNoInstalled(t *testing.T) {
	server := httptest.NewTLSServer(
		// It is expected that reading installed product certificates will not
		// trigger any REST API call
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for reading product certs, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	installedProducts, err := rhsmClient.readAllProductCertificates()
	if err != nil {
		t.Fatalf("reading of product certs failed: %s", err)
	}

	if len(installedProducts) != 1 {
		t.Fatalf("no product certs read from: %s", rhsmClient.RHSMConf.RHSM.DefaultProductCertDir)
	}
}

// TestReadNoProductCertificates tries to test the case, when no product
// certificates are installed or preinstalled on the system
func TestReadNoProductCertificates(t *testing.T) {
	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for reading product certs, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	installedProducts, err := rhsmClient.readAllProductCertificates()
	if err != nil {
		t.Fatalf("reading of product certs failed: %s", err)
	}

	if len(installedProducts) != 0 {
		t.Fatalf("some product certs read despite no product certificates installed in %s and %s",
			rhsmClient.RHSMConf.RHSM.ProductCertDir, rhsmClient.RHSMConf.RHSM.DefaultProductCertDir)
	}
}
