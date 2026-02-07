package pipeline

import (
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ProcessPackagesForRNVersion handles the processing of all packages for a specific React Native version
// This function is designed to be run in parallel for different RN versions
// Uses optimized package batching to reduce npm install/uninstall churn and database batching for efficiency
func ProcessPackagesForRNVersion(rnDir os.DirEntry, packages models.Packages, rnRoot string) ([]*models.PackageHash, error) {
	name := rnDir.Name()

	// Extract numeric part and convert it into a React Native version string
	digits := strings.TrimPrefix(name, "rn")
	num, err := strconv.Atoi(digits)
	if err != nil {
		utils.D.Printf("skipping folder %s: cannot parse version\n", name)
		return nil, fmt.Errorf("cannot parse version from folder %s: %v", name, err)
	}

	version := fmt.Sprintf("%.2f", float64(num)/100)
	reactNativeWorkingDirectoryPath := filepath.Join(rnRoot, name)
	hermesCompilerPath := filepath.Join(reactNativeWorkingDirectoryPath, "node_modules", "react-native", "sdks", "hermesc", "osx-bin", "hermesc")

	utils.L.Printf("Processing React Native version %s with %d packages\n", version, len(packages))

	// OPTIMIZATION Phase 3: Create database batcher for this RN environment
	batcher := NewDatabaseBatcher(100, version) // Updated batch size to 1000
	defer batcher.Flush()                       // Ensure all batched operations complete

	// OPTIMIZATION Phase 2: Group packages by name to minimize npm install/uninstall churn
	packageGroups := GroupPackagesByName(packages)
	utils.L.Printf("Grouped %d packages into %d package names for RN %s\n", len(packages), len(packageGroups), version)

	var allPackageHashes []*models.PackageHash
	var allErrors []error

	// Process packages grouped by name to reduce npm churn
	for packageName, packageVersions := range packageGroups {
		utils.L.Printf("Processing package group: %s (%d versions) for RN %s\n", packageName, len(packageVersions), version)

		packageHashes, errors := ProcessPackageVersionsBatchedWithBatcher(packageName, packageVersions, reactNativeWorkingDirectoryPath, hermesCompilerPath, version, batcher)

		// Collect results
		allPackageHashes = append(allPackageHashes, packageHashes...)
		allErrors = append(allErrors, errors...)

		if len(errors) > 0 {
			utils.F.Printf("Package group %s had %d errors out of %d versions\n", packageName, len(errors), len(packageVersions))
		}
	}

	// Log summary
	totalPackagesProcessed := len(packages)
	successfulHashes := len(allPackageHashes)
	totalErrors := len(allErrors)

	utils.S.Printf("Completed React Native version %s:\n", version)
	utils.S.Printf("  - Package groups processed: %d\n", len(packageGroups))
	utils.S.Printf("  - Total packages: %d\n", totalPackagesProcessed)
	utils.S.Printf("  - Successful hashes: %d\n", successfulHashes)
	utils.S.Printf("  - Total errors: %d\n", totalErrors)
	utils.S.Printf("  - Success rate: %.1f%%\n", float64(successfulHashes)/float64(totalPackagesProcessed)*100)

	// Return error only if NO packages were processed successfully
	if len(allPackageHashes) == 0 && len(allErrors) > 0 {
		return nil, fmt.Errorf("all packages failed for RN version %s: %d errors", version, len(allErrors))
	}

	return allPackageHashes, nil
}

// ProcessPackagesForRNVersionWithProgress orchestrates the fingerprinting pipeline for all
// pending packages within a single React Native environment, adding progress tracking and
// resume capability on top of the core processing logic.
//
// Before processing begins, it creates a pristine backup of the RN environment's node_modules
// via CreateBackup, then filters out already-completed packages using FilterPendingPackages
// and the supplied rnProgress state. Remaining packages are grouped by name via
// GroupPackagesByName to minimize npm install/uninstall churn, and each group is processed
// by ProcessPackageVersionsBatchedWithProgress with environment restoration (RestoreFromBackup)
// before every group to ensure a clean slate. Database writes are batched (100 ops/batch)
// via NewDatabaseBatcher for efficiency.
//
// If a backup restoration fails for a package group, all versions in that group are marked
// as failed in rnProgress and processing continues with the next group.
//
// Parameters:
//   - rnDir: directory entry for the React Native environment (e.g. "rn075"); the numeric
//     suffix is parsed into a version string (e.g. "0.75").
//   - packages: full list of packages to process; already-completed packages are filtered
//     out internally via rnProgress.
//   - rnRoot: parent directory containing all rnXXX environment directories.
//   - rnProgress: mutable progress tracker for this RN version; updated in place with
//     completed/failed package status for JSON-based resume capability.
//   - osHermes: Hermes binary platform directory name ("osx-bin" or "linux64-bin");
//     defaults to "osx-bin" if empty.
//
// Returns the collected PackageHash results for all successfully processed packages.
// Returns a non-nil error only if every package fails; partial failures are recorded
// in rnProgress and do not cause an error return.
func ProcessPackagesForRNVersionWithProgress(rnDir os.DirEntry, packages models.Packages, rnRoot string, rnProgress *RNEnvironmentProgress, osHermes string) ([]*models.PackageHash, error) {
	name := rnDir.Name()

	// Extract numeric part and convert it into a React Native version string
	digits := strings.TrimPrefix(name, "rn")
	num, err := strconv.Atoi(digits)
	if err != nil {
		utils.D.Printf("skipping folder %s: cannot parse version\n", name)
		return nil, fmt.Errorf("cannot parse version from folder %s: %v", name, err)
	}

	version := fmt.Sprintf("%.2f", float64(num)/100)
	reactNativeWorkingDirectoryPath := filepath.Join(rnRoot, name)
	if osHermes == "" {
		osHermes = "osx-bin"
	}
	hermesCompilerPath := filepath.Join(reactNativeWorkingDirectoryPath, "node_modules", "react-native", "sdks", "hermesc", osHermes, "hermesc")

	// Create a pristine backup of the environment before starting
	if err := CreateBackup(reactNativeWorkingDirectoryPath); err != nil {
		return nil, fmt.Errorf("failed to create environment backup for RN %s: %w", version, err)
	}

	// PHASE 4: Filter out already completed packages
	pendingPackages := FilterPendingPackages(packages, rnProgress)

	utils.L.Printf("Processing React Native version %s: %d pending packages (total %d)\n",
		version, len(pendingPackages), len(packages))

	// PHASE 4: Initialize progress if this is a fresh start
	if rnProgress.TotalPackages == 0 {
		rnProgress.InitializeRNProgress(len(packages))
	}

	// PHASE 3: Create database batcher for this RN environment
	batcher := NewDatabaseBatcher(100, version) // Updated batch size to 1000
	defer batcher.Flush()                       // Ensure all batched operations complete

	// PHASE 2: Group packages by name to minimize npm install/uninstall churn
	packageGroups := GroupPackagesByName(pendingPackages)
	utils.L.Printf("Grouped %d pending packages into %d package names for RN %s\n",
		len(pendingPackages), len(packageGroups), version)

	var allPackageHashes []*models.PackageHash
	var allErrors []error

	// Process packages grouped by name to reduce npm churn
	for packageName, packageVersions := range packageGroups {
		utils.L.Printf("Processing package group: %s (%d versions) for RN %s\n",
			packageName, len(packageVersions), version)

		// Restore the pristine environment before processing each package group
		if err := RestoreFromBackup(reactNativeWorkingDirectoryPath); err != nil {
			utils.F.Printf("FATAL: could not restore backup for %s: %v. Skipping package group.\n", packageName, err)
			for _, pkg := range packageVersions {
				rnProgress.MarkPackageFailed(pkg.PackageUniqueId, err.Error())
			}
			continue // Move to the next package group
		}

		packageHashes, errors := ProcessPackageVersionsBatchedWithProgress(packageName, packageVersions,
			reactNativeWorkingDirectoryPath, hermesCompilerPath, version, batcher, rnProgress)

		// Collect results
		allPackageHashes = append(allPackageHashes, packageHashes...)
		allErrors = append(allErrors, errors...)

		if len(errors) > 0 {
			utils.F.Printf("Package group %s had %d errors out of %d versions\n",
				packageName, len(errors), len(packageVersions))
		}
	}

	// Log summary with progress information
	utils.S.Printf("Completed React Native version %s: %s\n", version, rnProgress.GetProgressSummary())

	// Return error only if NO packages were processed successfully
	if len(allPackageHashes) == 0 && len(allErrors) > 0 {
		return nil, fmt.Errorf("all packages failed for RN version %s: %d errors", version, len(allErrors))
	}

	return allPackageHashes, nil
}
