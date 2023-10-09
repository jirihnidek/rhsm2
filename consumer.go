package rhsm2

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

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
