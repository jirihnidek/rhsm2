package rhsm2

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"
)

// defaultReposDir is the standard location for dnf/yum .repo files.
const defaultReposDir = "/etc/yum.repos.d"

// defaultReposOverrideDirs are the dnf5 repo override dirs in increasing
// precedence order; on a filename collision the later (user) dir wins, so it is
// listed last.
var defaultReposOverrideDirs = []string{
	"/usr/share/dnf5/repos.override.d",
	dnf5ReposOverrideDirPath, // /etc/dnf/repos.override.d
}

// EnabledRepository is a single enabled dnf/yum repository. BaseURLs holds the
// baseurl(s) as written; dnf variables are not expanded. Metalink and Mirrorlist
// carry the repo's location when it uses those instead of a baseurl.
type EnabledRepository struct {
	RepoID     string   `json:"repositoryid"`
	BaseURLs   []string `json:"baseurl"`
	Metalink   string   `json:"metalink,omitempty"`
	Mirrorlist string   `json:"mirrorlist,omitempty"`
}

// repoFileLoadOptions enables multi-line value parsing for dnf's
// indented-continuation format (e.g. a multi-value baseurl).
var repoFileLoadOptions = ini.LoadOptions{AllowPythonMultilineValues: true}

// stripURLCredentials removes embedded userinfo (user:password@) from a URL so
// baseurl credentials are not disclosed to candlepin. A value that does not parse
// as a URL is returned unchanged.
func stripURLCredentials(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User == nil {
		return rawURL
	}
	parsed.User = nil
	return parsed.String()
}

// splitBaseURLs splits a (possibly multi-line) baseurl value into individual
// URLs, trimming whitespace, dropping blank lines, and stripping any embedded
// credentials. The result is non-nil so a baseurl-less repo (e.g. metalink or
// mirrorlist) marshals as [] rather than null.
func splitBaseURLs(value string) []string {
	urls := []string{}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			urls = append(urls, stripURLCredentials(line))
		}
	}
	return urls
}

// isRepoEnabled interprets a repo section's "enabled" value using the same
// tokens dnf accepts. A missing key (empty string) means enabled by default.
func isRepoEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

// mergeReposFromINI merges the repos from a loaded INI file into repos
// (repoID -> key -> value). Later definitions of the same repo ID/key win.
func mergeReposFromINI(repos map[string]map[string]string, repoFile *ini.File) {
	for _, section := range repoFile.Sections() {
		// Skip the implicit DEFAULT section; it is not a real repository.
		if section.Name() == ini.DefaultSection {
			continue
		}

		repoID := section.Name()
		if repos[repoID] == nil {
			repos[repoID] = make(map[string]string)
		}
		for _, key := range section.Keys() {
			repos[repoID][key.Name()] = key.Value()
		}
	}
}

// overrideFileOrder returns the override files from overrideDirs in the order dnf
// applies them: sorted by name, with a later dir masking an identically named
// file in an earlier one. Missing directories are ignored.
func overrideFileOrder(overrideDirs []string) []string {
	// byName maps file name -> full path; a later dir masks an earlier one.
	byName := make(map[string]string)
	for _, dir := range overrideDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			// A missing override directory is not an error.
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			byName[entry.Name()] = filepath.Join(dir, entry.Name())
		}
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, byName[name])
	}
	return paths
}

// mergeRepoOverrides applies the dnf5 repo overrides from overrideDirs on top of
// repos, in dnf's order so the last value wins. Per dnf semantics, overrides only
// modify existing repos; they never introduce a new repo ID.
func mergeRepoOverrides(repos map[string]map[string]string, overrideDirs []string) error {
	for _, overrideFilePath := range overrideFileOrder(overrideDirs) {
		overrides, err := readDnf5RepoOverrides(overrideFilePath)
		if err != nil {
			// A single bad override file should not abort the rest.
			log.Warn().Err(err).Msgf("skipping unparseable repo override file %s", overrideFilePath)
			continue
		}

		for repoID, keys := range overrides {
			if repos[repoID] == nil {
				continue
			}
			for name, value := range keys {
				repos[repoID][name] = value
			}
		}
	}
	return nil
}

// enabledReposFromMap returns the enabled repositories from the merged repo map,
// sorted by RepoID.
func enabledReposFromMap(repos map[string]map[string]string) []EnabledRepository {
	var enabledRepos []EnabledRepository
	for repoID, keys := range repos {
		if !isRepoEnabled(keys["enabled"]) {
			continue
		}
		enabledRepos = append(enabledRepos, EnabledRepository{
			RepoID:     repoID,
			BaseURLs:   splitBaseURLs(keys["baseurl"]),
			Metalink:   strings.TrimSpace(keys["metalink"]),
			Mirrorlist: strings.TrimSpace(keys["mirrorlist"]),
		})
	}

	sort.Slice(enabledRepos, func(i, j int) bool {
		return enabledRepos[i].RepoID < enabledRepos[j].RepoID
	})

	return enabledRepos
}

// getEnabledRepos reads all *.repo files in reposDir, applies the overrides from
// overrideDirs, and returns the enabled repositories. Split from
// GetEnabledRepositories so tests can point it at fixture paths.
func getEnabledRepos(reposDir string, overrideDirs []string) ([]EnabledRepository, error) {
	pattern := filepath.Join(reposDir, "*.repo")
	repoFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("unable to list repo files in %s: %w", reposDir, err)
	}

	repos := make(map[string]map[string]string)
	for _, repoFilePath := range repoFiles {
		// Skip anything that is not a regular file.
		info, err := os.Stat(repoFilePath)
		if err != nil || info.IsDir() {
			continue
		}

		repoFile, err := ini.LoadSources(repoFileLoadOptions, repoFilePath)
		if err != nil {
			// A single bad .repo file should not abort the rest.
			log.Warn().Err(err).Msgf("skipping unparseable repo file %s", repoFilePath)
			continue
		}
		mergeReposFromINI(repos, repoFile)
	}

	if err := mergeRepoOverrides(repos, overrideDirs); err != nil {
		return nil, err
	}

	return enabledReposFromMap(repos), nil
}

// GetEnabledRepositories returns the enabled dnf/yum repositories on the system,
// reading /etc/yum.repos.d and applying the dnf5 overrides so the result reflects
// dnf's effective state.
func GetEnabledRepositories() ([]EnabledRepository, error) {
	return getEnabledRepos(defaultReposDir, defaultReposOverrideDirs)
}
