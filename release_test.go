package rhsm2

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_getListingPath(t *testing.T) {
	tests := []struct {
		name        string
		contentPath string
		want        string
		wantErr     bool
	}{
		{
			name:        "empty paths",
			contentPath: "",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "path without $releasever",
			contentPath: "/content",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "path with $releasever",
			contentPath: "/content/$releasever/foo",
			want:        "/content/",
			wantErr:     false,
		},
		{
			name:        "path with more $releasever",
			contentPath: "/content/$releasever/foo/$releasever/bar",
			want:        "/content/",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getListingPath(&tt.contentPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: getListingPath() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("%s: getListingPath() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func Test_parseListingFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		want    []string
	}{
		{
			name:    "empty file",
			path:    "/foo/bar/baz/listing",
			content: "",
			want:    []string{},
		},
		{
			name:    "one release",
			path:    "/foo/bar/baz/listing",
			content: "10.0",
			want:    []string{"10.0"},
		},
		{
			name:    "multiple releases",
			path:    "/foo/bar/baz/listing",
			content: "10.0\n10.1\n10.2",
			want:    []string{"10.0", "10.1", "10.2"},
		},
		{
			name:    "multiple releases with duplicated releases",
			path:    "/foo/bar/baz/listing",
			content: "10.0\n10.1\n10.1",
			want:    []string{"10.0", "10.1"},
		},
		{
			name:    "unordered multiple releases",
			path:    "/foo/bar/baz/listing",
			content: "10.2\n10\n10.0\n10.1",
			want:    []string{"10", "10.0", "10.1", "10.2"},
		},
		{
			name:    "multiple releases with empty lines",
			path:    "/foo/bar/baz/listing",
			content: "10.0\n\n10.1\n\n",
			want:    []string{"10.0", "10.1"},
		},
		{
			name:    "multiple releases with comments",
			path:    "/foo/bar/baz/listing",
			content: "10.0\n# comment\n10.1\n# comment\n",
			want:    []string{"10.0", "10.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseListingFileContent(&tt.content, &tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("%s: parseListingFileContent() = %v, want %v", tt.name, got, tt.want)
				return
			}
			for idx, want := range tt.want {
				if got[idx] != want {
					t.Errorf("%s: parseListingFileContent() = %v, want %v", tt.name, got, tt.want)
					return
				}
			}
		})
	}
}

func Test_getListingPathFromEngProducts(t *testing.T) {
	enabled := true
	disabled := false
	tests := []struct {
		name                 string
		engineeringProducts  map[int64][]EngineeringProduct
		productTags          []string
		expectedListingPaths map[string]struct{}
	}{
		{
			name:                 "empty map",
			engineeringProducts:  map[int64][]EngineeringProduct{},
			expectedListingPaths: map[string]struct{}{},
		},
		{
			name: "single product with single content",
			engineeringProducts: map[int64][]EngineeringProduct{
				1: {
					{
						Content: []Content{
							{
								Path:         "/content/$releasever/foo",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
						},
					},
				},
			},
			productTags: []string{"rhel-11"},
			expectedListingPaths: map[string]struct{}{
				"/content/": {},
			},
		},
		{
			name: "single product with single content without required tags",
			engineeringProducts: map[int64][]EngineeringProduct{
				1: {
					{
						Content: []Content{
							{
								Path:    "/content/$releasever/foo",
								Enabled: &enabled,
							},
						},
					},
				},
			},
			productTags: []string{"rhel-11"},
			expectedListingPaths: map[string]struct{}{
				"/content/": {},
			},
		},
		{
			name: "single product with single content with required tags not matching product tags",
			engineeringProducts: map[int64][]EngineeringProduct{
				1: {
					{
						Content: []Content{
							{
								Path:         "/content/$releasever/foo",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-12", "rhel-12-x86_68"},
							},
						},
					},
				},
			},
			productTags:          []string{"rhel-11"},
			expectedListingPaths: map[string]struct{}{},
		},
		{
			name: "single product with single content (enabled by default by nil)",
			engineeringProducts: map[int64][]EngineeringProduct{
				1: {
					{
						Content: []Content{
							{
								Path:         "/content/$releasever/foo",
								Enabled:      nil,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
						},
					},
				},
			},
			productTags: []string{"rhel-11"},
			expectedListingPaths: map[string]struct{}{
				"/content/": {},
			},
		},
		{
			name: "multiple products with multiple contents",
			engineeringProducts: map[int64][]EngineeringProduct{
				1: {
					{
						Content: []Content{
							{
								Path:         "/content/$releasever/foo",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
							{
								Path:         "/content2/$releasever/foo",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
						},
					},
				},
				2: {
					{
						Content: []Content{
							{
								Path:         "/content3/$releasever/foo",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
						},
					},
				},
			},
			productTags: []string{"rhel-11"},
			expectedListingPaths: map[string]struct{}{
				"/content/":  {},
				"/content2/": {},
				"/content3/": {},
			},
		},
		{
			name: "multiple products with different contents but same base path",
			engineeringProducts: map[int64][]EngineeringProduct{
				1: {
					{
						Content: []Content{
							{
								Path:         "/content/$releasever/foo",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
							{
								Path:         "/content/$releasever/bar",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
						},
					},
				},
				2: {
					{
						Content: []Content{
							{
								Path:         "/content/$releasever/baz",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
						},
					},
				},
			},
			productTags: []string{"rhel-11"},
			expectedListingPaths: map[string]struct{}{
				"/content/": {},
			},
		},
		{
			name: "disabled content paths",
			engineeringProducts: map[int64][]EngineeringProduct{
				1: {
					{
						Content: []Content{
							{
								Path:         "/content/$releasever/foo",
								Enabled:      &disabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
							{
								Path:         "/content2/$releasever/foo",
								Enabled:      &enabled,
								RequiredTags: []string{"rhel-11", "rhel-11-x86_68"},
							},
						},
					},
				},
			},
			productTags: []string{"rhel-11"},
			expectedListingPaths: map[string]struct{}{
				"/content2/": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getListingPathFromEngProducts(tt.engineeringProducts, tt.productTags)
			if len(got) != len(tt.expectedListingPaths) {
				t.Errorf("getListingPathFromEngProducts() got = %v, expected %v", got, tt.expectedListingPaths)
				return
			}
			for path := range tt.expectedListingPaths {
				if _, exists := got[path]; !exists {
					t.Errorf("getListingPathFromEngProducts() missing path %s in result %v", path, got)
				}
			}
		})
	}
}

func Test_GetCdnReleasesSameReleases(t *testing.T) {
	t.Parallel()
	cdnHandlerCounter := 0

	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed for generating redhat.repo, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create mock of CDN server
	cdnServer := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			reqURL := req.URL.String()
			if req.Method == http.MethodGet {
				if strings.HasSuffix(reqURL, "/listing") {
					cdnHandlerCounter += 1
					// Return code 200
					rw.WriteHeader(200)
					// Return a simple text file with some releases in all cases
					_, _ = rw.Write([]byte("# Some comment\n10 \n10.0 \n10.1 \n10.2\n\n"))
				}
			}
		}))
	defer cdnServer.Close()

	// Create the root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, cdnServer)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	releases, err := rhsmClient.GetCdnReleases(nil)

	// The testing entitlement certificate contains two different base baths. Thus, there should be
	// two REST API calls to the CDN server.
	if cdnHandlerCounter != 2 {
		t.Fatalf("unexpected number of CDN handlers called, expected 2, got %d", cdnHandlerCounter)
	}

	if err != nil {
		t.Fatalf("unable to get CDN releases: %s", err)
	}

	expectedReleases := map[string]struct{}{"10": {}, "10.0": {}, "10.1": {}, "10.2": {}}

	if len(releases) != len(expectedReleases) {
		t.Fatalf("unexpected number of CDN releases, expected %d, got %d", len(expectedReleases), len(releases))
	}

	for release := range releases {
		if _, exists := expectedReleases[release]; !exists {
			t.Fatalf("unexpected release %s", release)
		}
	}
}

func Test_GetCdnReleasesDifferentReleases(t *testing.T) {
	t.Parallel()
	cdnHandlerCounter := 0

	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call to candlepin needed for getting releases, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create mock of CDN server
	cdnServer := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			reqURL := req.URL.String()
			if req.Method == http.MethodGet {
				if strings.HasSuffix(reqURL, "/listing") {
					cdnHandlerCounter += 1
					// Return code 200
					rw.WriteHeader(200)
					// Return a simple text file with some releases in all cases
					if cdnHandlerCounter == 1 {
						_, _ = rw.Write([]byte("10 \n10.0 \n10.1 \n10.2\n\n"))
					} else {
						_, _ = rw.Write([]byte("10 \n10.1\n10.3\n\n"))
					}
				}
			}
		}))
	defer cdnServer.Close()

	// Create the root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, cdnServer)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	releases, err := rhsmClient.GetCdnReleases(nil)

	// The testing entitlement certificate contains two different base baths. Thus, there should be
	// two REST API calls to the CDN server.
	if cdnHandlerCounter != 2 {
		t.Fatalf("unexpected number of CDN handlers called, expected 2, got %d", cdnHandlerCounter)
	}

	if err != nil {
		t.Fatalf("unable to get CDN releases: %s", err)
	}

	expectedReleases := map[string]struct{}{"10": {}, "10.0": {}, "10.1": {}, "10.2": {}, "10.3": {}}

	if len(releases) != len(expectedReleases) {
		t.Fatalf("unexpected number of CDN releases, expected %d, got %d", len(expectedReleases), len(releases))
	}

	for release := range releases {
		if _, exists := expectedReleases[release]; !exists {
			t.Fatalf("unexpected release %s", release)
		}
	}
}

func Test_GetCdnReleasesCDNError(t *testing.T) {
	t.Parallel()
	cdnHandlerCounter := 0

	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call to candlepin needed for getting releases, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create mock of CDN server
	cdnServer := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			reqURL := req.URL.String()
			if req.Method == http.MethodGet {
				if strings.HasSuffix(reqURL, "/listing") {
					cdnHandlerCounter += 1
					// Return code 200
					rw.WriteHeader(400)
					// Return a simple text file with some releases in all cases
					_, _ = rw.Write([]byte("Some error"))
				}
			}
		}))
	defer cdnServer.Close()

	// Create the root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, true, true, true, true, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, cdnServer)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	releases, err := rhsmClient.GetCdnReleases(nil)

	// The testing entitlement certificate contains two different base baths. Thus, there should be
	// two REST API calls to the CDN server.
	if cdnHandlerCounter != 2 {
		t.Fatalf("unexpected number of CDN handlers called, expected 2, got %d", cdnHandlerCounter)
	}

	if err != nil {
		t.Fatalf("unable to get CDN releases: %s", err)
	}

	if len(releases) != 0 {
		t.Fatalf("unexpected number of CDN releases, expected 0, got %d", len(releases))
	}
}

func Test_GetCdnReleasesUnregistered(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(
		// There should be no REST API call in this case
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call to candlepin needed for getting releases, %s %s called",
				req.Method, req.URL.String())
		}))
	defer server.Close()

	// Create mock of CDN server
	cdnServer := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call to CDN needed when unregistered, %s %s called",
				req.Method, req.URL.String())
		}))
	defer cdnServer.Close()

	// Create the root directory for this test
	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, false)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, cdnServer)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	releases, err := rhsmClient.GetCdnReleases(nil)

	if err == nil {
		t.Fatalf("expected error when getting release on unregistered system, got nil")
	}

	if len(releases) != 0 {
		t.Fatalf("unexpected number of CDN releases, expected 0, got %d", len(releases))
	}
}

