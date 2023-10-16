package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetConsumerUUID(t *testing.T) {
	server := httptest.NewTLSServer(
		// It is expected that GetConsumerUUID() will not trigger any
		// REST API call
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for reading installed consumer cert, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	uuid, err := rhsmClient.GetConsumerUUID()
	if err != nil {
		t.Fatalf("unable to get consumer UUID from consumer cert: %s", err)
	} else {
		if *uuid != "5e9745d5-624d-4af1-916e-2c17df4eb4e8" {
			t.Fatalf("consumer UUID: '%s' != '5e9745d5-624d-4af1-916e-2c17df4eb4e8'", *uuid)
		}
	}
}

func TestGetOwner(t *testing.T) {
	server := httptest.NewTLSServer(
		// It is expected that GetOwner() will not trigger any
		// REST API call
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for reading installed consumer cert, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	orgID, err := rhsmClient.GetOwner()

	if err != nil {
		t.Fatalf("unable to get organization ID from consumer cert: %s", err)
	} else {
		if *orgID != "donaldduck" {
			t.Fatalf("org ID: '%s' != 'donaldduck'", *orgID)
		}
	}
}
