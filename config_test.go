package rhsm2

import (
	"reflect"
	"testing"
)

func TestLoadRHSMConf(t *testing.T) {
	confFilePath := "./test/rhsm.conf"
	rhsmConf, err := LoadRHSMConf(&confFilePath)
	if err != nil {
		t.Fatalf("unable to load configuration file: %s", err)
	} else {
		if rhsmConf.Server.Hostname != "subscription.rhsm.redhat.com" {
			t.Fatalf("server hostname: '%s' != 'subscription.rhsm.redhat.com'", rhsmConf.Server.Hostname)
		}
	}
}

func TestIsDefaultValue(t *testing.T) {
	confFilePath := "./test/rhsm.conf"
	rhsmConf, err := LoadRHSMConf(&confFilePath)
	if err != nil {
		t.Fatalf("unable to load configuration file: %s", err)
	} else {
		valuesOfRHSMConf := reflect.ValueOf(*rhsmConf)
		typesOfRHSMConf := valuesOfRHSMConf.Type()
		serverSection := valuesOfRHSMConf.FieldByName("Server")
		_, found := typesOfRHSMConf.FieldByName("Server")
		if found == false {
			t.Fatalf("'Server' type not found")
		}
		if serverSection.IsZero() == false {
			hostname := serverSection.FieldByName("Hostname")
			if hostname.IsZero() == false {
				serverTypes := serverSection.Type()
				hostnameType, found := serverTypes.FieldByName("Hostname")
				if found == false {
					t.Fatalf("'Hostname' type not found")
				}
				tag := hostnameType.Tag
				defaultValue, found := tag.Lookup("default")
				if found == true {
					isDefault, err := IsDefaultValue(&hostname, &defaultValue)
					if err != nil {
						t.Fatalf("unable to detect if values is default: %s", err)
					} else {
						if isDefault == false {
							t.Fatalf("server.hostname is not default: %s", hostname.String())
						}
					}
				} else {
					t.Fatalf("default tag not found")
				}
			} else {
				t.Fatalf("'Hostname' not found")
			}
		} else {
			t.Fatalf("'Server' section not found")
		}
	}
}

func TestIsValueAllowed(t *testing.T) {
	confFilePath := "./test/rhsm.conf"
	rhsmConf, err := LoadRHSMConf(&confFilePath)
	if err != nil {
		t.Fatalf("unable to load configuration file: %s", err)
	} else {
		valuesOfRHSMConf := reflect.ValueOf(*rhsmConf)
		typesOfRHSMConf := valuesOfRHSMConf.Type()
		serverSection := valuesOfRHSMConf.FieldByName("Server")
		_, found := typesOfRHSMConf.FieldByName("Server")
		if found == false {
			t.Fatalf("'Server' type not found")
		}
		if serverSection.IsZero() == false {
			hostname := serverSection.FieldByName("ProxyScheme")
			if hostname.IsZero() == false {
				serverTypes := serverSection.Type()
				proxyScheme, found := serverTypes.FieldByName("ProxyScheme")
				if found == false {
					t.Fatalf("'ProxyScheme' type not found")
				}
				tag := proxyScheme.Tag
				allowedValues, found := tag.Lookup("allowedValues")
				if found == true {
					isAllowed, err := IsValueAllowed(&hostname, &allowedValues)
					if err != nil {
						t.Fatalf("unable to detect if value is allowed: %s", err)
					} else {
						if isAllowed == false {
							t.Fatalf("server.proxy_scheme is not allowed: %s", hostname.String())
						}
					}
				} else {
					t.Fatalf("default tag not found")
				}
			} else {
				t.Fatalf("'Hostname' not found")
			}
		} else {
			t.Fatalf("'Server' section not found")
		}
	}
}