func Test_SetReleaseOnServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		releaseVer     string
		serverResponse string
		statusCode     int
		wantErr        bool
	}{
		{
			name:           "successful set",
			releaseVer:     "10.1",
			serverResponse: ``,
			statusCode:     204,
			wantErr:        false,
		},
		{
			name:           "successful unset",
			releaseVer:     "",
			serverResponse: ``,
			statusCode:     204,
			wantErr:        false,
		},
		{
			name:           "server error",
			releaseVer:     "10.1",
			serverResponse: "Internal Server Error",
			statusCode:     500,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			releaseVer:     "10.1",
			serverResponse: "Unauthorized",
			statusCode:     401,
			wantErr:        true,
		},
		{
			name:           "forbidden",
			releaseVer:     "10.1",
			serverResponse: "Forbidden",
			statusCode:     403,
			wantErr:        true,
		},
		{
			name:           "not found",
			releaseVer:     "10.1",
			serverResponse: "Not Found",
			statusCode:     404,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(
				http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodPut {
						t.Fatalf("unexpected HTTP method: %s", req.Method)
					}
					rw.WriteHeader(tt.statusCode)
					_, _ = rw.Write([]byte(tt.serverResponse))
				}))
			defer server.Close()

			tempDirFilePath := t.TempDir()

			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath,
				true,
				true,
				true,
				true,
				true,
			)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			err = rhsmClient.setReleaseOnServer(nil, tt.releaseVer)

			if (err != nil) != tt.wantErr {
				t.Errorf("%s: setReleaseOnServer() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
		})
	}
}

