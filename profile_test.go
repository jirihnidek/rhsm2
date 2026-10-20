package rhsm2

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
)

// TestSendPackageProfileDisabled verifies that when report_package_profile is
// disabled in rhsm.conf, SendPackageProfile is a no-op that returns nil without
// attempting to gather or upload anything.
func TestSendPackageProfileDisabled(t *testing.T) {
	t.Parallel()

	rhsmClient := &RHSMClient{
		RHSMConf: &RHSMConf{
			RHSM: RHSMConfRHSM{
				ReportPackageProfile: false,
			},
		},
	}

	if err := rhsmClient.SendPackageProfile(false, nil); err != nil {
		t.Errorf("SendPackageProfile(false) with reporting disabled = %v, want nil", err)
	}
	if err := rhsmClient.SendPackageProfile(true, nil); err != nil {
		t.Errorf("SendPackageProfile(true) with reporting disabled = %v, want nil", err)
	}
}

// TestSendPackageProfileAsyncReturnsImmediately verifies that the async mode
// returns nil without blocking on (or surfacing the result of) the background
// upload. The upload itself would fail here because no server is configured;
// async must swallow that error and return nil regardless.
//
// Not run in parallel: it mutates the shared listRpmPackages package variable
// and the profileUploadDoneSignal synchronization hook.
func TestSendPackageProfileAsyncReturnsImmediately(t *testing.T) {
	originalList := listRpmPackages
	// The background goroutine reads listRpmPackages, so wait for it (via the
	// profileUploadDoneSignal hook) before restoring the global to avoid a data race.
	done := make(chan struct{})
	profileUploadDoneSignal = done
	defer func() {
		<-done
		listRpmPackages = originalList
		profileUploadDoneSignal = nil
	}()

	// Make gathering fail fast so the background goroutine takes the error
	// path; the public call must still return nil immediately.
	listRpmPackages = func(string) ([]*rpmdb.PackageInfo, error) {
		return nil, errors.New("simulated rpm database failure")
	}

	rhsmClient := &RHSMClient{
		RHSMConf: &RHSMConf{
			RHSM: RHSMConfRHSM{ReportPackageProfile: true},
		},
	}

	if err := rhsmClient.SendPackageProfile(true, nil); err != nil {
		t.Errorf("SendPackageProfile(true) = %v, want nil (async must not surface errors)", err)
	}
}

// TestBuildProfileUploadPayload verifies the gathered CombinedProfile is
// converted into candlepin's expected wire envelope: an array of exactly two
// content-type blocks ("rpm" then "enabled_repos"), in that order, matching
// https://github.com/jirihnidek/rhsm2/issues/74.
func TestBuildProfileUploadPayload(t *testing.T) {
	t.Parallel()

	epoch := 0
	profile := &CombinedProfile{
		RpmPackages: []Package{
			{Name: "libgcc", Epoch: &epoch, Version: "14.2.1", Release: "7.el10", Arch: "x86_64", Vendor: "Red Hat, Inc."},
		},
		EnabledRepos: []EnabledRepository{
			{RepoID: "rhel-10-for-x86_64-baseos-rpms", BaseURLs: []string{"https://cdn.redhat.com/content/dist/rhel10/10/x86_64/baseos/os"}},
		},
	}

	payload := buildProfileUploadPayload(profile)

	if len(payload) != 2 {
		t.Fatalf("got %d content blocks, want 2", len(payload))
	}
	if payload[0].ContentType != rpmContentType {
		t.Errorf("payload[0].ContentType = %q, want %q", payload[0].ContentType, rpmContentType)
	}
	if payload[1].ContentType != enabledReposContentType {
		t.Errorf("payload[1].ContentType = %q, want %q", payload[1].ContentType, enabledReposContentType)
	}

	// Verify the actual marshaled JSON shape matches the documented wire
	// format exactly (field names/nesting), not just the Go struct shape.
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %s", err)
	}

	var decoded []struct {
		ContentType string `json:"content_type"`
		Profile     json.RawMessage
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload: %s", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("got %d decoded blocks, want 2", len(decoded))
	}

	var rpms []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Release string `json:"release"`
		Arch    string `json:"arch"`
		Epoch   int    `json:"epoch"`
		Vendor  string `json:"vendor"`
	}
	if err := json.Unmarshal(decoded[0].Profile, &rpms); err != nil {
		t.Fatalf("failed to unmarshal rpm profile block: %s", err)
	}
	if len(rpms) != 1 || rpms[0].Name != "libgcc" || rpms[0].Epoch != 0 {
		t.Errorf("unexpected rpm profile block: %+v", rpms)
	}

	var repos []struct {
		RepositoryID string   `json:"repositoryid"`
		BaseURL      []string `json:"baseurl"`
	}
	if err := json.Unmarshal(decoded[1].Profile, &repos); err != nil {
		t.Fatalf("failed to unmarshal enabled_repos profile block: %s", err)
	}
	if len(repos) != 1 || repos[0].RepositoryID != "rhel-10-for-x86_64-baseos-rpms" || len(repos[0].BaseURL) != 1 {
		t.Errorf("unexpected enabled_repos profile block: %+v", repos)
	}
}

