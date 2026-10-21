package rhsm2

import (
	"os"
	"path/filepath"
	"reflect"
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
			name:  "empty value",
			value: "",
			want:  nil,
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

// TestParseEnabledReposFromINI verifies enabled-repo extraction from in-memory
// INI content, covering explicit enable, disable, default-enabled, the
// boolean token form, and multi-baseurl (mirror fallback) repos.
func TestParseEnabledReposFromINI(t *testing.T) {
	t.Parallel()

	repoFile, err := ini.LoadSources(repoFileLoadOptions, []byte(sampleRepoFile))
	if err != nil {
		t.Fatalf("failed to load sample ini: %s", err)
	}

	repos := parseEnabledReposFromINI(repoFile)

	got := make(map[string][]string)
	for _, r := range repos {
		got[r.RepoID] = r.BaseURLs
	}

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
	if len(repos) != len(expected) {
		t.Errorf("got %d enabled repos, want %d (%+v)", len(repos), len(expected), repos)
	}
}

// TestGetEnabledReposFromDir verifies reading and merging .repo files from a
// directory, that results are sorted by RepoID, and that non-.repo files are
// ignored.
func TestGetEnabledReposFromDir(t *testing.T) {
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

	repos, err := getEnabledReposFromDir(tempDir)
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
}

// TestGetEnabledReposFromDirEmpty verifies that a directory with no .repo files
// yields an empty result and no error.
func TestGetEnabledReposFromDirEmpty(t *testing.T) {
	t.Parallel()

	repos, err := getEnabledReposFromDir(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(repos) != 0 {
		t.Errorf("got %d repos, want 0", len(repos))
	}
}
