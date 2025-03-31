package rhsm2

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetSystemPurpose tests the case, when system purpose is
// successfully read from configuration file
func TestGetSystemPurpose(t *testing.T) {
	t.Parallel()
	var syspurposeFilePath = "./testdata/etc/rhsm/syspurpose/syspurpose.json"
	syspurpose, err := getSystemPurpose(&syspurposeFilePath)
	if err != nil {
		t.Fatalf("reading of %s failed with error: %s", syspurposeFilePath, err)
	}

	expectedRole := "Red Hat Enterprise Linux Server"
	if syspurpose.Role != expectedRole {
		t.Fatalf("expected role: %s != %s", expectedRole, syspurpose.Role)
	}

	expectedUsage := "Development/Test"
	if syspurpose.Usage != expectedUsage {
		t.Fatalf("expected usage: %s != %s", expectedUsage, syspurpose.Usage)
	}

	expectedSLA := "Standard"
	if expectedSLA != syspurpose.ServiceLevelAgreement {
		t.Fatalf("expected SLA: %s != %s", expectedSLA, syspurpose.ServiceLevelAgreement)
	}
}

// TestMissingSystemPurposeFile test the case, when wrong path is provided
// or syspurpose.json file is missing
func TestMissingSystemPurposeFile(t *testing.T) {
	t.Parallel()
	var wrongFilePath = "./testdata/wrong/file/path/syspurpose.json"
	syspurpose, err := getSystemPurpose(&wrongFilePath)
	if err == nil {
		t.Fatalf("no error returned, when wrong file path: %s provided", wrongFilePath)
	}
	if syspurpose != nil {
		t.Fatalf("syspurpose object was returned despite wrong file path %s was provided", wrongFilePath)
	}
}

// TestCorruptedSystemPurposeFile test the case, when content of syspurpose.json
// is corrupted, and it is not possible to unmarshal content of JSON document
func TestCorruptedSystemPurposeFile(t *testing.T) {
	t.Parallel()
	// Create temporary file with corrupted content
	tempDirFilePath := t.TempDir()
	corruptedSyspurposeFilePath := filepath.Join(tempDirFilePath, "syspurpose.json")
	syspurposeFile, err := os.Create(corruptedSyspurposeFilePath)
	if err != nil {
		t.Fatalf("unable to create temporary %s file for testing", corruptedSyspurposeFilePath)
	}
	defer func() {
		err = syspurposeFile.Close()
		if err != nil {
			t.Fatalf("unable to close %s file for testing", corruptedSyspurposeFilePath)
		}
	}()
	corruptedContent := "[{]}foo:bar/%&@#"
	_, err = syspurposeFile.Write([]byte(corruptedContent))
	if err != nil {
		t.Fatalf("unable to write testing content to tempory testing %s file",
			corruptedSyspurposeFilePath)
	}

	syspurpose, err := getSystemPurpose(&corruptedSyspurposeFilePath)
	if err == nil {
		t.Fatalf("no error returned, when wrong file path to %s provided",
			corruptedSyspurposeFilePath)
	}
	if syspurpose != nil {
		t.Fatalf("syspurpose object was returned despite content of %s was corrupted",
			corruptedSyspurposeFilePath)
	}
}
