package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testConsumerResponse = `{
  "uuid" : "5e9745d5-624d-4af1-916e-2c17df4eb4e8",
  "name" : "localhost",
  "owner" : {
    "id" : "4028fcc68f4dcaa9018f4dcae87d0004",
    "key" : "donaldduck",
    "displayName" : "Donald Duck",
    "href" : "/owners/donaldduck",
    "contentAccessMode" : "org_environment"
  },
  "environments" : [
    {
      "created" : "2024-05-06T14:29:38+0000",
      "updated" : "2024-05-06T14:29:38+0000",
      "id" : "env-id-1",
      "name" : "env-name-1",
      "type" : "Opaque",
      "description" : "Testing environment #1",
      "contentPrefix" : "/content/dist/rhel9",
      "owner" : {
        "id" : "4028fcc68f4dcaa9018f4dcae87d0004",
        "key" : "donaldduck",
        "displayName" : "Donald Duck",
        "href" : "/owners/donaldduck",
        "contentAccessMode" : "org_environment"
      },
      "environmentContent" : [
        {
          "contentId" : "5001",
          "enabled" : true
        }
      ]
    },
    {
      "created" : "2024-05-06T14:30:02+0000",
      "updated" : "2024-05-06T14:30:02+0000",
      "id" : "env-id-2",
      "name" : "env-name-2",
      "type" : "Opaque",
      "description" : "Testing environment #2",
      "contentPrefix" : "/content/dist/rhel9",
      "owner" : {
        "id" : "4028fcc68f4dcaa9018f4dcae87d0004",
        "key" : "donaldduck",
        "displayName" : "Donald Duck",
        "href" : "/owners/donaldduck",
        "contentAccessMode" : "org_environment"
      },
      "environmentContent" : [
        {
          "contentId" : "5002",
          "enabled" : false
        }
      ]
    }
  ]
}`

const testConsumerResponseWithoutEnvironments = `{
  "uuid" : "5e9745d5-624d-4af1-916e-2c17df4eb4e8",
  "name" : "localhost",
  "owner" : {
    "key" : "donaldduck",
    "displayName" : "Donald Duck",
    "contentAccessMode" : "org_environment"
  },
  "environments" : null
}`

// TestGetConsumer test getting consumer data from the candlepin server
func TestGetConsumer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		wantUuid       string
		wantName       string
		wantErr        bool
		setupConsumer  bool
		checkEnvs      bool
	}{
		{
			name:           "successful response using consumer cert UUID",
			serverResponse: testConsumerResponse,
			statusCode:     200,
			wantUuid:       "5e9745d5-624d-4af1-916e-2c17df4eb4e8",
			wantName:       "localhost",
			wantErr:        false,
			setupConsumer:  true,
			checkEnvs:      true,
		},
		{
			name:           "successful response without environments",
			serverResponse: testConsumerResponseWithoutEnvironments,
			statusCode:     200,
			wantUuid:       "5e9745d5-624d-4af1-916e-2c17df4eb4e8",
			wantName:       "localhost",
			wantErr:        false,
			setupConsumer:  true,
			checkEnvs:      false,
		},
		{
			name:           "server error",
			serverResponse: "Internal Server Error",
			statusCode:     500,
			wantErr:        true,
			setupConsumer:  true,
		},
		{
			name:           "not found",
			serverResponse: response404,
			statusCode:     404,
			wantErr:        true,
			setupConsumer:  true,
		},
		{
			name:           "invalid json",
			serverResponse: `{"invalid": json}`,
			statusCode:     200,
			wantErr:        true,
			setupConsumer:  true,
		},
		{
			name:          "no consumer cert installed",
			wantErr:       true,
			setupConsumer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var expectedConsumerUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"

			server := httptest.NewTLSServer(
				http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodGet {
						t.Fatalf("unexpected HTTP method: %s", req.Method)
					}
					expectedURL := "/consumers/" + expectedConsumerUUID
					if req.URL.String() != expectedURL {
						t.Fatalf("expected request URL: %s, got: %s", expectedURL, req.URL.String())
					}
					rw.WriteHeader(tt.statusCode)
					_, _ = rw.Write([]byte(tt.serverResponse))
				}))
			defer server.Close()

			tempDirFilePath := t.TempDir()

			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath, false, tt.setupConsumer, false, false, true)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			consumerData, err := rhsmClient.GetConsumer(nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("%s: GetConsumer() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if consumerData == nil {
				t.Fatalf("%s: GetConsumer() returned nil consumer data without error", tt.name)
			}
			if consumerData.Uuid != tt.wantUuid {
				t.Errorf("%s: GetConsumer() uuid = %v, want %v", tt.name, consumerData.Uuid, tt.wantUuid)
			}
			if consumerData.Name != tt.wantName {
				t.Errorf("%s: GetConsumer() name = %v, want %v", tt.name, consumerData.Name, tt.wantName)
			}
			if consumerData.Owner.Key != "donaldduck" {
				t.Errorf("%s: GetConsumer() owner key = %v, want donaldduck", tt.name, consumerData.Owner.Key)
			}
			if tt.checkEnvs {
				assertConsumerEnvironments(t, tt.name, consumerData)
			} else if len(consumerData.Environments) > 0 {
				t.Errorf("%s: GetConsumer() expected no environments data, got environments=%+v",
					tt.name, consumerData.Environments)
			}
		})
	}
}

