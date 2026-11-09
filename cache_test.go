package rhsm2

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// readStatusCacheFile reads and parses the status cache envelope from the given path.
func readStatusCacheFile(t *testing.T, cacheFilePath string) *StatusCache {
	t.Helper()
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		t.Fatalf("unable to read status cache file %s: %s", cacheFilePath, err)
	}
	var cache StatusCache
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("unable to parse status cache file %s: %s", cacheFilePath, err)
	}
	return &cache
}

// writeStatusCacheFile marshals and writes the given status cache envelope to the
// given path. It is used to seed the cache with a controlled state (e.g. an
// expired entry).
func writeStatusCacheFile(t *testing.T, cacheFilePath string, cache *StatusCache) {
	t.Helper()
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("unable to marshal status cache: %s", err)
	}
	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		t.Fatalf("unable to write status cache file %s: %s", cacheFilePath, err)
	}
}

// assertCachedStatusVersion reads the status cache file and checks that the cached
// status holds the expected server version.
func assertCachedStatusVersion(t *testing.T, cacheFilePath string, expectedVersion string) {
	t.Helper()
	cache := readStatusCacheFile(t, cacheFilePath)
	if cache.Status.Version != expectedVersion {
		t.Fatalf("expected cached server version: %s, got: %s", expectedVersion, cache.Status.Version)
	}
}

// TestGetServerStatusWritesCacheCertAuth tests that the server status is written
// to the cache file when using a consumer-cert-auth connection (registered system).
func TestGetServerStatusWritesCacheCertAuth(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := newStatusServer(t, &handlerCounter)
	defer server.Close()

	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when the system is registered
	// (consumer cert/key installed)
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, true, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// The cache file should not exist before the first call
	if _, err := os.Stat(testingFiles.StatusCacheFilePath); err == nil {
		t.Fatalf("status cache file should not exist before the first call")
	}

	serverStatus, err := rhsmClient.GetServerStatus(nil)
	if err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for getting server status not called once, but called: %d", handlerCounter)
	}

	expectedServerVer := "4.3.8"
	if serverStatus.Version != expectedServerVer {
		t.Fatalf("expected server version: %s, got: %s", expectedServerVer, serverStatus.Version)
	}

	// The status should have been written to the cache file
	assertCachedStatusVersion(t, testingFiles.StatusCacheFilePath, expectedServerVer)

	// The cache envelope should record the consumer and the creation time
	cache := readStatusCacheFile(t, testingFiles.StatusCacheFilePath)
	if cache.ConsumerUUID == "" {
		t.Fatalf("expected non-empty consumer UUID in cache for cert-auth connection")
	}
	if time.Since(cache.Created) > time.Minute {
		t.Fatalf("expected recent cache creation time, got: %s", cache.Created)
	}
}

// TestGetServerStatusWritesCacheNoAuth tests that the server status is written to
// the cache file also when using a no-auth connection (unregistered system).
func TestGetServerStatusWritesCacheNoAuth(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := newStatusServer(t, &handlerCounter)
	defer server.Close()

	tempDirFilePath := t.TempDir()

	// Setup filesystem for the case, when the system is NOT registered
	// (no consumer cert/key installed)
	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	serverStatus, err := rhsmClient.GetServerStatus(nil)
	if err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}

	if handlerCounter != 1 {
		t.Fatalf("handler for getting server status not called once, but called: %d", handlerCounter)
	}

	expectedServerVer := "4.3.8"
	if serverStatus.Version != expectedServerVer {
		t.Fatalf("expected server version: %s, got: %s", expectedServerVer, serverStatus.Version)
	}

	// The status should have been written to the cache file, even for no-auth connection
	assertCachedStatusVersion(t, testingFiles.StatusCacheFilePath, expectedServerVer)

	// The cache envelope should record an empty consumer UUID for no-auth connection
	cache := readStatusCacheFile(t, testingFiles.StatusCacheFilePath)
	if cache.ConsumerUUID != "" {
		t.Fatalf("expected empty consumer UUID in cache for no-auth connection, got: %s", cache.ConsumerUUID)
	}
}

// TestGetServerStatusUsesValidCache tests that a valid (fresh) cache is reused, so
// a second call to GetServerStatus does not contact the server.
func TestGetServerStatusUsesValidCache(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := newStatusServer(t, &handlerCounter)
	defer server.Close()

	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// First call populates the cache from the server
	if _, err := rhsmClient.GetServerStatus(nil); err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}

	// Second call should be served from the fresh cache without contacting the server
	serverStatus, err := rhsmClient.GetServerStatus(nil)
	if err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}
	if serverStatus.Version != "4.3.8" {
		t.Fatalf("expected server version: 4.3.8, got: %s", serverStatus.Version)
	}

	if handlerCounter != 1 {
		t.Fatalf("expected exactly one server request (second served from cache), got: %d", handlerCounter)
	}
}

// TestRefreshServerStatusBypassesCache tests that RefreshServerStatus always
// contacts the server, even when a valid cache exists.
func TestRefreshServerStatusBypassesCache(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := newStatusServer(t, &handlerCounter)
	defer server.Close()

	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// First call populates the cache from the server
	if _, err := rhsmClient.GetServerStatus(nil); err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}

	// Forced refresh must contact the server again, ignoring the valid cache
	if _, err := rhsmClient.RefreshServerStatus(nil); err != nil {
		t.Fatalf("refreshing server status failed: %s", err)
	}

	if handlerCounter != 2 {
		t.Fatalf("expected two server requests (refresh bypasses cache), got: %d", handlerCounter)
	}
}

