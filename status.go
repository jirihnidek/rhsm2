package rhsm2

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

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
func (rhsmClient *RHSMClient) GetServerStatus() (*RHSMStatus, error) {
	var rhsmStatus RHSMStatus
	var connection *RHSMConnection

	var headers = make(map[string]string)
	headers["X-Correlation-ID"] = createCorrelationId()

	_, err := rhsmClient.GetConsumerUUID()
	if err == nil {
		connection = rhsmClient.ConsumerCertAuthConnection
	} else {
		// When no consumer has been installed, then we will
		// try to use no-auth connection. When server is available,
		// then this should work
		connection = rhsmClient.NoAuthConnection
	}

	res, err := connection.request(
		http.MethodGet,
		"status",
		"",
		"",
		&headers,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to get server status :%v", err)
	}

	// Server can respond only with 200 or 500 status code
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
