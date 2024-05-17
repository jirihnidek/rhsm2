package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// OrganizationData is structure used for parsing JSON document returned
// by candlepin. This structure represents one organization
type OrganizationData struct {
	Created                    string      `json:"created"`
	Updated                    string      `json:"updated"`
	Id                         string      `json:"id"`
	DisplayName                string      `json:"displayName"`
	Key                        string      `json:"key"`
	ContentPrefix              interface{} `json:"contentPrefix"`
	DefaultServiceLevel        interface{} `json:"defaultServiceLevel"`
	LogLevel                   interface{} `json:"logLevel"`
	ContentAccessMode          string      `json:"contentAccessMode"`
	ContentAccessModeList      string      `json:"contentAccessModeList"`
	AutobindHypervisorDisabled bool        `json:"autobindHypervisorDisabled"`
	AutobindDisabled           bool        `json:"autobindDisabled"`
	LastRefreshed              string      `json:"lastRefreshed"`
	ParentOwner                interface{} `json:"parentOwner"`
	UpstreamConsumer           interface{} `json:"upstreamConsumer"`
	Anonymous                  interface{} `json:"anonymous"`
	Claimed                    interface{} `json:"claimed"`
}

// GetOrgs tries to get list of available organizations for given username
func (rhsmClient *RHSMClient) GetOrgs(
	username string,
	password string,
	clientInfo *ClientInfo,
) ([]OrganizationData, error) {
	var organizations []OrganizationData
	var headers = make(map[string]string)

	headers["username"] = username
	headers["password"] = password

	if clientInfo == nil {
		clientInfo = &ClientInfo{"", "", ""}
	}
	clientInfo.xCorrelationId = createCorrelationId()

	res, err := rhsmClient.NoAuthConnection.request(
		http.MethodGet,
		"users/"+username+"/owners",
		"",
		"",
		&headers,
		nil,
		clientInfo)
	if err != nil {
		return organizations, fmt.Errorf("unable to get list of org IDs: %s", err)
	}

	resBody, err := getResponseBody(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(*resBody), &organizations)
	if err != nil {
		return organizations, fmt.Errorf("unable to unmarshal list of organizations: %s", err)
	}

	return organizations, nil
}