// TestGetServerStatusExpiredCacheRefetches tests that an expired cache is not used
// and the status is re-fetched from the server.
func TestGetServerStatusExpiredCacheRefetches(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := newStatusServer(t, &handlerCounter)
	defer server.Close()

	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Seed an expired cache entry (older than DefaultStatusCacheTimeout) that
	// otherwise matches the current no-auth context.
	writeStatusCacheFile(t, testingFiles.StatusCacheFilePath, &StatusCache{
		Created:        time.Now().Add(-DefaultStatusCacheTimeout - time.Hour),
		ConsumerUUID:   "",
		ServerHostname: rhsmClient.RHSMConf.Server.Hostname,
		ServerPort:     rhsmClient.RHSMConf.Server.Port,
		ServerPrefix:   rhsmClient.RHSMConf.Server.Prefix,
		Status:         RHSMStatus{Version: "0.0.0"},
	})

	// The expired cache must be ignored and the status re-fetched from the server
	serverStatus, err := rhsmClient.GetServerStatus(nil)
	if err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}
	if serverStatus.Version != "4.3.8" {
		t.Fatalf("expected freshly fetched server version: 4.3.8, got: %s", serverStatus.Version)
	}
	if handlerCounter != 1 {
		t.Fatalf("expected exactly one server request after expired cache, got: %d", handlerCounter)
	}
}

// TestGetServerStatusStaleCacheServerDownReturnsError tests that an error is
// returned when the cache is expired (invalid) and the server is not reachable.
// The stale cached status must never be returned, because it could lead to usage
// of stale data.
func TestGetServerStatusStaleCacheServerDownReturnsError(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := newStatusServer(t, &handlerCounter)

	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Seed an expired cache entry
	writeStatusCacheFile(t, testingFiles.StatusCacheFilePath, &StatusCache{
		Created:        time.Now().Add(-DefaultStatusCacheTimeout - time.Hour),
		ConsumerUUID:   "",
		ServerHostname: rhsmClient.RHSMConf.Server.Hostname,
		ServerPort:     rhsmClient.RHSMConf.Server.Port,
		ServerPrefix:   rhsmClient.RHSMConf.Server.Prefix,
		Status:         RHSMStatus{Version: "0.0.0"},
	})

	// Simulate an unreachable server
	server.Close()

	// The expired cache must be ignored; because the server is unreachable and
	// no valid cache exists, an error must be returned (stale data must not leak).
	if _, err := rhsmClient.GetServerStatus(nil); err == nil {
		t.Fatalf("expected error when cache is stale and server is unreachable")
	}
}

// TestGetServerStatusDifferentServerInvalidatesCache tests that a cache created
// for a different server is not used.
func TestGetServerStatusDifferentServerInvalidatesCache(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := newStatusServer(t, &handlerCounter)
	defer server.Close()

	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// Seed a fresh cache, but for a different server hostname
	writeStatusCacheFile(t, testingFiles.StatusCacheFilePath, &StatusCache{
		Created:        time.Now(),
		ConsumerUUID:   "",
		ServerHostname: "some-other-server.example.com",
		ServerPort:     rhsmClient.RHSMConf.Server.Port,
		ServerPrefix:   rhsmClient.RHSMConf.Server.Prefix,
		Status:         RHSMStatus{Version: "0.0.0"},
	})

	// The cache for a different server must be ignored and the status re-fetched
	serverStatus, err := rhsmClient.GetServerStatus(nil)
	if err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}
	if serverStatus.Version != "4.3.8" {
		t.Fatalf("expected freshly fetched server version: 4.3.8, got: %s", serverStatus.Version)
	}
	if handlerCounter != 1 {
		t.Fatalf("expected exactly one server request when cache is for a different server, got: %d", handlerCounter)
	}
}

// TestGetCachedServerStatus tests reading the cached status directly, without
// contacting the server.
func TestGetCachedServerStatus(t *testing.T) {
	t.Parallel()
	handlerCounter := 0

	server := newStatusServer(t, &handlerCounter)
	defer server.Close()

	tempDirFilePath := t.TempDir()

	testingFiles, err := setupTestingFileSystem(
		tempDirFilePath, false, false, false, false, true)
	if err != nil {
		t.Fatalf("unable to setup testing environment: %s", err)
	}

	rhsmClient, err := setupTestingRHSMClient(testingFiles, server, nil)
	if err != nil {
		t.Fatalf("unable to setup testing rhsm client: %s", err)
	}

	// There is no cache yet, so reading it must fail
	if _, err := rhsmClient.GetCachedServerStatus(); err == nil {
		t.Fatalf("expected error when reading non-existent status cache")
	}

	// Populate the cache from the server
	if _, err := rhsmClient.GetServerStatus(nil); err != nil {
		t.Fatalf("getting server status failed: %s", err)
	}

	// Now reading the cache must succeed without contacting the server
	cachedStatus, err := rhsmClient.GetCachedServerStatus()
	if err != nil {
		t.Fatalf("reading cached server status failed: %s", err)
	}

	expectedServerVer := "4.3.8"
	if cachedStatus.Version != expectedServerVer {
		t.Fatalf("expected cached server version: %s, got: %s", expectedServerVer, cachedStatus.Version)
	}

	// Reading the cache must not have triggered any additional server request
	if handlerCounter != 1 {
		t.Fatalf("expected exactly one server request, got: %d", handlerCounter)
	}
}
