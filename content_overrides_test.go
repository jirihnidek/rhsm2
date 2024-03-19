package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const contentOverridesList = `[ {
  "created" : "2023-10-23T11:54:30+0000",
  "updated" : "2023-10-23T11:54:30+0000",
  "name" : "enabled",
  "contentLabel" : "awesomeos-801",
  "value" : "1"
}, {
  "created" : "2023-10-23T11:54:30+0000",
  "updated" : "2023-10-23T11:54:30+0000",
  "name" : "enabled_metadata",
  "contentLabel" : "awesomeos-801",
  "value" : "1"
} ]
`

// TestGetContentOverrides test the case, when it is
// possible to get content overrides from server
func TestGetContentOverrides(t *testing.T) {
	t.Parallel()
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that getContentOverrides() will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID + "/content_overrides"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 200
			rw.WriteHeader(200)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(contentOverridesList))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	clientInfo := ClientInfo{"", "", "66bf0b7a-aaae-4b31-a7bf-bc22052afebf"}
	contentOverrides, err := rhsmClient.getContentOverrides(&clientInfo)
	if err != nil {
		t.Fatalf("unable to get list of content overrides: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler called: %d, expected 1 call", handlerCounter)
	}

	if len(contentOverrides) != 2 {
		t.Fatalf("expected length of content overrides: 2, got: %d", len(contentOverrides))
	}
}

// TestGetContentOverridesInsufficientPermissions test the case, when server
// response with 403 error
func TestGetContentOverridesInsufficientPermissions(t *testing.T) {
	t.Parallel()
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that getContentOverrides() will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID + "/content_overrides"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 404
			rw.WriteHeader(403)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(response403))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	clientInfo := ClientInfo{"", "", "66bf0b7a-aaae-4b31-a7bf-bc22052afebf"}
	_, err = rhsmClient.getContentOverrides(&clientInfo)
	if err == nil {
		t.Fatalf("no error raised, when server responses with 403 status code")
	}
}

// TestGetContentOverridesWrongConsumer test the case, when server
// response with 404 error
func TestGetContentOverridesWrongConsumer(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that getContentOverrides() will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID + "/content_overrides"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 404
			rw.WriteHeader(404)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(response404))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	clientInfo := ClientInfo{"", "", "66bf0b7a-aaae-4b31-a7bf-bc22052afebf"}
	_, err = rhsmClient.getContentOverrides(&clientInfo)
	if err == nil {
		t.Fatalf("no error raised, when server responses with 404 status code")
	}
}

// TestGetContentOverridesInternalServerError test the case, when server
// response with status code 500
func TestGetContentOverridesInternalServerError(t *testing.T) {
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		// It is expected that getContentOverrides() will call only
		// one REST API point
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Increase number of calls
			handlerCounter += 1

			// Test request method
			if req.Method != http.MethodGet {
				t.Fatalf("extepected request method: %s, got: %s", http.MethodGet, req.Method)
			}

			// Test that requested URL is correct
			expectedURL := "/consumers/" + expectedClientUUID + "/content_overrides"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			// Return code 404
			rw.WriteHeader(500)
			// Add some headers specific for candlepin server
			rw.Header().Add("x-candlepin-request-uuid", "168e3687-8498-46b2-af0a-272583d4d4ba")
			// Return empty body
			_, _ = rw.Write([]byte(response500))
		}))
	defer server.Close()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when system is only registered
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	clientInfo := ClientInfo{"", "", "66bf0b7a-aaae-4b31-a7bf-bc22052afebf"}
	_, err = rhsmClient.getContentOverrides(&clientInfo)
	if err == nil {
		t.Fatalf("no error raised, when server responses with 500 status code")
	}
}

// Test_createMapFromContentOverrides tests creating map from list of ContentOverrides
// returned form candlepin server
func Test_createMapFromContentOverrides(t *testing.T) {
	type args struct {
		contentOverrides []ContentOverride
	}
	tests := []struct {
		name string
		args args
		want map[string]map[string]string
	}{
		{
			"empty content overrides",
			args{contentOverrides: make([]ContentOverride, 0)},
			make(map[string]map[string]string),
		},
		{
			"one content override",
			args{
				contentOverrides: []ContentOverride{
					{
						ContentLabel: "awesome_os-801",
						Updated:      "2023-10-23T11:54:30+0000",
						Created:      "2023-10-23T11:54:30+0000",
						Name:         "enabled",
						Value:        "1",
					},
				},
			},
			map[string]map[string]string{"awesome_os-801": {"enabled": "1"}},
		},
		{
			"two content overrides",
			args{
				contentOverrides: []ContentOverride{
					{
						ContentLabel: "awesome_os-801",
						Updated:      "2023-10-23T11:54:30+0000",
						Created:      "2023-10-23T11:54:30+0000",
						Name:         "enabled",
						Value:        "1",
					},
					{
						ContentLabel: "awesome_os-801",
						Updated:      "2023-10-23T11:54:30+0000",
						Created:      "2023-10-23T11:54:30+0000",
						Name:         "enabled_metadata",
						Value:        "1",
					},
				},
			},
			map[string]map[string]string{"awesome_os-801": {"enabled": "1", "enabled_metadata": "1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createMapFromContentOverrides(tt.args.contentOverrides); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createMapFromContentOverrides() = %v, want %v", got, tt.want)
			}
		})
	}
}
