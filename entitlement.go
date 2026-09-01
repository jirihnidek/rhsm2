package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

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
func (rhsmClient *RHSMClient) getSCAEntitlementCertificates(metadata *RequestMetadata) ([]EntitlementCertificateKeyJSON, error) {
	consumerUuid, err := rhsmClient.GetConsumerUUID()

	if err != nil {
		return nil, fmt.Errorf("failed to get consumer certificate: %v", err)
	}

	var headers = make(map[string]string)

	connection, err := rhsmClient.getCertAuthConnection()
	if err != nil {
		return nil, fmt.Errorf("unable to get consumer cert auth connection: %v", err)
	}
	res, err := connection.request(
		rhsmClient.UserAgent,
		http.MethodGet,
		"consumers/"+*consumerUuid+"/certificates",
		"",
		"",
		&headers,
		nil,
		metadata)

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

type UpdatedEntitlementCertificate struct {
	ContentListing interface{} `json:"contentListing"`
	LastUpdate     string      `json:"lastUpdate"`
}

// UpdateEntitlementCertificate tries to update the installed entitlement certificate using
// GET method "/consumers/"+*consumerUuid+"/accessible_content" with If-Modified-Since header.
// When the SCA entitlement certificate has not been modified since the installation, then candlepin
// will return a 304 Not Modified response. If candlepin decide that it is necessary to update the entitlement
// certificate, then it will return a 200 OK response with the new certificate. Thus, it is necessary
// to install the new certificate and then delete the old one.
// When force is true, the If-Modified-Since header is omitted, so candlepin always returns a fresh
// certificate instead of a 304.
func (rhsmClient *RHSMClient) UpdateEntitlementCertificate(force bool, metadata *RequestMetadata) error {
	consumerUuid, err := rhsmClient.GetConsumerUUID()
	if err != nil {
		return fmt.Errorf("failed to get consumer certificate: %v", err)
	}

	installedCerts, err := rhsmClient.getInstalledEntitlementCertificateKeys()
	if err != nil {
		return fmt.Errorf("failed to get installed entitlement certificates: %v", err)
	}

	// Try to get the path of SCA entitlement certificate and key and the last modified time of the certificate.
	var lastModified time.Time
	var keyPath string
	var certPath string
	for _, certKey := range installedCerts {
		if certKey.CertPath != nil {
			fileInfo, err := os.Stat(*certKey.CertPath)
			if err != nil {
				log.Debug().Msgf("failed to get file info for %s: %v", *certKey.CertPath, err)
				continue
			}
			modTime := fileInfo.ModTime()
			if modTime.After(lastModified) {
				lastModified = modTime
			}
			keyPath = *certKey.KeyPath
			certPath = *certKey.CertPath
		}
	}

	var headers = make(map[string]string)
	if !force && !lastModified.IsZero() {
		headers["If-Modified-Since"] = lastModified.UTC().Format(time.RFC1123)
	}

	connection, err := rhsmClient.getCertAuthConnection()
	if err != nil {
		return fmt.Errorf("unable to get consumer cert auth connection: %v", err)
	}

	res, err := connection.request(
		rhsmClient.UserAgent,
		http.MethodGet,
		"consumers/"+*consumerUuid+"/accessible_content",
		"",
		"",
		&headers,
		nil,
		metadata)

	if err != nil {
		return fmt.Errorf("getting accessible content failed: %s", err)
	}

	if res.StatusCode == http.StatusNotModified {
		log.Debug().Msg("entitlement certificate is up to date")
		return nil
	}

	resBody, err := getResponseBody(res)
	if err != nil {
		return err
	}

	var updatedEntitlementCertificate UpdatedEntitlementCertificate

	if err := json.Unmarshal([]byte(*resBody), &updatedEntitlementCertificate); err != nil {
		return fmt.Errorf("failed to unmarshal accessible content response: %v", err)
	}

	var updateCerts = make(map[string][]string)

	contentListingValue := reflect.ValueOf(updatedEntitlementCertificate.ContentListing)

	if contentListingValue.Kind() == reflect.Map {
		mapKeys := contentListingValue.MapKeys()
		for _, key := range mapKeys {
			fieldName := key.String()
			mapValue := contentListingValue.MapIndex(key)

			// Handle the case where mapValue is an interface{}
			if mapValue.Kind() == reflect.Interface {
				mapValue = mapValue.Elem()
			}

			if mapValue.Kind() == reflect.Slice {
				stringSlice := make([]string, mapValue.Len())
				for j := 0; j < mapValue.Len(); j++ {
					elem := mapValue.Index(j)
					if elem.Kind() == reflect.Interface {
						elem = elem.Elem()
					}
					if elem.Kind() == reflect.String {
						stringSlice[j] = elem.String()
					} else {
						stringSlice[j] = fmt.Sprintf("%v", elem.Interface())
					}
				}
				updateCerts[fieldName] = stringSlice
				break
			} else {
				return fmt.Errorf("accessible content does not contain expected fields")
			}
		}
	} else {
		return fmt.Errorf("accessible content does not contain expected fields")
	}

	if len(updateCerts) == 0 {
		return fmt.Errorf("accessible content %v does not contain any new certificates", updatedEntitlementCertificate.ContentListing)
	}

	// Get the first certificate from the map of certificates IDs
	newEntCertInstalled := false
	var newCertId string
	var newCertPath string
	var foundCertId string
	var foundCertContent string
	for certId, stringSlice := range updateCerts {
		if len(stringSlice) > 0 {
			foundCertId = certId
			// Join the string slice into a single string, because each item of the slice
			// can contain one or multiple blocks of the certificate.
			foundCertContent = strings.Join(stringSlice, "")
			break
		}
	}

	if foundCertId != "" {
		certFilePath := filepath.Join(rhsmClient.RHSMConf.RHSM.EntitlementCertDir, foundCertId+".pem")
		err := writePemFile(&certFilePath, &foundCertContent, nil)
		if err != nil {
			return fmt.Errorf("unable to write entitlement certificate %s: %s", certFilePath, err)
		}
		log.Info().Msgf("wrote a new entitlement certificate %s", certFilePath)
		newEntCertInstalled = true
		newCertId = foundCertId
		newCertPath = certFilePath
	}

	// When a new entitlement certificate is installed, rename the existing key file and then
	// delete the old certificate. When the reissued certificate keeps the same serial number,
	// newCertPath is the same file that was just written above, so renaming/removing would
	// destroy the certificate that was just installed -- skip both in that case.
	if newEntCertInstalled && newCertPath != certPath {
		newKeyFilePath := filepath.Join(rhsmClient.RHSMConf.RHSM.EntitlementCertDir, newCertId+"-key.pem")
		err = os.Rename(keyPath, newKeyFilePath)
		if err != nil {
			// When it is not possible to rename the key file, remove the new certificate
			_ = os.Remove(newCertPath)
			return fmt.Errorf("unable to rename entitlement key from %s to %s: %s", keyPath, newKeyFilePath, err)
		}
		log.Info().Msgf("renamed entitlement key to %s", newKeyFilePath)
		err = os.Remove(certPath)
		if err != nil {
			log.Warn().Msgf("unable to remove old entitlement certificate %s: %s", certPath, err)
		} else {
			log.Info().Msgf("removed old entitlement certificate %s", certPath)
		}
	}

	return nil
}
