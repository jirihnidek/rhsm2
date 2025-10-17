package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const environmentListResponse = `[
    {
        "created": "2024-05-06T14:29:38+0000",
        "updated": "2024-05-06T14:29:38+0000",
        "id": "env-id-1",
        "name": "env-name-1",
        "type": null,
        "description": "Testing environment #1",
        "contentPrefix": null,
        "owner": {
            "id": "4028fcc68f4dcaa9018f4dcae87d0004",
            "key": "donaldduck",
            "displayName": "Donald Duck",
            "href": "/owners/donaldduck",
            "contentAccessMode": "org_environment"
        },
        "environmentContent": []
    },
    {
        "created": "2024-05-06T14:30:02+0000",
        "updated": "2024-05-06T14:30:02+0000",
        "id": "env-id-2",
        "name": "env-name-2",
        "type": null,
        "description": "Testing environment #2",
        "contentPrefix": null,
        "owner": {
            "id": "4028fcc68f4dcaa9018f4dcae87d0004",
            "key": "donaldduck",
            "displayName": "Donald Duck",
            "href": "/owners/donaldduck",
            "contentAccessMode": "org_environment"
        },
        "environmentContent": []
    },
    {
        "created": "2024-05-17T09:47:16+0000",
        "updated": "2024-05-17T09:47:16+0000",
        "id": "env-id-3",
        "name": "env-name-3",
        "type": null,
        "description": "Testing environment #3",
        "contentPrefix": null,
        "owner": {
            "id": "4028fcc68f4dcaa9018f4dcae87d0004",
            "key": "donaldduck",
            "displayName": "Donald Duck",
            "href": "/owners/donaldduck",
            "contentAccessMode": "org_environment"
        },
        "environmentContent": []
    }
]`

// TestGetEnvironments test the case, when client tries to get
// list of all environments
func TestGetEnvironments(t *testing.T) {
	t.Parallel()
	handlerCounterGetEnvironments := 0

	username := "admin"
	password := "admin"
	organization := "donaldduck"

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			reqURL := req.URL.String()

			if req.Method == http.MethodGet && reqURL == "/owners/"+organization+"/environments" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetEnvironments += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with list of environments
				_, _ = rw.Write([]byte(environmentListResponse))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	environments, err := rhsmClient.GetEnvironments(username, password, organization, nil)
	if err != nil {
		t.Fatalf("getting environments failed: %s", err)
	}

	if len(environments) != 3 {
		t.Fatalf("expected 3 environments in the returned list, got: %d", len(environments))
	}

	for _, environment := range environments {
		if environment.Owner.Key != organization {
			t.Fatalf("expected organization to be owned by %s, got: %s", organization, environment.Owner.Key)
		}
	}

	if handlerCounterGetEnvironments != 1 {
		t.Fatalf("REST API point POST /owners/%s/environments not called once", organization)
	}

}

// environmentListResponseCorrupted is used for the case, when candlepin server
// returns corrupted data
const environmentListResponseCorrupted = "[ { ] }"

// TestGetEnvironmentsCorruptedList test the case, when client tries to get
// list of all environments, but server returns corrupted data
func TestGetEnvironmentsCorruptedList(t *testing.T) {
	t.Parallel()
	handlerCounterGetEnvironments := 0

	username := "admin"
	password := "admin"
	organization := "donaldduck"

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			reqURL := req.URL.String()

			if req.Method == http.MethodGet && reqURL == "/owners/"+organization+"/environments" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetEnvironments += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with list of environments
				_, _ = rw.Write([]byte(environmentListResponseCorrupted))
			} else {
				t.Fatalf("unexpected REST API call: %s %s", req.Method, reqURL)
			}
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	environments, err := rhsmClient.GetEnvironments(username, password, organization, nil)
	if err == nil {
		t.Fatalf("receiving of corrupted data did not cause error")
	}

	if len(environments) != 0 {
		t.Fatalf("expected no environment in the returned list, got: %d", len(environments))
	}

	if handlerCounterGetEnvironments != 1 {
		t.Fatalf("REST API point POST /owners/%s/environments not called once", organization)
	}

}