func Test_isAnyRequiredTagProvided(t *testing.T) {
	tests := []struct {
		name         string
		requiredTags []string
		providedTags []string
		want         bool
	}{
		{
			name:         "empty required tags and empty provided tags",
			requiredTags: []string{},
			providedTags: []string{},
			want:         true,
		},
		{
			name:         "empty required tags",
			requiredTags: []string{},
			providedTags: []string{"rhel-11"},
			want:         true,
		},
		{
			name:         "empty provided tags",
			requiredTags: []string{"rhel-11"},
			providedTags: []string{},
			want:         false,
		},
		{
			name:         "matching tags",
			requiredTags: []string{"rhel-11", "rhel-11-x86_64"},
			providedTags: []string{"rhel-11"},
			want:         true,
		},
		{
			name:         "non-matching tags",
			requiredTags: []string{"rhel-11", "rhel-11-x86_64"},
			providedTags: []string{"rhel-9", "rhel-9-aarch64"},
			want:         false,
		},
		{
			name:         "nil required tags",
			requiredTags: nil,
			providedTags: []string{"rhel-11"},
			want:         true,
		},
		{
			name:         "nil provided tags",
			requiredTags: []string{"rhel-11"},
			providedTags: nil,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAnyRequiredTagProvided(tt.requiredTags, tt.providedTags)
			if got != tt.want {
				t.Errorf("%s: isAnyRequiredTagProvided() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func Test_SetReleaseOnServerUnregistered(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("no REST API call needed when system is not registered: %s %s",
				req.Method, req.URL.String())
		}))
	defer server.Close()

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

	err = rhsmClient.setReleaseOnServer(nil, "10.1")
	if err == nil {
		t.Fatal("expected error when setting release on unregistered system")
	}
}

