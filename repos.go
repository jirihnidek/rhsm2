package rhsm2

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"
)

// defaultReposDir is the standard location for dnf/yum .repo files.
const defaultReposDir = "/etc/yum.repos.d"

// EnabledRepository represents a single enabled dnf/yum repository read from
// the local .repo files. RepoID is the section name; BaseURLs holds the
// baseurl(s) as written (dnf variables are not expanded). A repo can have
// multiple baseurls (fallback mirrors), hence a slice.
type EnabledRepository struct {
	RepoID   string   `json:"repositoryid"`
	BaseURLs []string `json:"baseurl"`
}

// repoFileLoadOptions enables multi-line value parsing, needed to read dnf's
// indented-continuation format for multi-value keys such as baseurl.
var repoFileLoadOptions = ini.LoadOptions{AllowPythonMultilineValues: true}

// splitBaseURLs splits a (possibly multi-line) baseurl value into individual
// URLs, trimming whitespace and dropping blank lines.
func splitBaseURLs(value string) []string {
	var urls []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			urls = append(urls, line)
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

// parseEnabledReposFromINI extracts the enabled repositories from a loaded INI
// file. Split out from the file-walking logic so it can be unit-tested against
// in-memory content.
func parseEnabledReposFromINI(repoFile *ini.File) []EnabledRepository {
	var repos []EnabledRepository
	for _, section := range repoFile.Sections() {
		// Skip the implicit DEFAULT section; it is not a real repository.
		if section.Name() == ini.DefaultSection {
			continue
		}

		enabledValue := ""
		if section.HasKey("enabled") {
			enabledValue = section.Key("enabled").String()
		}
		if !isRepoEnabled(enabledValue) {
			continue
		}

		var baseURLs []string
		if section.HasKey("baseurl") {
			baseURLs = splitBaseURLs(section.Key("baseurl").String())
		}

		repos = append(repos, EnabledRepository{
			RepoID:   section.Name(),
			BaseURLs: baseURLs,
		})
	}
	return repos
}

// getEnabledReposFromDir reads all *.repo files in the given directory and
// returns the enabled repositories across all of them, sorted by RepoID. Split
// out from GetEnabledRepositories so tests can point it at a fixture directory.
func getEnabledReposFromDir(reposDir string) ([]EnabledRepository, error) {
	pattern := filepath.Join(reposDir, "*.repo")
	repoFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("unable to list repo files in %s: %w", reposDir, err)
	}

	var enabledRepos []EnabledRepository
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
		enabledRepos = append(enabledRepos, parseEnabledReposFromINI(repoFile)...)
	}

	sort.Slice(enabledRepos, func(i, j int) bool {
		return enabledRepos[i].RepoID < enabledRepos[j].RepoID
	})

	return enabledRepos, nil
}

// GetEnabledRepositories returns all enabled dnf/yum repositories on the system
// by reading the .repo files in /etc/yum.repos.d.
func GetEnabledRepositories() ([]EnabledRepository, error) {
	return getEnabledReposFromDir(defaultReposDir)
}
