package rhsm2

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const DefaultRepoFilePath = "/etc/yum.repos.d/redhat.repo"

// EngineeringProduct is structure containing information about one engineering product.
// This structure is unmarshalled from entitlement certificate
type EngineeringProduct struct {
	Id            string        `json:"id"`
	Name          string        `json:"name"`
	Version       string        `json:"version"`
	Architectures []interface{} `json:"architectures"`
	Content       []struct {
		Id             string   `json:"id"`
		Type           string   `json:"type"`
		Name           string   `json:"name" ini:"name"`
		Label          string   `json:"label"`
		Vendor         string   `json:"vendor"`
		Path           string   `json:"path"`
		Enabled        bool     `json:"enabled,omitempty"`
		Arches         []string `json:"arches"`
		GpgUrl         string   `json:"gpg_url,omitempty"`
		MetadataExpire int      `json:"metadata_expire,omitempty" ini:"metadata_expire,omitempty"`
		RequiredTags   []string `json:"required_tags,omitempty"`
	} `json:"content"`
}

// EntitlementContentJSON is structure containing information about content (decoded from entitlement certificate)
type EntitlementContentJSON struct {
	Consumer     string `json:"consumer"`
	Subscription struct {
		Sku  string `json:"sku"`
		Name string `json:"name"`
	} `json:"subscription"`
	Order struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"order"`
	Products []EngineeringProduct `json:"products"`
	Pool     struct {
	} `json:"pool"`
}

// writeRepoFile tries to write map of products to repo file
func (rhsmClient *RHSMClient) writeRepoFile(
	productsMap map[int64][]EngineeringProduct,
	contentOverrides map[string]map[string]string,
) error {
	file := ini.Empty()

	ini.PrettyFormat = false

	for serial, products := range productsMap {
		for _, product := range products {
			for _, content := range product.Content {
				// Identifier of the section. Something like [rhel-9-for-x86_64-baseos-rpms]
				section, err := file.NewSection(content.Id)
				if err != nil {
					return fmt.Errorf("unable to add section: %s: %s", content.Id, err)
				}

				// name
				_, _ = section.NewKey("name", content.Name)

				// baseurl
				baseURL, err := url.Parse(rhsmClient.RHSMConf.RHSM.BaseURL + content.Path)
				if err != nil {
					return fmt.Errorf("unable to create parse base URL: %s", err)
				}
				_, _ = section.NewKey("baseurl", baseURL.String())

				// enabled
				var enabled string
				if content.Enabled {
					enabled = "1"
				} else {
					enabled = "0"
				}
				_, _ = section.NewKey("enabled", enabled)
				_, _ = section.NewKey("enabled_metadata", enabled)

				// gpg
				if len(content.GpgUrl) > 0 {
					_, _ = section.NewKey("gpgcheck", "1")
					_, _ = section.NewKey("gpgkey", content.GpgUrl)
				} else {
					_, _ = section.NewKey("gpgcheck", "0")
				}

				// ssl
				_, _ = section.NewKey("sslverify", "1")
				_, _ = section.NewKey("sslcacert", rhsmClient.RHSMConf.RHSM.RepoCACertificate)
				keyPath := rhsmClient.entKeyPath(serial)
				certPath := rhsmClient.entCertPath(serial)
				_, _ = section.NewKey("sslclientkey", *keyPath)
				_, _ = section.NewKey("sslclientcert", *certPath)

				// metadata
				_, _ = section.NewKey("metadata_expire", strconv.Itoa(content.MetadataExpire))

				// arches
				if len(content.Arches) > 0 {
					var arches string
					for _, arch := range content.Arches {
						arches = arches + arch
					}
					_, _ = section.NewKey("arches", arches)
				}

				// sslverifystatus
				_, _ = section.NewKey("sslverifystatus", "1")

				// Apply content overrides
				if override, exists := contentOverrides[content.Name]; exists {
					for key, value := range override {
						_, err := section.NewKey(key, value)
						if err != nil {
							log.Error().Msgf("unable to apply content override for repository: %s", content.Name)
							log.Error().Msgf("unable to add value: %v for key: %s: %s", value, key, err)
						}
					}
				}
			}
		}
	}

	err := file.SaveTo(rhsmClient.RHSMConf.yumRepoFilePath)
	if err != nil {
		return fmt.Errorf("unable to write to %s: %s",
			rhsmClient.RHSMConf.yumRepoFilePath, err)
	}
	return nil
}

// generateRepoFileFromInstalledEntitlementCerts tries to generate redhat.repo file
// from installed entitlement certificate(s) and content overrides
func (rhsmClient *RHSMClient) generateRepoFileFromInstalledEntitlementCerts(
	contentOverrides map[string]map[string]string,
) error {
	entCertDirPath := rhsmClient.RHSMConf.RHSM.EntitlementCertDir
	entCertsFilePaths, err := os.ReadDir(entCertDirPath)
	if err != nil {
		return fmt.Errorf("unable to read content of %s: %s", entCertDirPath, err)
	}

	var engineeringProductsMap = make(map[int64][]EngineeringProduct)

	for _, file := range entCertsFilePaths {
		fileName := file.Name()
		filePath := filepath.Join(entCertDirPath, fileName)
		// Skip the file if it is key file
		if strings.HasSuffix(filePath, "-key.pem") {
			continue
		}
		// Other pem file should be entitlement cert file
		if strings.HasSuffix(filePath, ".pem") {
			engineeringProduct, err := getContentFromEntCertFile(&filePath)
			if err != nil {
				log.Debug().Msgf("skipping reading content from %s, %s", filePath, err)
			}
			serialNumberStr := strings.TrimSuffix(fileName, ".pem")
			serialNumber, err := strconv.ParseInt(serialNumberStr, 10, 64)
			if err != nil {
				log.Debug().Msgf("unable to convert %s to int: %s", fileName, err)
			}
			engineeringProductsMap[serialNumber] = engineeringProduct
		}
	}

	return rhsmClient.writeRepoFile(engineeringProductsMap, contentOverrides)
}

// getContentFromEntCertFile tries to load entitlement certificate from given file
func getContentFromEntCertFile(filePath *string) ([]EngineeringProduct, error) {
	entCertFileContent, err := os.ReadFile(*filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read entitlement certificate: %s, %s", *filePath, err)
	}

	content := string(entCertFileContent)
	engineeringProducts, err := getContentFromEntCert(&content)
	if err != nil {
		return nil, err
	}

	return engineeringProducts, nil
}

// getContentFromEntCert tries to get content definition from content of entitlement certificate
func getContentFromEntCert(entCertContent *string) ([]EngineeringProduct, error) {
	data := []byte(*entCertContent)
	blockEntitlementDataFound := false
	var engineeringProducts []EngineeringProduct

	// Go through the entitlement certificate and try to get block "ENTITLEMENT DATA"
	for data != nil {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		} else {
			// Content is saved in this block type
			if block.Type == "ENTITLEMENT DATA" {
				blockEntitlementDataFound = true
				// The "block.Bytes" is already base64 decoded. We can try to un-compress.
				b := bytes.NewReader(block.Bytes)
				zReader, err := zlib.NewReader(b)
				if err != nil {
					return nil, fmt.Errorf("unable to create new zlib readed for ENTITLEMENT DATA: %s", err)
				}
				p, err := io.ReadAll(zReader)
				if err != nil {
					return nil, fmt.Errorf("unable to uncompress ENTITLEMENT DATA: %s", err)
				}
				_ = zReader.Close()

				// Try to unmarshal string to the list of repo definitions
				var entitlementContents EntitlementContentJSON
				err = json.Unmarshal(p, &entitlementContents)
				if err != nil {
					return nil, err
				}

				engineeringProducts = append(engineeringProducts, entitlementContents.Products...)
			}
		}
		data = rest
	}

	if !blockEntitlementDataFound {
		return nil, fmt.Errorf("unable to get content, because no block \"ENTITLEMENT DATA\" found")
	}

	return engineeringProducts, nil
}
