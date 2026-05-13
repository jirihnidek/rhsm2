package rhsm2

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/henvic/httpretty"
	"github.com/jeandeaual/go-locale"
	"github.com/jirihnidek/rhsm2/constants"
	"github.com/rs/zerolog/log"
)

// AuthType is type used for specifying authentication type of connection
type AuthType int

// Constants of authentication types
const (
	// NoAuth does not require any authentication. It can use base64 encoded
	// username:password in HTTP header for authentication of client
	NoAuth AuthType = iota

	// ConsumerCertAuth uses consumer certificate for client authentication
	ConsumerCertAuth

	// EntitlementCertAuth uses entitlement certificate for client authentication
	EntitlementCertAuth
)

// RHSMConnection contains information about connection to server
// This is typically connection to candlepin server, but it can be also
// connection to CDN, when we try to get information about release
type RHSMConnection struct {
	AuthType       AuthType
	Client         *http.Client
	ServerHostname *string
	ServerPort     *string
	ServerPrefix   *string
}

// createCorrelationId
func createCorrelationId() string {
	return uuid.New().String()
}

// UserAgentInfo holds information about current client connected
// to candlepin server
type UserAgentInfo struct {
	BaseString string
	Command    string
}

// ClientInfo holds information about current client triggering
// given HTTP request. Information in this structure could not
// be stored in rhsmClient, because RHSM client could be also
// rhsm2.service providing D-Bus API and each D-Bus client
// communicating over D-Bus can have different preferences
// (e.g. locale).
type ClientInfo struct {
	Locale         string
	DBusSender     string
	xCorrelationId string
}

var (
	// UserAgent is the HTTP header used in each HTTP request
	UserAgent = UserAgentInfo{
		"RHSM/" + constants.ApiVersion,
		"",
	}
)

// SetUserAgentCmd set command of UserAgent
func SetUserAgentCmd(userAgentCmd string) {
	UserAgent.Command = userAgentCmd
}

// String returns textual representation of UserAgent
func (userAgent UserAgentInfo) String() string {
	if userAgent.Command != "" {
		return userAgent.BaseString + " (cmd=" + userAgent.Command + ")"
	}
	return userAgent.BaseString
}

// request tries to call HTTP request to candlepin server
func (connection *RHSMConnection) request(
	method string,
	path string,
	query string,
	fragment string,
	headers *map[string]string,
	body *[]byte,
	clientInfo *ClientInfo,
) (*http.Response, error) {

	requestURL := url.URL{
		Scheme:   "https",
		Host:     *connection.ServerHostname + ":" + *connection.ServerPort,
		Path:     *connection.ServerPrefix + "/" + path,
		RawQuery: query,
		Fragment: fragment,
	}

	requestUrl := requestURL.String()

	var buffer *bytes.Buffer
	if body != nil {
		buffer = bytes.NewBuffer(*body)
	} else {
		buffer = &bytes.Buffer{}
	}

	req, err := http.NewRequest(method, requestUrl, buffer)
	if err != nil {
		return nil, fmt.Errorf("unable to create http request %s: %s", method, err)
	}

	// When connection without cert/key auth is used, then it is possible to
	// use basic authentication username/password
	if connection.AuthType == NoAuth && headers != nil {
		// Set username and password for basic authentication
		username, usernameExist := (*headers)["username"]
		password, passwordExist := (*headers)["password"]
		if usernameExist && passwordExist {
			req.SetBasicAuth(username, password)
		}
		// Remove username and password from map of headers
		if usernameExist {
			delete(*headers, "username")
		}
		if passwordExist {
			delete(*headers, "password")
		}
	}

	// Always add HTTP header UserAgent
	if clientInfo != nil && clientInfo.DBusSender != "" {
		req.Header.Add(
			"User-Agent",
			fmt.Sprintf("%s (dbus_sender=%s)", UserAgent.String(), clientInfo.DBusSender),
		)
	} else {
		req.Header.Add("User-Agent", UserAgent.String())
	}

	// Always add HTTP header Accept-Language
	if clientInfo != nil && clientInfo.Locale != "" {
		req.Header.Add("Accept-Language", clientInfo.Locale)
	} else {
		userLocale, err := locale.GetLocale()
		if err != nil || userLocale == "" {
			req.Header.Add("Accept-Language", "c")
		} else {
			req.Header.Add("Accept-Language", userLocale)
		}
	}

	// Try to add HTTP header X-Correlation-Id
	if clientInfo != nil && clientInfo.xCorrelationId != "" {
		req.Header.Add("X-Correlation-Id", clientInfo.xCorrelationId)
	}

	// If "Accept" header is not specified, then request JSON in response
	var acceptExists = false
	if headers != nil {
		_, acceptExists = (*headers)["Accept"]
	}
	if !acceptExists {
		req.Header.Add("Accept", "application/json")
	}

	if headers != nil {
		for key, value := range *headers {
			req.Header.Add(key, value)
		}
	}

	res, err := connection.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request %s: %s", method, err)
	}

	return res, nil
}

