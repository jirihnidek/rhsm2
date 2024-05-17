package rhsm2

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SystemFacts is collection of system facts necessary during registration
type SystemFacts struct {
	SystemCertificateVersion string `json:"system.certificate_version"`
}

type Environment struct {
	Id string `json:"id"`
}

// RegisterData is structure representing JSON data used for register request
type RegisterData struct {
	Type              string             `json:"type"`
	Name              string             `json:"name"`
	Facts             *SystemFacts       `json:"facts"`
	InstalledProducts []InstalledProduct `json:"installedProducts"`
	ContentTags       []string           `json:"contentTags"`
	Role              string             `json:"role"`
	AddOns            []interface{}      `json:"addOns"`
	Usage             string             `json:"usage"`
	ServiceLevel      string             `json:"serviceLevel"`
	Environments      []string           `json:"environments"`
}

// ConsumerData is structure used for parsing JSON data returned during registration
// when system was successfully registered and consumer was created
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
	Environment      interface{} `json:"environment"`
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
	Environments   interface{}   `json:"environments"`
}

// RegisterError is structure used for parsing JSON document returned
// by candlepin server, when registration is not successful
type RegisterError struct {
	DisplayMessage string `json:"displayMessage"`
	RequestUuid    string `json:"requestUuid"`
}

// registerSystem tries to register system
func (rhsmClient *RHSMClient) registerSystem(
	headers map[string]string,
	query string,
	clientInfo *ClientInfo,
) (*ConsumerData, error) {
	isActivationKeyUsed := false
	if strings.Contains(query, "&activation_keys=") {
		isActivationKeyUsed = true
	}

	// It is necessary to set system certificate version to value 3.0 or higher
	facts := SystemFacts{
		SystemCertificateVersion: "3.2",
		// TODO: try to get some real facts.
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to get hostname: %s", err)
	}

	var sysPurpose *SysPurposeJSON
	sysPurpose, err = getSystemPurpose(&rhsmClient.RHSMConf.syspurposeFilePath)
	if err != nil {
		log.Warn().Msgf("unable to read syspurpose: %s", err)
		defaultSysPurpose := getDefaultSystemPurpose()
		sysPurpose = &defaultSysPurpose
		log.Info().Msgf("using default syspurpose values")
	}

	installedProducts := rhsmClient.getInstalledProducts()

	contentTags := createListOfContentTags(installedProducts)

	// Create body for the register request
	headers["Content-type"] = "application/json"
	registerData := RegisterData{
		Type:              "system",
		Name:              hostname,
		Facts:             &facts,
		Role:              sysPurpose.Role,
		Usage:             sysPurpose.Usage,
		ServiceLevel:      sysPurpose.ServiceLevelAgreement,
		InstalledProducts: installedProducts,
		ContentTags:       contentTags,
	}
	body, err := json.Marshal(registerData)
	if err != nil {
		return nil, err
	}

	res, err := rhsmClient.NoAuthConnection.request(
		http.MethodPost,
		"consumers",
		query,
		"",
		&headers,
		&body,
		clientInfo)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to register, status code: %d (unable to read response body)",
				res.StatusCode,
			)
		}
		var regError RegisterError
		err = json.Unmarshal(resBody, &regError)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to register, status code: %d (unable to parse response body)",
				res.StatusCode,
			)
		}
		return nil, fmt.Errorf(
			"unable to register, status code: %d, error message: %s",
			res.StatusCode,
			regError.DisplayMessage,
		)
	}

	resBody, err := getResponseBody(res)
	if err != nil {
		return nil, err
	}

	consumerData := ConsumerData{}
	err = json.Unmarshal([]byte(*resBody), &consumerData)
	if err != nil {
		return nil, fmt.Errorf("unable to parse consumer object: %s, %s", *resBody, err)
	}

	err = writeConsumerCert(rhsmClient.consumerCertPath(), &consumerData.IdCert.Cert)
	if err != nil {
		return nil, err
	}

	err = writeConsumerKey(rhsmClient.consumerKeyPath(), &consumerData.IdCert.Key)
	if err != nil {
		return nil, err
	}

	log.Info().Msg("System registered")

	certFilePath := filepath.Join(rhsmClient.RHSMConf.RHSM.ConsumerCertDir, "cert.pem")
	keyFilePath := filepath.Join(rhsmClient.RHSMConf.RHSM.ConsumerCertDir, "key.pem")
	err = rhsmClient.createCertAuthConnection(
		&rhsmClient.RHSMConf.Server.Hostname,
		&rhsmClient.RHSMConf.Server.Port,
		&rhsmClient.RHSMConf.Server.Prefix,
		&certFilePath,
		&keyFilePath,
	)
	if err != nil {
		return nil, err
	}

	// TODO: send signal to virt-who

	// When we are in SCA mode, then we can get entitlement cert(s) and generate content
	if consumerData.Owner.ContentAccessMode == "org_environment" {
		// When at least one activation is used for registration, then
		// try to get content override during registration, because
		// there can be some content override associated with one of
		// activation keys
		getContentOverrides := isActivationKeyUsed
		err = rhsmClient.enableContent(getContentOverrides, clientInfo)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("organization %s does not use Simple Content Access Mode",
			consumerData.Owner.DisplayName)
	}

	return &consumerData, nil
}

