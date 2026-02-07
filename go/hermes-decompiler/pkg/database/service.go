package database

import (
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.philip.id/mongopiet/pkg/bulk"
	"go.philip.id/mongopiet/pkg/db"
)

// AddPackageToDatabase inserts a package record into the "packages" MongoDB collection
// if no document with the same package_name and package_version already exists. It
// performs a find-before-insert to enforce uniqueness: if the package is not found
// (mongo.ErrNoDocuments), it inserts the document; if it already exists, it returns an
// error indicating the duplicate. Any unexpected query or insert failure is wrapped and
// returned.
func AddPackageToDatabase(pkg *models.PackageWithVulnerabilitiesModel) error {

	existingPkg, err := db.FindOne[models.PackageWithVulnerabilitiesModel]("packages", bson.M{"package_name": pkg.PackageName, "package_version": pkg.PackageVersion})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err = db.InsertOne("packages", pkg)
			if err != nil {
				return fmt.Errorf("error adding package to database: %v", err)
			}
			utils.S.Println("Package added to database " + pkg.PackageName + "@" + pkg.PackageVersion)
			return nil
		} else {
			return fmt.Errorf("error checking if package [%s] already exists: %v", pkg.PackageName+"@"+pkg.PackageVersion, err)
		}
	} else {
		if existingPkg != nil {
			utils.D.Printf("Package [%s] already exists, skipping...\n", pkg.PackageName+"@"+pkg.PackageVersion)
			return fmt.Errorf("package [%s] already exists", pkg.PackageName+"@"+pkg.PackageVersion)
		}
	}

	return nil
}

// checkIfPackageHasHashes checks if the package has hashes for a given react native version
func CheckIfPackageHasHashes(pkg *models.PackageWithVulnerabilitiesModel, rnVersion string) (bool, error) {
	existingHash, err := db.FindOne[models.FunctionObjectHashModel]("hashes", bson.M{"package_id": pkg.ID, "react_native_version": rnVersion})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, fmt.Errorf("error checking if package has hashes: %v", err)
	}

	if existingHash != nil {
		return true, nil
	}

	return false, nil
}

// AddHashesToDatabase adds hashes to the database in a bulk operation by batching them with 2000 documents at a time
func AddHashesToDatabase(hashes []*models.FunctionObjectHashModel) error {

	models := make([]bulk.Write, len(hashes))

	for i, hash := range hashes {
		hash.ID = primitive.NewObjectID()
		models[i] = &bulk.Insert{Document: hash}
	}

	utils.L.Printf("Adding %d hashes to database\n", len(hashes))

	_, err := db.BulkWrite("hashes", models)
	if err != nil {
		return fmt.Errorf("error adding hashes to database: %v", err)
	}

	return nil
}

func GetPackages() ([]*models.PackageWithVulnerabilitiesModel, error) {
	return db.Find[models.PackageWithVulnerabilitiesModel]("packages", bson.M{})
}

func UpdatePackageErrorLog(pkg *models.PackageWithVulnerabilitiesModel) error {
	_, err := db.UpdateOne("packages", bson.M{"_id": pkg.ID}, bson.M{"has_error": pkg.HasError, "error": pkg.Error}, nil)
	if err != nil {
		return fmt.Errorf("error updating package error log: %v", err)
	}

	return nil
}
