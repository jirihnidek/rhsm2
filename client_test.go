package rhsm2

import "testing"

// TestCreateRHSMClient test the case, when client is
// successfully created using given configuration file
func TestCreateRHSMClient(t *testing.T) {
	t.Parallel()
	confFilePath := "./testdata/etc/rhsm/rhsm.conf"

	rhsmClient, err := createRHSMClient(&confFilePath)

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

		if rhsmClient.NoAuthConnection == nil {
			t.Fatal("no-auth connection has not been created")
		}

		if rhsmClient.ConsumerCertAuthConnection == nil {
			t.Fatal("consumer cert auth connection has not been created")
		}
	}
}

// TestGetRHSMClient test the case, when client tries to get
// RHSMClient multiple times. It should be still the same instance
func TestGetRHSMClient(t *testing.T) {
	confFilePath := "./testdata/etc/rhsm/rhsm.conf"

	rhsmClient01, err := GetRHSMClient(&confFilePath)
	if err != nil {
		t.Fatalf("unable to get instance of RHSM client: %s", err)
	} else {
		rhsmClient02, err := GetRHSMClient(&confFilePath)
		if err != nil {
			t.Fatalf("unable to get another instance of RHSM client: %s", err)
		} else {
			if rhsmClient01 != rhsmClient02 {
				t.Fatalf("instances of RHSM client are not the same")
			}
		}
	}
}
