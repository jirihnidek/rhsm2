package rhsm2

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// RHSMClient contains information about client. It can hold up to 3 different
// type of connections, but usually it is necessary to use only ConsumerCertAuthConnection.
// The NoAuthConnection is used only during registration process, when no consumer
// certificate/key is installed. Note: we do not create special connection for
// "Base Auth", because it is actually NoAuthConnection with special HTTP header.
// EntitlementCertAuthConnection could be used for communication with CDN.
type RHSMClient struct {
	RHSMConf                      *RHSMConf
	NoAuthConnection              *RHSMConnection
	ConsumerCertAuthConnection    *RHSMConnection
	EntitlementCertAuthConnection *RHSMConnection
}

var singletonRhsmClient *RHSMClient
var once sync.Once

// GetRHSMClient tries to return instance of RHSMClient. If the instance
// already exist, then existing instance is returned. The confFilePath
// is used only in the first call of the function. It is just ignored
// in any other next call.
func GetRHSMClient(confFilePath *string) (*RHSMClient, error) {
	var err error
	once.Do(func() {
		singletonRhsmClient, err = createRHSMClient(confFilePath)
	})
	if err != nil {
		return nil, err
	}
	return singletonRhsmClient, nil
}

// createRHSMClient tries to create structure holding information about RHSM client
func createRHSMClient(confFilePath *string) (*RHSMClient, error) {
	var err error
	var rhsmConf *RHSMConf

	// Try to load configuration file
	if confFilePath != nil {
		rhsmConf, err = LoadRHSMConf(*confFilePath)
	} else {
		rhsmConf, err = LoadRHSMConf(DefaultRHSMConfFilePath)
	}
	if err != nil {
		return nil, err
	}

	rhsmClient := &RHSMClient{
		RHSMConf:                      rhsmConf,
		NoAuthConnection:              nil,
		ConsumerCertAuthConnection:    nil,
		EntitlementCertAuthConnection: nil,
	}

	// Try to create connection without authentication
	// Note: It doesn't do any TCP/TLS handshake ATM
	err = rhsmClient.createNoAuthConnection(
		&rhsmConf.Server.Hostname,
		&rhsmConf.Server.Port,
		&rhsmConf.Server.Prefix)
	if err != nil {
		return nil, err
	}

	// When the consumer key and the certificate exist, then it is possible
	// to create a connection using consumer cert/key for authentication
	consumerCertFilePath := filepath.Join(rhsmConf.RHSM.ConsumerCertDir, "cert.pem")
	if _, err := os.Stat(consumerCertFilePath); err == nil {
		consumerKeyFilePath := filepath.Join(rhsmConf.RHSM.ConsumerCertDir, "key.pem")
		if _, err := os.Stat(consumerKeyFilePath); err == nil {
			err = rhsmClient.createCertAuthConnection(
				&rhsmConf.Server.Hostname,
				&rhsmConf.Server.Port,
				&rhsmConf.Server.Prefix,
				&consumerCertFilePath,
				&consumerKeyFilePath,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	// When the entitlement key and the certificate exist, then it is possible
	// to create a connection using entitlement cert/key for authentication to CDN
	certKeys, err := rhsmClient.getInstalledEntitlementCertificateKeys()
	if err != nil {
		return nil, err
	}
	
	for _, certKey := range certKeys {
		if certKey.CertPath == nil {
			log.Debug().Msgf("cert path is nil")
			continue
		}
		if certKey.KeyPath == nil {
			log.Debug().Msgf("key path is nil")
			continue
		}
		if _, err := os.Stat(*certKey.KeyPath); err == nil {
			if _, err := os.Stat(*certKey.CertPath); err == nil {
				cdnURL, err := url.Parse(rhsmConf.RHSM.BaseURL)
				if err != nil {
					return nil, err
				}
				log.Debug().Msgf("cdnURL: %s, host: %s, port %s", cdnURL, cdnURL.Host, cdnURL.Port())
				cdnPort := cdnURL.Port()
				cdnHost := cdnURL.Host
				if cdnPort == "" {
					cdnPort = "443"
				} else {
					// Split host and port
					cdnHost = strings.Split(cdnURL.Host, ":")[0]
				}
				err = rhsmClient.createEntitlementCertAuthConnection(
					&cdnHost,
					&cdnPort,
					&cdnURL.Path,
					certKey.CertPath,
					certKey.KeyPath,
				)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return rhsmClient, nil
}

// consumerPEMFile returns a full path to a PEM file in the consumer certificate directory
// fileName: name of the PEM file to locate
func (rhsmClient *RHSMClient) consumerPEMFile(fileName string) *string {
	consumerCerDir := rhsmClient.RHSMConf.RHSM.ConsumerCertDir
	consumerCertPath := filepath.Join(consumerCerDir, fileName)
	return &consumerCertPath
}

// entitlementPEMFile returns a full path to a PEM file in the entitlement certificate directory
// fileName: name of the PEM file to locate
func (rhsmClient *RHSMClient) entitlementPEMFile(fileName string) *string {
	entCerDir := rhsmClient.RHSMConf.RHSM.EntitlementCertDir
	entCertPath := filepath.Join(entCerDir, fileName)
	return &entCertPath
}

// entCertPath tries to return path of entitlement certificate for given serial number
func (rhsmClient *RHSMClient) entCertPath(serialNum int64) *string {
	return rhsmClient.entitlementPEMFile(strconv.FormatInt(serialNum, 10) + ".pem")
}

// entKeyPath tries to return path of entitlement key for given serial number
func (rhsmClient *RHSMClient) entKeyPath(serialNum int64) *string {
	return rhsmClient.entitlementPEMFile(strconv.FormatInt(serialNum, 10) + "-key.pem")
}

// consumerCertPath tries to return path of consumer certificate
func (rhsmClient *RHSMClient) consumerCertPath() *string {
	return rhsmClient.consumerPEMFile("cert.pem")
}

// consumerCertPath tries to return path of consumer certificate
func (rhsmClient *RHSMClient) consumerKeyPath() *string {
	return rhsmClient.consumerPEMFile("key.pem")
}
