package pipeline

import (
	"encoding/json"
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/utils"
	"os"
	"strings"
	"sync"
	"time"
)

// RNEnvironmentProgress tracks the progress of processing packages for a specific React Native version
type RNEnvironmentProgress struct {
	RNVersion         string            `json:"rn_version"`
	CompletedPackages map[string]bool   `json:"completed_packages"` // package@version -> bool
	FailedPackages    map[string]string `json:"failed_packages"`    // package@version -> error message
	TotalPackages     int               `json:"total_packages"`
	CompletedCount    int               `json:"completed_count"`
	FailedCount       int               `json:"failed_count"`
	StartTime         time.Time         `json:"start_time"`
	EndTime           *time.Time        `json:"end_time,omitempty"`
	IsCompleted       bool              `json:"is_completed"`
}

// GlobalProgress tracks the overall progress across all RN environments
type GlobalProgress struct {
	mutex               sync.RWMutex                      `json:"-"` // Protects map access
	RNEnvironments      map[string]*RNEnvironmentProgress `json:"rn_environments"`
	TotalRNVersions     int                               `json:"total_rn_versions"`
	CompletedRNVersions int                               `json:"completed_rn_versions"`
	StartTime           time.Time                         `json:"start_time"`
	LastUpdated         time.Time                         `json:"last_updated"`
	saveFile            string                            `json:"-"` // Don't serialize the file path
}

// NewGlobalProgress creates a new global progress tracker
func NewGlobalProgress(saveFile string) *GlobalProgress {
	return &GlobalProgress{
		RNEnvironments:      make(map[string]*RNEnvironmentProgress),
		TotalRNVersions:     0,
		CompletedRNVersions: 0,
		StartTime:           time.Now(),
		LastUpdated:         time.Now(),
		saveFile:            saveFile,
	}
}

// LoadProgress loads existing progress from file, returns error if file doesn't exist
func (gp *GlobalProgress) LoadProgress() error {
	data, err := os.ReadFile(gp.saveFile)
	if err != nil {
		if os.IsNotExist(err) {
			utils.L.Printf("No existing progress file found at %s, starting fresh\n", gp.saveFile)
			return nil // File doesn't exist, start fresh
		}
		return fmt.Errorf("error reading progress file: %v", err)
	}

	err = json.Unmarshal(data, gp)
	if err != nil {
		return fmt.Errorf("error parsing progress file: %v", err)
	}

	utils.L.Printf("Loaded progress: %d/%d RN versions completed\n", gp.CompletedRNVersions, gp.TotalRNVersions)
	return nil
}

// SaveProgress saves current progress to file
func (gp *GlobalProgress) SaveProgress() error {
	gp.LastUpdated = time.Now()

	// Lock for reading during serialization
	gp.mutex.RLock()
	data, err := json.MarshalIndent(gp, "", "  ")
	gp.mutex.RUnlock()

	if err != nil {
		return fmt.Errorf("error marshaling progress: %v", err)
	}

	err = os.WriteFile(gp.saveFile, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing progress file: %v", err)
	}

	utils.D.Printf("Progress saved to %s\n", gp.saveFile)
	return nil
}

// GetOrCreateRNProgress gets existing RN progress or creates a new one
func (gp *GlobalProgress) GetOrCreateRNProgress(rnDirName string) *RNEnvironmentProgress {
	// First try with read lock to see if it exists
	gp.mutex.RLock()
	if progress, exists := gp.RNEnvironments[rnDirName]; exists {
		gp.mutex.RUnlock()
		return progress
	}
	gp.mutex.RUnlock()

	// Need to create new progress, get write lock
	gp.mutex.Lock()
	defer gp.mutex.Unlock()

	// Double-check in case another goroutine created it while we were waiting
	if progress, exists := gp.RNEnvironments[rnDirName]; exists {
		return progress
	}

	// Create new RN environment progress
	progress := &RNEnvironmentProgress{
		RNVersion:         rnDirName,
		CompletedPackages: make(map[string]bool),
		FailedPackages:    make(map[string]string),
		TotalPackages:     0,
		CompletedCount:    0,
		FailedCount:       0,
		StartTime:         time.Now(),
		IsCompleted:       false,
	}

	gp.RNEnvironments[rnDirName] = progress
	return progress
}

// MarkRNVersionComplete marks an RN version as completed
func (gp *GlobalProgress) MarkRNVersionComplete(rnDirName string) {
	gp.mutex.RLock()
	progress, exists := gp.RNEnvironments[rnDirName]
	gp.mutex.RUnlock()

	if exists && !progress.IsCompleted {
		progress.IsCompleted = true
		endTime := time.Now()
		progress.EndTime = &endTime

		// This increment needs to be protected too
		gp.mutex.Lock()
		gp.CompletedRNVersions++
		gp.mutex.Unlock()

		utils.S.Printf("RN version %s completed: %d successful, %d failed packages\n",
			rnDirName, progress.CompletedCount, progress.FailedCount)
	}
}

