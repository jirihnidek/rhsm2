package rhsm2

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isDirEmpty tries to check if directory is empty
func isDirEmpty(name *string) (bool, error) {
	f, err := os.Open(*name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// copyFile tries to copy file
func copyFile(srcFilePath *string, dstFilePath *string) error {
	pemIn, err := os.Open(*srcFilePath)
	if err != nil {
		return fmt.Errorf("unable to open file: %s: %s", *srcFilePath, err)
	}

	pemOut, err := os.Create(*dstFilePath)
	if err != nil {
		return fmt.Errorf("unable to create file: %s: %s", *dstFilePath, err)
	}

	_, err = io.Copy(pemOut, pemIn)
	if err != nil {
		return fmt.Errorf("unable to copy %s to %s: %s", *srcFilePath, *dstFilePath, err)
	}
	return nil
}

// TestingFileSystem is structure holding information about file paths
// used for testing
type TestingFileSystem struct {
	ConsumerDirFilePath       string
	EntitlementDirFilePath    string
	ProductDirFilePath        string
	ProductDefaultDirFilePath string
	YumRepoFilePath           string
}

// setupTestingFiles tries to copy and generate testing files to testing directories
func setupTestingFiles(tempDirFilePath string, testingFileSystem *TestingFileSystem) error {

	// Copy consumer key to temporary directory
	srcConsumerKeyFilePath := "./test/pki/consumer/key.pem"
	dstConsumerKeyFilePath := filepath.Join(testingFileSystem.ConsumerDirFilePath, "key.pem")
	err := copyFile(&srcConsumerKeyFilePath, &dstConsumerKeyFilePath)
	if err != nil {
		return fmt.Errorf(
			"unable to create testing consumer key file: %s", err)
	}
	// Copy consumer cert to temporary directory
	srcConsumerCertFilePath := "test/pki/consumer/cert.pem"
	dstConsumerCertFilePath := filepath.Join(testingFileSystem.ConsumerDirFilePath, "cert.pem")
	err = copyFile(&srcConsumerCertFilePath, &dstConsumerCertFilePath)
	if err != nil {
		return fmt.Errorf(
			"unable to create testing consumer cert file: %s", err)
	}

	// Copy entitlement key to temporary directory
	srcEntitlementKeyFilePath := "./test/pki/entitlement/6490061114713729830-key.pem"
	dstEntitlementKeyFilePath := filepath.Join(testingFileSystem.EntitlementDirFilePath, "6490061114713729830-key.pem")
	err = copyFile(&srcEntitlementKeyFilePath, &dstEntitlementKeyFilePath)
	if err != nil {
		return fmt.Errorf(
			"unable to create testing entitlement key file: %s", err)
	}
	// Copy entitlement cert to temporary directory
	srcEntitlementCertFilePath := "./test/pki/entitlement/6490061114713729830.pem"
	dstEntitlementCertFilePath := filepath.Join(testingFileSystem.EntitlementDirFilePath, "6490061114713729830.pem")
	err = copyFile(&srcEntitlementCertFilePath, &dstEntitlementCertFilePath)
	if err != nil {
		return fmt.Errorf("unable to create testing entitlement cert file: %s", err)
	}

	// Copy product cert to temporary directory
	srcProductCertFilePath := "./test/pki/product/900.pem"
	dstProductCertFilePath := filepath.Join(testingFileSystem.ProductDirFilePath, "900.pem")
	err = copyFile(&srcProductCertFilePath, &dstProductCertFilePath)
	if err != nil {
		return fmt.Errorf("unable to create testing product cert file: %s", err)
	}

	// Copy default product cert to temporary directory
	srcDefaultProductCertFilePath := "./test/pki/product-default/5050.pem"
	dstDefaultProductCertFilePath := filepath.Join(testingFileSystem.ProductDefaultDirFilePath, "5050.pem")
	err = copyFile(&srcDefaultProductCertFilePath, &dstDefaultProductCertFilePath)
	if err != nil {
		return fmt.Errorf("unable to create testing default product cert file: %s", err)
	}

	// Create empty redhat.repo
	yumRepoFilePath := filepath.Join(tempDirFilePath, "redhat.repo")
	_, err = os.Create(yumRepoFilePath)
	if err != nil {
		return fmt.Errorf("unable to create %s: %s", yumRepoFilePath, err)
	}
	// TODO: populate redhat.repo with content from installed entitlement certificate
	testingFileSystem.YumRepoFilePath = yumRepoFilePath

	return nil
}

// setupTestingDirectories tries to set up directories for testing filesystem
func setupTestingDirectories(tempDirFilePath string) (*TestingFileSystem, error) {
	testingFileSystem := TestingFileSystem{}
	// Create temporary directory for consumer certificates
	consumerDirFilePath := filepath.Join(tempDirFilePath, "pki/consumer")
	err := os.MkdirAll(consumerDirFilePath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", consumerDirFilePath, err)
	}
	testingFileSystem.ConsumerDirFilePath = consumerDirFilePath

	// Create temporary directory for entitlement certificates
	entitlementDirFilePath := filepath.Join(tempDirFilePath, "pki/entitlement")
	err = os.MkdirAll(entitlementDirFilePath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", entitlementDirFilePath, err)
	}
	testingFileSystem.EntitlementDirFilePath = entitlementDirFilePath

	// Create temporary directory for product certificates
	productDirFilePath := filepath.Join(tempDirFilePath, "pki/product")
	err = os.MkdirAll(productDirFilePath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", productDirFilePath, err)
	}
	testingFileSystem.ProductDirFilePath = productDirFilePath

	// Create temporary directory for product certificates
	productDefaultDirFilePath := filepath.Join(tempDirFilePath, "pki/product-default")
	err = os.MkdirAll(productDefaultDirFilePath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", productDefaultDirFilePath, err)
	}
	testingFileSystem.ProductDefaultDirFilePath = productDefaultDirFilePath
	return &testingFileSystem, nil
}

// setupTestingFileSystem tries to set up directories and files for testing and mock system
// that is fully installed
func setupTestingFileSystem(tempDirFilePath string, registered bool) (*TestingFileSystem, error) {
	testingFileSystem, err := setupTestingDirectories(tempDirFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create testing directories: %s", err)
	}

	if registered {
		err = setupTestingFiles(tempDirFilePath, testingFileSystem)
		if err != nil {
			return nil, fmt.Errorf("unable to copy testing file to testing directories: %s", err)
		}
	}

	return testingFileSystem, nil
}

// setupTestingRHSMClient tries to set up testing instance of RHSMClient
func setupTestingRHSMClient(testingFiles *TestingFileSystem, server *httptest.Server) (*RHSMClient, error) {
	// Get the hostname, port and prefix from the fake server
	// It will be used for configuring rhsm client
	parsedURL, err := url.Parse(server.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse server URL: %s: %s", server.URL, err)
	}
	hostname := parsedURL.Hostname()
	port := parsedURL.Port()
	prefix := parsedURL.Path

	// Create instance of RHSM client
	rhsmClient := RHSMClient{}

	// Fill rhsm conf with fake data and temporary paths
	rhsmClient.RHSMConf = &RHSMConf{
		yumRepoFilePath: testingFiles.YumRepoFilePath,
		Server: RHSMConfServer{
			Hostname: hostname,
			Port:     port,
			Prefix:   prefix,
		},
		RHSM: RHSMConfRHSM{
			ConsumerCertDir:    testingFiles.ConsumerDirFilePath,
			EntitlementCertDir: testingFiles.EntitlementDirFilePath,
			ProductCertDir:     testingFiles.ProductDirFilePath,
		},
	}

	// Mock connections to server with mock server
	rhsmClient.NoAuthConnection = &RHSMConnection{
		AuthType:       NoAuth,
		Client:         server.Client(),
		ServerHostname: &hostname,
		ServerPort:     &port,
		ServerPrefix:   &prefix,
	}
	rhsmClient.ConsumerCertAuthConnection = &RHSMConnection{
		AuthType:       ConsumerCertAuth,
		Client:         server.Client(),
		ServerHostname: &hostname,
		ServerPort:     &port,
		ServerPrefix:   &prefix,
	}

	// TODO: populate rhsm.conf with paths of temporary files and server hostname, port, prefix, etc.

	return &rhsmClient, nil
}

// helperTestInstalledFilesRemoved is helper function that test that files that have to be
// removed after unregister, has been really successfully removed and files that have to
// be kept, has not been removed.
func helperTestInstalledFilesRemoved(t *testing.T, testingFiles *TestingFileSystem) {
	// redhat.repo should be deleted
	if _, err := os.Stat(testingFiles.YumRepoFilePath); err == nil {
		t.Fatalf("redhat.repo has not been deleted during unregister process")
	}

	// Directory with content cert & key should be empty
	isEmpty, err := isDirEmpty(&testingFiles.ConsumerDirFilePath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ConsumerDirFilePath, err)
	}
	if isEmpty == false {
		t.Fatalf("not all files have been deleted from %s during unregister process",
			testingFiles.ConsumerDirFilePath)
	}

	// Directory with entitlement cert & key should be empty
	isEmpty, err = isDirEmpty(&testingFiles.EntitlementDirFilePath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.EntitlementDirFilePath, err)
	}
	if isEmpty == false {
		t.Fatalf("not all files have been deleted from %s during unregister process",
			testingFiles.EntitlementDirFilePath)
	}

	// Directory with default product certs should be still populated. Such files should never have been
	// deleted from this directory, because it is protected directory.
	isEmpty, err = isDirEmpty(&testingFiles.ProductDefaultDirFilePath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ProductDefaultDirFilePath, err)
	}
	if isEmpty == true {
		t.Fatalf("certs cannot be deleted from %s during unregister process",
			testingFiles.ProductDefaultDirFilePath)
	}

	// Directory with installed product certs should be still populated too
	isEmpty, err = isDirEmpty(&testingFiles.ProductDirFilePath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ProductDirFilePath, err)
	}
	if isEmpty == true {
		t.Fatalf("certs cannot be deleted from %s during unregister process",
			testingFiles.ProductDirFilePath)
	}
}

