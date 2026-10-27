package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"

	"gopkg.in/ini.v1"
)

// TestGetEngineeringProducts tests the case, when engineering products are
// successfully loaded from the list of entitlement certificates
func TestGetEngineeringProducts(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create the root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	engineeringProducts, err := rhsmClient.getEngineeringProducts()
	if err != nil {
		t.Fatalf("unable to get engineering products from files: %s", err)
	}

	if len(engineeringProducts) != 1 {
		t.Fatalf("files does not contain one engineering product")
	}

	var s = testEntCertSerialNumber

	testEntSerialNumber, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		t.Fatalf("unable to parse test entitlement serial number: %s", err)
	}

	if engProducts, has := engineeringProducts[testEntSerialNumber]; has {
		for _, engProduct := range engProducts {
			if engProduct.Name != " Content Access" {
				t.Fatalf("expected product name: ' Content Access', got: '%s'", engProduct.Name)
			}
		}
	}

}

// TestGetEngineeringProductsError tests the case when engineering products cannot
// be loaded from the empty file system
func TestGetEngineeringProductsError(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create the root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, false)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	engineeringProducts, err := rhsmClient.getEngineeringProducts()
	if err != nil {
		t.Fatalf("unable to get engineering products from files: %s", err)
	}

	if len(engineeringProducts) != 0 {
		t.Fatalf("expected no engineering product, got: %d", len(engineeringProducts))
	}
}

// TestGetContentFromEntCert tests the case, when content is successfully loaded
// from entitlement certificate
func TestGetContentFromEntCert(t *testing.T) {
	t.Parallel()
	filePath := "testdata/etc/pki/entitlement/" + testEntCertSerialNumber + ".pem"
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

	if len(engProduct.Content) == 0 {
		t.Fatalf("expected non-zero number of product content definitions, got: '%v'", len(engProduct.Content))
	}

	// Note: Following test depends on the content of the entitlement certificate
	//       in the testdata directory ./testdata/etc/pki/entitlement/. If you will update
	//       the entitlement certificate, then you will need to update the test as well.

	// Test content definition only for the first content
	content := engProduct.Content[0]

	// Id
	expectedContentId := "2134123412366401"
	if content.Id != expectedContentId {
		t.Fatalf("expected content id: '%s', got: '%s'", expectedContentId, content.Id)
	}
	// Name
	expectedContentName := "awesomeos-x86_64-only-content-213412341236"
	if content.Name != expectedContentName {
		t.Fatalf("expected content name: '%s', got: '%s'", expectedContentName, content.Name)
	}
	// Label
	expectedContentLabel := "awesomeos-x86_64-only-content-213412341236"
	if content.Label != expectedContentLabel {
		t.Fatalf("expected content label: '%s', got: '%s'", expectedContentLabel, content.Label)
	}
	// Path
	expectedContentPath := "/path/to/awesomeos/x86_64_content/213412341236-6401"
	if content.Path != expectedContentPath {
		t.Fatalf("expected content path: '%s', got: '%s'", expectedContentPath, content.Path)
	}

	// Note: if you are bored, then add tests for remaining fields of 'content' struct
}

type TestRepo struct {
	Name            string `ini:"name"`
	BaseURL         string `ini:"baseurl"`
	Enabled         bool   `ini:"enabled"`
	EnabledMetadata bool   `ini:"enabled_metadata"`
	GPGCheck        bool   `ini:"gpgcheck"`
	GPGKey          string `ini:"gpgkey"`
	SSLVerify       bool   `ini:"sslverify"`
	SSLCACert       string `ini:"sslcacert"`
	SSLClientCert   string `ini:"sslclientcert"`
	SSLClientKey    string `ini:"sslclientkey"`
	MetadataExpire  int    `ini:"metadata_expire"`
}