// createHTTPsClient tries to create instance of http.Client and configure to use TLS.
// When certFile and keyFile are not nil, then these two file will be used for client
// authentication.
func (rhsmClient *RHSMClient) createHTTPsClient(certFile *string, keyFile *string) (*http.Client, error) {
	insecure := rhsmClient.RHSMConf.Server.Insecure
	caDir := rhsmClient.RHSMConf.RHSM.CACertDir

	// First try to read directory with CA PEM files
	caFiles, err := os.ReadDir(caDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read PEM files from CA directory: %w", err)
	}
	// Create empty pool of CA cert, because we do not want to load CA certs installed
	// in the system
	caCertPool := x509.NewCertPool()
	// Try to add all PEM files from this directory to the pool
	for _, file := range caFiles {
		caFilePath := filepath.Join(caDir, file.Name())
		data, err := os.ReadFile(caFilePath)
		if err != nil {
			return nil, fmt.Errorf("cannot read CA PEM file %s : %w", caFilePath, err)
		}
		ok := caCertPool.AppendCertsFromPEM(data)
		if !ok {
			return nil, fmt.Errorf("cannot append CA PEM file: %s", caFilePath)
		}
	}

	var tlsConfig *tls.Config
	// When cert and key file are not null, then try to configure using cert and key
	// files for client authentication
	if certFile != nil && keyFile != nil {
		// Try to load client certificate and key
		keyPair, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			return nil, fmt.Errorf("unable to load client certificate and key: %s", err)
		}
		tlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{keyPair},
			RootCAs:            caCertPool,
			InsecureSkipVerify: insecure,
		}
	} else {
		tlsConfig = &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: insecure,
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig

	if rhsmClient.RHSMConf.Server.ProxyHostname != "" {
		log.Debug().Msgf("using proxy configuration from rhsm.conf")
		proxyHostname := rhsmClient.RHSMConf.Server.ProxyHostname
		proxyPort := rhsmClient.RHSMConf.Server.ProxyPort
		proxyUser := rhsmClient.RHSMConf.Server.ProxyUser
		proxyPassword := rhsmClient.RHSMConf.Server.ProxyPassword

		// Detect if proxyHostname is raw IPv6 address and assembly
		// proxyHostnamePort, because url.URL does not have Port field
		var proxyHostnamePort string
		ipAddress := net.ParseIP(proxyHostname)
		if ipAddress != nil && len(ipAddress) == net.IPv6len {
			proxyHostnamePort = "[" + proxyHostname + "]:" + proxyPort
		} else {
			proxyHostnamePort = proxyHostname + ":" + proxyPort
		}

		var proxyURL url.URL
		if proxyUser != "" || proxyPassword != "" {
			proxyURL = url.URL{
				Scheme: "https",
				Host:   proxyHostnamePort,
				User:   url.UserPassword(proxyUser, proxyPassword),
			}
		} else {
			proxyURL = url.URL{
				Scheme: "https",
				Host:   proxyHostnamePort,
			}
		}

		log.Debug().Msgf("using proxy: %s", proxyURL.String())

		transport.Proxy = http.ProxyURL(&proxyURL)
	}

	var client *http.Client

	// If env variables are set, then client will pretty print some
	// debug information to stdout using
	printReq := os.Getenv("SUBMAN_DEBUG_PRINT_REQUEST")
	printRes := os.Getenv("SUBMAN_DEBUG_PRINT_RESPONSE")
	if printRes != "" || printReq != "" {
		logger := &httpretty.Logger{
			Time:            true,
			TLS:             true,
			RequestHeader:   printReq != "",
			RequestBody:     printReq != "",
			ResponseHeader:  printRes != "",
			ResponseBody:    printRes != "",
			Colors:          true,
			Formatters:      []httpretty.Formatter{&httpretty.JSONFormatter{}},
			MaxRequestBody:  1024 * 1024,
			MaxResponseBody: 1024 * 1024,
			SkipSanitize:    true,
		}
		roundTripper := logger.RoundTripper(transport)
		client = &http.Client{Transport: roundTripper}
	} else {
		client = &http.Client{Transport: transport}
	}

	return client, nil
}

