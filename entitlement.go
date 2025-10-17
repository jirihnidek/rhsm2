package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// EntitlementCertificateKeyJSON is structure used for un-marshaling of JSON returned from candlepin server.
// JSON document includes list of this objects
type EntitlementCertificateKeyJSON struct {
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
}

type EntitlementCertificateKey struct {
	KeyPath  *string
	CertPath *string
}

// getInstalledEntitlementCertificateKeys retrieves a map of installed entitlement certificate keys and paths or an error.
func (rhsmClient *RHSMClient) getInstalledEntitlementCertificateKeys() (map[int64]EntitlementCertificateKey, error) {
	var installedCertKeys = make(map[int64]EntitlementCertificateKey)

	entCertDirPath := rhsmClient.RHSMConf.RHSM.EntitlementCertDir
	entCertsFilePaths, err := os.ReadDir(entCertDirPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read content of %s: %s", entCertDirPath, err)
	}

	// Iterate over all files in the entitlement certificate directory and try to find certificate and key files
	for _, file := range entCertsFilePaths {
		fileName := file.Name()
		filePath := filepath.Join(entCertDirPath, fileName)

		if strings.HasSuffix(filePath, "-key.pem") {
			serialNumberStr := strings.TrimSuffix(fileName, "-key.pem")
			serialNumber, err := strconv.ParseInt(serialNumberStr, 10, 64)
			if err != nil {
				log.Debug().Msgf("failed to parse serial number from file name: %s", fileName)
				continue
			}
			if entry, exist := installedCertKeys[serialNumber]; exist {
				entry.KeyPath = &filePath
				installedCertKeys[serialNumber] = entry
			} else {
				installedCertKeys[serialNumber] = EntitlementCertificateKey{
					KeyPath:  &filePath,
					CertPath: nil,
				}
			}
		} else if strings.HasSuffix(filePath, ".pem") {
			serialNumberStr := strings.TrimSuffix(fileName, ".pem")
			serialNumber, err := strconv.ParseInt(serialNumberStr, 10, 64)
			if err != nil {
				log.Debug().Msgf("failed to parse serial number from file name: %s", fileName)
				continue
			}
			if entry, exist := installedCertKeys[serialNumber]; exist {
				entry.CertPath = &filePath
				installedCertKeys[serialNumber] = entry
			} else {
				installedCertKeys[serialNumber] = EntitlementCertificateKey{
					KeyPath:  nil,
					CertPath: &filePath,
				}
			}
		}
	}

	// Remove entries without a certificate or key
	for serial, certKey := range installedCertKeys {
		if certKey.KeyPath == nil {
			log.Debug().Msgf("key is missing, removing serial: %d from the list", serial)
			delete(installedCertKeys, serial)
		}
		if certKey.CertPath == nil {
			log.Debug().Msgf("cert is missing, removing serial: %d from the list", serial)
			delete(installedCertKeys, serial)
		}
	}

	return installedCertKeys, nil
}

// getEntitlementCertificate tries to get all SCA entitlement certificate(s) from candlepin server.
// When it is possible to get entitlement certificate(s), then write these certificate(s) to file.
// Note: candlepin server returns only one SCA entitlement certificate ATM, but REST API allows to
// return more entitlement certificates.
func (rhsmClient *RHSMClient) getSCAEntitlementCertificates(clientInfo *ClientInfo) ([]EntitlementCertificateKeyJSON, error) {
	consumerUuid, err := rhsmClient.GetConsumerUUID()

	if err != nil {
		return nil, fmt.Errorf("failed to get consumer certificate: %v", err)
	}

	var headers = make(map[string]string)

	res, err := rhsmClient.ConsumerCertAuthConnection.request(
		http.MethodGet,
		"consumers/"+*consumerUuid+"/certificates",
		"",
		"",
		&headers,
		nil,
		clientInfo)

	if err != nil {
		return nil, fmt.Errorf("getting entitlement certificates failed: %s", err)
	}

	resBody, err := getResponseBody(res)
	if err != nil {
		return nil, err
	}

	// Try to get SCA entitlement certificate(s). It should be only one certificate,
	// but it is returned in the list (due to backward compatibility).
	var entCertKeys []EntitlementCertificateKeyJSON
	err = json.Unmarshal([]byte(*resBody), &entCertKeys)
	if err != nil {
		return nil, err
	}

	// When one entitlement certificate was returned, then generate redhat.repo from this
	// entitlement certificate
	l := len(entCertKeys)
	if l != 1 {
		if l == 0 {
			return nil, fmt.Errorf("no SCA entitlement certificate returned from server")
		}
		if l > 0 {
			log.Warn().Msgf("more than one SCA (%d) entitlement certificates installed", l)
		}
	}

	// Write certificate(s) and key(s) to file(s)
	for _, entCertKey := range entCertKeys {
		entCertFilePath, err := rhsmClient.writeEntitlementCert(&entCertKey.Cert, entCertKey.Serial.Serial)
		if err != nil {
			log.Error().Msgf("unable to install entitlement certificate: %s", err)
			continue
		}
		_, err = rhsmClient.writeEntitlementKey(&entCertKey.Key, entCertKey.Serial.Serial)
		if err != nil {
			log.Error().Msgf("unable to write entitlement key: %s", err)

			// When it is not possible to install key, then remove certificate file, because
			// certificate is useless without key
			err = os.Remove(*entCertFilePath)
			if err != nil {
				log.Error().Msgf("unable to remove entitlement certificate: %s", err)
			}
		}
	}

	return entCertKeys, nil
}

// writeEntitlementCert tries to write entitlement certificate. It is
// typically /etc/pki/entitlement/<serial_number>.pem
func (rhsmClient *RHSMClient) writeEntitlementCert(entCert *string, serialNum int64) (*string, error) {
	entCertFilePath := rhsmClient.entCertPath(serialNum)
	return entCertFilePath, writePemFile(entCertFilePath, entCert, nil)
}

// writeEntitlementCert tries to write entitlement certificate. It is
// typically /etc/pki/entitlement/<serial_number>-key.pem
func (rhsmClient *RHSMClient) writeEntitlementKey(entKey *string, serialNum int64) (*string, error) {
	entKeyFilePath := rhsmClient.entKeyPath(serialNum)
	return entKeyFilePath, writePemFile(entKeyFilePath, entKey, nil)
}