func Test_SetDnfVarsRelease(t *testing.T) {
	tests := []struct {
		name         string
		releaseVer   string
		setupFiles   bool
		wantErr      bool
		expectedFile string
	}{
		{
			name:         "successful set release with major and minor version",
			releaseVer:   "11.1",
			wantErr:      false,
			expectedFile: "11.1",
		},
		{
			name:         "successful set release with major version",
			releaseVer:   "43",
			wantErr:      false,
			expectedFile: "43",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDirFilePath := t.TempDir()

			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath,
				true,
				true,
				true,
				true,
				true,
			)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, nil, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			err = rhsmClient.setDnfVarsRelease(tt.releaseVer)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetReleaseOnHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Try to read the content of the release version file
				content, err := os.ReadFile(testingFiles.DnfVarsReleaseFilePath)
				if err != nil {
					if tt.expectedFile != "" {
						t.Errorf("unable to read release version file: %s", err)
					}
				} else {
					gotContent := string(content)
					if gotContent != tt.expectedFile {
						t.Errorf("unexpected content of release version file: got %s, want %s",
							gotContent, tt.expectedFile)
					}
				}
			}
		})
	}
}

func Test_GetReleaseFromServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		wantRelease    string
		wantErr        bool
	}{
		{
			name:           "successful response",
			serverResponse: `{"releaseVer":"10.1"}`,
			statusCode:     200,
			wantRelease:    "10.1",
			wantErr:        false,
		},
		{
			name:           "empty response",
			serverResponse: `{}`,
			statusCode:     200,
			wantRelease:    "",
			wantErr:        false,
		},
		{
			name:           "server error",
			serverResponse: "Internal Server Error",
			statusCode:     500,
			wantRelease:    "",
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			serverResponse: "Unauthorized",
			statusCode:     401,
			wantRelease:    "",
			wantErr:        true,
		},
		{
			name:           "forbidden",
			serverResponse: "Forbidden",
			statusCode:     403,
			wantRelease:    "",
			wantErr:        true,
		},
		{
			name:           "not found",
			serverResponse: "Not Found",
			statusCode:     404,
			wantRelease:    "",
			wantErr:        true,
		},
		{
			name:           "invalid json",
			serverResponse: `{"invalid": json}`,
			statusCode:     200,
			wantRelease:    "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(
				http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodGet {
						t.Fatalf("unexpected HTTP method: %s", req.Method)
					}
					if !strings.HasSuffix(req.URL.Path, "/release") {
						t.Fatalf("unexpected URL path: %s", req.URL.Path)
					}
					rw.WriteHeader(tt.statusCode)
					_, _ = rw.Write([]byte(tt.serverResponse))
				}))
			defer server.Close()

			tempDirFilePath := t.TempDir()

			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath, true, true, true, true, true)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			release, err := rhsmClient.GetReleaseFromServer(nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("%s: GetReleaseFromServer() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if release != tt.wantRelease {
				t.Errorf("%s: GetReleaseFromServer() = %v, want %v", tt.name, release, tt.wantRelease)
			}
		})
	}
}