// getNoAuthConnection establishes or retrieves a no-authentication connection to the RHSM server.
func (rhsmClient *RHSMClient) getNoAuthConnection() (*RHSMConnection, error) {
	if rhsmClient.noAuthConnection != nil {
		return rhsmClient.noAuthConnection, nil
	}

	hostname := &rhsmClient.RHSMConf.Server.Hostname
	port := &rhsmClient.RHSMConf.Server.Port
	prefix := &rhsmClient.RHSMConf.Server.Prefix

	err := rhsmClient.createNoAuthConnection(hostname, port, prefix)
	if err != nil {
		return nil, fmt.Errorf("unable to create no-auth connection: %v", err)
	}

	return rhsmClient.noAuthConnection, nil
}

// createNoAuthConnection tries to create connection not using any cert authentication of client
func (rhsmClient *RHSMClient) createNoAuthConnection(
	hostname *string,
	port *string,
	prefix *string,
) error {
	client, err := rhsmClient.createHTTPsClient(nil, nil)

	if err != nil {
		return fmt.Errorf("unable to create no-auth connection: %v", err)
	}

	rhsmClient.noAuthConnection = &RHSMConnection{
		AuthType:       NoAuth,
		Client:         client,
		ServerHostname: hostname,
		ServerPort:     port,
		ServerPrefix:   prefix,
	}

	return nil
}

// getCertAuthConnection tries to get the current consumer cert auth connection. When the connection
// does not exist, it creates a new one using the provided configuration.
func (rhsmClient *RHSMClient) getCertAuthConnection() (*RHSMConnection, error) {
	if rhsmClient.consumerCertAuthConnection != nil {
		return rhsmClient.consumerCertAuthConnection, nil
	}

	hostname := &rhsmClient.RHSMConf.Server.Hostname
	port := &rhsmClient.RHSMConf.Server.Port
	prefix := &rhsmClient.RHSMConf.Server.Prefix
	consumerCertFilePath := filepath.Join(rhsmClient.RHSMConf.RHSM.ConsumerCertDir, "cert.pem")
	if _, err := os.Stat(consumerCertFilePath); err != nil {
		return nil, fmt.Errorf("consumer certificate %s does not exists", consumerCertFilePath)
	}
	consumerKeyFilePath := filepath.Join(rhsmClient.RHSMConf.RHSM.ConsumerCertDir, "key.pem")
	if _, err := os.Stat(consumerKeyFilePath); err != nil {
		return nil, fmt.Errorf("consumer key %s does not exists", consumerKeyFilePath)
	}
	err := rhsmClient.createCertAuthConnection(hostname, port, prefix, &consumerCertFilePath, &consumerKeyFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create consumer cert auth connection: %v", err)
	}

	return rhsmClient.consumerCertAuthConnection, nil
}

