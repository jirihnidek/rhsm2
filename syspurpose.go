package rhsm2

import (
	"encoding/json"
	"fmt"
	"os"
)

const DefaultSystemPurposeFilePath = "/etc/rhsm/syspurpose/syspurpose.json"

// SysPurposeJSON is structure holding system purpose attributes
type SysPurposeJSON struct {
	Role                  string `json:"role"`
	ServiceLevelAgreement string `json:"service_level_agreement"`
	Usage                 string `json:"usage"`
}

// getSystemPurpose tries to load system purpose from given file
func getSystemPurpose(filePath *string) (*SysPurposeJSON, error) {
	var sysPurpose SysPurposeJSON

	sysPurposeContent, err := os.ReadFile(*filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read system purpose file: %s, %s", *filePath, err)
	}

	err = json.Unmarshal(sysPurposeContent, &sysPurpose)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal system purpose file: %s: %s", *filePath, err)
	}

	return &sysPurpose, nil
}

// getDefaultSystemPurpose return structure with default values
func getDefaultSystemPurpose() SysPurposeJSON {
	return SysPurposeJSON{"", "", ""}
}
