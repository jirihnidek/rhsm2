package rhsm2

import (
	"fmt"
	"github.com/creasty/defaults"
	"gopkg.in/ini.v1"
	"reflect"
	"strconv"
	"strings"
)

const DefaultRHSMConfFilePath = "/etc/rhsm/rhsm.conf"

// RHSMConfServer represents section [server] in rhsm.conf
type RHSMConfServer struct {
	// Basic settings for connection to candlepin server
	Hostname string `ini:"hostname" default:"subscription.rhsm.redhat.com"`
	Prefix   string `ini:"prefix" default:"/subscription"`
	Port     string `ini:"port" default:"443"`
	Insecure bool   `ini:"insecure" default:"false"`
	Timeout  int64  `ini:"server_timeout" default:"180"`

	// Proxy settings
	ProxyHostname string `ini:"proxy_hostname" default:""`
	ProxyScheme   string `ini:"proxy_scheme" default:"http" allowedValues:"http,https"`
	ProxyPort     string `ini:"proxy_port" default:"3128"`
	ProxyUser     string `ini:"proxy_user" default:""`
	ProxyPassword string `ini:"proxy_password" default:""`

	// Comma separated list of hostnames, when connection should not go
	// through proxy server
	NoProxy string `ini:"no_proxy" default:""`
}

// RHSMConfRHSM represents section [rhsm] in rhsm.conf
type RHSMConfRHSM struct {
	// Directories used for certificates
	CACertDir          string `ini:"ca_cert_dir" default:"/etc/rhsm/ca/"`
	ConsumerCertDir    string `ini:"consumer_cert_dir" default:"/etc/pki/consumer"`
	EntitlementCertDir string `ini:"entitlement_cert_dir" default:"/etc/pki/entitlement"`
	ProductCertDir     string `ini:"product_cert_dir" default:"/etc/pki/product"`

	// Configuration options related to RPMs and repositories
	BaseURL              string `ini:"baseurl" default:"https://cdn.redhat.com"`
	ReportPackageProfile bool   `ini:"report_package_profile" default:"true"`
	RepoCACertificate    string `ini:"repo_ca_cert" default:"/etc/rhsm/ca/redhat-uep.pem"`
	ManageRepos          bool   `ini:"manage_repos" default:"true"`

	// Configuration options related to DNF plugins
	AutoEnableYumPlugins  bool `ini:"auto_enable_yum_plugins" default:"true"`
	PackageProfileOnTrans bool `ini:"package_profile_on_trans" default:"false"`
}

// RHSMConfRHSMCertDaemon represents section [rhsmcertd] in rhsm.conf
type RHSMConfRHSMCertDaemon struct {
	AutoRegistration         bool  `ini:"auto_registration" default:"false"`
	AutoRegistrationInterval int64 `ini:"auto_registration_interval" default:"60"`
	Splay                    bool  `ini:"splay" default:"true"`
}

type RHSMConfLogging struct {
	DefaultLogLevel string `ini:"default_log_level" default:"INFO" allowedValues:"ERROR,WARN,INFO,DEBUG"`
}

// RHSMConf is structure intended for storing configuration
// that is typically read from /etc/rhsm/rhsm.conf. We try to
type RHSMConf struct {
	// Not public attributes

	// filePath is file path of configuration file
	filePath string

	// yumRepoFilePath is path
	yumRepoFilePath string

	// Server represents section [server]
	Server RHSMConfServer `ini:"server"`

	// RHSM represents section [rhsm]
	RHSM RHSMConfRHSM `ini:"rhsm"`

	// RHSMCertDaemon represents section [rhsmcertd]
	RHSMCertDaemon RHSMConfRHSMCertDaemon `ini:"rhsmcertd"`

	// Logging represents section [logging]
	Logging RHSMConfLogging `ini:"logging"`
}

// setDefaultValues tries to set default values specified in tags
func (rhsmConf *RHSMConf) setDefaultValues() error {
	err := defaults.Set(rhsmConf)
	if err != nil {
		return err
	}
	return nil
}

// load tries to load configuration file (usually /etc/rhsm/rhsm.conf)
func (rhsmConf *RHSMConf) load() error {
	cfg, err := ini.Load(rhsmConf.filePath)
	if err != nil {
		return err
	}

	// First set default values
	err = rhsmConf.setDefaultValues()
	if err != nil {
		return err
	}

	// Then try to load values from given configuration file.
	// Note that parsing errors are ignored and default values are used in case of some errors.
	err = cfg.MapTo(rhsmConf)
	if err != nil {
		return err
	}

	return nil
}

// IsDefaultValue tries to say if given value is default value or not
func IsDefaultValue(value *reflect.Value, defaultValue *string) (bool, error) {
	switch value.Kind() {
	case reflect.String:
		return value.String() == *defaultValue, nil
	case reflect.Int, reflect.Int64:
		intVal := value.Int()
		defaultIntVal, err := strconv.ParseInt(*defaultValue, 10, 64)
		if err != nil {
			return false, err
		}
		return intVal == defaultIntVal, nil
	case reflect.Bool:
		boolVal := value.Bool()
		defaultBoolVal, err := strconv.ParseBool(*defaultValue)
		if err != nil {
			return false, err
		}
		return boolVal == defaultBoolVal, nil
	default:
		return false, fmt.Errorf("unsupported type of value: %s", value.Kind())
	}
}

// IsValueAllowed tries to say if given value is allowed or not. The allowedValues
// is string with comma separated values
func IsValueAllowed(value *reflect.Value, allowedValues *string) (bool, error) {
	allowedValuesSlice := strings.Split(*allowedValues, ",")

	// Only strings are supported at this moment, because I cannot imagine another use case
	// with another e.g. allowed integer values. When it will be necessary, then it is easy to add
	// support for another type
	switch value.Kind() {
	case reflect.String:
		val := value.String()
		for _, allowedValue := range allowedValuesSlice {
			if val == allowedValue {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported type of allowed value: %s", value.Kind())
	}
}

// LoadRHSMConf tries to load given configuration file to
// RHSMConf structure
func LoadRHSMConf(confFilePath string) (*RHSMConf, error) {
	rhsmConf := &RHSMConf{
		filePath:        confFilePath,
		yumRepoFilePath: DefaultRepoFilePath,
	}

	err := rhsmConf.load()
	if err != nil {
		return nil, err
	}

	return rhsmConf, nil
}
