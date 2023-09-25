package rhsm2

import (
	"os"
	"path/filepath"
	"strconv"
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

// CreateRHSMClient tries to create structure holding information
func CreateRHSMClient(confFilePath *string) (*RHSMClient, error) {
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

	// When consumer key and certificate exist, then it is possible
	// to create connection using consumer cert/key for authentication
	certFilePath := filepath.Join(rhsmConf.RHSM.ConsumerCertDir, "cert.pem")
	if _, err := os.Stat(certFilePath); err == nil {
		keyFilePath := filepath.Join(rhsmConf.RHSM.ConsumerCertDir, "key.pem")
		if _, err := os.Stat(keyFilePath); err == nil {
			err = rhsmClient.createCertAuthConnection(
				&rhsmConf.Server.Hostname,
				&rhsmConf.Server.Port,
				&rhsmConf.Server.Prefix,
				&certFilePath,
				&keyFilePath,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	return rhsmClient, nil
}

func (rhsmClient *RHSMClient) consumerPEMFile(fileName string) *string {
	consumerCerDir := rhsmClient.RHSMConf.RHSM.ConsumerCertDir
	consumerCertPath := filepath.Join(consumerCerDir, fileName)
	return &consumerCertPath
}

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
