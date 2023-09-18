package rhsm2

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"path/filepath"
)

// Unregister tries to unregister system
func (rhsmClient *RHSMClient) Unregister() error {
	consumerKeyFile := rhsmClient.consumerKeyPath()

	uuid, err := rhsmClient.GetConsumerUUID(nil)

	if err != nil {
		return err
	}

	res, err := rhsmClient.ConsumerCertAuthConnection.request(
		http.MethodDelete,
		"consumers/"+*uuid,
		"",
		"",
		nil,
		nil)
	if err != nil {
		return fmt.Errorf("unable to unregister system: %s", err)
	}

	// TODO: handle unusual state in better way
	if res.Status != "204" {
		log.Error().Msgf("system unregistered, status code: %d", res.StatusCode)
	}

	err = os.Remove(*rhsmClient.consumerCertPath())
	if err != nil {
		return fmt.Errorf("unable to remove consumer certificate: %s", err)
	}

	err = os.Remove(*consumerKeyFile)
	if err != nil {
		return fmt.Errorf("unable to remove consumer key: %s", err)
	}

	// Note: Any of following error is not critical, because system is technically
	//       unregistered and any other error should be only logged

	// Remove entitlement certificate(s) and keys
	entCertDir := &rhsmClient.RHSMConf.RHSM.EntitlementCertDir
	pemFiles, err := os.ReadDir(*entCertDir)
	if err != nil {
		log.Error().Msgf("unable to read directory %s with entitlement certs/keys: %s", *entCertDir, err)
	}

	for _, pemFile := range pemFiles {
		pemFilePath := filepath.Join(*entCertDir, pemFile.Name())
		err = os.Remove(pemFilePath)
		if err != nil {
			log.Error().Msgf("unable to remove %s: %s", pemFilePath, err)
		}
	}

	// Remove redhat.repo file
	err = os.Remove(DefaultRepoFilePath)
	if err != nil {
		log.Error().Msgf("unable to remove %s: %s", DefaultRepoFilePath, err)
	}

	return nil
}