// TestBuildProfileUploadPayloadEmptyProfile verifies that an empty
// CombinedProfile still produces the two-block envelope shape (candlepin
// expects both content_type blocks to always be present).
func TestBuildProfileUploadPayloadEmptyProfile(t *testing.T) {
	t.Parallel()

	payload := buildProfileUploadPayload(&CombinedProfile{})
	if len(payload) != 2 {
		t.Fatalf("got %d content blocks, want 2", len(payload))
	}

	// Nil slices must marshal as [] (empty array), not null, so candlepin does
	// not receive "profile": null.
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %s", err)
	}
	for _, block := range []string{`"profile":[]`} {
		if !strings.Contains(string(data), block) {
			t.Errorf("payload %s does not contain %s (nil slice marshaled as null?)", data, block)
		}
	}
}

// TestGatherCombinedProfileFailsWhenRpmQueryFails verifies the all-or-nothing
// contract: if the rpm package query fails, gathering returns an error and no
// profile, rather than producing a partial profile. Uploading a partial profile
// would cause candlepin to replace the stored profile with an empty rpm block,
// wiping last-known-good server state; failing early avoids that.
//
// Not run in parallel: it mutates the shared listRpmPackages package variable.
func TestGatherCombinedProfileFailsWhenRpmQueryFails(t *testing.T) {
	originalQuery := listRpmPackages
	defer func() { listRpmPackages = originalQuery }()

	listRpmPackages = func(string) ([]*rpmdb.PackageInfo, error) {
		return nil, errors.New("simulated rpm query failure")
	}

	rhsmClient := &RHSMClient{}
	profile, err := rhsmClient.GatherCombinedProfile()
	if err == nil {
		t.Fatal("GatherCombinedProfile() returned nil error on rpm-query failure, want an error")
	}
	if profile != nil {
		t.Errorf("GatherCombinedProfile() returned a non-nil profile on rpm-query failure, want nil: %+v", profile)
	}
}

// testProfile is a small, deterministic CombinedProfile used by
// uploadCombinedProfile tests. uploadCombinedProfile takes an
// already-gathered profile as a parameter specifically so these tests don't
// need to touch the live system's RPM database or repo files (see
// uploadCombinedProfile's doc comment for why that matters).
func testProfile() *CombinedProfile {
	epoch := 0
	return &CombinedProfile{
		RpmPackages: []Package{
			{Name: "bash", Epoch: &epoch, Version: "5.2.26", Release: "1.fc40", Arch: "x86_64", Vendor: "Fedora Project"},
		},
		EnabledRepos: []EnabledRepository{
			{RepoID: "test-repo", BaseURLs: []string{"https://example.com"}},
		},
	}
}

// TestUploadCombinedProfile verifies the happy path: a 204 response from the
// server is treated as success, and the request is well-formed (PUT method,
// correct consumers/{uuid}/profiles path, JSON content type).
func TestUploadCombinedProfile(t *testing.T) {
	t.Parallel()
	var expectedClientUUID = "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
	handlerCounter := 0

	server := httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			handlerCounter += 1

			if req.Method != http.MethodPut {
				t.Fatalf("expected request method: %s, got: %s", http.MethodPut, req.Method)
			}

			expectedURL := "/consumers/" + expectedClientUUID + "/profiles"
			reqURL := req.URL.String()
			if reqURL != expectedURL {
				t.Fatalf("expected request URL: %s, got: %s", expectedURL, reqURL)
			}

			if req.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("expected Content-Type: application/json, got: %s", req.Header.Get("Content-Type"))
			}

			// Status code confirmed by direct testing against production
			// candlepin: 204 on a valid payload (see jirihnidek/rhsm2#74).
			rw.WriteHeader(204)
		}))
	defer server.Close()

	tempDirFilePath := t.TempDir()
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, true, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	if err := rhsmClient.uploadCombinedProfile(testProfile(), nil); err != nil {
		t.Fatalf("unable to upload combined profile: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler called: %d, expected 1 call", handlerCounter)
	}
}

