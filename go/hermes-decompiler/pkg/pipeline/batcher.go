package pipeline

import (
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/utils"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.philip.id/mongopiet/pkg/bulk"
	"go.philip.id/mongopiet/pkg/db"
)

var CHANGE_COLLECTION = false

// DatabaseBatcher handles batched database operations to improve performance
type DatabaseBatcher struct {
	packages      []models.Package
	packageHashes []*models.PackageHash
	mutex         sync.Mutex
	batchSize     int
	rnVersion     string
}

// NewDatabaseBatcher creates a new database batcher with the specified batch size
func NewDatabaseBatcher(batchSize int, rnVersion string) *DatabaseBatcher {
	return &DatabaseBatcher{
		packages:      make([]models.Package, 0),
		packageHashes: make([]*models.PackageHash, 0),
		batchSize:     batchSize,
		rnVersion:     rnVersion,
	}
}

// AddPackage adds a package to the batch for insertion
func (dbBatcher *DatabaseBatcher) AddPackage(pkg models.Package) {
	dbBatcher.mutex.Lock()
	defer dbBatcher.mutex.Unlock()

	dbBatcher.packages = append(dbBatcher.packages, pkg)

	if len(dbBatcher.packages) >= dbBatcher.batchSize {
		dbBatcher.flushPackages()
	}
}

// AddPackageHash adds package hashes to the batch for insertion
func (dbBatcher *DatabaseBatcher) AddPackageHash(pkgHash *models.PackageHash) {
	dbBatcher.mutex.Lock()
	defer dbBatcher.mutex.Unlock()

	dbBatcher.packageHashes = append(dbBatcher.packageHashes, pkgHash)

	if len(dbBatcher.packageHashes) >= dbBatcher.batchSize {
		dbBatcher.flushPackageHashes()
	}
}

// Flush ensures all remaining batched operations are executed
func (dbBatcher *DatabaseBatcher) Flush() {
	dbBatcher.mutex.Lock()
	defer dbBatcher.mutex.Unlock()

	dbBatcher.flushPackages()
	dbBatcher.flushPackageHashes()
}

// flushPackages performs batch insertion of packages after filtering existing ones
func (dbBatcher *DatabaseBatcher) flushPackages() {
	if len(dbBatcher.packages) == 0 {
		return
	}

	utils.L.Printf("Flushing %d packages to database\n", len(dbBatcher.packages))

	// Pre-filter existing packages with single batch query
	existingIds := dbBatcher.getExistingPackageIds(dbBatcher.packages)
	newPackages := dbBatcher.filterNewPackages(dbBatcher.packages, existingIds)

	if len(newPackages) > 0 {
		writes := make([]bulk.Write, len(newPackages))
		for i, pkg := range newPackages {
			writes[i] = &bulk.Insert{Document: pkg}
		}

		collectionName := "packages"
		if CHANGE_COLLECTION {
			collectionName = "packages_ghsa"
		}

		_, err := db.BulkWrite(collectionName, writes)
		if err != nil {
			utils.F.Printf("Error in batch package insert: %v\n", err)
		} else {
			utils.S.Printf("Batch inserted %d packages\n", len(newPackages))
		}
	} else {
		utils.D.Printf("All %d packages already exist, skipping batch insert\n", len(dbBatcher.packages))
	}

	dbBatcher.packages = dbBatcher.packages[:0] // Reset slice
}

// flushPackageHashes performs batch insertion of package hashes after filtering existing ones
func (dbBatcher *DatabaseBatcher) flushPackageHashes() {
	if len(dbBatcher.packageHashes) == 0 {
		return
	}

	utils.L.Printf("Flushing %d package hash groups to database\n", len(dbBatcher.packageHashes))

	// Pre-check existing hashes in batch
	existingKeys := dbBatcher.getExistingHashKeys(dbBatcher.packageHashes)

	writes := []bulk.Write{}
	totalHashes := 0

	for _, pkgHash := range dbBatcher.packageHashes {
		key := fmt.Sprintf("%s_%s", pkgHash.PackageID.Hex(), pkgHash.ReactNativeVersion)
		if existingKeys[key] {
			utils.D.Printf("Package hash for %s already exists, skipping\n", key)
			continue // Skip existing
		}

		for _, hash := range pkgHash.Hashes {
			writes = append(writes, &bulk.Insert{
				Document: models.PackageHashModel{
					ID:                 primitive.NewObjectID(),
					PackageID:          pkgHash.PackageID,
					ReactNativeVersion: pkgHash.ReactNativeVersion,
					Hash:               hash,
				},
			})
			totalHashes++
		}
	}

	if len(writes) > 0 {

		collectionName := "hashes"
		if CHANGE_COLLECTION {
			collectionName = "hashes_ghsa"
		}

		_, err := db.BulkWrite(collectionName, writes)
		if err != nil {
			utils.F.Printf("Error in batch package hash insert: %v\n", err)
		} else {
			utils.S.Printf("Batch inserted %d package hashes\n", totalHashes)
		}
	} else {
		utils.D.Printf("All package hashes already exist, skipping batch insert\n")
	}

	dbBatcher.packageHashes = dbBatcher.packageHashes[:0] // Reset slice
}

// getExistingPackageIds queries for existing package unique IDs to avoid duplicates
func (dbBatcher *DatabaseBatcher) getExistingPackageIds(packages []models.Package) map[string]bool {
	if len(packages) == 0 {
		return make(map[string]bool)
	}

	uniqueIds := make([]string, len(packages))
	for i, pkg := range packages {
		uniqueIds[i] = pkg.PackageUniqueId
	}

	// Use simplified approach: check each package individually
	// This is more reliable than cursor-based approach for now
	existingIds := make(map[string]bool)
	for _, uniqueId := range uniqueIds {
		collectionName := "packages"
		if CHANGE_COLLECTION {
			collectionName = "packages_ghsa"
		}

		_, err := db.FindOne[models.Package](collectionName, bson.M{"package_unique_id": uniqueId})
		if err == nil {
			existingIds[uniqueId] = true
		}
	}

	utils.D.Printf("Found %d existing packages out of %d queried\n", len(existingIds), len(uniqueIds))
	return existingIds
}

// filterNewPackages returns only packages that don't already exist
func (dbBatcher *DatabaseBatcher) filterNewPackages(packages []models.Package, existingIds map[string]bool) []models.Package {
	var newPackages []models.Package
	for _, pkg := range packages {
		if !existingIds[pkg.PackageUniqueId] {
			newPackages = append(newPackages, pkg)
		}
	}
	return newPackages
}

// getExistingHashKeys queries for existing package hash keys to avoid duplicates
func (dbBatcher *DatabaseBatcher) getExistingHashKeys(packageHashes []*models.PackageHash) map[string]bool {
	if len(packageHashes) == 0 {
		return make(map[string]bool)
	}

	// Use simplified approach: check each package hash individually
	existingKeys := make(map[string]bool)
	for _, pkgHash := range packageHashes {
		collectionName := "hashes"
		if CHANGE_COLLECTION {
			collectionName = "hashes_ghsa"
		}

		_, err := db.FindOne[models.PackageHashModel](collectionName, bson.M{
			"package_id":           pkgHash.PackageID,
			"react_native_version": pkgHash.ReactNativeVersion,
		})
		if err == nil {
			key := fmt.Sprintf("%s_%s", pkgHash.PackageID.Hex(), pkgHash.ReactNativeVersion)
			existingKeys[key] = true
		}
	}

	utils.D.Printf("Found %d existing hash keys out of %d queried\n", len(existingKeys), len(packageHashes))
	return existingKeys
}
