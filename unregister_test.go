package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// helperTestInstalledFilesRemoved is helper function that test that files that have to be
// removed after unregister, has been really successfully removed and files that have to
// be kept, has not been removed.
func helperTestInstalledFilesRemoved(t *testing.T, testingFiles *TestingFileSystem) {
	// redhat.repo should be deleted
	if _, err := os.Stat(testingFiles.YumRepoFilePath); err == nil {
		t.Fatalf("redhat.repo has not been deleted during unregister process")
	}

	// Directory with content cert & key should be empty
	isEmpty, err := isDirEmpty(&testingFiles.ConsumerDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ConsumerDirPath, err)
	}
	if isEmpty == false {
		t.Fatalf("not all files have been deleted from %s during unregister process",
			testingFiles.ConsumerDirPath)
	}

	// Directory with entitlement cert & key should be empty
	isEmpty, err = isDirEmpty(&testingFiles.EntitlementDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.EntitlementDirPath, err)
	}
	if isEmpty == false {
		t.Fatalf("not all files have been deleted from %s during unregister process",
			testingFiles.EntitlementDirPath)
	}

	// Directory with default product certs should be still populated. Such files should never have been
	// deleted from this directory, because it is protected directory.
	isEmpty, err = isDirEmpty(&testingFiles.ProductDefaultDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ProductDefaultDirPath, err)
	}
	if isEmpty == true {
		t.Fatalf("certs cannot be deleted from %s during unregister process",
			testingFiles.ProductDefaultDirPath)
	}

	// Directory with installed product certs should be still populated too
	isEmpty, err = isDirEmpty(&testingFiles.ProductDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ProductDirPath, err)
	}
	if isEmpty == true {
		t.Fatalf("certs cannot be deleted from %s during unregister process",
			testingFiles.ProductDirPath)
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
	isEmpty, err := isDirEmpty(&testingFiles.ConsumerDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ConsumerDirPath, err)
	}
	if isEmpty == true {
		t.Fatalf("no consumer cert or key in: %s", testingFiles.ConsumerDirPath)
	}

	// Directory with entitlement cert & key should be installed
	isEmpty, err = isDirEmpty(&testingFiles.EntitlementDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.EntitlementDirPath, err)
	}
	if isEmpty == true {
		t.Fatalf("no entitlement key or cert in: %s",
			testingFiles.EntitlementDirPath)
	}

	// Directory with default product certs should be still populated. Such files should never have been
	// deleted from this directory, because it is protected directory.
	isEmpty, err = isDirEmpty(&testingFiles.ProductDefaultDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ProductDefaultDirPath, err)
	}
	if isEmpty == true {
		t.Fatalf("certs cannot be deleted from %s during unregister process",
			testingFiles.ProductDefaultDirPath)
	}

	// Directory with installed product certs should be still populated too
	isEmpty, err = isDirEmpty(&testingFiles.ProductDirPath)
	if err != nil {
		t.Fatalf("unable to read content of: %s: %s", testingFiles.ProductDirPath, err)
	}
	if isEmpty == true {
		t.Fatalf("certs cannot be deleted from %s during unregister process",
			testingFiles.ProductDirPath)
	}
}

// TestUnregisterRegisteredSystem tries to test unregistering of registered system
// using function RHSMClient.Unregister()
func TestUnregisterRegisteredSystem(t *testing.T) {
	t.Parallel()
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

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Calling tested function!
	err = rhsmClient.Unregister(nil)

	if err != nil {
		t.Fatalf("unregistering failed with error: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}

	// Test that installed files were removed
	helperTestInstalledFilesRemoved(t, testingFiles)
}

// TestClenRegisteredSystem tries to test cleaning of filesystem without calling
// any REST API call
func TestClenRegisteredSystem(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer( // It is expected that Unregister() method will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API should be called during Clean()")
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Calling tested function!
	err = rhsmClient.Clean()

	if err != nil {
		t.Fatalf("cleaning failed with error: %s", err)
	}

	// Test that installed files were removed
	helperTestInstalledFilesRemoved(t, testingFiles)
}

// TestUnregisterRegisteredSystemReadOnlyFileSystem tries to test unregistering of registered system
// using function RHSMClient.Unregister(). This case cover the case, when all files are only read-only
func TestUnregisterRegisteredSystemReadOnlyFileSystem(t *testing.T) {
	t.Parallel()
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

	// Add cleanup function for resetting permissions of files and directories
	t.Cleanup(func() {
		err := fixPermissionsOfDirsAndFiles(tempDirFilePath)
		if err != nil {
			t.Fatalf("unable to make file system read-write again: %s", err)
		}
	})

	testingFiles, err := setupTestingFileSystemReadOnly(
		tempDirFilePath, false, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Calling tested function!
	err = rhsmClient.Unregister(nil)
	if err != nil {
		t.Fatalf("unregistering failed with error: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}
}

// TestUnregisterUnRegisteredSystem tries to test unregistering of un-registered system
// using function RHSMClient.Unregister()
func TestUnregisterUnRegisteredSystem(t *testing.T) {
	t.Parallel()
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

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.Unregister(nil)
	if err == nil {
		t.Fatalf("unregistering failed with no error")
	}

	if handlerCounter != 0 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}
}

// TestUnregisterDeletedConsumer tries to test unregistering of registered system
// using function RHSMClient.Unregister(). This case is focused on the case, when
// consumer has been already deleted by some other DELETE /consumer/{consumer_uuid}
// by some other tool
func TestUnregisterDeletedConsumer(t *testing.T) {
	t.Parallel()
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
			_, _ = rw.Write([]byte(response410))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.Unregister(nil)
	if err == nil {
		t.Fatalf("unregistering failed with no error")
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}

	helperTestInstalledFilesRemoved(t, testingFiles)
}

// TestUnregisterWrongConsumer tries to test unregistering of registered system
// using function RHSMClient.Unregister(). This case is focused on the case, when
// system tries to delete consumer, but user does not have permission doing that.
func TestUnregisterWrongConsumer(t *testing.T) {
	t.Parallel()
	var expectedConsumerUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
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
			expectedURL := "/consumers/" + expectedConsumerUUID
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 403
			rw.WriteHeader(403)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return JSON document with error
			_, _ = rw.Write([]byte(response403))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.Unregister(nil)
	if err == nil {
		t.Fatalf("unregistering failed with no error")
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}

	// Test that no installed files were removed
	helperTestInstalledFilesNotRemoved(t, testingFiles)
}

// TestUnregisterInternalServerError tries to test unregistering of registered system
// using function RHSMClient.Unregister(). This case is focused on the case, when
// there is some internal server error.
func TestUnregisterInternalServerError(t *testing.T) {
	t.Parallel()
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
			_, _ = rw.Write([]byte(response500))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	err = rhsmClient.Unregister(nil)
	if err == nil {
		t.Fatalf("unregistering failed with no error")
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for unregister REST API pointed not called once, but called: %d", handlerCounter)
	}

	// Test that no installed files were removed
	helperTestInstalledFilesNotRemoved(t, testingFiles)
}
