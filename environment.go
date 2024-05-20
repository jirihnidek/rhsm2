package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Environment contains information about environment
// returned form candlepin server. The Owner is pointer
// on structure, because this structure is also used
// during registration and only ID of environment is
// really necessary
type Environment struct {
	Created       string      `json:"created,omitempty"`
	Updated       string      `json:"updated,omitempty"`
	Id            string      `json:"id"`
	Name          string      `json:"name,omitempty"`
	Type          interface{} `json:"type,omitempty"`
	Description   string      `json:"description,omitempty"`
	ContentPrefix interface{} `json:"contentPrefix,omitempty"`
	Owner         *struct {
		Id                string `json:"id"`
		Key               string `json:"key,omitempty"`
		DisplayName       string `json:"displayName,omitempty"`
		Href              string `json:"href,omitempty"`
		ContentAccessMode string `json:"contentAccessMode,omitempty"`
	} `json:"owner,omitempty"`
	EnvironmentContent []interface{} `json:"environmentContent,omitempty"`
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