// TestWriteRepoFile test the case, when content definition from
// entitlement certificate is successfully written to repo file
func TestWriteRepoFile(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	cdnServer := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer cdnServer.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, cdnServer)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.GenerateRepoFileFromInstalledEntitlementCerts()

	if err != nil {
		t.Fatalf("unable to generate '%s': %s", testingFiles.YumRepoFilePath, err)
	}

	repoFile, err := ini.Load(testingFiles.YumRepoFilePath)
	if err != nil {
		t.Fatalf("unable to read '%s' as INI: %s", testingFiles.YumRepoFilePath, err)
	}

	baseURL := rhsmClient.RHSMConf.RHSM.BaseURL
	expectedURLFound := false
	for _, repo := range repoFile.Sections() {
		var testRepo TestRepo
		err = repo.MapTo(&testRepo)
		if err != nil {
			t.Fatalf("unable to map repo: %s", err)
		}

		if testRepo.Name == "awesomeos-x86_64-only-content-213412341236" {
			expectedURL, err := url.Parse(baseURL + "/path/to/awesomeos/x86_64_content/213412341236-6401")
			if err != nil {
				t.Fatalf("unable to parse expected URL: %s", err)
			}
			if expectedURL.String() != testRepo.BaseURL {
				t.Fatalf("expected URL %s, got %s", expectedURL, testRepo.BaseURL)
			}
			expectedURLFound = true
			// Only one test is enough
			break
		}
	}

	if !expectedURLFound {
		t.Fatalf("expected URL not found")
	}
}

// TestWriteRepoFileFullURL test the case, when content definition in
// entitlement certificate contains full URL
func TestWriteRepoFileFullURL(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	cdnServer := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer cdnServer.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, false)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	srcEntCert := "./testdata/use_cases/ent_cert_with_full_url/174185085859391424.pem"
	dstEntCert := filepath.Join(testingFiles.EntitlementDirPath, "174185085859391424.pem")
	err = copyFile(&srcEntCert, &dstEntCert, nil)
	if err != nil {
		t.Fatalf("unable to copy entitlement certificate: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, cdnServer)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.GenerateRepoFileFromInstalledEntitlementCerts()

	if err != nil {
		t.Fatalf("unable to generate '%s': %s", testingFiles.YumRepoFilePath, err)
	}

	repoFile, err := ini.Load(testingFiles.YumRepoFilePath)
	if err != nil {
		t.Fatalf("unable to load '%s': %s", testingFiles.YumRepoFilePath, err)
	}

	fullURLCounter := 0
	for _, repo := range repoFile.Sections() {
		var testRepo TestRepo
		err = repo.MapTo(&testRepo)
		if err != nil {
			t.Fatalf("unable to map repo: %s", err)
		}
		switch testRepo.Name {
		case "EPEL 9 Everything x86_64":
			expectedURL := "https://dl.fedoraproject.org/pub/epel/9/Everything/x86_64"
			if testRepo.BaseURL != expectedURL {
				t.Fatalf("For %s, expected: %s, got: %s", testRepo.Name, expectedURL, testRepo.BaseURL)
			}
			fullURLCounter++
		case "Red Hat Enterprise Linux 9 for x86_64 - AppStream (RPMs)":
			expectedURL := "https://cert.console.redhat.com/api/pulp-content/ccac33ac/templates/7ec325b9-d3bb-407b-ab05-58ce63b4bb24/content/dist/rhel9/$releasever/x86_64/appstream/os"
			if testRepo.BaseURL != expectedURL {
				t.Fatalf("For %s, expected: %s, got: %s", testRepo.Name, expectedURL, testRepo.BaseURL)
			}
			fullURLCounter++
		case "Red Hat Enterprise Linux 9 for x86_64 - BaseOS (RPMs)":
			expectedURL := "https://cert.console.redhat.com/api/pulp-content/ccac33ac/templates/7ec325b9-d3bb-407b-ab05-58ce63b4bb24/content/dist/rhel9/$releasever/x86_64/baseos/os"
			if testRepo.BaseURL != expectedURL {
				t.Fatalf("For %s, expected: %s, got: %s", testRepo.Name, expectedURL, testRepo.BaseURL)
			}
			fullURLCounter++
		default:
		}
	}

	if fullURLCounter != 3 {
		t.Fatalf("Expected 3 full URLs, got %d", fullURLCounter)
	}
}

// TestWriteRepoFileNoEntCert test the case, when content definition from
// entitlement certificate is not possible to read, because there is no
// entitlement certificate installed
func TestWriteRepoFileNoEntCert(t *testing.T) {
	t.Parallel()
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
		tempDirFilePath, false, true, false, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.GenerateRepoFileFromInstalledEntitlementCerts()

	if err != nil {
		t.Fatalf("when no entitlement certificate installed, error returned: %s", err)
	}
}