func Test_UnsetRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupFiles     bool
		setupHTTP      bool
		statusCode     int
		serverResponse string
		wantErr        bool
	}{
		{
			name:           "successful unset on registered system",
			setupFiles:     true,
			setupHTTP:      true,
			statusCode:     204,
			serverResponse: ``,
			wantErr:        false,
		},
		{
			name:       "unset on unregistered system",
			setupFiles: false,
			setupHTTP:  false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if tt.setupHTTP {
				server = httptest.NewTLSServer(
					http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
						if !tt.setupHTTP {
							t.Fatalf("No REST API call needed when system is not registered: %s %s",
								req.Method, req.URL.Path)
						}
						if req.Method != http.MethodPut {
							t.Fatalf("unexpected HTTP method: %s", req.Method)
						}
						rw.WriteHeader(tt.statusCode)
						_, _ = rw.Write([]byte(tt.serverResponse))
					}))
				defer server.Close()
			}

			tempDirFilePath := t.TempDir()

			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath,
				tt.setupFiles,
				tt.setupFiles,
				tt.setupFiles,
				tt.setupFiles,
				tt.setupFiles,
			)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			// Create a test release file on registered and unregistered systems
			dnfVarsDirPath := filepath.Dir(testingFiles.DnfVarsReleaseFilePath)
			err = os.MkdirAll(dnfVarsDirPath, 0755)
			if err != nil {
				t.Fatalf("unable to create directory %s: %s", dnfVarsDirPath, err)
			}
			err = os.WriteFile(testingFiles.DnfVarsReleaseFilePath, []byte("11.6"), 0644)
			if err != nil {
				t.Fatalf("unable to create test release file: %s", err)
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			err = rhsmClient.UnsetRelease()
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: UnsetRelease() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}

			// The file should have been removed on registered and unregistered systems
			_, err = os.Stat(testingFiles.DnfVarsReleaseFilePath)
			if !os.IsNotExist(err) {
				t.Error("DNF vars release file should have been removed")
			}
		})
	}
}

