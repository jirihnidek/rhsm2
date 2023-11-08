package rhsm2

import "testing"

// TestCreateRHSMClient test the case, when client is
// successfully created using given configuration file
func TestCreateRHSMClient(t *testing.T) {
	t.Parallel()
	confFilePath := "./testdata/etc/rhsm/rhsm.conf"

	rhsmClient, err := CreateRHSMClient(&confFilePath)

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
