package rhsm2

import (
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
)

// This is JSON document returned by candlepin in body of response,
// when status code is 410
const consumerAlreadyDeleted = `{
  "displayMessage": "Consumer with 5e9745d5-624d-4af1-916e-2c17df4eb4e8 is already deleted.",
  "requestUuid": "c4347004-8792-41fe-a4d8-fccaa0d3898a"
  "deletedId": "5e9745d5-624d-4af1-916e-2c17df4eb4e8"
}`

// This is JSON document returned by candlepin in body of response,
// when status code is 500
const internalServerError = `{
  "displayMessage": "An unexpected exception has occurred",
  "requestUuid": "c4347004-8792-41fe-a4d8-fccaa0d3898a"
}`

// isDirEmpty tries to check if directory is empty
func isDirEmpty(name *string) (bool, error) {
	f, err := os.Open(*name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// copyFile tries to copy file
func copyFile(srcFilePath *string, dstFilePath *string) error {
	pemIn, err := os.Open(*srcFilePath)
	if err != nil {
		return fmt.Errorf("unable to open file: %s: %s", *srcFilePath, err)
	}

	pemOut, err := os.Create(*dstFilePath)
	if err != nil {
		return fmt.Errorf("unable to create file: %s: %s", *dstFilePath, err)
	}

	_, err = io.Copy(pemOut, pemIn)
	if err != nil {
		return fmt.Errorf("unable to copy %s to %s: %s", *srcFilePath, *dstFilePath, err)
	}
	return nil
}

// TestingFileSystem is structure holding information about file paths
// used for testing
type TestingFileSystem struct {
	CACertDirPath         string
	ConsumerDirPath       string
	EntitlementDirPath    string
	ProductDirPath        string
	ProductDefaultDirPath string
	SyspurposeDirPath     string
	SyspurposeFilePath    string
	YumReposDirPath       string
	YumRepoFilePath       string
}

// setupTestingFiles tries to copy and generate testing files to testing directories
func setupTestingFiles(
	testingFileSystem *TestingFileSystem,
	syspurposeFilesInstalled bool,
	consumerCertInstalled bool,
	entCertsInstalled bool,
	prodCertsInstalled bool,
	defaultProdCertsInstalled bool,
) error {
	if syspurposeFilesInstalled {
		// Copy syspurpose file to temporary directory
		srcSyspurposeFilePath := "./test/etc/rhsm/syspurpose/syspurpose.json"
		dstSyspurposeFilePath := filepath.Join(testingFileSystem.SyspurposeDirPath, "syspurpose.json")
		err := copyFile(&srcSyspurposeFilePath, &dstSyspurposeFilePath)
		if err != nil {
			return fmt.Errorf(
				"unable to create testing consumer key file: %s", err)
		}
		testingFileSystem.SyspurposeFilePath = dstSyspurposeFilePath
	}

	if consumerCertInstalled {
		// Copy consumer key to temporary directory
		srcConsumerKeyFilePath := "./test/etc/pki/consumer/key.pem"
		dstConsumerKeyFilePath := filepath.Join(testingFileSystem.ConsumerDirPath, "key.pem")
		err := copyFile(&srcConsumerKeyFilePath, &dstConsumerKeyFilePath)
		if err != nil {
			return fmt.Errorf(
				"unable to create testing consumer key file: %s", err)
		}
		// Copy consumer cert to temporary directory
		srcConsumerCertFilePath := "test/etc/pki/consumer/cert.pem"
		dstConsumerCertFilePath := filepath.Join(testingFileSystem.ConsumerDirPath, "cert.pem")
		err = copyFile(&srcConsumerCertFilePath, &dstConsumerCertFilePath)
		if err != nil {
			return fmt.Errorf(
				"unable to create testing consumer cert file: %s", err)
		}
	}

	if entCertsInstalled {
		// Copy entitlement key to temporary directory
		srcEntitlementKeyFilePath := "./test/etc/pki/entitlement/6490061114713729830-key.pem"
		dstEntitlementKeyFilePath := filepath.Join(testingFileSystem.EntitlementDirPath, "6490061114713729830-key.pem")
		err := copyFile(&srcEntitlementKeyFilePath, &dstEntitlementKeyFilePath)
		if err != nil {
			return fmt.Errorf(
				"unable to create testing entitlement key file: %s", err)
		}
		// Copy entitlement cert to temporary directory
		srcEntitlementCertFilePath := "./test/etc/pki/entitlement/6490061114713729830.pem"
		dstEntitlementCertFilePath := filepath.Join(testingFileSystem.EntitlementDirPath, "6490061114713729830.pem")
		err = copyFile(&srcEntitlementCertFilePath, &dstEntitlementCertFilePath)
		if err != nil {
			return fmt.Errorf("unable to create testing entitlement cert file: %s", err)
		}
	}

	// Copy product cert to temporary directory
	if prodCertsInstalled {
		srcProductCertFilePath := "./test/etc/pki/product/900.pem"
		dstProductCertFilePath := filepath.Join(testingFileSystem.ProductDirPath, "900.pem")
		err := copyFile(&srcProductCertFilePath, &dstProductCertFilePath)
		if err != nil {
			return fmt.Errorf("unable to create testing product cert file: %s", err)
		}
	}

	// Copy default product cert to temporary directory
	// Note: There is always at least one default product certificate on RHEL system,
	// but there are other Linux distributions without preinstalled product certificates
	// like Fedora or Centos Stream
	if defaultProdCertsInstalled {
		srcDefaultProductCertFilePath := "./test/etc/pki/product-default/5050.pem"
		dstDefaultProductCertFilePath := filepath.Join(testingFileSystem.ProductDefaultDirPath, "5050.pem")
		err := copyFile(&srcDefaultProductCertFilePath, &dstDefaultProductCertFilePath)
		if err != nil {
			return fmt.Errorf("unable to create testing default product cert file: %s", err)
		}
	}

	// Create only empty redhat.repo ATM
	yumRepoFilePath := filepath.Join(testingFileSystem.YumReposDirPath, "redhat.repo")
	_, err := os.Create(yumRepoFilePath)
	if err != nil {
		return fmt.Errorf("unable to create %s: %s", yumRepoFilePath, err)
	}
	testingFileSystem.YumRepoFilePath = yumRepoFilePath

	return nil
}

// setupTestingDirectories tries to set up directories for testing filesystem
func setupTestingDirectories(tempDirFilePath string) (*TestingFileSystem, error) {
	testingFileSystem := TestingFileSystem{}

	// Create temporary directory for CA certificate
	caCertDirPath := filepath.Join(tempDirFilePath, "etc/rhsm/ca")
	err := os.MkdirAll(caCertDirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", caCertDirPath, err)
	}
	testingFileSystem.CACertDirPath = caCertDirPath

	// Create temporary directory for consumer certificates
	consumerDirPath := filepath.Join(tempDirFilePath, "etc/pki/consumer")
	err = os.MkdirAll(consumerDirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", consumerDirPath, err)
	}
	testingFileSystem.ConsumerDirPath = consumerDirPath

	// Create temporary directory for entitlement certificates
	entitlementDirPath := filepath.Join(tempDirFilePath, "etc/pki/entitlement")
	err = os.MkdirAll(entitlementDirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", entitlementDirPath, err)
	}
	testingFileSystem.EntitlementDirPath = entitlementDirPath

	// Create temporary directory for product certificates
	productDirPath := filepath.Join(tempDirFilePath, "etc/pki/product")
	err = os.MkdirAll(productDirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", productDirPath, err)
	}
	testingFileSystem.ProductDirPath = productDirPath

	// Create temporary directory for product certificates
	productDefaultDirPath := filepath.Join(tempDirFilePath, "etc/pki/product-default")
	err = os.MkdirAll(productDefaultDirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", productDefaultDirPath, err)
	}
	testingFileSystem.ProductDefaultDirPath = productDefaultDirPath

	// Create temporary directory for syspurpose configuration files
	syspurposeDirPath := filepath.Join(tempDirFilePath, "etc/rhsm/syspurpose")
	err = os.MkdirAll(syspurposeDirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", syspurposeDirPath, err)
	}
	testingFileSystem.SyspurposeDirPath = syspurposeDirPath

	// Create directory for redhat.repo
	yumReposDirPath := filepath.Join(tempDirFilePath, "etc/yum.repos.d")
	err = os.MkdirAll(yumReposDirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf(
			"unable to create temporary directory: %s: %s", yumReposDirPath, err)
	}
	testingFileSystem.YumReposDirPath = yumReposDirPath

	return &testingFileSystem, nil
}

// setupTestingFileSystem tries to set up directories and files for testing and mock system
// that is fully installed
func setupTestingFileSystem(
	tempDirFilePath string,
	syspurposeFilesInstalled bool,
	consumerCertInstalled bool,
	entCertsInstalled bool,
	prodCertsInstalled bool,
	defaultProdCertsInstalled bool,
) (*TestingFileSystem, error) {
	testingFileSystem, err := setupTestingDirectories(tempDirFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create testing directories: %s", err)
	}

	err = setupTestingFiles(testingFileSystem, syspurposeFilesInstalled, consumerCertInstalled, entCertsInstalled, prodCertsInstalled, defaultProdCertsInstalled)
	if err != nil {
		return nil, fmt.Errorf("unable to copy testing file to testing directories: %s", err)
	}

	return testingFileSystem, nil
}

// setupTestingRHSMClient tries to set up testing instance of RHSMClient
func setupTestingRHSMClient(testingFiles *TestingFileSystem, server *httptest.Server) (*RHSMClient, error) {
	// Get the hostname, port and prefix from the fake server
	// It will be used for configuring rhsm client
	parsedURL, err := url.Parse(server.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse server URL: %s: %s", server.URL, err)
	}
	hostname := parsedURL.Hostname()
	port := parsedURL.Port()
	prefix := parsedURL.Path

	// Create instance of RHSM client
	rhsmClient := RHSMClient{}

	// Fill rhsm conf with fake data and temporary paths
	rhsmClient.RHSMConf = &RHSMConf{
		yumRepoFilePath:    testingFiles.YumRepoFilePath,
		syspurposeFilePath: testingFiles.SyspurposeFilePath,
		Server: RHSMConfServer{
			Hostname: hostname,
			Port:     port,
			Prefix:   prefix,
		},
		RHSM: RHSMConfRHSM{
			ConsumerCertDir:       testingFiles.ConsumerDirPath,
			EntitlementCertDir:    testingFiles.EntitlementDirPath,
			ProductCertDir:        testingFiles.ProductDirPath,
			DefaultProductCertDir: testingFiles.ProductDefaultDirPath,
			CACertDir:             testingFiles.CACertDirPath,
		},
	}

	// Mock connections to server with mock server
	rhsmClient.NoAuthConnection = &RHSMConnection{
		AuthType:       NoAuth,
		Client:         server.Client(),
		ServerHostname: &hostname,
		ServerPort:     &port,
		ServerPrefix:   &prefix,
	}
	rhsmClient.ConsumerCertAuthConnection = &RHSMConnection{
		AuthType:       ConsumerCertAuth,
		Client:         server.Client(),
		ServerHostname: &hostname,
		ServerPort:     &port,
		ServerPrefix:   &prefix,
	}

	// TODO: populate rhsm.conf with paths of temporary files and server hostname, port, prefix, etc.

	return &rhsmClient, nil
}