func Test_SetRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		releaseVer     string
		setupFiles     bool
		setupHTTP      bool
		statusCode     int
		serverResponse string
		wantErr        bool
	}{
		{
			name:           "successful set on registered system",
			releaseVer:     "11.5",
			setupFiles:     true,
			setupHTTP:      true,
			statusCode:     204,
			serverResponse: ``,
			wantErr:        false,
		},
		{
			name:           "server error on registered system",
			releaseVer:     "11.5",
			setupFiles:     true,
			setupHTTP:      true,
			statusCode:     500,
			serverResponse: "Internal Server Error",
			wantErr:        false,
		},
		{
			name:       "set on unregistered system",
			releaseVer: "11.5",
			setupFiles: false,
			setupHTTP:  false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if tt.setupHTTP {
				server = httptest.NewTLSServer(
					http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
						if !tt.setupHTTP {
							t.Fatalf("No REST API call needed when system is not registered: %s %s",
								req.Method, req.URL.Path)
						}
						if req.Method != http.MethodPut {
							t.Fatalf("unexpected HTTP method: %s", req.Method)
						}
						rw.WriteHeader(tt.statusCode)
						_, _ = rw.Write([]byte(tt.serverResponse))
					}))
				defer server.Close()
			}

			tempDirFilePath := t.TempDir()

			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath,
				tt.setupFiles,
				tt.setupFiles,
				tt.setupFiles,
				tt.setupFiles,
				tt.setupFiles,
			)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			err = rhsmClient.SetRelease(tt.releaseVer)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				content, err := os.ReadFile(testingFiles.DnfVarsReleaseFilePath)
				if err != nil {
					t.Errorf("unable to read release version file: %s", err)
					return
				}
				gotContent := string(content)
				if gotContent != tt.releaseVer {
					t.Errorf("unexpected content of release version file: got %s, want %s",
						gotContent, tt.releaseVer)
				}
			}
		})
	}
}