// RegisterOrgActivationKeys tries to register system using organization id and activation keys
func (rhsmClient *RHSMClient) RegisterOrgActivationKeys(
	org *string,
	activationKeys []string,
	clientInfo *ClientInfo,
) (*ConsumerData, error) {
	var headers = make(map[string]string)

	if clientInfo == nil {
		clientInfo = &ClientInfo{"", "", ""}
	}
	clientInfo.xCorrelationId = createCorrelationId()

	var strActivationKeys string
	for idx, activationKey := range activationKeys {
		strActivationKeys += activationKey
		if idx < len(activationKeys)-1 {
			strActivationKeys += ","
		}
	}

	query := "owner=" + *org + "&activation_keys=" + strActivationKeys

	return rhsmClient.registerSystem(headers, query, clientInfo)
}

// RegisterUsernamePasswordOrg tries to register system using organization id, username and password
func (rhsmClient *RHSMClient) RegisterUsernamePasswordOrg(
	username *string,
	password *string,
	org *string,
	clientInfo *ClientInfo,
) (*ConsumerData, error) {
	var headers = make(map[string]string)

	headers["username"] = *username
	headers["password"] = *password

	if clientInfo == nil {
		clientInfo = &ClientInfo{"", "", ""}
	}
	clientInfo.xCorrelationId = createCorrelationId()

	var query string
	if *org != "" {
		query = "owner=" + *org
	} else {
		query = ""
	}

	return rhsmClient.registerSystem(headers, query, clientInfo)
}

// createProductMap tries to create map of entitlement certificates
func createProductMap(entCertKeys []EntitlementCertificateKeyJSON) map[int64][]EngineeringProduct {
	var engineeringProducts = make(map[int64][]EngineeringProduct)
	for _, entCertKey := range entCertKeys {
		serial := entCertKey.Serial.Serial
		certContent := &entCertKey.Cert
		products, err := getContentFromEntCert(certContent)
		if err != nil {
			log.Warn().Msgf("unable to get content from entitlement certificate: %s", err)
			continue
		}
		engineeringProducts[serial] = products
	}
	return engineeringProducts
}

// EntCertKeysResult is structure used in enableContent function
type EntCertKeysResult struct {
	entCertKeyJSONList []EntitlementCertificateKeyJSON
	err                error
}

// ContentOverridesResult is structure used in enableContent function
type ContentOverridesResult struct {
	contentOverridesList []ContentOverride
	err                  error
}

