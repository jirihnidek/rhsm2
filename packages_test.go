package rhsm2

import (
	"errors"
	"testing"

	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
)

// TestConvertRpmPackages verifies the mapping from go-rpmdb records to our
// Package type, including a vendor value containing whitespace.
func TestConvertRpmPackages(t *testing.T) {
	t.Parallel()

	in := []*rpmdb.PackageInfo{
		{Name: "fonts-filesystem", Version: "5.0.0", Release: "2.fc44", Arch: "noarch", Vendor: "Fedora Project"},
	}

	packages := convertRpmPackages(in)
	if len(packages) != 1 {
		t.Fatalf("got %d packages, want 1", len(packages))
	}

	got := packages[0]
	if got.Name != "fonts-filesystem" || got.Version != "5.0.0" ||
		got.Release != "2.fc44" || got.Arch != "noarch" || got.Vendor != "Fedora Project" {
		t.Errorf("convertRpmPackages() = %+v, unexpected fields", got)
	}
}

// TestConvertRpmPackagesEmpty verifies that no records yields no packages.
func TestConvertRpmPackagesEmpty(t *testing.T) {
	t.Parallel()

	if got := convertRpmPackages(nil); len(got) != 0 {
		t.Errorf("got %d packages, want 0", len(got))
	}
}

// TestConvertRpmPackagesSkipsNil verifies that a nil record in the slice is
// skipped rather than causing a panic.
func TestConvertRpmPackagesSkipsNil(t *testing.T) {
	t.Parallel()

	in := []*rpmdb.PackageInfo{
		{Name: "bash", Version: "5.2.26", Release: "1.fc44", Arch: "x86_64"},
		nil,
	}
	got := convertRpmPackages(in)
	if len(got) != 1 || got[0].Name != "bash" {
		t.Errorf("convertRpmPackages() = %+v, want one bash package", got)
	}
}

// TestGetInstalledPackages verifies the full path with a substituted
// listRpmPackages, confirming records are converted into Packages.
//
// Not run in parallel: it mutates the shared listRpmPackages package variable.
func TestGetInstalledPackages(t *testing.T) {
	original := listRpmPackages
	defer func() { listRpmPackages = original }()

	listRpmPackages = func() ([]*rpmdb.PackageInfo, error) {
		return []*rpmdb.PackageInfo{
			{Name: "bash", Epoch: nil, Version: "5.2.26", Release: "1.fc44", Arch: "x86_64", Vendor: "Fedora Project"},
		}, nil
	}

	packages, err := GetInstalledPackages()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(packages) != 1 || packages[0].Name != "bash" {
		t.Errorf("GetInstalledPackages() = %+v, want one bash package", packages)
	}
}

// TestGetInstalledPackagesQueryError verifies that a failure to read the rpm
// database is surfaced as an error.
//
// Not run in parallel: it mutates the shared listRpmPackages package variable.
func TestGetInstalledPackagesQueryError(t *testing.T) {
	original := listRpmPackages
	defer func() { listRpmPackages = original }()

	listRpmPackages = func() ([]*rpmdb.PackageInfo, error) {
		return nil, errors.New("rpm database not found")
	}

	if _, err := GetInstalledPackages(); err == nil {
		t.Fatal("expected an error when reading the rpm database fails, got nil")
	}
}
