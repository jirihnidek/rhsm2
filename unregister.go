package rhsm2

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UnregisterServerError is structure representing error
// returned from server
type UnregisterServerError struct {
	DisplayMessage string `json:"displayMessage"`
	RequestUuid    string `json:"requestUuid"`
	StatusCode     int
	ParsingError   error
}

// Error interface
func (unregisterServerError UnregisterServerError) Error() string {
	return unregisterServerError.DisplayMessage
}

// parseServerResponse tries to parse response from server and set corresponding fields
// in UnregisterServerError structure
func parseServerResponse(unregisterServerError *UnregisterServerError, res *http.Response) {
	unregisterServerError.StatusCode = res.StatusCode
	resBody, err := getResponseBody(res)
	if err != nil {
		unregisterServerError.ParsingError = fmt.Errorf("unable to get body from %d response", res.StatusCode)
	}
	err = json.Unmarshal([]byte(*resBody), &unregisterServerError)
	if err != nil {
		unregisterServerError.ParsingError = fmt.Errorf(
			"unable to parse JSON document returned by candlepin server")
	}
	unregisterServerError.ParsingError = nil
}

// removeInstalledFiles tries to remove all installed files. When all
// files have been removed, then nil is returned. When some files is not
// possible to remove, then log error is written, but removing of other
// is not terminated. When at least one files is not possible to remove,
// then error is returned.
func (rhsmClient *RHSMClient) removeInstalledFiles() error {
	removedAll := true

	// Remove consumer certificate and key
	log.Debug().Msgf("removing consumer certificate: %s", *rhsmClient.consumerCertPath())
	err := os.Remove(*rhsmClient.consumerCertPath())
	if err != nil {
		log.Error().Msgf("unable to remove consumer certificate: %s", err)
		removedAll = false
	}
	log.Debug().Msgf("removing consumer key: %s", *rhsmClient.consumerKeyPath())
	err = os.Remove(*rhsmClient.consumerKeyPath())
	if err != nil {
		log.Error().Msgf("unable to remove consumer key: %s", err)
		removedAll = false
	}

	// Remove entitlement certificate(s) and keys
	entCertDir := &rhsmClient.RHSMConf.RHSM.EntitlementCertDir
	entPemFiles, err := os.ReadDir(*entCertDir)
	if err != nil {
		log.Error().Msgf("unable to read directory %s with entitlement certs/keys: %s", *entCertDir, err)
		removedAll = false
	} else {
		log.Debug().Msgf("removing installed entitlement certs & keys from %s", *entCertDir)
		for _, entPemFile := range entPemFiles {
			entPemFilePath := filepath.Join(*entCertDir, entPemFile.Name())
			if strings.HasSuffix(entPemFilePath, "-key.pem") {
				log.Debug().Msgf("removing entitlement key: %s", entPemFilePath)
			} else {
				log.Debug().Msgf("removing entitlement cert: %s", entPemFilePath)
			}
			err = os.Remove(entPemFilePath)
			if err != nil {
				log.Error().Msgf("unable to remove %s: %s", entPemFilePath, err)
				removedAll = false
			}
		}
	}

	// Remove redhat.repo file
	if rhsmClient.RHSMConf.yumRepoFilePath != "" {
		err = os.Remove(rhsmClient.RHSMConf.yumRepoFilePath)
		if err != nil {
			log.Error().Msgf("unable to remove %s: %s", rhsmClient.RHSMConf.yumRepoFilePath, err)
			removedAll = false
		}
	}

	if !removedAll {
		return fmt.Errorf("unable to remove all installed files")
	}

	return nil
}

// Clean tries to clean all installed files, but do not try to
// remove consumer object from candlepin server
func (rhsmClient *RHSMClient) Clean() error {
	log.Warn().Msg("removing installed files without removing consumer from candlepin server")
	return rhsmClient.removeInstalledFiles()
}

// Unregister tries to unregister system
func (rhsmClient *RHSMClient) Unregister(clientInfo *ClientInfo) error {
	consumerUuid, err := rhsmClient.GetConsumerUUID()

	if err != nil {
		return err
	}

	var headers = make(map[string]string)

	if clientInfo == nil {
		clientInfo = &ClientInfo{"", "", ""}
	}
	clientInfo.xCorrelationId = createCorrelationId()

	res, err := rhsmClient.ConsumerCertAuthConnection.request(
		http.MethodDelete,
		"consumers/"+*consumerUuid,
		"",
		"",
		&headers,
		nil,
		clientInfo,
	)

	// When we are not able to call REST API call, then cancel registration process.
	// We should not delete installed data, because we would not be able to unregister
	// system in the future anymore, when candlepin get available again.
	if err != nil {
		return fmt.Errorf("unable to unregister system on candlepin server: %s", err)
	}

	// Server can respond with following codes
	var unregisterServerError UnregisterServerError
	switch res.StatusCode {
	case 204: // Consumer was successfully deleted from the server
		log.Info().Msgf("system successfully unregistered on server")
		// Try to remove all installed files.
		err = rhsmClient.removeInstalledFiles()
		// If it is not possible to remove any installed file, then only
		// log it as error, but do not return error from this function, because
		// system is technically unregistered at this moment
		if err != nil {
			log.Error().Msgf("%s", err)
		}
		return nil
	case 403: // Not enough permission to delete consumer on server
		// Do not remove installed files, because removing consumer was refused by server
		parseServerResponse(&unregisterServerError, res)
		log.Error().Msgf("unable to unregister: %s",
			unregisterServerError.DisplayMessage)
		return unregisterServerError
	case 410: // Consumer has been already deleted on the server
		_ = rhsmClient.removeInstalledFiles()
		parseServerResponse(&unregisterServerError, res)
		log.Warn().Msgf("already unregistered: %s", unregisterServerError.DisplayMessage)
		return unregisterServerError
	case 500: // Internal server error
		parseServerResponse(&unregisterServerError, res)
		log.Error().Msgf("unable to unregister: %s",
			unregisterServerError.DisplayMessage)
		return unregisterServerError
	default:
		log.Warn().Msgf("unknown status code %d returned during unregistering", res.StatusCode)
	}

	return nil
}