// TestUploadCombinedProfileErrors verifies that non-204 responses are
// reported as errors. 403/500 are confirmed-observed behaviors (500 was
// directly reproduced against production candlepin via malformed/invalid
// input; see jirihnidek/rhsm2#74); 404 and an arbitrary unexpected code are
// covered for completeness, following the same convention as the sibling
// SendContentOverrides.
func TestUploadCombinedProfileErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		response   string
	}{
		{"forbidden", 403, response403},
		{"not found", 404, response404},
		{"internal server error", 500, response500},
		{"unexpected status code", 418, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewTLSServer(
				http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					rw.WriteHeader(tt.statusCode)
					_, _ = rw.Write([]byte(tt.response))
				}))
			defer server.Close()

			tempDirFilePath := t.TempDir()
			testingFiles, err := setupTestingFileSystem(
				tempDirFilePath, false, true, true, false, true)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}

			rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			err = rhsmClient.uploadCombinedProfile(testProfile(), nil)
			if err == nil {
				t.Fatalf("no error raised when server responds with %d status code", tt.statusCode)
			}
		})
	}
}

// statusServer returns an httptest TLS server that answers GET /status with a
// managerCapabilities list containing exactly the given capabilities.
func statusServer(t *testing.T, capabilities []string) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(
		http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.String() != "/status" {
				t.Fatalf("expected GET /status, got %s %s", req.Method, req.URL.String())
			}
			body, _ := json.Marshal(map[string]interface{}{
				"result":              true,
				"managerCapabilities": capabilities,
			})
			rw.WriteHeader(200)
			_, _ = rw.Write(body)
		}))
}

// TestHasCapability verifies that HasCapability reports true only when the
// server's managerCapabilities list contains the requested capability.
func TestHasCapability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		capabilities []string
		query        string
		want         bool
	}{
		{"present", []string{"cores", "combined_reporting", "syspurpose"}, "combined_reporting", true},
		{"absent", []string{"cores", "syspurpose"}, "combined_reporting", false},
		{"empty", []string{}, "combined_reporting", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := statusServer(t, tt.capabilities)
			defer server.Close()

			testingFiles, err := setupTestingFileSystem(
				t.TempDir(), false, true, true, false, true)
			if err != nil {
				t.Fatalf("unable to setup testing environment: %s", err)
			}
			rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
			if err != nil {
				t.Fatalf("unable to setup testing rhsm client: %s", err)
			}

			got, err := rhsmClient.HasCapability(tt.query, nil)
			if err != nil {
				t.Fatalf("HasCapability() unexpected error: %s", err)
			}
			if got != tt.want {
				t.Errorf("HasCapability(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

// TestSendPackageProfileNotSupported verifies that when the server does not
// advertise the combined_reporting capability, sendPackageProfile returns a
// typed ProfileNotSupportedError (so the caller can map it to the Varlink
// ProfileNotSupported error) rather than attempting an upload.
func TestSendPackageProfileNotSupported(t *testing.T) {
	t.Parallel()

	// /status is served without combined_reporting. If the code wrongly tried
	// to upload, it would PUT /consumers/.../profiles, which this handler would
	// flag as an unexpected path.
	server := statusServer(t, []string{"cores", "syspurpose"})
	defer server.Close()

	testingFiles, err := setupTestingFileSystem(
		t.TempDir(), false, true, true, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}
	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}
	// Ensure profile reporting is enabled so the code reaches the capability
	// check rather than returning early on the report_package_profile gate.
	rhsmClient.RHSMConf.RHSM.ReportPackageProfile = true

	err = rhsmClient.sendPackageProfile(nil)
	var notSupported *ProfileNotSupportedError
	if !errors.As(err, &notSupported) {
		t.Fatalf("sendPackageProfile() error = %v, want *ProfileNotSupportedError", err)
	}
}
