package rhsm2

import "testing"

func TestGetConsumerUUID(t *testing.T) {
	consumerCertFilePath := "./test/pki/consumer/cert.pem"
	uuid, err := GetConsumerUUID(&consumerCertFilePath)
	if err != nil {
		t.Fatalf("unable to get consumer UUID from consumer cert: %s", err)
	} else {
		if *uuid != "5e9745d5-624d-4af1-916e-2c17df4eb4e8" {
			t.Fatalf("consumer UUID: '%s' != '5e9745d5-624d-4af1-916e-2c17df4eb4e8'", *uuid)
		}
	}
}

func TestGetOwner(t *testing.T) {
	consumerCertFilePath := "./test/pki/consumer/cert.pem"
	orgID, err := GetOwner(&consumerCertFilePath)
	if err != nil {
		t.Fatalf("unable to get organization ID from consumer cert: %s", err)
	} else {
		if *orgID != "donaldduck" {
			t.Fatalf("org ID: '%s' != 'donaldduck'", *orgID)
		}
	}
}
