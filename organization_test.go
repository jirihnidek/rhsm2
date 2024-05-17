package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// orgListResponse is response of candlepin server, when client asks for list of
// organizations the given user is member of.
const orgListResponse = `[ {
  "created" : "2023-10-31T09:25:10+0000",
  "updated" : "2023-10-31T09:25:16+0000",
  "id" : "4028fcc68b850d06018b850d1cb20002",
  "displayName" : "Admin Owner",
  "key" : "admin",
  "contentPrefix" : null,
  "defaultServiceLevel" : null,
  "logLevel" : null,
  "contentAccessMode" : "entitlement",
  "contentAccessModeList" : "entitlement",
  "autobindHypervisorDisabled" : false,
  "autobindDisabled" : false,
  "lastRefreshed" : "2023-10-31T09:25:16+0000",
  "parentOwner" : null,
  "upstreamConsumer" : null
}, {
  "created" : "2023-10-31T09:25:11+0000",
  "updated" : "2023-10-31T09:25:19+0000",
  "id" : "4028fcc68b850d06018b850d1d420003",
  "displayName" : "Snow White",
  "key" : "snowwhite",
  "contentPrefix" : null,
  "defaultServiceLevel" : null,
  "logLevel" : null,
  "contentAccessMode" : "entitlement",
  "contentAccessModeList" : "entitlement,org_environment",
  "autobindHypervisorDisabled" : false,
  "autobindDisabled" : false,
  "lastRefreshed" : "2023-10-31T09:25:19+0000",
  "parentOwner" : null,
  "upstreamConsumer" : null
}, {
  "created" : "2023-10-31T09:25:11+0000",
  "updated" : "2023-10-31T09:25:21+0000",
  "id" : "4028fcc68b850d06018b850d1d860004",
  "displayName" : "Donald Duck",
  "key" : "donaldduck",
  "contentPrefix" : null,
  "defaultServiceLevel" : null,
  "logLevel" : null,
  "contentAccessMode" : "org_environment",
  "contentAccessModeList" : "entitlement,org_environment",
  "autobindHypervisorDisabled" : false,
  "autobindDisabled" : false,
  "lastRefreshed" : "2023-10-31T09:25:21+0000",
  "parentOwner" : null,
  "upstreamConsumer" : null
} ]`

// TestGetOrganizations test the case, when client tries to get
// list of all organizations
func TestGetOrganizations(t *testing.T) {
	t.Parallel()
	handlerCounterGetOwners := 0

	username := "admin"
	password := "admin"

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodGet && reqURL == "/users/"+username+"/owners" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetOwners += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with list of organizations
				_, _ = rw.Write([]byte(orgListResponse))
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

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	orgs, err := rhsmClient.GetOrgs(username, password, nil)
	if err != nil {
		t.Fatalf("registration failed: %s", err)
	}

	if len(orgs) != 3 {
		t.Fatalf("expected 3 organizations in the returned list, got: %d", len(orgs))
	}

	if handlerCounterGetOwners != 1 {
		t.Fatalf("REST API point POST /users/%s/owners not called once", username)
	}

}

// orgListResponseCorrupted is corrupted JSON document (obvious reason)
const orgListResponseCorrupted = "{ [ } ]"

// TestGetOrganizationsCorruptedData test the case, when client tries to get
// list of all organizations, but the list is corrupted for some reason
func TestGetOrganizationsCorruptedData(t *testing.T) {
	t.Parallel()
	handlerCounterGetOwners := 0

	username := "admin"
	password := "admin"

	server := httptest.NewTLSServer(
		// It is expected that Register() method will call only
		// two REST API points
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Handler has to be a little bit more sophisticated in this
			// case, because we have to handle two types of REST API calls

			reqURL := req.URL.String()

			if req.Method == http.MethodGet && reqURL == "/users/"+username+"/owners" {
				// Increase number of calls of this REST API endpoint
				handlerCounterGetOwners += 1

				// Return code 200
				rw.WriteHeader(200)
				// Add some headers specific for candlepin server
				rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
				// Return JSON document with list of organizations
				_, _ = rw.Write([]byte(orgListResponseCorrupted))
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

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// TODO: try to use secure connection
	rhsmClient.RHSMConf.Server.Insecure = true

	orgs, err := rhsmClient.GetOrgs(username, password, nil)
	if err == nil {
		t.Fatalf("expected that registration will fail, when corrupted data returned")
	}

	if len(orgs) != 0 {
		t.Fatalf("expected no organization in the returned list, got: %d", len(orgs))
	}

	if handlerCounterGetOwners != 1 {
		t.Fatalf("REST API point POST /users/%s/owners not called once", username)
	}

}
