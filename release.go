package rhsm2

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// isAnyRequiredTagProvided tries to find if any of the required tags is provided in the list
// of release tags. The function returns true if any of the required tags is provided in the
// release tags. Otherwise, it returns false.
//
// Example:
//
//	requiredTags = ["rhel-11", "rhel-11-x86_64"]
//	releaseTags = ["rhel-11-x86_64"]
//	isAnyRequiredTagProvided(requiredTags, releaseTags) -> true
func isAnyRequiredTagProvided(requiredTags []string, releaseTags []string) bool {
	// If no tags are required, then return true
	if len(requiredTags) == 0 {
		return true
	}
	// Check if any of the required tags is provided in the release tags
	for _, requiredTag := range requiredTags {
		for _, releaseTag := range releaseTags {
			// It is enough to check if the required tag starts with the release tag.
			if strings.HasPrefix(releaseTag, requiredTag) {
				log.Debug().Msgf("required tag %s matches release tags: %s", requiredTag, releaseTags)
				return true
			}
		}
	}
	return false
}

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

const DefaultOsReleaseFilePath = "/etc/os-release"

type OSRelease struct {
	ID           string
	VersionID    string
	VersionMajor string
	VersionMinor string
}

// parseOSRelease tries to parse the content of the /etc/os-release file.
// It reads only the ID and VERSION_ID attributes.
func parseOSRelease(content *[]byte) (*OSRelease, error) {
	release := OSRelease{}
	lines := strings.Split(string(*content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		switch key {
		case "ID":
			release.ID = value
		case "VERSION_ID":
			release.VersionID = value
		}
	}

	if release.ID == "" || release.VersionID == "" {
		return nil, fmt.Errorf("unable to parse ID or VERSION_ID from os release file")
	}

	// Split the version ID into the major and minor version
	versionParts := strings.Split(release.VersionID, ".")
	if len(versionParts) > 0 {
		release.VersionMajor = versionParts[0]
		if len(versionParts) > 1 {
			release.VersionMinor = versionParts[1]
		}
	}

	return &release, nil
}

// filterInstalledProductsUsingOSRelease tries to filter the list of installed product certificates
// using the current release of Linux distribution.
//
// Example:
//
//		installedProducts = [
//			{
//				filePath: "/etc/pki/product/698.pem",
//				providedTags: ["rhel-11", "rhel-11-x86_64"]
//			},
//			{
//				filePath: "/etc/pki/product/69.pem",
//				providedTags: ["rhel-7", "rhel-7-x86_64"]
//			},
//	 ]
//
//		osRelease = {
//			ID: "rhel",
//			VersionID: "11"
//		}
//
//		filterInstalledProductsUsingOSRelease(installedProducts, osRelease) -> [
//			{
//				filePath: "/etc/pki/product/698.pem",
//				providedTags: ["rhel-11", "rhel-11-x86_64"]
//			},
//
// ]
func (rhsmClient *RHSMClient) filterInstalledProductsUsingOSRelease(installedProducts []InstalledProduct) ([]InstalledProduct, error) {
	content, err := os.ReadFile(rhsmClient.RHSMConf.osReleaseFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s: %s", rhsmClient.RHSMConf.osReleaseFilePath, err)
	}

	release, err := parseOSRelease(&content)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("os release: %s-%s parsed from: %v",
		release.ID,
		release.VersionMajor,
		rhsmClient.RHSMConf.osReleaseFilePath,
	)

	// Tag in product certificates should follow the pattern: <os-release>-<major-version>*
	osReleaseTag := strings.ToLower(release.ID) + "-" + release.VersionMajor

	var filteredProducts []InstalledProduct
	for _, product := range installedProducts {
		found := false
		// Look for tags that match the current OS release version
		// E.g.: When running RHEL 11, look for rhel-11 tags
		for _, tag := range product.providedTags {
			if strings.HasPrefix(tag, osReleaseTag) {
				filteredProducts = append(filteredProducts, product)
				found = true
				break
			}
		}
		if !found {
			log.Warn().Msgf(
				"skipping product: %s; its tags: %s do not match os release: %s",
				product.filePath,
				product.providedTags,
				osReleaseTag,
			)
		}
	}

	if len(filteredProducts) == 0 {
		return filteredProducts, fmt.Errorf(
			"no installed product certificate matches os release: %s", osReleaseTag,
		)
	}

	return filteredProducts, nil
}

// getListingFile tries to get the content of the 'listing' file from CDN
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
	ReleaseVer string `json:"releaseVer"`
}

