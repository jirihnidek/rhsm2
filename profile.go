package rhsm2

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// content_type values candlepin expects for each block of the combined
// profile upload payload.
const (
	rpmContentType          = "rpm"
	enabledReposContentType = "enabled_repos"
)

// combinedReportingCapability is the candlepin managerCapability advertised when
// the server supports the combined profile upload.
const combinedReportingCapability = "combined_reporting"

// ProfileNotSupportedError is returned when the candlepin server does not
// support uploading the combined package profile.
type ProfileNotSupportedError struct{}

func (e *ProfileNotSupportedError) Error() string {
	return "candlepin server does not support combined profile upload"
}

// CombinedProfile is the combined package profile gathered from the local
// system: installed RPM packages and enabled repositories. Modules are not
// included (DNF5 dropped module support).
type CombinedProfile struct {
	RpmPackages  []Package
	EnabledRepos []EnabledRepository
}

// profileContentBlock is one element of the candlepin combined-profile upload
// payload: a content type tag plus the list of entries of that type.
type profileContentBlock struct {
	ContentType string      `json:"content_type"`
	Profile     interface{} `json:"profile"`
}

// buildProfileUploadPayload converts a CombinedProfile into the array of
// content-type blocks candlepin expects.
func buildProfileUploadPayload(profile *CombinedProfile) []profileContentBlock {
	return []profileContentBlock{
		{ContentType: rpmContentType, Profile: profile.RpmPackages},
		{ContentType: enabledReposContentType, Profile: profile.EnabledRepos},
	}
}

// GatherCombinedProfile collects the combined package profile (installed RPM
// packages and enabled repositories) from the local system. Gathering is
// all-or-nothing: if either read fails, no profile is produced, so a partial
// profile is never uploaded.
func (rhsmClient *RHSMClient) GatherCombinedProfile() (*CombinedProfile, error) {
	packages, err := GetInstalledPackages()
	if err != nil {
		return nil, fmt.Errorf("unable to gather installed packages: %w", err)
	}

	repos, err := GetEnabledRepositories()
	if err != nil {
		return nil, fmt.Errorf("unable to gather enabled repositories: %w", err)
	}

	return &CombinedProfile{
		RpmPackages:  packages,
		EnabledRepos: repos,
	}, nil
}

// uploadCombinedProfile sends an already-gathered CombinedProfile to the
// candlepin server via PUT consumers/{uuid}/profiles.
func (rhsmClient *RHSMClient) uploadCombinedProfile(profile *CombinedProfile, info *RequestMetadata) error {
	consumerUuid, err := rhsmClient.GetConsumerUUID()
	if err != nil {
		return err
	}

	info = sanitizeMetadata(info)

	body, err := json.Marshal(buildProfileUploadPayload(profile))
	if err != nil {
		return fmt.Errorf("unable to marshal package profile: %s", err)
	}

	headers := map[string]string{"Content-type": "application/json"}

	connection, err := rhsmClient.getCertAuthConnection()
	if err != nil {
		return fmt.Errorf("unable to get consumer cert auth connection: %v", err)
	}
	res, err := connection.request(
		rhsmClient.UserAgent,
		http.MethodPut,
		"consumers/"+*consumerUuid+"/profiles",
		"",
		"",
		&headers,
		&body,
		info,
	)
	if err != nil {
		return fmt.Errorf("unable to put package profile: %s", err)
	}

	switch res.StatusCode {
	case 204:
		return nil
	case 403:
		log.Error().Msgf("insufficient permissions")
		return fmt.Errorf("unable to put package profile: insufficient permissions")
	case 404:
		log.Error().Msgf("consumer with UUID: %s could not be found", *consumerUuid)
		return fmt.Errorf("unable to put package profile: consumer not found")
	case 500:
		log.Error().Msgf("an unexpected exception has occurred")
		return fmt.Errorf("unable to put package profile: internal server error")
	default:
		return fmt.Errorf("unable to put package profile: unexpected status code %d", res.StatusCode)
	}
}

// sendPackageProfile gathers the combined package profile and uploads it to the
// candlepin server, returning any error. It is the synchronous core of
// SendPackageProfile.
func (rhsmClient *RHSMClient) sendPackageProfile(info *RequestMetadata) error {
	// Respect the report_package_profile rhsm.conf setting.
	if !rhsmClient.RHSMConf.RHSM.ReportPackageProfile {
		return nil
	}

	// Only upload when the server supports the combined profile; otherwise
	// return a typed error callers can distinguish from an upload failure.
	supported, err := rhsmClient.HasCapability(combinedReportingCapability, info)
	if err != nil {
		return fmt.Errorf("unable to determine server profile capability: %w", err)
	}
	if !supported {
		return &ProfileNotSupportedError{}
	}

	profile, err := rhsmClient.GatherCombinedProfile()
	if err != nil {
		return err
	}

	return rhsmClient.uploadCombinedProfile(profile, info)
}

// SendPackageProfile gathers the combined package profile and uploads it to the
// candlepin server. When async is false it runs synchronously and returns any
// error; when async is true it runs in the background, returns nil immediately,
// and logs the result.
func (rhsmClient *RHSMClient) SendPackageProfile(async bool, info *RequestMetadata) error {
	if !async {
		return rhsmClient.sendPackageProfile(info)
	}

	go func() {
		// Signal completion so tests can wait for the async upload.
		if profileUploadDoneSignal != nil {
			defer close(profileUploadDoneSignal)
		}
		if err := rhsmClient.sendPackageProfile(info); err != nil {
			log.Error().Err(err).Msg("async package profile upload failed")
		} else {
			log.Info().Msg("async package profile upload succeeded")
		}
	}()

	return nil
}

// profileUploadDoneSignal is a test-only hook; when non-nil the async goroutine
// closes it after the upload completes. It is nil in production.
var profileUploadDoneSignal chan struct{}
