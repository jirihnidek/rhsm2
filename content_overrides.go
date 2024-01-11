package rhsm2

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

// ContentOverride is structure containing information about content
// override for given repository
type ContentOverride struct {
	Created      string `json:"created"`
	Updated      string `json:"updated"`
	Name         string `json:"name"`
	ContentLabel string `json:"contentLabel"`
	Value        string `json:"value"`
}

// GetContentOverrides tries to get content overrides from server
func (rhsmClient *RHSMClient) GetContentOverrides() ([]ContentOverride, error) {
	var contentOverrides []ContentOverride

	uuid, err := rhsmClient.GetConsumerUUID()

	if err != nil {
		return nil, err
	}

	res, err := rhsmClient.ConsumerCertAuthConnection.request(
		http.MethodGet,
		"consumers/"+*uuid+"/content_overrides",
		"",
		"",
		nil,
		nil,
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
		log.Error().Msgf("consumer with UUID: %s could no be found", *uuid)
		return nil, fmt.Errorf("unable to get content overrides")
	case 500:
		log.Error().Msgf("an unexpected exception has occurred")
		return nil, fmt.Errorf("unable to get content overrides")
	}

	return contentOverrides, nil
}

// createMapFromContentOverrides creates map with content overrides from list of
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