// SetReleaseOnServer tries to set the release on the candlepin server only (not on the host in the variable
// file in /etc/dnf/vars/).
func (rhsmClient *RHSMClient) SetReleaseOnServer(clientInfo *ClientInfo, release string) error {
	consumerUuid, err := rhsmClient.GetConsumerUUID()

	if err != nil {
		return err
	}

	var headers = make(map[string]string)

	if clientInfo == nil {
		clientInfo = &ClientInfo{"", "", ""}
	}
	clientInfo.xCorrelationId = createCorrelationId()

	headers["Content-type"] = "application/json"
	consumerData := Release{
		ReleaseVer: release,
	}
	body, err := json.Marshal(consumerData)
	if err != nil {
		return err
	}

	res, err := rhsmClient.ConsumerCertAuthConnection.request(
		http.MethodPut,
		"consumers/"+*consumerUuid,
		"",
		"",
		&headers,
		&body,
		clientInfo,
	)

	if err != nil {
		return err
	}

	if res.StatusCode != 204 {
		return fmt.Errorf("unable to set release: %d", res.StatusCode)
	}

	return nil
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

	return release.ReleaseVer, nil
}

// getReleaseTags tries to get the list of tags from installed product certificates.
func (rhsmClient *RHSMClient) getReleaseTags() ([]string, error) {
	installedProducts := rhsmClient.getInstalledProducts()
	if len(installedProducts) == 0 {
		return nil, errors.New("no installed product certificate found")
	}

	// Use only product certificate that matches current release of Linux distribution.
	// E.g.: When the current major release is RHEL 11, use only product certificates
	// that contain tags for RHEL 11
	filteredInstalledProducts, err := rhsmClient.filterInstalledProductsUsingOSRelease(installedProducts)
	if err != nil {
		return nil, err
	}

	var installedProductFilePaths []string
	for _, product := range filteredInstalledProducts {
		installedProductFilePaths = append(installedProductFilePaths, product.filePath)
	}
	log.Debug().Msgf("trying to get release tags from installed products: %v", installedProductFilePaths)

	requiredTags := createListOfContentTags(installedProducts)
	return requiredTags, nil
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

	// Get the list tags from installed product certificates
	// These tags will be used for filtering the content path
	// used for getting the list of available releases
	releaseTags, err := rhsmClient.getReleaseTags()
	if err != nil {
		log.Debug().Msgf("unable to get release tags: %s", err)
		return nil, err
	}
	log.Debug().Msgf("release tags: %v", releaseTags)

	listingPaths := getListingPathFromEngProducts(engineeringProductsMap, releaseTags)

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
// Each item of the map contains the list of content labels. The list of release tags is used for
// filtering the content.
func getListingPathFromEngProducts(
	engineeringProductsMap map[int64][]EngineeringProduct,
	releaseTags []string,
) map[string]struct{} {
	// Go through all products and get all unique base content paths
	listingPaths := make(map[string]struct{})
	for _, products := range engineeringProductsMap {
		for _, product := range products {
			for _, content := range product.Content {
				// If the content->enabled is not defined in the entitlement certificate,
				// then the content is considered as enabled by default.
				if content.Enabled == nil || *content.Enabled {
					// Check if any of tag required by content is provided in the release tags
					if !isAnyRequiredTagProvided(content.RequiredTags, releaseTags) {
						log.Debug().Msgf(
							"skipping content: '%s'; no of its required tags: %s found in release tags: %s",
							content.Label, content.RequiredTags, releaseTags,
						)
						continue
					}
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
