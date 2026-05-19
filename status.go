package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// RHSMEndPoints is structure used for storing GET response from
// REST API endpoint "/". This endpoint can be called using no-auth
// or consumer-cert-auth connection
type RHSMEndPoints struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// GetServerEndpoints tries to get list of supported server endpoints
func (rhsmClient *RHSMClient) GetServerEndpoints(metadata *RequestMetadata) (*[]RHSMEndPoints, error) {
	var rhsmEndPoints []RHSMEndPoints
	var connection *RHSMConnection

	var headers = make(map[string]string)

	metadata = sanitizeMetadata(metadata)

	_, err := rhsmClient.GetConsumerUUID()
	if err == nil {
		connection, err = rhsmClient.getCertAuthConnection()
		if err != nil {
			return nil, fmt.Errorf("unable to get consumer cert auth connection: %v", err)
		}
	} else {
		// When no consumer has been installed, then we will
		// try to use no-auth connection. When server is available,
		// then this should work
		connection, err = rhsmClient.getNoAuthConnection()
		if err != nil {
			return nil, fmt.Errorf("unable to get no-auth connection: %v", err)
		}
	}

	res, err := connection.request(
		rhsmClient.UserAgent,
		http.MethodGet,
		"",
		"",
		"",
		&headers,
		nil,
		metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to get server endpoints :%v", err)
	}

	defer func() {
		// We can ignore error returning, by Close(), because we only
		// read content of body
		_ = res.Body.Close()
	}()

	resBody, err := getResponseBody(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(*resBody), &rhsmEndPoints)
	if err != nil {
		return nil, fmt.Errorf("unable to parse server endpoints: %s", err)
	}

	return &rhsmEndPoints, nil
}

// RHSMStatus is structure used for storing GET response from REST API
// endpoint "/status". This endpoint can be called using no-auth or
// consumer-cert-auth connection
type RHSMStatus struct {
	Mode           string      `json:"mode"`
	ModeReason     interface{} `json:"modeReason"`
	ModeChangeTime interface{} `json:"modeChangeTime"`
	Result         bool        `json:"result"`
	Version        string      `json:"version"`
	Release        string      `json:"release"`
	Standalone     bool        `json:"standalone"`
	// Note: json module cannot unmarshal timeUTC into time.Time
	// for this reason: https://github.com/golang/go/issues/47353
	// Because we do not need to use timeUTC for anything ATM.
	// It is parsed as normal string.
	TimeUTC             string      `json:"timeUTC"`
	RulesSource         string      `json:"rulesSource"`
	RulesVersion        string      `json:"rulesVersion"`
	ManagerCapabilities []string    `json:"managerCapabilities"`
	KeycloakRealm       interface{} `json:"keycloakRealm"`
	KeycloakAuthUrl     interface{} `json:"keycloakAuthUrl"`
	KeycloakResource    interface{} `json:"keycloakResource"`
	DeviceAuthRealm     interface{} `json:"deviceAuthRealm"`
	DeviceAuthUrl       interface{} `json:"deviceAuthUrl"`
	DeviceAuthClientId  interface{} `json:"deviceAuthClientId"`
	DeviceAuthScope     interface{} `json:"deviceAuthScope"`
}

// GetServerStatus tries to get status from the server. This
// method is possible to call, when server is connected or not
func (rhsmClient *RHSMClient) GetServerStatus(metadata *RequestMetadata) (*RHSMStatus, error) {
	var rhsmStatus RHSMStatus
	var connection *RHSMConnection

	var headers = make(map[string]string)

	metadata = sanitizeMetadata(metadata)

	_, err := rhsmClient.GetConsumerUUID()
	if err == nil {
		connection, err = rhsmClient.getCertAuthConnection()
		if err != nil {
			return nil, fmt.Errorf("unable to get consumer cert auth connection: %v", err)
		}
	} else {
		// When no consumer has been installed, then we will
		// try to use no-auth connection. When server is available,
		// then this should work
		connection, err = rhsmClient.getNoAuthConnection()
		if err != nil {
			return nil, fmt.Errorf("unable to get no-auth connection: %v", err)
		}
	}

	res, err := connection.request(
		rhsmClient.UserAgent,
		http.MethodGet,
		"status",
		"",
		"",
		&headers,
		nil,
		metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to get server status :%v", err)
	}

	// Server can respond only with 200 or 500 status codes
	if res.StatusCode == 500 {
		var unregisterServerError UnregisterServerError
		parseServerResponse(&unregisterServerError, res)
		log.Error().Msgf("unable to server status: %s",
			unregisterServerError.DisplayMessage)
		return nil, unregisterServerError
	}

	resBody, err := getResponseBody(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(*resBody), &rhsmStatus)
	if err != nil {
		return nil, fmt.Errorf("unable to parse server status: %s", err)
	}

	return &rhsmStatus, nil
}