// createCertAuthConnection tries to create connection using some cert for authentication.
// Consumer cert/key is used for auth against candlepin server and entitlement
// cert/key is used
func (rhsmClient *RHSMClient) createCertAuthConnection(
	hostname *string,
	port *string,
	prefix *string,
	certFilePath *string,
	keyFilePath *string,
) error {
	client, err := rhsmClient.createHTTPsClient(certFilePath, keyFilePath)

	if err != nil {
		return fmt.Errorf("unable to create consumer cert auth connection: %v", err)
	}

	rhsmClient.consumerCertAuthConnection = &RHSMConnection{
		AuthType:       ConsumerCertAuth,
		Client:         client,
		ServerHostname: hostname,
		ServerPort:     port,
		ServerPrefix:   prefix,
	}

	return nil
}

// getEntitlementCertAuthConnection returns the entitlement cert auth connection if it exists
// and it tries to create it if it doesn't exist
func (rhsmClient *RHSMClient) getEntitlementCertAuthConnection() (*RHSMConnection, error) {
	if rhsmClient.entitlementCertAuthConnection != nil {
		return rhsmClient.entitlementCertAuthConnection, nil
	}

	cdnHost, cdnPort, cdnPath, err := parseBaseURL(rhsmClient.RHSMConf.RHSM.BaseURL)
	if err != nil {
		return nil, err
	}

	certKeys, err := rhsmClient.getInstalledEntitlementCertificateKeys()
	if err != nil {
		return nil, err
	}

	var certPath, keyPath *string
	for _, certKey := range certKeys {
		if certKey.CertPath == nil {
			log.Debug().Msgf("cert path is nil")
			continue
		}
		if certKey.KeyPath == nil {
			log.Debug().Msgf("key path is nil")
			continue
		}
		if _, err := os.Stat(*certKey.KeyPath); err != nil {
			log.Debug().Msgf("key path %s does not exist: %s", *certKey.KeyPath, err)
			continue
		}
		if _, err := os.Stat(*certKey.CertPath); err != nil {
			log.Debug().Msgf("cert path %s does not exist: %s", *certKey.CertPath, err)
			continue
		}
		if certKey.CertPath != nil && certKey.KeyPath != nil {
			certPath = certKey.CertPath
			keyPath = certKey.KeyPath
			break
		}
	}

	if keyPath == nil || certPath == nil {
		return nil, fmt.Errorf("no entitlement certificate and key found")
	}

	err = rhsmClient.createEntitlementCertAuthConnection(
		&cdnHost,
		&cdnPort,
		&cdnPath,
		certPath,
		keyPath,
	)
	if err != nil {
		return nil, fmt.Errorf("entitlement cert auth connection not initialized: %s", err)
	}

	return rhsmClient.entitlementCertAuthConnection, nil
}

// createEntitlementCertAuthConnection tries to create a connection using entitlement cert/key
// for authentication. It is typically used when we want to communicate with CDN.
// E.g. when we want to get information about release.
func (rhsmClient *RHSMClient) createEntitlementCertAuthConnection(
	hostname *string,
	port *string,
	prefix *string,
	certFilePath *string,
	keyFilePath *string,
) error {
	client, err := rhsmClient.createHTTPsClient(certFilePath, keyFilePath)

	if err != nil {
		return fmt.Errorf("unable to create entitlement cert auth connection: %v", err)
	}

	rhsmClient.entitlementCertAuthConnection = &RHSMConnection{
		AuthType:       EntitlementCertAuth,
		Client:         client,
		ServerHostname: hostname,
		ServerPort:     port,
		ServerPrefix:   prefix,
	}

	return nil
}

// getResponseBody tries to get response body
func getResponseBody(response *http.Response) (*string, error) {
	resBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error: reading response body: %s", err)
	}

	retBody := string(resBody[:])

	return &retBody, nil
}
