package rhsm2

import "testing"

func TestCreateRHSMClient(t *testing.T) {
	confFilePath := "./test/etc/rhsm/rhsm.conf"
	rhsmClient, err := CreateRHSMClient(&confFilePath)
	if err != nil {
		t.Fatalf("unable to create RHSM client: %s", err)
	} else {

		consumerCertPath := rhsmClient.consumerCertPath()
		if *consumerCertPath != "test/etc/pki/consumer/cert.pem" {
			t.Fatalf("consumer cert file path: '%s' != 'test/pki/consumer/cert.pem'", *consumerCertPath)
		}

		consumerKeyPath := rhsmClient.consumerKeyPath()
		if *consumerKeyPath != "test/etc/pki/consumer/key.pem" {
			t.Fatalf("consumer key file path: '%s' != 'test/pki/consumer/key.pem'", *consumerKeyPath)
		}

		if rhsmClient.NoAuthConnection == nil {
			t.Fatal("no-auth connection has not been created")
		}

		if rhsmClient.ConsumerCertAuthConnection == nil {
			t.Fatal("consumer cert auth connection has not been created")
		}
	}
}
