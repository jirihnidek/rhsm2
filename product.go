package rhsm2

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

const RedhatOidNamespace = "1.3.6.1.4.1.2312.9"

// InstalledProduct is product certificate installed to /etc/pki/product or
// /etc/pki/product-default. DNF plugin installs product certificates
// to /etc/pki/product and there is typically one pre-installed product
// certificate in /etc/pki/product-default, when pre-installed operating
// system is part of some product (Red Hat Enterprise Linux)
type InstalledProduct struct {
	// Following attributes are sent in the report.
	Id           string `json:"productId"`
	Name         string `json:"productName"`
	Version      string `json:"version"`
	Architecture string `json:"arch"`
	// Following attributes are not sent in the report. Thus, it is not necessary to export them
	providedTags []string
	brandType    string
	brandName    string
	filePath     string
}

// DirectoryDefaultProductCertificate is directory containing default
// product certificate. This certificate is pre-installed on the system.
// The path cannot be altered in configuration file rhsm.conf
const DirectoryDefaultProductCertificate = "/etc/pki/product-default"

// parseASN1Value tries to parse ASN.1 value of product certificate. It should contain only
// one string
func parseASN1Value(oid *string, extensionValue *[]byte) (*string, error) {
	// Value of extension is ASN.1 value, and it is necessary to unmarshal it
	var asn1Value asn1.RawValue
	_, err := asn1.Unmarshal(*extensionValue, &asn1Value)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ASN.1 value of extension: %s: %s", *oid, err)
	}

	var value string
	switch asn1Value.Tag {
	case asn1.TagUTF8String:
		value = string(asn1Value.Bytes)
	case asn1.TagOctetString:
		value = string(asn1Value.Bytes)
	default:
		return nil, fmt.Errorf("extension: %s contains unsupported tag type: %d", *oid, asn1Value.Tag)
	}

	return &value, nil
}

// parseProductCertificateExtension tries to parse one extension of Red Hat product certificate
func parseProductCertificateExtension(installedProduct *InstalledProduct, extension *pkix.Extension) error {
	oid := extension.Id.String()

	// We are interested only in extensions with "Red Hat" namespace
	// which is in this case: "1.3.6.1.4.1.2312.9"
	// Red Hat extensions uses following OID scheme in product certificates:
	// "1.3.6.1.4.1.2312.9." + "1." + <product_id> + "." + <extension_id>

	// Do not try to parse extension, when OID does not begin with Red Hat namespace
	if !strings.HasPrefix(oid, RedhatOidNamespace) {
		return nil
	}

	// Try to get <product_id> and <extension_id> form OID
	oidSuffix := strings.TrimPrefix(oid, RedhatOidNamespace+".1.")
	ids := strings.Split(oidSuffix, ".")
	var productId string
	var extensionId string
	if len(ids) == 2 {
		productId = ids[0]
		extensionId = ids[1]
	} else {
		return fmt.Errorf("OID does not contain product ID and extension ID")
	}

	// Try to get ASN.1 value from extension
	value, err := parseASN1Value(&oid, &extension.Value)
	if err != nil {
		return err
	}

	// Set product id only once
	if installedProduct.Id == "" {
		installedProduct.Id = productId
	}

	// Set product attributes according extension IDs
	switch extensionId {
	case "1":
		installedProduct.Name = *value
	case "2":
		installedProduct.Version = *value
	case "3":
		installedProduct.Architecture = *value
	case "4":
		installedProduct.providedTags = strings.Split(*value, ",")
	case "5":
		installedProduct.brandType = *value
	case "6":
		installedProduct.brandName = *value
	}

	return nil
}

// parseProductCertificateContent tries to parse content of product certificate
func parseProductCertificateContent(productCertFilePath *string, productCertContent *[]byte) (*InstalledProduct, error) {
	// Go through all blocks of product certificate and try to find block "CERTIFICATE"
	for productCertContent != nil {
		block, rest := pem.Decode(*productCertContent)
		if block == nil {
			break
		} else {
			if block.Type == "CERTIFICATE" {
				var installedProduct InstalledProduct
				certificate, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					return nil, fmt.Errorf("failed to parse (block CERTIFICATE) of PEM file: %s: %v",
						*productCertFilePath, err)
				}

				// Read all extension
				for _, extension := range certificate.Extensions {
					err = parseProductCertificateExtension(&installedProduct, &extension)
					if err != nil {
						return nil, fmt.Errorf("failed to parse extension of PEM file: %s: %v",
							*productCertFilePath, err)
					}
				}
				installedProduct.filePath = *productCertFilePath

				return &installedProduct, nil
			}
		}
		*productCertContent = rest
	}

	return nil, fmt.Errorf("PEM file: %s does not contain block CERTIFICATE", *productCertFilePath)
}

// readProductCertificate tries to parse information from extensions of product
// certificate and store it in InstalledProduct structure
func readProductCertificate(productCertFilePath *string) (*InstalledProduct, error) {
	productCertContent, err := os.ReadFile(*productCertFilePath)

	if err != nil {
		return nil, fmt.Errorf("failed to read product certificate: %v", err)
	}

	return parseProductCertificateContent(productCertFilePath, &productCertContent)
}

// readProductCertificatesInDir tries to read all product certificates from given directory
func readProductCertificatesInDir(productCertDirPath string) ([]InstalledProduct, error) {
	var productCerts []InstalledProduct

	productCertsFilePaths, err := os.ReadDir(productCertDirPath)
	if err != nil {
		return productCerts, fmt.Errorf("unable to read directory: %s with product certificates: %v",
			productCertDirPath, err)
	}
	for _, file := range productCertsFilePaths {
		filePath := filepath.Join(productCertDirPath, file.Name())
		productCert, err := readProductCertificate(&filePath)
		if err != nil {
			log.Warn().Msgf("skipping product certificate: %s: %v", filePath, err)
			continue
		}
		productCerts = append(productCerts, *productCert)
	}
	return productCerts, nil
}

// readAllProductCertificates tries to read all product certificates in given directory
func (rhsmClient *RHSMClient) readAllProductCertificates() ([]InstalledProduct, error) {
	var productCerts []InstalledProduct
	var err1, err2 error

	// Try to read installed product certificates
	productCertDirPath := rhsmClient.RHSMConf.RHSM.ProductCertDir
	productCerts, err1 = readProductCertificatesInDir(productCertDirPath)
	if err1 != nil {
		log.Warn().Msgf("%s", err1)
	}

	// Try to read preinstalled (default) product certificates
	defaultProductCertDirPath := rhsmClient.RHSMConf.RHSM.DefaultProductCertDir
	defaultProductCerts, err2 := readProductCertificatesInDir(defaultProductCertDirPath)
	if err2 != nil {
		log.Warn().Msgf("%s", err2)
	}

	if err1 != nil && err2 != nil {
		return productCerts, fmt.Errorf("unable to read any product cert: %s, %s", err1, err2)
	}

	productCerts = append(productCerts, defaultProductCerts...)

	return productCerts, nil
}
