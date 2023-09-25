package rhsm2

// RHSMStatus is structure used for storing GET response from REST API
// endpoint "/status". This endpoint can be called using no-auth or
// consumer-cert-auth connection
type RHSMStatus struct {
	Mode           string      `json:"mode"`
	ModeReason     interface{} `json:"modeReason"`
	ModeChangeTime interface{} `json:"modeChangeTime"`
	Result         bool        `json:"result"`
	Version        string      `json:"version"`
	Release        string      `json:"release"`
	Standalone     bool        `json:"standalone"`
	// Note: json module cannot unmarshal timeUTC into time.Time
	// for this reason: https://github.com/golang/go/issues/47353
	// Because we do not need to use timeUTC for anything ATM.
	// It is parsed as normal string.
	TimeUTC             string      `json:"timeUTC"`
	RulesSource         string      `json:"rulesSource"`
	RulesVersion        string      `json:"rulesVersion"`
	ManagerCapabilities []string    `json:"managerCapabilities"`
	KeycloakRealm       interface{} `json:"keycloakRealm"`
	KeycloakAuthUrl     interface{} `json:"keycloakAuthUrl"`
	KeycloakResource    interface{} `json:"keycloakResource"`
	DeviceAuthRealm     interface{} `json:"deviceAuthRealm"`
	DeviceAuthUrl       interface{} `json:"deviceAuthUrl"`
	DeviceAuthClientId  interface{} `json:"deviceAuthClientId"`
	DeviceAuthScope     interface{} `json:"deviceAuthScope"`
}
