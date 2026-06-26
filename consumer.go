package rhsm2

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
)

// ConsumerData is structure used for parsing JSON data returned by candlepin server
// for consumer objects
type ConsumerData struct {
	Created             string        `json:"created"`
	Updated             string        `json:"updated"`
	Id                  string        `json:"id"`
	Uuid                string        `json:"uuid"`
	Name                string        `json:"name"`
	Username            string        `json:"username"`
	EntitlementStatus   string        `json:"entitlementStatus"`
	ServiceLevel        string        `json:"serviceLevel"`
	Role                string        `json:"role"`
	Usage               string        `json:"usage"`
	AddOns              []interface{} `json:"addOns"`
	SystemPurposeStatus string        `json:"systemPurposeStatus"`
	ReleaseVer          struct {
		ReleaseVer interface{} `json:"releaseVer"`
	} `json:"releaseVer"`
	Owner struct {
		Id                string `json:"id"`
		Key               string `json:"key"`
		DisplayName       string `json:"displayName"`
		Href              string `json:"href"`
		ContentAccessMode string `json:"contentAccessMode"`
	} `json:"owner"`
	Environment      Environment `json:"environment"`
	EntitlementCount int         `json:"entitlementCount"`
	Facts            struct {
	} `json:"facts"`
	LastCheckin       interface{} `json:"lastCheckin"`
	InstalledProducts interface{} `json:"installedProducts"`
	CanActivate       bool        `json:"canActivate"`
	Capabilities      interface{} `json:"capabilities"`
	HypervisorId      interface{} `json:"hypervisorId"`
	ContentTags       interface{} `json:"contentTags"`
	Autoheal          bool        `json:"autoheal"`
	Annotations       interface{} `json:"annotations"`
	ContentAccessMode interface{} `json:"contentAccessMode"`
	Type              struct {
		Created  interface{} `json:"created"`
		Updated  interface{} `json:"updated"`
		Id       string      `json:"id"`
		Label    string      `json:"label"`
		Manifest bool        `json:"manifest"`
	} `json:"type"`
	IdCert struct {
		Created string `json:"created"`
		Updated string `json:"updated"`
		Id      string `json:"id"`
		Key     string `json:"key"`
		Cert    string `json:"cert"`
		Serial  struct {
			Created    string `json:"created"`
			Updated    string `json:"updated"`
			Id         int64  `json:"id"`
			Serial     int64  `json:"serial"`
			Expiration string `json:"expiration"`
			Revoked    bool   `json:"revoked"`
		} `json:"serial"`
	} `json:"idCert"`
	GuestIds       []interface{} `json:"guestIds"`
	Href           string        `json:"href"`
	ActivationKeys []interface{} `json:"activationKeys"`
	ServiceType    interface{}   `json:"serviceType"`
	Environments   []Environment `json:"environments"`
}

// GetConsumer tries to get consumer data from the candlepin server
// The consumer UUID is read from the installed consumer certificate and
// consumer cert auth is used for the request.
func (rhsmClient *RHSMClient) GetConsumer(metadata *RequestMetadata) (*ConsumerData, error) {
	uuid, err := rhsmClient.GetConsumerUUID()

	if err != nil {
		return nil, fmt.Errorf("unable to get consumer uuid: %v", err)
	}

	var headers = make(map[string]string)

	connection, err := rhsmClient.getCertAuthConnection()
	if err != nil {
		return nil, fmt.Errorf("unable to get consumer cert auth connection: %v", err)
	}
	res, err := connection.request(
		rhsmClient.UserAgent,
		http.MethodGet,
		"consumers/"+*uuid,
		"",
		"",
		&headers,
		nil,
		metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to get consumer: %s", err)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unable to get consumer: %d", res.StatusCode)
	}

	resBody, err := getResponseBody(res)
	if err != nil {
		return nil, err
	}

	consumerData := ConsumerData{}
	err = json.Unmarshal([]byte(*resBody), &consumerData)
	if err != nil {
		return nil, fmt.Errorf("unable to parse consumer object: %s", err)
	}

	return &consumerData, nil
}

// GetConsumerUUID tries to get consumer UUID from installed consumer certificate
func (rhsmClient *RHSMClient) GetConsumerUUID() (*string, error) {
	consumerCertFilePath := rhsmClient.consumerCertPath()
	consumerCert, err := os.ReadFile(*consumerCertFilePath)

	if err != nil {
		return nil, fmt.Errorf("failed to read consumer certificate: %v", err)
	}

	block, _ := pem.Decode(consumerCert)
	if block == nil {
		return nil, fmt.Errorf("failed to parse: %s (PEM block containing the public key)", *consumerCertFilePath)
	}

	if block.Type == "CERTIFICATE" {
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PEM certificate: %s: %v", *consumerCertFilePath, err)
		}

		return &certificate.Subject.CommonName, nil
	}

	return nil, fmt.Errorf("file %s does not contain CERTIFICATE block", *consumerCertFilePath)
}

// GetOwner tries to get owner from installed consumer certificate
func (rhsmClient *RHSMClient) GetOwner() (*string, error) {
	consumerCertFilePath := rhsmClient.consumerCertPath()

	consumerCert, err := os.ReadFile(*rhsmClient.consumerCertPath())

	if err != nil {
		return nil, fmt.Errorf("failed to read consumer certificate: %v", err)
	}

	block, _ := pem.Decode(consumerCert)
	if block == nil {
		return nil, fmt.Errorf("failed to parse: %s (PEM block containing the public key)", *consumerCertFilePath)
	}

	if block.Type == "CERTIFICATE" {
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PEM certificate: %s: %v", *consumerCertFilePath, err)
		}

		return &certificate.Subject.Organization[0], nil
	}

	return nil, fmt.Errorf("file %s does not contain CERTIFICATE block", *consumerCertFilePath)
}