// helperTestInstalledFilesNotRemoved is helper function for testing that installed
// files are still installed on the system.
func helperTestInstalledFilesNotRemoved(t *testing.T, testingFiles *TestingFileSystem) {
	// redhat.repo should not be deleted
	if _, err := os.Stat(testingFiles.YumRepoFilePath); err != nil {
		t.Fatalf("redhat.repo was deleted")
	}

	// Directory with content cert & key should be installed
	isEmpty, err := isDirEmpty(&testingFiles.ConsumerDirFilePath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ConsumerDirFilePath, err)
	}
	if isEmpty == true {
		t.Fatalf("no consumer cert or key in: %s", testingFiles.ConsumerDirFilePath)
	}

	// Directory with entitlement cert & key should be installed
	isEmpty, err = isDirEmpty(&testingFiles.EntitlementDirFilePath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.EntitlementDirFilePath, err)
	}
	if isEmpty == true {
		t.Fatalf("no entitlement key or cert in: %s",
			testingFiles.EntitlementDirFilePath)
	}

	// Directory with default product certs should be still populated. Such files should never have been
	// deleted from this directory, because it is protected directory.
	isEmpty, err = isDirEmpty(&testingFiles.ProductDefaultDirFilePath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ProductDefaultDirFilePath, err)
	}
	if isEmpty == true {
		t.Fatalf("certs cannot be deleted from %s during unregister process",
			testingFiles.ProductDefaultDirFilePath)
	}

	// Directory with installed product certs should be still populated too
	isEmpty, err = isDirEmpty(&testingFiles.ProductDirFilePath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ProductDirFilePath, err)
	}
	if isEmpty == true {
		t.Fatalf("certs cannot be deleted from %s during unregister process",
			testingFiles.ProductDirFilePath)
	}
}