// enableContent tries to get SCA entitlement certificate and generate redhat.repo from these
// certificates. Note: candlepin returns only one SCA certificate, but it returns it
// in the list. Thus, in theory more certificates could be returned.
func (rhsmClient *RHSMClient) enableContent(getContentOverrides bool, info *ClientInfo) error {
	var waitGroup sync.WaitGroup

	// Try to get SCA entitlement certificate and key asynchronously
	entCertKeysChan := make(chan EntCertKeysResult, 1)
	// Add following go routine to wait group
	waitGroup.Add(1)
	go func(wg *sync.WaitGroup, result chan EntCertKeysResult) {
		defer wg.Done()
		entCertKeys, err := rhsmClient.getSCAEntitlementCertificates(info)
		entCertKeyResult := EntCertKeysResult{entCertKeys, err}
		result <- entCertKeyResult
	}(&waitGroup, entCertKeysChan)

	// If getting content overrides is request, then try to do it asynchronously
	// because entitlement certificate is not necessary for that
	contentOverridesChan := make(chan ContentOverridesResult, 1)
	if getContentOverrides {
		// Add following go routine to wait group
		waitGroup.Add(1)
		go func(wg *sync.WaitGroup, result chan ContentOverridesResult) {
			defer wg.Done()
			contentOverridesList, err := rhsmClient.getContentOverrides(info)
			contentOverridesResult := ContentOverridesResult{contentOverridesList, err}
			result <- contentOverridesResult
		}(&waitGroup, contentOverridesChan)
	}

	// Wait for result of both REST API calls
	waitGroup.Wait()

	// Create empty maps of products and content overrides for corner cases
	engineeringProducts := make(map[int64][]EngineeringProduct)
	contentOverrides := make(map[string]map[string]string)

	// Try to get entitlement certs and keys from channel
	entCertKeysResult, ok := <-entCertKeysChan
	if ok {
		if entCertKeysResult.err != nil {
			return entCertKeysResult.err
		}
		// Get content from entitlement certificates
		engineeringProducts = createProductMap(entCertKeysResult.entCertKeyJSONList)
	}

	// Try to get content overrides from channel
	if getContentOverrides {
		contentOverridesResult, ok := <-contentOverridesChan
		if ok {
			if contentOverridesResult.err != nil {
				return contentOverridesResult.err
			}
			if len(contentOverridesResult.contentOverridesList) > 0 {
				contentOverrides = createMapFromContentOverrides(contentOverridesResult.contentOverridesList)
			}
		}
	}

	// Write content to redhat.repo file
	if len(engineeringProducts) > 0 {
		err := rhsmClient.writeRepoFile(engineeringProducts, contentOverrides)
		if err != nil {
			return fmt.Errorf("unable to write repo file: %s: %s",
				rhsmClient.RHSMConf.yumRepoFilePath, err)
		}
		log.Info().Msgf("%s generated", rhsmClient.RHSMConf.yumRepoFilePath)
	} else {
		log.Debug().Msgf("skipping writing repo file %s, because the list of engineering products is empty",
			rhsmClient.RHSMConf.yumRepoFilePath)
	}

	return nil
}

// getInstalledProducts tries to get all installed products. Typically, from directories:
// /etc/pki/product and /etc/pki/product-default
func (rhsmClient *RHSMClient) getInstalledProducts() []InstalledProduct {
	installedProducts, err := rhsmClient.readAllProductCertificates()
	if err != nil {
		log.Warn().Msgf("unable to read any director with product certificates: %s\n", err)
	}
	return installedProducts
}

// createListOfContentTags creates list of unique tags from the list of installed products
func createListOfContentTags(installedProducts []InstalledProduct) []string {
	var contentTags []string

	// We use map, because there is nothing like set
	var contentTagsMap = make(map[string]bool)
	for _, prod := range installedProducts {
		for _, tag := range prod.providedTags {
			_, exists := contentTagsMap[tag]
			if !exists {
				contentTagsMap[tag] = true
			}
		}
	}
	// Create list from the map (we care only about keys)
	for tagName := range contentTagsMap {
		contentTags = append(contentTags, tagName)
	}
	return contentTags
}

// writeConsumerCert tries to write consumer certificate. It is
// typically /etc/pki/consumer/cert.pem
func writeConsumerCert(consumerCertFilePath *string, consumerCert *string) error {
	var mode os.FileMode = 0640
	return writePemFile(consumerCertFilePath, consumerCert, &mode)
}

// writeConsumerKey tries to write consumer key. It is typically
// /etc/pki/consumer/key.pem
func writeConsumerKey(consumerKeyFilePath *string, consumerKey *string) error {
	var mode os.FileMode = 0640
	return writePemFile(consumerKeyFilePath, consumerKey, &mode)
}
