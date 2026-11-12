package rhsm2

import "testing"

// TestCreateRHSMClient test the case, when client is
// successfully created using given configuration file
func TestCreateRHSMClient(t *testing.T) {
	t.Parallel()
	appName := "unit-tester/0.1"
	confFilePath := "./testdata/etc/rhsm/rhsm.conf"

	rhsmClient, err := createRHSMClient(&appName, &confFilePath)

	if err != nil {
		t.Fatalf("unable to create RHSM client: %s", err)
	} else {

		consumerCertPath := rhsmClient.consumerCertPath()
		if *consumerCertPath != "testdata/etc/pki/consumer/cert.pem" {
			t.Fatalf("consumer cert file path: '%s' != 'testdata/pki/consumer/cert.pem'", *consumerCertPath)
		}

		consumerKeyPath := rhsmClient.consumerKeyPath()
		if *consumerKeyPath != "testdata/etc/pki/consumer/key.pem" {
			t.Fatalf("consumer key file path: '%s' != 'testdata/pki/consumer/key.pem'", *consumerKeyPath)
		}

		if rhsmClient.noAuthConnection == nil {
			t.Fatal("no-auth connection has not been created")
		}

		if rhsmClient.consumerCertAuthConnection == nil {
			t.Fatal("consumer cert auth connection has not been created")
		}
	}
}

// TestCreateRHSMClientWithBrokenEntitlementPair tests the case when the only installed
// entitlement cert/key pair is broken (the certificate and key do not belong together).
// The entitlement connection is used only for communication with CDN and is not critical
// for the RHSM client, so a failure to create it must not prevent the client from starting.
// See testdata/use_cases/broken_pair_of_ent_cert_and_key for details about the broken pair.
func TestCreateRHSMClientWithBrokenEntitlementPair(t *testing.T) {
	t.Parallel()

	// Create root directory for this test
	tempDirFilePath := t.TempDir()

	// Set up a filesystem with a consumer cert/key installed, but without any valid
	// entitlement cert/key pair (entCertsInstalled is false).
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath,
		true,
		true,
		false,
		true,
		true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	// Rewrite the testing rhsm.conf so that all certificate directories point to the
	// temporary directories created above. createRHSMClient loads this configuration file.
	if _, err := setupTestingRHSMClient(testingFiles, nil, nil); err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Install only a broken entitlement cert/key pair, whose certificate and key do not
	// belong together and therefore cannot be loaded as a TLS key pair.
	if err := testingFiles.setupBrokenEntitlementCertKey(nil); err != nil {
		t.Fatalf("unable to install broken entitlement cert/key pair: %s", err)
	}

	appName := "unit-tester/0.1"

	// Creating the client must succeed despite the broken entitlement pair. Before the fix
	// the broken pair caused tls.LoadX509KeyPair to fail and createRHSMClient returned an
	// error, preventing the whole client from starting.
	rhsmClient, err := createRHSMClient(&appName, &testingFiles.RHSMConfFilePath)
	if err != nil {
		t.Fatalf("createRHSMClient must not fail due to a broken entitlement pair, got: %s", err)
	}
	if rhsmClient == nil {
		t.Fatalf("rhsm client should not be nil")
	}

	// The no-auth and consumer cert auth connections must still be created.
	if rhsmClient.noAuthConnection == nil {
		t.Fatal("no-auth connection has not been created")
	}
	if rhsmClient.consumerCertAuthConnection == nil {
		t.Fatal("consumer cert auth connection has not been created")
	}

	// The broken entitlement pair must be skipped and, as it is the only one installed,
	// no entitlement cert auth connection should be created.
	if rhsmClient.entitlementCertAuthConnection != nil {
		t.Fatal("entitlement cert auth connection should not be created from a broken pair")
	}
}

// TestGetRHSMClient test the case, when client tries to get
// RHSMClient multiple times. It should be still the same instance
func TestGetRHSMClient(t *testing.T) {
	appName := "unit-tester/0.1"
	confFilePath := "./testdata/etc/rhsm/rhsm.conf"

	rhsmClient01, err := GetRHSMClient(&appName, &confFilePath)
	if err != nil {
		t.Fatalf("unable to get instance of RHSM client: %s", err)
	} else {
		rhsmClient02, err := GetRHSMClient(&appName, &confFilePath)
		if err != nil {
			t.Fatalf("unable to get another instance of RHSM client: %s", err)
		} else {
			if rhsmClient01 != rhsmClient02 {
				t.Fatalf("instances of RHSM client are not the same")
			}
		}
	}
}
