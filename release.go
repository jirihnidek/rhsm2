package rhsm2

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// getListingPath tries to get a listing path from the content path. The listing path is the path
// where is probably stored the 'listing' file containing the list of available releases.
func getListingPath(contentPath *string) (string, error) {
	if !strings.Contains(*contentPath, "$releasever") {
		return "", fmt.Errorf("content path: '%s' does not contain '$releasever'", *contentPath)
	}

	parts := strings.SplitN(*contentPath, "$releasever", 2)
	if len(parts) > 0 {
		return parts[0], nil
	}

	return "", fmt.Errorf("cannot split: '%s' using '$releasever' keyword", *contentPath)
}

func (rhsmClient *RHSMClient) getListingFile(listingPath string) (*string, error) {
	resp, err := rhsmClient.EntitlementCertAuthConnection.request(
		http.MethodGet,
		listingPath,
		"",
		"",
		nil,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unable to get content listing: %d", resp.StatusCode)
	}

	respBody, err := getResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("unable to get content listing: %s", err)
	}

	return respBody, nil
}

// Release represents the release object returned from candlepin server
type Release struct {
	Version string `json:"releaseVer"`
}

// GetReleaseFromServer tries to get the latest release from the candlepin server.
func (rhsmClient *RHSMClient) GetReleaseFromServer(clientInfo *ClientInfo) (string, error) {
	consumerUuid, err := rhsmClient.GetConsumerUUID()

	if err != nil {
		return "", err
	}

	var headers = make(map[string]string)

	if clientInfo == nil {
		clientInfo = &ClientInfo{"", "", ""}
	}
	clientInfo.xCorrelationId = createCorrelationId()

	res, err := rhsmClient.ConsumerCertAuthConnection.request(
		http.MethodGet,
		"consumers/"+*consumerUuid+"/release",
		"",
		"",
		&headers,
		nil,
		clientInfo,
	)

	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("unable to get latest release: %d", res.StatusCode)
	}
	resBody, err := getResponseBody(res)
	if err != nil {
		return "", err
	}

	var release Release
	err = json.Unmarshal([]byte(*resBody), &release)
	if err != nil {
		return "", err
	}

	return release.Version, nil
}

// GetCdnReleases tries to get the list of available releases from CDN. The list of releases is
// should include only unique values of releases. There should not be any duplicates.
func (rhsmClient *RHSMClient) GetCdnReleases(clientInfo *ClientInfo) (map[string]struct{}, error) {
	// If the connection to the repository does not exist, return error
	if rhsmClient.EntitlementCertAuthConnection == nil || rhsmClient.EntitlementCertAuthConnection.Client == nil {
		return nil, errors.New("connection to repository does not exist")
	}

	// Get all engineering products
	engineeringProductsMap, err := rhsmClient.getEngineeringProducts()
	if err != nil {
		return nil, err
	}

	if len(engineeringProductsMap) == 0 {
		return nil, errors.New("no engineering products found")
	}

	listingPaths := getListingPathFromEngProducts(engineeringProductsMap)

	releases := rhsmClient.getAllReleasesFromPaths(listingPaths)

	return releases, nil
}

// getAllReleasesFromPaths tries to get the list of available releases from given content paths.
// The list of releases should include only unique values of releases. There should not be
// any duplicates.
func (rhsmClient *RHSMClient) getAllReleasesFromPaths(listingPaths map[string]struct{}) map[string]struct{} {
	var releaseMap = make(map[string]struct{})
	for path := range listingPaths {
		listingPath := filepath.Join(path, "/listing")
		respBody, err := rhsmClient.getListingFile(listingPath)
		if err != nil {
			log.Warn().Msgf("failed to retrieve listing file from path: %s: %s", path, err)
			continue
		}

		releases := parseListingFileContent(respBody, &listingPath)

		log.Debug().Msgf("got %v releases from path: %s", releases, path)

		for _, release := range releases {
			if _, exists := releaseMap[release]; !exists {
				releaseMap[release] = struct{}{}
			}
		}
	}
	return releaseMap
}

// getListingPathFromEngProducts tries to get the content path, which should contain the 'listing' file.
// We detect candidates if the content path contains the '$releasever' keyword.
// The list of content paths is returned as a map. Thus, there should not be any duplicates.
// Each item of the map contains the list of content labels.
func getListingPathFromEngProducts(engineeringProductsMap map[int64][]EngineeringProduct) map[string]struct{} {
	// Go through all products and get all unique base content paths
	listingPaths := make(map[string]struct{})
	for _, products := range engineeringProductsMap {
		for _, product := range products {
			for _, content := range product.Content {
				if content.Enabled == nil || *content.Enabled {
					basePath, err := getListingPath(&content.Path)
					if err != nil {
						continue
					}
					if _, exists := listingPaths[basePath]; !exists {
						log.Debug().Msgf("adding path %s to the list of listing paths", basePath)
						listingPaths[basePath] = struct{}{}
					}
				}
			}
		}
	}
	return listingPaths
}

// parseListingFileContent tries to parse the content of the listing file. The result map should
// contain only unique values of releases. There should not be any duplicates in the result.
// Empty lines and lines starting with '#' are ignored.
//
// The listing file could look like this:
// # The listing file for Red Hat Enterprise Linux Server 10.x
// 10
// 10.1
// 10.2
// 10.3
func parseListingFileContent(respBody *string, listingPath *string) []string {
	releasesMap := make(map[string]struct{})
	lines := strings.Split(*respBody, "\n")
	for _, line := range lines {
		// Remove whitespaces
		line = strings.TrimSpace(line)
		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}
		if line != "" {
			if _, exists := releasesMap[line]; !exists {
				releasesMap[line] = struct{}{}
			} else {
				log.Warn().Msgf("duplicate release found: %s in %s", line, *listingPath)
			}
		}
	}
	var releases []string
	for release := range releasesMap {
		releases = append(releases, release)
	}
	sort.Strings(releases)
	return releases
}