// InitializeRNProgress sets up the total package count for an RN environment
func (rnProgress *RNEnvironmentProgress) InitializeRNProgress(totalPackages int) {
	rnProgress.TotalPackages = totalPackages
	rnProgress.StartTime = time.Now()

	utils.L.Printf("Initialized RN %s progress: %d total packages\n", rnProgress.RNVersion, totalPackages)
}

// MarkPackageCompleted marks a package as successfully completed
func (rnProgress *RNEnvironmentProgress) MarkPackageCompleted(packageUniqueId string) {
	if !rnProgress.CompletedPackages[packageUniqueId] {
		rnProgress.CompletedPackages[packageUniqueId] = true
		rnProgress.CompletedCount++

		// Remove from failed if it was previously failed
		if _, wasFailed := rnProgress.FailedPackages[packageUniqueId]; wasFailed {
			delete(rnProgress.FailedPackages, packageUniqueId)
			rnProgress.FailedCount--
		}

		utils.D.Printf("Package %s completed (%d/%d) [RN %s]\n", packageUniqueId, rnProgress.CompletedCount, rnProgress.TotalPackages, rnProgress.RNVersion)
	}
}

// MarkPackageFailed marks a package as failed with error message
func (rnProgress *RNEnvironmentProgress) MarkPackageFailed(packageUniqueId string, errorMsg string) {
	if !rnProgress.CompletedPackages[packageUniqueId] {
		if _, alreadyFailed := rnProgress.FailedPackages[packageUniqueId]; !alreadyFailed {
			rnProgress.FailedPackages[packageUniqueId] = errorMsg
			rnProgress.FailedCount++
		} else {
			// Update error message for existing failure
			rnProgress.FailedPackages[packageUniqueId] = errorMsg
		}

		utils.D.Printf("Package %s failed: %s (%d failed total)\n", packageUniqueId, errorMsg, rnProgress.FailedCount)
	}
}

// IsPackageCompleted checks if a package has already been successfully processed
func (rnProgress *RNEnvironmentProgress) IsPackageCompleted(packageUniqueId string) bool {
	return rnProgress.CompletedPackages[packageUniqueId]
}

// ShouldSkipPackage determines if a package should be skipped (already completed)
func (rnProgress *RNEnvironmentProgress) ShouldSkipPackage(packageUniqueId string) bool {
	return rnProgress.IsPackageCompleted(packageUniqueId)
}

// GetProgressSummary returns a human-readable progress summary
func (rnProgress *RNEnvironmentProgress) GetProgressSummary() string {
	completionRate := float64(rnProgress.CompletedCount) / float64(rnProgress.TotalPackages) * 100

	duration := time.Since(rnProgress.StartTime)
	if rnProgress.EndTime != nil {
		duration = rnProgress.EndTime.Sub(rnProgress.StartTime)
	}

	return fmt.Sprintf("RN %s: %d/%d completed (%.1f%%), %d failed, duration: %v",
		rnProgress.RNVersion, rnProgress.CompletedCount, rnProgress.TotalPackages,
		completionRate, rnProgress.FailedCount, duration.Round(time.Second))
}

// FilterPendingRNVersions filters out RN versions that are already completed
func FilterPendingRNVersions(allRNDirs []os.DirEntry, globalProgress *GlobalProgress) []os.DirEntry {
	var pendingDirs []os.DirEntry

	for _, rnDir := range allRNDirs {
		if !rnDir.IsDir() || !strings.HasPrefix(rnDir.Name(), "rn") {
			continue
		}

		// Check if this RN version is already completed
		globalProgress.mutex.RLock()
		progress, exists := globalProgress.RNEnvironments[rnDir.Name()]
		globalProgress.mutex.RUnlock()

		if exists && progress.IsCompleted {
			utils.L.Printf("RN version %s already completed, skipping\n", rnDir.Name())
			continue
		}

		pendingDirs = append(pendingDirs, rnDir)
	}

	utils.L.Printf("Filtered RN versions: %d pending out of %d total\n", len(pendingDirs), len(allRNDirs))
	return pendingDirs
}

// FilterPendingPackages filters out packages that are already completed for a specific RN version
func FilterPendingPackages(allPackages models.Packages, rnProgress *RNEnvironmentProgress) models.Packages {
	var pendingPackages models.Packages

	for _, pkg := range allPackages {
		if !rnProgress.ShouldSkipPackage(pkg.PackageUniqueId) {
			pendingPackages = append(pendingPackages, pkg)
		}
	}

	skippedCount := len(allPackages) - len(pendingPackages)
	utils.L.Printf("Filtered packages for RN %s: %d pending, %d already completed\n",
		rnProgress.RNVersion, len(pendingPackages), skippedCount)

	return pendingPackages
}
