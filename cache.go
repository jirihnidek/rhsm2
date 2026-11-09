package rhsm2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultStatusCacheFilePath is the default file path of the cache file holding
// the last successful response of the "/status" REST API endpoint. The result is
// cached here for both consumer-cert-auth and no-auth connections.
const DefaultStatusCacheFilePath = "/var/cache/rhsm/status.json"

// DefaultStatusCacheTimeout is the maximum age of a cached "/status" response.
// The response of the "/status" endpoint changes only when the server is updated
// or reconfigured, so the status can be cached for a relatively long time to
// minimize the number of requests to the server.
const DefaultStatusCacheTimeout = 4 * time.Hour

// StatusCache is the on-disk envelope stored in the status cache file
// (see DefaultStatusCacheFilePath). Besides the cached "/status" response it
// stores the context in which the response was obtained, so the cache can be
// invalidated when that context no longer matches.
type StatusCache struct {
	// Created is the UTC time when this cache entry was written. It is used
	// together with DefaultStatusCacheTimeout to expire stale entries.
	Created time.Time `json:"created"`

	// ConsumerUUID is the UUID of the consumer used to obtain the status.
	// It is empty when the status was obtained over a no-auth connection
	// (i.e. the system is not registered).
	ConsumerUUID string `json:"consumerUUID"`

	// Server identity (hostname, port and prefix) under which the status was
	// obtained. When the configuration is changed to point to a different
	// server, then the cache is considered invalid.
	ServerHostname string `json:"serverHostname"`
	ServerPort     string `json:"serverPort"`
	ServerPrefix   string `json:"serverPrefix"`

	// Status is the cached response body of the "/status" endpoint.
	Status RHSMStatus `json:"status"`
}

// statusCacheFilePath returns the file path of the status cache file. When the
// path is not configured (e.g. the RHSMConf was created manually), then the
// default path is used.
func (rhsmClient *RHSMClient) statusCacheFilePath() string {
	if rhsmClient.RHSMConf != nil && rhsmClient.RHSMConf.statusCacheFilePath != "" {
		return rhsmClient.RHSMConf.statusCacheFilePath
	}
	return DefaultStatusCacheFilePath
}

// currentConsumerUUID returns the UUID of the currently installed consumer. When
// no consumer certificate is installed (the system is not registered), then an
// empty string is returned.
func (rhsmClient *RHSMClient) currentConsumerUUID() string {
	consumerUUID, err := rhsmClient.GetConsumerUUID()
	if err != nil || consumerUUID == nil {
		return ""
	}
	return *consumerUUID
}

// writeServerStatusCache stores the given server status to the status cache file
// (see DefaultStatusCacheFilePath) together with the context in which it was
// obtained. The parent directory is created when it does not exist. The result is
// cached for both consumer-cert-auth and no-auth connections.
func (rhsmClient *RHSMClient) writeServerStatusCache(status *RHSMStatus, consumerUUID string) error {
	cacheFilePath := rhsmClient.statusCacheFilePath()

	cache := StatusCache{
		Created:        time.Now().UTC(),
		ConsumerUUID:   consumerUUID,
		ServerHostname: rhsmClient.RHSMConf.Server.Hostname,
		ServerPort:     rhsmClient.RHSMConf.Server.Port,
		ServerPrefix:   rhsmClient.RHSMConf.Server.Prefix,
		Status:         *status,
	}

	data, err := json.MarshalIndent(&cache, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal server status cache: %s", err)
	}

	cacheDirPath := filepath.Dir(cacheFilePath)
	if err := os.MkdirAll(cacheDirPath, 0755); err != nil {
		return fmt.Errorf("unable to create cache directory %s: %s", cacheDirPath, err)
	}

	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		return fmt.Errorf("unable to write server status cache %s: %s", cacheFilePath, err)
	}

	return nil
}

// readServerStatusCache reads the status cache file and returns the stored cache
// envelope. An error is returned when the cache file does not exist or cannot be
// parsed.
func (rhsmClient *RHSMClient) readServerStatusCache() (*StatusCache, error) {
	cacheFilePath := rhsmClient.statusCacheFilePath()

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read server status cache %s: %s", cacheFilePath, err)
	}

	var cache StatusCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("unable to parse server status cache %s: %s", cacheFilePath, err)
	}

	return &cache, nil
}

// isStatusCacheValid reports whether the given cache entry can be used instead of
// contacting the server. The cache is valid only when it has not expired and when
// it was obtained in the same context (same consumer and same server) as the
// current one.
func (rhsmClient *RHSMClient) isStatusCacheValid(cache *StatusCache, consumerUUID string) bool {
	if cache == nil {
		return false
	}

	// The cache must not be older than the configured timeout
	if time.Since(cache.Created) > DefaultStatusCacheTimeout {
		return false
	}

	// The cache must have been obtained by the same consumer (an empty UUID
	// means a no-auth connection was used)
	if cache.ConsumerUUID != consumerUUID {
		return false
	}

	// The cache must have been obtained from the currently configured server
	if cache.ServerHostname != rhsmClient.RHSMConf.Server.Hostname ||
		cache.ServerPort != rhsmClient.RHSMConf.Server.Port ||
		cache.ServerPrefix != rhsmClient.RHSMConf.Server.Prefix {
		return false
	}

	return true
}

// GetCachedServerStatus returns the server status stored in the local cache file
// without contacting the server and without any validity check. An error is
// returned when no cached status is available. Note: the returned status may be
// stale; use GetServerStatus() to get an up-to-date status.
func (rhsmClient *RHSMClient) GetCachedServerStatus() (*RHSMStatus, error) {
	cache, err := rhsmClient.readServerStatusCache()
	if err != nil {
		return nil, err
	}
	return &cache.Status, nil
}
