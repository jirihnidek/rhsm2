package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"
)

const dnf5ReposOverrideDirPath = "/etc/dnf/repos.override.d"
const dnf5ReposOverrideFileName = "98-redhat.repo"
const Dnf5RedHatReposOverrideFilePath = dnf5ReposOverrideDirPath + "/" + dnf5ReposOverrideFileName

// ContentOverride is a structure containing information about content
// override for a given repository
type ContentOverride struct {
	Created      string `json:"created,omitempty"`
	Updated      string `json:"updated,omitempty"`
	Name         string `json:"name"`
	ContentLabel string `json:"contentLabel"`
	Value        string `json:"value"`
}

// GetContentOverrides tries to get content overrides from server
func (rhsmClient *RHSMClient) GetContentOverrides(info *RequestMetadata) ([]ContentOverride, error) {
	var contentOverrides []ContentOverride

	consumerUuid, err := rhsmClient.GetConsumerUUID()

	if err != nil {
		return nil, err
	}

	var headers = make(map[string]string)

	connection, err := rhsmClient.getCertAuthConnection()
	if err != nil {
		return nil, fmt.Errorf("unable to get consumer cert auth connection: %v", err)
	}
	res, err := connection.request(
		rhsmClient.UserAgent,
		http.MethodGet,
		"consumers/"+*consumerUuid+"/content_overrides",
		"",
		"",
		&headers,
		nil,
		info,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to get content overrides: %s", err)
	}

	switch res.StatusCode {
	case 200:
		resBody, err := getResponseBody(res)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(*resBody), &contentOverrides)
		if err != nil {
			return nil, err
		}
	case 403:
		log.Error().Msgf("insufficient permissions")
		return nil, fmt.Errorf("unable to get content overrides")
	case 404:
		log.Error().Msgf("consumer with UUID: %s could no be found", *consumerUuid)
		return nil, fmt.Errorf("unable to get content overrides")
	case 500:
		log.Error().Msgf("an unexpected exception has occurred")
		return nil, fmt.Errorf("unable to get content overrides")
	}

	return contentOverrides, nil
}

// readDnf5RepoOverrides tries to read content overrides from dnf5 repo override file.
// We try to read repo overrides using ini package. Hopefully, it will work without any issue.
// It returns a map of repoId -> {key: value} representing the INI sections and their keys.
func readDnf5RepoOverrides(filePath string) (map[string]map[string]string, error) {
	repo, err := ini.Load(filePath)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]string)
	for _, section := range repo.Sections() {
		// Skip the DEFAULT section added by ini package
		if section.Name() == ini.DefaultSection {
			continue
		}
		sectionMap := make(map[string]string)
		for _, key := range section.Keys() {
			sectionMap[key.Name()] = key.Value()
		}
		result[section.Name()] = sectionMap
	}
	return result, nil
}

// ReadLocalContentOverrides reads the DNF5 repo override file at the given path
// and returns its contents as a slice of ContentOverride structs.
func ReadLocalContentOverrides(filePath string) ([]ContentOverride, error) {
	repoOverrides, err := readDnf5RepoOverrides(filePath)
	if err != nil {
		return nil, err
	}

	var overrides []ContentOverride
	for label, override := range repoOverrides {
		for name, value := range override {
			overrides = append(overrides, ContentOverride{
				ContentLabel: label,
				Name:         name,
				Value:        value,
			})
		}
	}
	return overrides, nil
}

// WriteDnf5RepoOverrides writes content overrides to a dnf5 repo override file in INI format.
func WriteDnf5RepoOverrides(contentOverrides []ContentOverride, filePath string) error {
	// First, create empty ini file object
	repo := ini.Empty()

	// Convert the list of content overrides to a map
	mapContentOverrides := createMapFromContentOverrides(contentOverrides)

	// Fill ini file (repo file) with repo override from the map
	for contentLabel, contentOverrideMap := range mapContentOverrides {
		var section *ini.Section
		var err error

		if repo.HasSection(contentLabel) {
			section = repo.Section(contentLabel)
		} else {
			section, err = repo.NewSection(contentLabel)
			if err != nil {
				return err
			}
		}
		for contentName, contentValue := range contentOverrideMap {
			var key *ini.Key
			if section.HasKey(contentName) {
				key = section.Key(contentName)
			} else {
				key, err = section.NewKey(contentName, contentValue)
				if err != nil {
					return err
				}
			}
			key.SetValue(contentValue)
		}
	}

	// Write repo override to the file
	err := repo.SaveTo(filePath)
	if err != nil {
		return err
	}

	return nil
}

// SendContentOverrides sends content overrides to the candlepin server via PUT request.
// The overrides are sent to the consumers/{uuid}/content_overrides endpoint.
func (rhsmClient *RHSMClient) SendContentOverrides(overrides []ContentOverride, info *RequestMetadata) error {
	consumerUuid, err := rhsmClient.GetConsumerUUID()
	if err != nil {
		return err
	}

	headers := map[string]string{"Content-type": "application/json"}

	info = sanitizeMetadata(info)

	body, err := json.Marshal(overrides)
	if err != nil {
		return fmt.Errorf("unable to marshal content overrides: %s", err)
	}

	connection, err := rhsmClient.getCertAuthConnection()
	if err != nil {
		return fmt.Errorf("unable to get consumer cert auth connection: %v", err)
	}
	res, err := connection.request(
		rhsmClient.UserAgent,
		http.MethodPut,
		"consumers/"+*consumerUuid+"/content_overrides",
		"",
		"",
		&headers,
		&body,
		info,
	)
	if err != nil {
		return fmt.Errorf("unable to put content overrides: %s", err)
	}

	switch res.StatusCode {
	case 200:
		return nil
	case 403:
		log.Error().Msgf("insufficient permissions")
		return fmt.Errorf("unable to put content overrides: insufficient permissions")
	case 404:
		log.Error().Msgf("consumer with UUID: %s could not be found", *consumerUuid)
		return fmt.Errorf("unable to put content overrides: consumer not found")
	case 500:
		log.Error().Msgf("an unexpected exception has occurred")
		return fmt.Errorf("unable to put content overrides: internal server error")
	default:
		return fmt.Errorf("unable to put content overrides: unexpected status code %d", res.StatusCode)
	}
}

// createMapFromContentOverrides creates the map with content overrides from the list of
// content overrides returned from candlepin server
func createMapFromContentOverrides(contentOverrides []ContentOverride) map[string]map[string]string {
	mapContentOverrides := make(map[string]map[string]string)
	for _, contentOverride := range contentOverrides {
		if _, exist := mapContentOverrides[contentOverride.ContentLabel]; !exist {
			mapContentOverrides[contentOverride.ContentLabel] = make(map[string]string)
		}
		mapContentOverrides[contentOverride.ContentLabel][contentOverride.Name] = contentOverride.Value
	}
	return mapContentOverrides
}