func Test_UnsetDnfVarsRelease(t *testing.T) {
	tests := []struct {
		name       string
		setupFile  bool
		wantErr    bool
		fileExists bool
	}{
		{
			name:       "successful unset of existing file",
			setupFile:  true,
			wantErr:    false,
			fileExists: false,
		},
		{
			name:       "unset when file does not exist",
			setupFile:  false,
			wantErr:    false,
			fileExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDirFilePath := t.TempDir()

			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath,
				true,
				true,
				true,
				true,
				true,
			)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			if tt.setupFile {
				dnfVarsDirPath := filepath.Dir(testingFiles.DnfVarsReleaseFilePath)
				err := os.MkdirAll(dnfVarsDirPath, 0755)
				if err != nil {
					t.Fatalf("unable to create directory %s: %s", dnfVarsDirPath, err)
				}
				err = os.WriteFile(testingFiles.DnfVarsReleaseFilePath, []byte("11.6"), 0644)
				if err != nil {
					t.Fatalf("unable to create test release file: %s", err)
				}
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, nil, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			err = rhsmClient.unsetDnfVarsRelease()
			if (err != nil) != tt.wantErr {
				t.Errorf("unsetDnfVarsRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			_, err = os.Stat(testingFiles.DnfVarsReleaseFilePath)
			fileExists := !os.IsNotExist(err)
			if fileExists != tt.fileExists {
				t.Errorf("File exists = %v, want %v", fileExists, tt.fileExists)
			}
		})
	}
}

func Test_GetDnfVarsRelease(t *testing.T) {
	tests := []struct {
		name      string
		setupFile bool
		content   string
		want      string
		wantErr   bool
	}{
		{
			name:      "successful read of existing file",
			setupFile: true,
			content:   "11.6",
			want:      "11.6",
			wantErr:   false,
		},
		{
			name:      "read when file does not exist",
			setupFile: false,
			content:   "",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDirFilePath := t.TempDir()

			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath,
				true,
				true,
				true,
				true,
				true,
			)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			if tt.setupFile {
				dnfVarsDirPath := filepath.Dir(testingFiles.DnfVarsReleaseFilePath)
				err := os.MkdirAll(dnfVarsDirPath, 0755)
				if err != nil {
					t.Fatalf("unable to create directory %s: %s", dnfVarsDirPath, err)
				}
				err = os.WriteFile(testingFiles.DnfVarsReleaseFilePath, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("unable to create test release file: %s", err)
				}
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, nil, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			got, err := rhsmClient.GetDnfVarsRelease()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDnfVarsRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDnfVarsRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseOSRelease(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    OSRelease
		wantErr bool
	}{
		{
			name:    "empty content",
			content: "",
			want:    OSRelease{},
			wantErr: true,
		},
		{
			name: "valid RHEL content of os-release file",
			content: `NAME="Red Hat Enterprise Linux"
VERSION="11.5 (Foo)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="11.5"
PLATFORM_ID="platform:el11"
PRETTY_NAME="Red Hat Enterprise Linux 11.5 (Foo)"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:redhat:enterprise_linux:11::baseos"
HOME_URL="https://www.redhat.com/"
BUG_REPORT_URL="https://bugzilla.redhat.com/"
REDHAT_BUGZILLA_PRODUCT="Red Hat Enterprise Linux 11"
REDHAT_BUGZILLA_PRODUCT_VERSION=11.5
REDHAT_SUPPORT_PRODUCT="Red Hat Enterprise Linux"
REDHAT_SUPPORT_PRODUCT_VERSION="11.5"`,
			want: OSRelease{
				ID:           "rhel",
				VersionID:    "11.5",
				VersionMajor: "11",
				VersionMinor: "5",
			},
			wantErr: false,
		},
		{
			name: "valid RHEL content of os-release file with more sub-minor versions",
			content: `NAME="Red Hat Enterprise Linux"
VERSION="12.3.1 (Bar)"
ID="rhel"
ID_LIKE="fedora"
VERSION_ID="12.3.1"
PLATFORM_ID="platform:el12"
PRETTY_NAME="Red Hat Enterprise Linux 12.3.1 (Bar)"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:redhat:enterprise_linux:12::baseos"
HOME_URL="https://www.redhat.com/"
BUG_REPORT_URL="https://bugzilla.redhat.com/"
REDHAT_BUGZILLA_PRODUCT="Red Hat Enterprise Linux 11"
REDHAT_BUGZILLA_PRODUCT_VERSION=12.3.1
REDHAT_SUPPORT_PRODUCT="Red Hat Enterprise Linux"
REDHAT_SUPPORT_PRODUCT_VERSION="12.3.1"`,
			want: OSRelease{
				ID:           "rhel",
				VersionID:    "12.3.1",
				VersionMajor: "12",
				VersionMinor: "3",
			},
			wantErr: false,
		},
		{
			name: "valid Fedora content of os-release file (no minor version)",
			content: `NAME="Fedora Linux"
VERSION="43 (Workstation Edition)"
RELEASE_TYPE=stable
ID=fedora
VERSION_ID=43
VERSION_CODENAME=""
PRETTY_NAME="Fedora Linux 43 (Workstation Edition)"
ANSI_COLOR="0;38;2;60;110;180"
LOGO=fedora-logo-icon
CPE_NAME="cpe:/o:fedoraproject:fedora:43"
DEFAULT_HOSTNAME="fedora"
HOME_URL="https://fedoraproject.org/"
DOCUMENTATION_URL="https://docs.fedoraproject.org/en-US/fedora/f43/"
SUPPORT_URL="https://ask.fedoraproject.org/"
BUG_REPORT_URL="https://bugzilla.redhat.com/"
REDHAT_BUGZILLA_PRODUCT="Fedora"
REDHAT_BUGZILLA_PRODUCT_VERSION=43
REDHAT_SUPPORT_PRODUCT="Fedora"
REDHAT_SUPPORT_PRODUCT_VERSION=43
SUPPORT_END=2026-12-02
VARIANT="Workstation Edition"
VARIANT_ID=workstation`,
			want: OSRelease{
				ID:           "fedora",
				VersionID:    "43",
				VersionMajor: "43",
				VersionMinor: "",
			},
			wantErr: false,
		},
		{
			name: "invalid content of os-release file",
			content: `NAME=Red Hat Enterprise Linux
VERSION=11.5 (Foo)
INVALID_LINE
ID=rhel`,
			want:    OSRelease{},
			wantErr: true,
		},
		{
			name: "missing required fields in os-release file",
			content: `ANSI_COLOR="0;31"
HOME_URL="https://www.redhat.com/"`,
			want:    OSRelease{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			byteContent := []byte(tt.content)
			got, err := parseOSRelease(&byteContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOSRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got != nil && *got != tt.want {
				t.Errorf("parseOSRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}
