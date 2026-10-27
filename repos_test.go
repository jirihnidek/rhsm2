package rhsm2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/ini.v1"
)

// TestIsRepoEnabled checks the truthy/falsy token handling, including the
// dnf convention that an absent (empty) value means enabled.
func TestIsRepoEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  bool
	}{
		{"", true}, // absent key defaults to enabled
		{"1", true},
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"yes", true},
		{"on", true},
		{"enabled", true},
		{"  1  ", true}, // surrounding whitespace tolerated
		{"0", false},
		{"false", false},
		{"False", false},
		{"no", false},
		{"off", false},
		{"disabled", false},
		{"garbage", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()
			if got := isRepoEnabled(tt.value); got != tt.want {
				t.Errorf("isRepoEnabled(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestSplitBaseURLs verifies splitting of a (possibly multi-line) baseurl
// value into individual URLs, including trimming and dropping blank lines.
func TestSplitBaseURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{
			name:  "single url",
			value: "https://example.com/one",
			want:  []string{"https://example.com/one"},
		},
		{
			name:  "multi-line continuation",
			value: "https://example.com/one\n    https://example.com/two\n    https://example.com/three",
			want:  []string{"https://example.com/one", "https://example.com/two", "https://example.com/three"},
		},
		{
			name:  "blank continuation lines are dropped",
			value: "https://example.com/one\n\n   \nhttps://example.com/two",
			want:  []string{"https://example.com/one", "https://example.com/two"},
		},
		{
			name:  "embedded credentials are stripped",
			value: "https://user:pass@example.com/one\n    https://example.com/two",
			want:  []string{"https://example.com/one", "https://example.com/two"},
		},
		{
			// A repo using metalink/mirrorlist has no baseurl; the result must be
			// an empty (non-nil) slice so it marshals as [] rather than null.
			name:  "empty value",
			value: "",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := splitBaseURLs(tt.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitBaseURLs(%q) = %#v, want %#v", tt.value, got, tt.want)
			}
		})
	}
}

// TestEnabledRepoMetalink verifies that an enabled repo using metalink (no
// baseurl) reports the metalink URL and serializes with "baseurl":[] rather than
// "baseurl":null.
func TestEnabledRepoMetalink(t *testing.T) {
	t.Parallel()

	repos := map[string]map[string]string{
		"rhel-metalink": {
			"enabled":  "1",
			"metalink": "https://example.com/metalink?repo=rhel",
		},
	}

	enabled := enabledReposFromMap(repos)
	if len(enabled) != 1 {
		t.Fatalf("got %d repos, want 1: %+v", len(enabled), enabled)
	}
	if enabled[0].BaseURLs == nil {
		t.Errorf("BaseURLs is nil, want a non-nil empty slice so it marshals as []")
	}
	if enabled[0].Metalink != "https://example.com/metalink?repo=rhel" {
		t.Errorf("Metalink = %q, want the repo's metalink URL", enabled[0].Metalink)
	}

	data, err := json.Marshal(enabled[0])
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}
	if strings.Contains(string(data), `"baseurl":null`) {
		t.Errorf("metalink repo marshaled with null baseurl: %s", data)
	}
	if !strings.Contains(string(data), `"baseurl":[]`) {
		t.Errorf("metalink repo baseurl = %s, want \"baseurl\":[]", data)
	}
	if !strings.Contains(string(data), `"metalink":"https://example.com/metalink?repo=rhel"`) {
		t.Errorf("metalink repo = %s, want the metalink field reported", data)
	}
}

// TestEnabledRepoMirrorlist verifies that a repo using mirrorlist reports the
// mirrorlist URL, and that a plain baseurl repo omits the metalink/mirrorlist
// fields entirely.
func TestEnabledRepoMirrorlist(t *testing.T) {
	t.Parallel()

	repos := map[string]map[string]string{
		"with-mirrorlist": {
			"enabled":    "1",
			"mirrorlist": "https://example.com/mirrorlist",
		},
		"with-baseurl": {
			"enabled": "1",
			"baseurl": "https://example.com/base",
		},
	}

	enabled := enabledReposFromMap(repos)
	byID := make(map[string]EnabledRepository)
	for _, r := range enabled {
		byID[r.RepoID] = r
	}

	if byID["with-mirrorlist"].Mirrorlist != "https://example.com/mirrorlist" {
		t.Errorf("Mirrorlist = %q, want the repo's mirrorlist URL", byID["with-mirrorlist"].Mirrorlist)
	}

	// A plain baseurl repo must not carry empty metalink/mirrorlist keys.
	data, err := json.Marshal(byID["with-baseurl"])
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}
	if strings.Contains(string(data), "metalink") || strings.Contains(string(data), "mirrorlist") {
		t.Errorf("baseurl repo should omit metalink/mirrorlist: %s", data)
	}
}

