package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Environment contains information about environment
// returned form candlepin server
type Environment struct {
	Created       string      `json:"created"`
	Updated       string      `json:"updated"`
	Id            string      `json:"id"`
	Name          string      `json:"name"`
	Type          interface{} `json:"type"`
	Description   string      `json:"description"`
	ContentPrefix interface{} `json:"contentPrefix"`
	Owner         struct {
		Id                string `json:"id"`
		Key               string `json:"key"`
		DisplayName       string `json:"displayName"`
		Href              string `json:"href"`
		ContentAccessMode string `json:"contentAccessMode"`
	} `json:"owner"`
	EnvironmentContent []interface{} `json:"environmentContent"`
}

// GetEnvironments tries to get list of environments from candlepin server
func (rhsmClient *RHSMClient) GetEnvironments(
	username string,
	password string,
	organization string,
	clientInfo *ClientInfo,
) ([]Environment, error) {
	var environments []Environment
	var headers = make(map[string]string)

	headers["username"] = username
	headers["password"] = password

	if clientInfo == nil {
		clientInfo = &ClientInfo{"", "", ""}
	}
	clientInfo.xCorrelationId = createCorrelationId()

	res, err := rhsmClient.NoAuthConnection.request(
		http.MethodGet,
		"owners/"+organization+"/environments",
		"",
		"",
		&headers,
		nil,
		clientInfo)

	if err != nil {
		return environments, fmt.Errorf("unable to get list of environments: %s", err)
	}

	resBody, err := getResponseBody(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(*resBody), &environments)
	if err != nil {
		return environments, fmt.Errorf("unable to unmarshal list of environments: %s", err)
	}

	return environments, nil
}
