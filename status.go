package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/rs/zerolog/log"
)

// RHSMEndPoints is structure used for storing GET response from
// REST API endpoint "/". This endpoint can be called using no-auth
// or consumer-cert-auth connection
type RHSMEndPoints struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// getAvailableConnection tries to get available connection. Return authenticated
// connection, when it is available. Otherwise, return no-auth connection. No-auth
// connection is available in all cases.
func (rhsmClient *RHSMClient) getAvailableConnection() (*RHSMConnection, error) {
	var connection *RHSMConnection
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
	return connection, err
}

// GetServerEndpoints tries to get list of supported server endpoints
func (rhsmClient *RHSMClient) GetServerEndpoints(metadata *RequestMetadata) (*[]RHSMEndPoints, error) {
	connection, err := rhsmClient.getAvailableConnection()
	if err != nil {
		return nil, fmt.Errorf("unable to get connection: %v", err)
	}

	var headers = make(map[string]string)
	metadata = sanitizeMetadata(metadata)
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

	var rhsmEndPoints []RHSMEndPoints
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
	connection, err := rhsmClient.getAvailableConnection()
	if err != nil {
		return nil, fmt.Errorf("unable to get connection: %v", err)
	}

	metadata = sanitizeMetadata(metadata)
	var headers = make(map[string]string)
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

	var rhsmStatus RHSMStatus
	err = json.Unmarshal([]byte(*resBody), &rhsmStatus)
	if err != nil {
		return nil, fmt.Errorf("unable to parse server status: %s", err)
	}

	return &rhsmStatus, nil
}

// HasCapability reports whether candlepin advertises the given capability in its
// GET /status response.
func (rhsmClient *RHSMClient) HasCapability(capability string, metadata *RequestMetadata) (bool, error) {
	status, err := rhsmClient.GetServerStatus(metadata)
	if err != nil {
		return false, err
	}
	return slices.Contains(status.ManagerCapabilities, capability), nil
}
