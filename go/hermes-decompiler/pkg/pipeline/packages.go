package pipeline

import (
	"encoding/json"
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/utils"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.philip.id/mongopiet/pkg/bulk"
	"go.philip.id/mongopiet/pkg/db"
)

// A package in the react-native directory json file will have all its versions as an array of strings in the package_version field
// every map-field will map its values to the package version
type DirectoryPackageJson struct {
	PackageName             string                       `json:"package_name"`
	PackageVersion          []string                     `json:"package_version"`
	PackageAuthor           string                       `json:"package_author,omitempty"`
	PackageHomepage         string                       `json:"package_homepage,omitempty"`
	PackageRepository       string                       `json:"package_repository,omitempty"`
	PackageDependencies     map[string]map[string]string `json:"package_dependencies,omitempty"`
	PackageDevDependencies  map[string]map[string]string `json:"package_dev_dependencies,omitempty"`
	PackagePeerDependencies map[string]map[string]string `json:"package_peer_dependencies,omitempty"`
	PackageScripts          map[string]map[string]string `json:"package_scripts,omitempty"`
}

type PackageWithVulnerabilitiesJson struct {
	PackageName     string                  `json:"name"`
	PackageVersions []PackageVersionsStruct `json:"versions"`
}

type PackageVersionsStruct struct {
	Version      string   `json:"version"`
	AdvisoryUrls []string `json:"urls"`
}

func GetPackagesFromVulnerabilitiesJson(vulnerabilitiesJsonPath string) ([]models.PackageWithVulnerabilitiesModel, error) {
	vulnerabilitiesJson, err := os.ReadFile(vulnerabilitiesJsonPath)
	if err != nil {
		return nil, err
	}

	var packages []PackageWithVulnerabilitiesJson
	err = json.Unmarshal(vulnerabilitiesJson, &packages)
	if err != nil {
		return nil, err
	}

	return convertToPackageWithVulnerabilitiesModel(packages), nil
}

func convertToPackageWithVulnerabilitiesModel(packages []PackageWithVulnerabilitiesJson) []models.PackageWithVulnerabilitiesModel {
	var result []models.PackageWithVulnerabilitiesModel
	for _, pkg := range packages {
		for _, version := range pkg.PackageVersions {
			packageWithVulnerabilities := models.PackageWithVulnerabilitiesModel{
				ID:             primitive.NewObjectID(),
				PackageName:    pkg.PackageName,
				PackageVersion: version.Version,
				AdvisoryUrls:   version.AdvisoryUrls,
			}
			result = append(result, packageWithVulnerabilities)
		}
	}
	return result
}

// The react-native directory json file is an array of json objects, each object is a package
// meaning []DirectoryPackageJson
func GetPackagesFromReactNativeDirectoryJson(reactNativeDirectoryJsonPath string) (models.Packages, error) {
	reactNativeDirectoryJson, err := os.ReadFile(reactNativeDirectoryJsonPath)
	if err != nil {
		return nil, err
	}

	var packages []DirectoryPackageJson
	err = json.Unmarshal(reactNativeDirectoryJson, &packages)
	if err != nil {
		return nil, err
	}

	return convertDirectoryPackageJsonToPackage(packages), nil
}

// for every version of a package, we need to create a package model
func convertDirectoryPackageJsonToPackage(directoryPackageJson []DirectoryPackageJson) models.Packages {
	var result models.Packages

	for _, dirPkg := range directoryPackageJson {
		for _, version := range dirPkg.PackageVersion {
			// retrieve version-specific maps; they may be absent
			deps := getNestedMapValue(dirPkg.PackageDependencies, version)
			devDeps := getNestedMapValue(dirPkg.PackageDevDependencies, version)
			peerDeps := getNestedMapValue(dirPkg.PackagePeerDependencies, version)
			scripts := getNestedMapValue(dirPkg.PackageScripts, version)

			// Assemble the final Package model
			pkg := models.Package{
				ID:              primitive.NewObjectID(),
				PackageName:     dirPkg.PackageName,
				PackageVersion:  version,
				PackageUniqueId: dirPkg.PackageName + "@" + version,
				Metadata: models.Metadata{
					PackageAuthor:           dirPkg.PackageAuthor,
					PackageHomepage:         dirPkg.PackageHomepage,
					PackageRepository:       dirPkg.PackageRepository,
					PackageDependencies:     deps,
					PackageDevDependencies:  devDeps,
					PackagePeerDependencies: peerDeps,
					PackageScripts:          scripts,
				},
				ErrorLog: models.ErrorLog{
					HasError: false,
					Error:    "",
				},
			}

			result = append(result, pkg)
		}
	}

	return result
}

