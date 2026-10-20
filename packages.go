package rhsm2

import (
	"fmt"

	// Register the pure-Go sqlite driver under the name "sqlite", which
	// go-rpmdb uses via sql.Open to read the sqlite rpmdb backend.
	_ "github.com/glebarez/go-sqlite"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
)

// rpmDBPath is the location of the system rpm database.
// DNF5 systems use the sqlite rpmdb backend at /usr/lib/sysimage/rpm.
const rpmDBPath = "/usr/lib/sysimage/rpm/rpmdb.sqlite"

// Package represents a single installed RPM package.
type Package struct {
	Name    string `json:"name"`
	Epoch   *int   `json:"epoch,omitempty"`
	Version string `json:"version"`
	Release string `json:"release"`
	Arch    string `json:"arch"`
	Vendor  string `json:"vendor"`
}

// listRpmPackages opens the rpm database at dbPath and lists installed packages.
// It is a var so tests can substitute canned records.
var listRpmPackages = func(dbPath string) ([]*rpmdb.PackageInfo, error) {
	db, err := rpmdb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open rpm database %s: %w", dbPath, err)
	}
	defer func() { _ = db.Close() }()

	pkgs, err := db.ListPackages()
	if err != nil {
		return nil, fmt.Errorf("unable to list packages from rpm database %s: %w", dbPath, err)
	}
	return pkgs, nil
}

// convertRpmPackages maps go-rpmdb records into our Package type.
func convertRpmPackages(pkgs []*rpmdb.PackageInfo) []Package {
	var packages []Package
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		packages = append(packages, Package{
			Name:    pkg.Name,
			Epoch:   pkg.Epoch,
			Version: pkg.Version,
			Release: pkg.Release,
			Arch:    pkg.Arch,
			Vendor:  pkg.Vendor,
		})
	}
	return packages
}

// GetInstalledPackages returns all RPM packages installed on the system by
// reading the rpm database directly via go-rpmdb.
func GetInstalledPackages() ([]Package, error) {
	pkgs, err := listRpmPackages(rpmDBPath)
	if err != nil {
		return nil, err
	}
	return convertRpmPackages(pkgs), nil
}