// TestUnregisterRegisteredSystem tries to test unregistering of registered system
// using function RHSMClient.Unregister()
func TestUnregisterRegisteredSystem(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodDelete {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodDelete, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 204
			rw.WriteHeader(204)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(""))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(tempDirFilePath, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Calling tested function!
	err = rhsmClient.Unregister()
	if err != nil {
		t.Fatalf("unregistering failed with error: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}

	// Test that installed files were removed
	helperTestInstalledFilesRemoved(t, testingFiles)
}

// TestUnregisterUnRegisteredSystem tries to test unregistering of un-registered system
// using function RHSMClient.Unregister()
func TestUnregisterUnRegisteredSystem(t *testing.T) {
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will not be called at all
		// in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodDelete {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodDelete, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/"
			reqURL := req.URL.String()
			if strings.HasPrefix(reqURL, expectedURL) {
				t.Fatalf("URL %s does not start with: %s", reqURL, expectedURL)
			}

			t.Fatalf("REST API endpoing DELETE /consumers/{consumer_uuid}" +
				"should not be called, when system is unregistered")
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(tempDirFilePath, false)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.Unregister()
	if err == nil {
		t.Fatalf("unregistering failed with no error")
	}

	if handlerCounter != 0 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}
}

const consumerAlreadyDeleted = "{\n  \"displayMessage\": \"Consumer with this UUID is already deleted.\",\n  \"requestUuid\": \"c4347004-8792-41fe-a4d8-fccaa0d3898a\"\n}"

// TestUnregisterDeletedConsumer tries to test unregistering of registered system
// using function RHSMClient.Unregister(). This case is focused on the case, when
// consumer has been already deleted by some other DELETE /consumer/{consumer_uuid}
// by some other tool
func TestUnregisterDeletedConsumer(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodDelete {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodDelete, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 410
			rw.WriteHeader(410)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return JSON document with error
			_, _ = rw.Write([]byte(consumerAlreadyDeleted))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(tempDirFilePath, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.Unregister()
	if err == nil {
		t.Fatalf("unregistering failed with no error")
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}

	helperTestInstalledFilesRemoved(t, testingFiles)
}

const insufficientPermissions = "{\n  \"displayMessage\": \"Consumer could not be deleted due to insufficient permissions.\",\n  \"requestUuid\": \"c4347004-8792-41fe-a4d8-fccaa0d3898a\"\n}"

// TestUnregisterWrongConsumer tries to test unregistering of registered system
// using function RHSMClient.Unregister(). This case is focused on the case, when
// system tries to delete consumer, but user does not have permission doing that.
func TestUnregisterWrongConsumer(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodDelete {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodDelete, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 403
			rw.WriteHeader(403)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return JSON document with error
			_, _ = rw.Write([]byte(insufficientPermissions))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(tempDirFilePath, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.Unregister()
	if err == nil {
		t.Fatalf("unregistering failed with no error")
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}

	// Test that no installed files were removed
	helperTestInstalledFilesNotRemoved(t, testingFiles)
}

const internalServerError = "{\n  \"displayMessage\": \"An unexpected exception has occurred\",\n  \"requestUuid\": \"c4347004-8792-41fe-a4d8-fccaa0d3898a\"\n}"

// TestUnregisterInternalServerError tries to test unregistering of registered system
// using function RHSMClient.Unregister(). This case is focused on the case, when
// there is some internal server error.
func TestUnregisterInternalServerError(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodDelete {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodDelete, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 500
			rw.WriteHeader(500)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return JSON document with error
			_, _ = rw.Write([]byte(internalServerError))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(tempDirFilePath, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.Unregister()
	if err == nil {
		t.Fatalf("unregistering failed with no error")
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}

	// Test that no installed files were removed
	helperTestInstalledFilesNotRemoved(t, testingFiles)
}