// getNestedMapValue safely fetches the value for a version key from a map-of-maps.
// It returns nil if the outer map or the requested inner map is absent.
func getNestedMapValue(m map[string]map[string]string, version string) map[string]string {
	if m == nil {
		return nil
	}
	if val, ok := m[version]; ok {
		return val
	}
	return nil
}

func AddPackageToDatabase(pkg *models.Package) error { // TODO: need to add check about react native version here
	// check if package already exists
	existingPkg, err := db.FindOne[models.Package]("packages", bson.M{"package_unique_id": pkg.PackageUniqueId})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err = db.InsertOne("packages", pkg)
			if err != nil {
				return fmt.Errorf("error adding package to database: %v", err)
			}
			utils.S.Println("Package added to database " + pkg.PackageUniqueId)
			return nil
		} else {
			return fmt.Errorf("error checking if package [%s] already exists: %v", pkg.PackageUniqueId, err)
		}
	} else {
		if existingPkg != nil {
			utils.D.Printf("Package [%s] already exists, skipping...\n", pkg.PackageUniqueId)
			return fmt.Errorf("package [%s] already exists", pkg.PackageUniqueId)
		}
	}

	return nil
}

func AddPackageHashesToDatabase(pkgHashes *models.PackageHash, packageUniqueId string) error {
	utils.L.Println("Adding package hashes to database " + packageUniqueId)

	// check if package hashes already exists
	existingPkg, err := db.FindOne[models.PackageHash]("hashes", bson.M{"package_id": pkgHashes.PackageID, "react_native_version": pkgHashes.ReactNativeVersion})
	if err != nil {
		if err == mongo.ErrNoDocuments {

			write := []bulk.Write{}

			collectionName := "hashes"
			if CHANGE_COLLECTION {
				collectionName = "hashes_ghsa"
			}

			for i, hashObj := range pkgHashes.Hashes {
				write = append(write, &bulk.Insert{Document: models.PackageHashModel{
					ID:                 primitive.NewObjectID(),
					PackageID:          pkgHashes.PackageID,
					ReactNativeVersion: pkgHashes.ReactNativeVersion,
					Hash:               hashObj,
				}})

				if i%1000 == 0 && i > 0 {

					_, err = db.BulkWrite(collectionName, write)
					if err != nil {
						return fmt.Errorf("error adding package hashes to database: %v", err)
					}
					write = []bulk.Write{}
				}
			}

			_, err = db.BulkWrite(collectionName, write)
			if err != nil {
				return fmt.Errorf("error adding package hashes to database: %v", err)
			}
			utils.S.Println("Package hashes added to database " + packageUniqueId)
			return nil
		} else {
			return fmt.Errorf("error checking if package hashes [%s] already exists: %v", packageUniqueId, err)
		}
	} else {
		if existingPkg != nil {
			utils.D.Printf("Package hashes [%s] already exists, skipping...\n", packageUniqueId)
			return nil
		}
	}

	return nil
}

func UpdatePackageErrorLog(pkg *models.Package, loggedError error) error {
	pkg.ErrorLog.HasError = true
	pkg.ErrorLog.Error = loggedError.Error()

	_, err := db.UpdateOne("packages", bson.M{"package_unique_id": pkg.PackageUniqueId}, pkg, nil)
	if err != nil {
		return fmt.Errorf("error updating package error log: %v", err)
	}

	return nil
}