// TestStripURLCredentials verifies that embedded userinfo is removed from a URL
// while the rest (path, query) is preserved, and that non-URL or
// credential-free values are returned unchanged.
func TestStripURLCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"user and password", "https://user:pass@example.com/repo", "https://example.com/repo"},
		{"user only", "https://user@example.com/repo", "https://example.com/repo"},
		{"no credentials", "https://example.com/repo", "https://example.com/repo"},
		{"query preserved, creds stripped", "https://user:pass@example.com/repo?arch=x86_64", "https://example.com/repo?arch=x86_64"},
		{"no scheme, no creds", "example.com/repo", "example.com/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := stripURLCredentials(tt.in); got != tt.want {
				t.Errorf("stripURLCredentials(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

const sampleRepoFile = `[repo-enabled-explicit]
name = Enabled Explicitly
baseurl = https://example.com/enabled
enabled = 1

[repo-disabled]
name = Disabled
baseurl = https://example.com/disabled
enabled = 0

[repo-enabled-default]
name = Enabled By Default
baseurl = https://example.com/default

[repo-enabled-bool]
name = Enabled Bool
baseurl = https://example.com/bool
enabled = true

[repo-multi-baseurl]
name = Multiple Mirrors
baseurl = https://example.com/mirror1
    https://example.com/mirror2
enabled = 1
`

// mergeAndCollect is a small test helper: it merges the given INI content into a
// fresh repo map and returns the enabled repositories keyed by RepoID.
func mergeAndCollect(t *testing.T, content string) map[string][]string {
	t.Helper()
	repoFile, err := ini.LoadSources(repoFileLoadOptions, []byte(content))
	if err != nil {
		t.Fatalf("failed to load sample ini: %s", err)
	}
	repos := make(map[string]map[string]string)
	mergeReposFromINI(repos, repoFile)

	got := make(map[string][]string)
	for _, r := range enabledReposFromMap(repos) {
		got[r.RepoID] = r.BaseURLs
	}
	return got
}

// TestMergeReposFromINI verifies enabled-repo extraction from in-memory INI
// content, covering explicit enable, disable, default-enabled, the boolean token
// form, and multi-baseurl (mirror fallback) repos.
func TestMergeReposFromINI(t *testing.T) {
	t.Parallel()

	got := mergeAndCollect(t, sampleRepoFile)

	if _, disabled := got["repo-disabled"]; disabled {
		t.Error("disabled repository should not be present in the result")
	}

	expected := map[string][]string{
		"repo-enabled-explicit": {"https://example.com/enabled"},
		"repo-enabled-default":  {"https://example.com/default"},
		"repo-enabled-bool":     {"https://example.com/bool"},
		"repo-multi-baseurl":    {"https://example.com/mirror1", "https://example.com/mirror2"},
	}
	for id, baseURLs := range expected {
		if !reflect.DeepEqual(got[id], baseURLs) {
			t.Errorf("repo %q: baseurl = %#v, want %#v", id, got[id], baseURLs)
		}
	}
	if len(got) != len(expected) {
		t.Errorf("got %d enabled repos, want %d (%+v)", len(got), len(expected), got)
	}
}

// TestMergeReposFromINIDuplicateRepoID verifies that when the same repo ID
// appears more than once, the entries collapse to a single effective repo with
// the last definition winning (per key).
func TestMergeReposFromINIDuplicateRepoID(t *testing.T) {
	t.Parallel()

	content := `[dup]
baseurl = https://example.com/first
enabled = 1

[dup]
baseurl = https://example.com/second
enabled = 1
`
	got := mergeAndCollect(t, content)

	if len(got) != 1 {
		t.Fatalf("got %d repos, want 1 (duplicate repo IDs should collapse): %+v", len(got), got)
	}
	if !reflect.DeepEqual(got["dup"], []string{"https://example.com/second"}) {
		t.Errorf("dup baseurl = %#v, want last definition to win", got["dup"])
	}
}

// TestGetEnabledRepos verifies reading and merging .repo files from a directory,
// that results are sorted by RepoID, that duplicate repo IDs across files
// collapse, and that non-.repo files are ignored.
func TestGetEnabledRepos(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	fileA := `[zeta]
baseurl = https://example.com/zeta
enabled = 1

[alpha]
baseurl = https://example.com/alpha
enabled = 1
`
	fileB := `[mango]
baseurl = https://example.com/mango

[nope]
baseurl = https://example.com/nope
enabled = 0

[alpha]
baseurl = https://example.com/alpha-override
enabled = 1
`
	if err := os.WriteFile(filepath.Join(tempDir, "a.repo"), []byte(fileA), 0644); err != nil {
		t.Fatalf("failed to write a.repo: %s", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "b.repo"), []byte(fileB), 0644); err != nil {
		t.Fatalf("failed to write b.repo: %s", err)
	}
	// A non-.repo file must be ignored even if it looks like INI.
	if err := os.WriteFile(filepath.Join(tempDir, "notes.txt"), []byte("[ignored]\nenabled=1\n"), 0644); err != nil {
		t.Fatalf("failed to write notes.txt: %s", err)
	}

	// No override dirs for this case.
	repos, err := getEnabledRepos(tempDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	wantOrder := []string{"alpha", "mango", "zeta"}
	if len(repos) != len(wantOrder) {
		t.Fatalf("got %d repos, want %d (%+v)", len(repos), len(wantOrder), repos)
	}
	for i, id := range wantOrder {
		if repos[i].RepoID != id {
			t.Errorf("repos[%d].RepoID = %q, want %q (result not sorted?)", i, repos[i].RepoID, id)
		}
	}
	// alpha is defined in both files; the later file wins.
	if !reflect.DeepEqual(repos[0].BaseURLs, []string{"https://example.com/alpha-override"}) {
		t.Errorf("alpha baseurl = %#v, want the b.repo definition to win", repos[0].BaseURLs)
	}
}

// TestGetEnabledReposWithOverrides verifies that dnf5 repo overrides are applied
// on top of the base .repo files: an override can enable a repo that is disabled
// by default and can change a repo's baseurl.
func TestGetEnabledReposWithOverrides(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	baseRepo := `[base-disabled]
baseurl = https://example.com/base-disabled
enabled = 0

[base-enabled]
baseurl = https://example.com/base-enabled
enabled = 1
`
	if err := os.WriteFile(filepath.Join(tempDir, "base.repo"), []byte(baseRepo), 0644); err != nil {
		t.Fatalf("failed to write base.repo: %s", err)
	}

	// Override enables base-disabled and rewrites base-enabled's baseurl. The
	// override lives in its own directory, separate from the base .repo dir.
	overrideDir := t.TempDir()
	override := `[base-disabled]
enabled = 1

[base-enabled]
baseurl = https://example.com/overridden
`
	if err := os.WriteFile(filepath.Join(overrideDir, "override.repo"), []byte(override), 0644); err != nil {
		t.Fatalf("failed to write override: %s", err)
	}

	repos, err := getEnabledRepos(tempDir, []string{overrideDir})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	got := make(map[string][]string)
	for _, r := range repos {
		got[r.RepoID] = r.BaseURLs
	}

	// base-disabled is enabled by the override.
	if !reflect.DeepEqual(got["base-disabled"], []string{"https://example.com/base-disabled"}) {
		t.Errorf("base-disabled = %#v, want enabled by override", got["base-disabled"])
	}
	// base-enabled's baseurl comes from the override.
	if !reflect.DeepEqual(got["base-enabled"], []string{"https://example.com/overridden"}) {
		t.Errorf("base-enabled = %#v, want overridden baseurl", got["base-enabled"])
	}
	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2 (%+v)", len(repos), repos)
	}
}

// TestGetEnabledReposCanDisableViaOverride verifies that an override can disable
// a repo that is enabled in the base .repo file.
func TestGetEnabledReposCanDisableViaOverride(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	baseRepo := `[repo]
baseurl = https://example.com/repo
enabled = 1
`
	if err := os.WriteFile(filepath.Join(tempDir, "base.repo"), []byte(baseRepo), 0644); err != nil {
		t.Fatalf("failed to write base.repo: %s", err)
	}

	overrideDir := t.TempDir()
	override := `[repo]
enabled = 0
`
	if err := os.WriteFile(filepath.Join(overrideDir, "override.repo"), []byte(override), 0644); err != nil {
		t.Fatalf("failed to write override: %s", err)
	}

	repos, err := getEnabledRepos(tempDir, []string{overrideDir})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(repos) != 0 {
		t.Errorf("got %d repos, want 0 (override should have disabled the repo): %+v", len(repos), repos)
	}
}

// TestGetEnabledReposEmpty verifies that a directory with no .repo files yields
// an empty result and no error.
func TestGetEnabledReposEmpty(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	repos, err := getEnabledRepos(tempDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(repos) != 0 {
		t.Errorf("got %d repos, want 0", len(repos))
	}
}

// TestMergeRepoOverridesMissingDir verifies that a missing override directory is
// not an error and leaves the base repo map untouched.
func TestMergeRepoOverridesMissingDir(t *testing.T) {
	t.Parallel()

	repos := map[string]map[string]string{
		"repo": {"enabled": "1", "baseurl": "https://example.com/repo"},
	}
	if err := mergeRepoOverrides(repos, []string{filepath.Join(t.TempDir(), "absent-dir")}); err != nil {
		t.Fatalf("missing override dir should not error, got: %s", err)
	}
	if repos["repo"]["baseurl"] != "https://example.com/repo" {
		t.Errorf("base repo map was modified by a missing override dir: %+v", repos["repo"])
	}
}

// TestMergeRepoOverridesAppliesOnTop verifies that overrides change existing keys
// but, per dnf semantics, do not introduce a repo ID absent from the base map.
func TestMergeRepoOverridesAppliesOnTop(t *testing.T) {
	t.Parallel()

	repos := map[string]map[string]string{
		"base": {"enabled": "0", "baseurl": "https://example.com/base"},
	}

	override := `[base]
enabled = 1

[override-only]
baseurl = https://example.com/override-only
enabled = 1
`
	overrideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(overrideDir, "override.repo"), []byte(override), 0644); err != nil {
		t.Fatalf("failed to write override: %s", err)
	}

	if err := mergeRepoOverrides(repos, []string{overrideDir}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Existing key updated.
	if repos["base"]["enabled"] != "1" {
		t.Errorf("base.enabled = %q, want 1 (override not applied)", repos["base"]["enabled"])
	}
	// base.baseurl untouched by the override.
	if repos["base"]["baseurl"] != "https://example.com/base" {
		t.Errorf("base.baseurl = %q, want unchanged", repos["base"]["baseurl"])
	}
	// Override-only repo is NOT introduced: overrides modify existing repos only.
	if _, exists := repos["override-only"]; exists {
		t.Errorf("override-only repo should not be introduced by an override: %+v", repos["override-only"])
	}
}

// TestMergeRepoOverridesPrecedence verifies that when the same key is overridden
// by files in different directories, the later (user) directory wins, and that
// files are applied in alphabetical order by name.
func TestMergeRepoOverridesPrecedence(t *testing.T) {
	t.Parallel()

	repos := map[string]map[string]string{
		"repo": {"enabled": "1", "baseurl": "https://example.com/base"},
	}

	distroDir := t.TempDir()
	userDir := t.TempDir()

	// Distro override sets one baseurl; user override (later dir) wins on the
	// same filename.
	if err := os.WriteFile(filepath.Join(distroDir, "50-x.repo"),
		[]byte("[repo]\nbaseurl = https://example.com/distro\n"), 0644); err != nil {
		t.Fatalf("write distro override: %s", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "50-x.repo"),
		[]byte("[repo]\nbaseurl = https://example.com/user\n"), 0644); err != nil {
		t.Fatalf("write user override: %s", err)
	}

	// overrideDirs lists distro first, user second (increasing precedence).
	if err := mergeRepoOverrides(repos, []string{distroDir, userDir}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if repos["repo"]["baseurl"] != "https://example.com/user" {
		t.Errorf("baseurl = %q, want the user dir to win on identical filename", repos["repo"]["baseurl"])
	}
}

// TestGetEnabledReposSkipsUnparseableFile verifies that a malformed .repo file is
// skipped while the valid ones are still returned.
func TestGetEnabledReposSkipsUnparseableFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	good := `[good]
baseurl = https://example.com/good
enabled = 1
`
	if err := os.WriteFile(filepath.Join(tempDir, "good.repo"), []byte(good), 0644); err != nil {
		t.Fatalf("failed to write good.repo: %s", err)
	}
	// An INI parser error (unclosed section header) in one file must not abort
	// the whole gather.
	if err := os.WriteFile(filepath.Join(tempDir, "bad.repo"), []byte("[unclosed\nenabled = 1\n"), 0644); err != nil {
		t.Fatalf("failed to write bad.repo: %s", err)
	}

	repos, err := getEnabledRepos(tempDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(repos) != 1 || repos[0].RepoID != "good" {
		t.Errorf("got %+v, want only the good repo", repos)
	}
}