func assertConsumerEnvironments(t *testing.T, testName string, consumerData *ConsumerData) {
	t.Helper()

	if len(consumerData.Environments) != 2 {
		t.Fatalf("%s: GetConsumer() environments count = %d, want 2", testName, len(consumerData.Environments))
	}

	firstEnvironment := consumerData.Environments[0]
	if firstEnvironment.Id != "env-id-1" {
		t.Errorf("%s: GetConsumer() environments[0].id = %v, want env-id-1", testName, firstEnvironment.Id)
	}
	if firstEnvironment.Name != "env-name-1" {
		t.Errorf("%s: GetConsumer() environments[0].name = %v, want env-name-1", testName, firstEnvironment.Name)
	}
	if firstEnvironment.Description != "Testing environment #1" {
		t.Errorf("%s: GetConsumer() environments[0].description = %v, want Testing environment #1",
			testName, firstEnvironment.Description)
	}
	if firstEnvironment.ContentPrefix != "/content/dist/rhel9" {
		t.Errorf("%s: GetConsumer() environments[0].contentPrefix = %v, want /content/dist/rhel9",
			testName, firstEnvironment.ContentPrefix)
	}
	if firstEnvironment.Owner.Key != "donaldduck" {
		t.Errorf("%s: GetConsumer() environments[0].owner.key = %v, want donaldduck",
			testName, firstEnvironment.Owner.Key)
	}
	if len(firstEnvironment.EnvironmentContent) != 1 {
		t.Fatalf("%s: GetConsumer() environments[0].environmentContent count = %d, want 1",
			testName, len(firstEnvironment.EnvironmentContent))
	}
	firstEnvironmentContent, ok := firstEnvironment.EnvironmentContent[0].(map[string]interface{})
	if !ok {
		t.Fatalf("%s: GetConsumer() environments[0].environmentContent[0] is not a map", testName)
	}
	if firstEnvironmentContent["contentId"] != "5001" {
		t.Errorf("%s: GetConsumer() environments[0].environmentContent[0].contentId = %v, want 5001",
			testName, firstEnvironmentContent["contentId"])
	}
	if firstEnvironmentContent["enabled"] != true {
		t.Errorf("%s: GetConsumer() environments[0].environmentContent[0].enabled = %v, want true",
			testName, firstEnvironmentContent["enabled"])
	}

	secondEnvironment := consumerData.Environments[1]
	if secondEnvironment.Id != "env-id-2" {
		t.Errorf("%s: GetConsumer() environments[1].id = %v, want env-id-2", testName, secondEnvironment.Id)
	}
	if secondEnvironment.Name != "env-name-2" {
		t.Errorf("%s: GetConsumer() environments[1].name = %v, want env-name-2", testName, secondEnvironment.Name)
	}
	if len(secondEnvironment.EnvironmentContent) != 1 {
		t.Fatalf("%s: GetConsumer() environments[1].environmentContent count = %d, want 1",
			testName, len(secondEnvironment.EnvironmentContent))
	}
	secondEnvironmentContent, ok := secondEnvironment.EnvironmentContent[0].(map[string]interface{})
	if !ok {
		t.Fatalf("%s: GetConsumer() environments[1].environmentContent[0] is not a map", testName)
	}
	if secondEnvironmentContent["contentId"] != "5002" {
		t.Errorf("%s: GetConsumer() environments[1].environmentContent[0].contentId = %v, want 5002",
			testName, secondEnvironmentContent["contentId"])
	}
	if secondEnvironmentContent["enabled"] != false {
		t.Errorf("%s: GetConsumer() environments[1].environmentContent[0].enabled = %v, want false",
			testName, secondEnvironmentContent["enabled"])
	}
}

// TestGetConsumerUUID test getting consumer UUID from
// installed consumer certificate
func TestGetConsumerUUID(t *testing.T) {
	t.Parallel()
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

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	consumerUuid, err := rhsmClient.GetConsumerUUID()
	if err != nil {
		t.Fatalf("unable to get consumer UUID from consumer cert: %s", err)
	} else {
		if *consumerUuid != "5e9745d5-624d-4af1-916e-2c17df4eb4e8" {
			t.Fatalf("consumer UUID: '%s' != '5e9745d5-624d-4af1-916e-2c17df4eb4e8'", *consumerUuid)
		}
	}
}

// TestGetOwner test getting owner (organization ID) from installed
// consumer certificate
func TestGetOwner(t *testing.T) {
	t.Parallel()
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

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
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
