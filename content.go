package rhsm2

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"gopkg.in/ini.v1"
	"io"
	"net/url"
	"strconv"
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
func (rhsmClient *RHSMClient) writeRepoFile(filePath string, productsMap map[int64][]EngineeringProduct) error {
	file := ini.Empty()

	ini.PrettyFormat = false

	for serial, products := range productsMap {
		for _, product := range products {
			for _, content := range product.Content {
				section, err := file.NewSection(content.Name)
				if err != nil {
					return fmt.Errorf("unable to add section: %s: %s", content.Name, err)
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
			}
		}
	}

	err := file.SaveTo(filePath)
	if err != nil {
		return fmt.Errorf("unable to write to %s: %s", filePath, err)
	}
	return nil
}

// getContentFromEntCert tries to get content definition from content of entitlement certificate
func getContentFromEntCert(entCert *string) ([]EngineeringProduct, error) {
	data := []byte(*entCert)
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
