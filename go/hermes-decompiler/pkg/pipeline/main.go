package pipeline

import (
	"crypto/sha256"
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/global"
	"hermes-decompiler/pkg/hbc"
	"hermes-decompiler/pkg/hbc/types"
	"hermes-decompiler/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.philip.id/mongopiet/pkg/db"
)

// CheckPackageHashesExistInDatabase checks if package hashes already exist for a given package and RN version
// This prevents unnecessary pipeline processing when the work is already done
func CheckPackageHashesExistInDatabase(packageUniqueId, rnVersion string) (bool, error) {
	// Step 1: Find the package in the database to get its actual ID
	existingPkg, err := db.FindOne[models.Package]("packages", bson.M{"package_unique_id": packageUniqueId})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Package doesn't exist in database yet, so hashes definitely don't exist
			return false, nil
		}
		return false, fmt.Errorf("error checking package existence: %v", err)
	}

	// Step 2: Check if hashes exist for this package ID + RN version combination
	_, err = db.FindOne[models.PackageHashModel]("hashes", bson.M{
		"package_id":           existingPkg.ID,
		"react_native_version": rnVersion,
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Hashes don't exist for this package + RN version combination
			return false, nil
		}
		return false, fmt.Errorf("error checking package hashes existence: %v", err)
	}

	// Hashes exist!
	return true, nil
}

// Two pipelines: Baseline ()
// Baseline:
// - Create a baseline database entry for a given React Native version
// - Add the baseline database entry to the database
// Package:
// - Create a package database entry for a given package for every react native and hermes version
// - Add the package database entry to the database

// GroupPackagesByName groups packages by their name to minimize npm install/uninstall churn
func GroupPackagesByName(packages models.Packages) map[string][]models.Package {
	groups := make(map[string][]models.Package)
	for _, pkg := range packages {
		groups[pkg.PackageName] = append(groups[pkg.PackageName], pkg)
	}
	return groups
}

// InstallPackage installs a single package version
func InstallPackage(workingDir, packageName, packageVersion string) error {
	utils.L.Printf("Installing package %s@%s\n", packageName, packageVersion)
	installPackageCommand := exec.Command("npm", "install", packageName+"@"+packageVersion, "--no-audit")
	installPackageCommand.Dir = workingDir
	if global.Verbose {
		installPackageCommand.Stdout = os.Stdout
		installPackageCommand.Stderr = os.Stderr
	}
	err := installPackageCommand.Run()
	if err != nil {
		return fmt.Errorf("error installing package [%s@%s]: %v", packageName, packageVersion, err)
	}
	utils.S.Printf("Package %s@%s installed successfully\n", packageName, packageVersion)
	return nil
}

// RunSinglePackagePipeline runs the pipeline for a single package without database operations
// This is used for optimized batch processing where database operations are handled separately
func RunSinglePackagePipeline(reactNativeWorkingDirectoryPath, hermesCompilerPath, version string, pkg models.Package) (*models.PackageHash, error) {
	bundlePath, err := UseMetroToBundleOptimized(reactNativeWorkingDirectoryPath, version, pkg.PackageName, pkg.PackageVersion)
	if err != nil {
		return nil, fmt.Errorf("error using Metro to bundle: %v", err)
	}

	compiledBundlePath, hermesVersion, err := UseHermesToCompile(bundlePath, hermesCompilerPath, reactNativeWorkingDirectoryPath, version, false)
	if err != nil {
		return nil, fmt.Errorf("error using Hermes to compile: %v", err)
	}

	fois, err := UseHermesDecompilerToDisassemble(filepath.Join(reactNativeWorkingDirectoryPath, compiledBundlePath))
	if err != nil {
		return nil, fmt.Errorf("error disassembling Hermes bytecode file: %v", err)
	}

	return CreatePackageHashesDatabaseEntry(fois, version, hermesVersion, pkg), nil
}

// ProcessPackageVersionsBatched processes all versions of a single package name in batch
// This reduces npm install/uninstall churn by processing all versions of a package together
func ProcessPackageVersionsBatched(packageName string, packageVersions []models.Package, workingDir, hermesPath, rnVersion string) ([]*models.PackageHash, []error) {
	var results []*models.PackageHash
	var errors []error

	utils.L.Printf("Processing package %s with %d versions for RN %s\n", packageName, len(packageVersions), rnVersion)

	for _, pkg := range packageVersions {
		// Add package to database first
		err := AddPackageToDatabase(&pkg)
		if err != nil {
			utils.F.Printf("error adding package to database for RN %s: %v\n", rnVersion, err)
			errors = append(errors, fmt.Errorf("database error for %s: %v", pkg.PackageUniqueId, err))
			continue
		}

		// Install package version
		err = InstallPackage(workingDir, pkg.PackageName, pkg.PackageVersion)
		if err != nil {
			utils.F.Printf("error installing package %s: %v\n", pkg.PackageUniqueId, err)
			UpdatePackageErrorLog(&pkg, err)
			errors = append(errors, fmt.Errorf("install error for %s: %v", pkg.PackageUniqueId, err))
			continue
		}

		// Process for this RN version
		pkgHash, err := RunSinglePackagePipeline(workingDir, hermesPath, rnVersion, pkg)
		if err != nil {
			utils.F.Printf("error running pipeline for package %s: %v\n", pkg.PackageUniqueId, err)
			UpdatePackageErrorLog(&pkg, err)
			errors = append(errors, fmt.Errorf("pipeline error for %s: %v", pkg.PackageUniqueId, err))
		} else {
			// Add package hashes to database
			err = AddPackageHashesToDatabase(pkgHash, pkg.PackageUniqueId)
			if err != nil {
				utils.F.Printf("error adding package hashes to database: %v\n", err)
				UpdatePackageErrorLog(&pkg, err)
				errors = append(errors, fmt.Errorf("database hash error for %s: %v", pkg.PackageUniqueId, err))
			} else {
				results = append(results, pkgHash)
			}
		}
	}

	utils.S.Printf("Completed package %s: processed %d versions, %d successful, %d errors\n",
		packageName, len(packageVersions), len(results), len(errors))

	return results, errors
}

// ProcessPackageVersionsBatchedWithBatcher processes all versions of a single package name using database batcher
// This reduces both npm install/uninstall churn AND database operation overhead
func ProcessPackageVersionsBatchedWithBatcher(packageName string, packageVersions []models.Package, workingDir, hermesPath, rnVersion string, batcher *DatabaseBatcher) ([]*models.PackageHash, []error) {
	var results []*models.PackageHash
	var errors []error

	utils.L.Printf("Processing package %s with %d versions for RN %s (with database batcher)\n", packageName, len(packageVersions), rnVersion)

	for _, pkg := range packageVersions {
		// DATABASE CHECK: Check if package hashes already exist in database before running pipeline
		hashesExist, err := CheckPackageHashesExistInDatabase(pkg.PackageUniqueId, rnVersion)
		if err != nil {
			utils.F.Printf("Error checking database for package %s: %v\n", pkg.PackageUniqueId, err)
			// Continue processing since we can't determine if it exists
		} else if hashesExist {
			utils.D.Printf("Package %s already exists in database for RN %s, skipping pipeline\n", pkg.PackageUniqueId, rnVersion)
			continue
		}

		// Add package to batcher instead of direct database
		batcher.AddPackage(pkg)

		// Install package version
		err = InstallPackage(workingDir, pkg.PackageName, pkg.PackageVersion)
		if err != nil {
			utils.F.Printf("error installing package %s: %v\n", pkg.PackageUniqueId, err)
			// Update error log directly since we can't use batcher for this
			UpdatePackageErrorLog(&pkg, err)
			errors = append(errors, fmt.Errorf("install error for %s: %v", pkg.PackageUniqueId, err))
			continue
		}

		// Process for this RN version
		pkgHash, err := RunSinglePackagePipeline(workingDir, hermesPath, rnVersion, pkg)
		if err != nil {
			utils.F.Printf("error running pipeline for package %s: %v\n", pkg.PackageUniqueId, err)
			UpdatePackageErrorLog(&pkg, err)
			errors = append(errors, fmt.Errorf("pipeline error for %s: %v", pkg.PackageUniqueId, err))
		} else {
			// Add package hashes to batcher instead of direct database
			batcher.AddPackageHash(pkgHash)
			results = append(results, pkgHash)
		}
	}

	utils.S.Printf("Completed package %s with batcher: processed %d versions, %d successful, %d errors\n",
		packageName, len(packageVersions), len(results), len(errors))

	return results, errors
}

// ProcessPackageVersionsBatchedWithProgress processes all versions of a single package name with progress tracking
// This reduces npm install/uninstall churn, database operation overhead AND provides resume capability
func ProcessPackageVersionsBatchedWithProgress(packageName string, packageVersions []models.Package, workingDir, hermesPath, rnVersion string, batcher *DatabaseBatcher, rnProgress *RNEnvironmentProgress) ([]*models.PackageHash, []error) {
	var results []*models.PackageHash
	var errors []error

	utils.L.Printf("Processing package %s with %d versions for RN %s (with progress tracking)\n", packageName, len(packageVersions), rnVersion)

	for _, pkg := range packageVersions {
		// PHASE 4: Check if this package is already completed
		if rnProgress.ShouldSkipPackage(pkg.PackageUniqueId) {
			utils.D.Printf("Package %s already completed, skipping\n", pkg.PackageUniqueId)
			continue
		}

		// DATABASE CHECK: Check if package hashes already exist in database before running pipeline
		hashesExist, err := CheckPackageHashesExistInDatabase(pkg.PackageUniqueId, rnVersion)
		if err != nil {
			utils.F.Printf("Error checking database for package %s: %v\n", pkg.PackageUniqueId, err)
			// Continue processing since we can't determine if it exists
		} else if hashesExist {
			utils.D.Printf("Package %s already exists in database for RN %s, skipping pipeline\n", pkg.PackageUniqueId, rnVersion)
			// Mark as completed in progress since it's already in database
			rnProgress.MarkPackageCompleted(pkg.PackageUniqueId)
			continue
		}

		// Add package to batcher instead of direct database
		batcher.AddPackage(pkg)

		// Install package version
		err = InstallPackage(workingDir, pkg.PackageName, pkg.PackageVersion)
		if err != nil {
			utils.F.Printf("error installing package %s: %v\n", pkg.PackageUniqueId, err)
			// PHASE 4: Mark package as failed in progress
			rnProgress.MarkPackageFailed(pkg.PackageUniqueId, err.Error())
			// Update error log directly since we can't use batcher for this
			UpdatePackageErrorLog(&pkg, err)
			errors = append(errors, fmt.Errorf("install error for %s: %v", pkg.PackageUniqueId, err))
			continue
		}

		// Process for this RN version
		pkgHash, err := RunSinglePackagePipeline(workingDir, hermesPath, rnVersion, pkg)
		if err != nil {
			utils.F.Printf("error running pipeline for package %s: %v\n", pkg.PackageUniqueId, err)
			// PHASE 4: Mark package as failed in progress
			rnProgress.MarkPackageFailed(pkg.PackageUniqueId, err.Error())
			UpdatePackageErrorLog(&pkg, err)
			errors = append(errors, fmt.Errorf("pipeline error for %s: %v", pkg.PackageUniqueId, err))
		} else {
			// Add package hashes to batcher instead of direct database
			batcher.AddPackageHash(pkgHash)
			results = append(results, pkgHash)
			// PHASE 4: Mark package as completed in progress
			rnProgress.MarkPackageCompleted(pkg.PackageUniqueId)
		}
	}

	utils.S.Printf("Completed package %s with progress: processed %d versions, %d successful, %d errors\n",
		packageName, len(packageVersions), len(results), len(errors))

	return results, errors
}

// RunBaselinePipeline generates a fingerprint baseline for a single React Native environment.
// It executes the full four-step pipeline: Metro bundle (baseline_entry.js) → Hermes compile →
// HBC disassemble → SHA256 hash all functions, then stores the result in the "baselines_v3"
// MongoDB collection. Baselines capture the framework-only function fingerprints so the analyzer
// can filter them out when scanning application bundles, preventing false-positive matches against
// React Native internals.
//
// Parameters:
//   - reactNativeWorkingDirectoryPath: absolute path to a pipeline/react-natives/rnXXX/ directory
//   - hermesCompilerPath: absolute path to the hermesc binary for this RN version
//   - version: React Native version identifier (e.g. "rn075")
//
// Returns an error if any pipeline step fails (Metro bundling, Hermes compilation,
// HBC disassembly, or database insertion).
func RunBaselinePipeline(reactNativeWorkingDirectoryPath, hermesCompilerPath, version string) error {
	bundlePath, err := UseMetroToBundle(reactNativeWorkingDirectoryPath, version, "", "", true)
	if err != nil {
		return fmt.Errorf("error using Metro to bundle: %v", err)
	}
	compiledBundlePath, hermesVersion, err := UseHermesToCompile(bundlePath, hermesCompilerPath, reactNativeWorkingDirectoryPath, version, true)
	if err != nil {
		return fmt.Errorf("error using Hermes to compile: %v", err)
	}
	fois, err := UseHermesDecompilerToDisassemble(filepath.Join(reactNativeWorkingDirectoryPath, compiledBundlePath))
	if err != nil {
		return fmt.Errorf("error disassembling Hermes bytecode file: %v", err)
	}
	baselineEntry := CreateBaselineDatabaseEntry(fois, version, hermesVersion)
	err = AddBaselineDatabaseEntryToDatabase(baselineEntry)
	if err != nil {
		return fmt.Errorf("error adding baseline database entry to database: %v", err)
	}

	return nil
}

// RunPackagePipeline generates function fingerprints for a single npm package version within
// one React Native environment. It executes the four-step pipeline: Metro bundle (package_entry.js
// with the target package imported) → Hermes compile → HBC disassemble → SHA256 hash all functions.
// Unlike RunSinglePackagePipeline, this function handles npm install and uninstall of the package
// internally via UseMetroToBundle.
//
// Parameters:
//   - reactNativeWorkingDirectoryPath: absolute path to a pipeline/react-natives/rnXXX/ directory
//   - hermesCompilerPath: absolute path to the hermesc binary for this RN version
//   - version: React Native version identifier (e.g. "rn075")
//   - pkg: package metadata including PackageName and PackageVersion for npm install
//
// Returns the PackageHash containing per-function SHA256 fingerprints ready for database insertion,
// or an error if any pipeline step fails (npm install, Metro bundling, Hermes compilation, or
// HBC disassembly).
func RunPackagePipeline(reactNativeWorkingDirectoryPath, hermesCompilerPath, version string, pkg models.Package) (*models.PackageHash, error) {
	bundlePath, err := UseMetroToBundle(reactNativeWorkingDirectoryPath, version, pkg.PackageName, pkg.PackageVersion, false)
	if err != nil {
		return nil, fmt.Errorf("error using Metro to bundle: %v", err)
	}
	compiledBundlePath, hermesVersion, err := UseHermesToCompile(bundlePath, hermesCompilerPath, reactNativeWorkingDirectoryPath, version, false)
	if err != nil {
		return nil, fmt.Errorf("error using Hermes to compile: %v", err)
	}
	fois, err := UseHermesDecompilerToDisassemble(filepath.Join(reactNativeWorkingDirectoryPath, compiledBundlePath))
	if err != nil {
		return nil, fmt.Errorf("error disassembling Hermes bytecode file: %v", err)
	}

	return CreatePackageHashesDatabaseEntry(fois, version, hermesVersion, pkg), nil
}

func writeBundleScriptFile(reactNativeWorkingDirectoryPath, version string, baseline bool) (string, error) {

	outputFileName := ""
	entryFileName := ""
	if baseline {
		outputFileName = fmt.Sprintf("baseline_out_%s.js", version)
		entryFileName = "baseline_entry.js"
	} else {
		outputFileName = fmt.Sprintf("package_out_%s.js", version)
		entryFileName = "package_entry.js"
	}

	bundleScriptFileContent := fmt.Sprintf(`import Metro from 'metro';
	const config = await Metro.loadConfig();
	config.resetCache = true;
	await Metro.runBuild(config, {
		entry: '%s',
		platform: 'ios',
		minify: false,
		out: '%s'
	});`, entryFileName, outputFileName)

	bundleScriptFilePath := filepath.Join(reactNativeWorkingDirectoryPath, "bundle-script.js")
	err := os.WriteFile(bundleScriptFilePath, []byte(bundleScriptFileContent), 0644)
	if err != nil {
		return "", err
	}

	return outputFileName, nil
}

func writePackageEntryFile(reactNativeWorkingDirectoryPath, packageName string) error {
	packageEntryFilePath := filepath.Join(reactNativeWorkingDirectoryPath, "package_entry.js")
	packageEntryFileContent := fmt.Sprintf(`import * as ModuleToBundle from '%s';

	global.MODULE_TO_BUNDLE = ModuleToBundle;`, packageName)

	err := os.WriteFile(packageEntryFilePath, []byte(packageEntryFileContent), 0644)
	if err != nil {
		return err
	}

	return nil
}

// step 1: use metro to bundle from entry file
func UseMetroToBundle(reactNativeWorkingDirectoryPath, version, packageName, packageVersion string, baseline bool) (string, error) {
	utils.L.Println("Using Metro to bundle with React Native version " + version)

	if !baseline {
		utils.L.Println("Writing package entry file for package " + packageName)
		err := writePackageEntryFile(reactNativeWorkingDirectoryPath, packageName)
		if err != nil {
			return "", fmt.Errorf("error writing package entry file [%s]: %v", fmt.Sprintf("%s@%s", packageName, packageVersion), err)
		}
		utils.S.Println("Package entry file written for package " + packageName)

		utils.L.Println("Installing package " + packageName + " package version " + packageVersion)
		installPackageCommand := exec.Command("npm", "install", packageName+"@"+packageVersion)
		installPackageCommand.Dir = reactNativeWorkingDirectoryPath
		if global.Verbose {
			installPackageCommand.Stdout = os.Stdout
			installPackageCommand.Stderr = os.Stderr
		}
		err = installPackageCommand.Run()
		if err != nil {
			return "", fmt.Errorf("error installing package [%s]: %v", fmt.Sprintf("%s@%s", packageName, packageVersion), err)
		}
		utils.S.Println("Package " + packageName + " installed successfully")
	}

	bundledOutputFileName, err := writeBundleScriptFile(reactNativeWorkingDirectoryPath, version, baseline)
	if err != nil {
		return "", fmt.Errorf("error writing bundle script file: %v", err)
	}
	utils.S.Println("Bundle script file written for version " + version)

	bundleCommandArgs := []string{"node", "bundle-script.js"}

	bundleCommand := exec.Command(bundleCommandArgs[0], bundleCommandArgs[1:]...)
	bundleCommand.Dir = reactNativeWorkingDirectoryPath
	if global.Verbose {
		bundleCommand.Stdout = os.Stdout
		bundleCommand.Stderr = os.Stderr
	}

	err = bundleCommand.Run()
	if err != nil {
		return "", fmt.Errorf("error bundling with Metro: %v", err)
	}
	utils.S.Println("Metro bundled successfully to " + bundledOutputFileName)

	if !baseline {
		utils.L.Println("Uninstalling package " + packageName + " package version " + packageVersion)
		// Uninstall is no longer needed here as we restore a clean environment for each package group.
	}

	return bundledOutputFileName, nil
}

// UseMetroToBundleOptimized assumes the package is already installed and skips npm operations
// This is used by the optimized pipeline where npm operations are handled externally
func UseMetroToBundleOptimized(reactNativeWorkingDirectoryPath, version, packageName, packageVersion string) (string, error) {
	utils.L.Printf("Using Metro to bundle (optimized) with React Native version %s for package %s@%s\n", version, packageName, packageVersion)

	// Write package entry file
	utils.D.Println("Writing package entry file for package " + packageName)
	err := writePackageEntryFile(reactNativeWorkingDirectoryPath, packageName)
	if err != nil {
		return "", fmt.Errorf("error writing package entry file [%s]: %v", fmt.Sprintf("%s@%s", packageName, packageVersion), err)
	}
	utils.D.Println("Package entry file written for package " + packageName)

	// Write bundle script
	bundledOutputFileName, err := writeBundleScriptFile(reactNativeWorkingDirectoryPath, version, false)
	if err != nil {
		return "", fmt.Errorf("error writing bundle script file: %v", err)
	}
	utils.D.Println("Bundle script file written for version " + version)

	// Run Metro bundling (package assumed to be already installed)
	bundleCommandArgs := []string{"node", "bundle-script.js"}

	bundleCommand := exec.Command(bundleCommandArgs[0], bundleCommandArgs[1:]...)
	bundleCommand.Dir = reactNativeWorkingDirectoryPath
	if global.Verbose {
		bundleCommand.Stdout = os.Stdout
		bundleCommand.Stderr = os.Stderr
	}

	err = bundleCommand.Run()
	if err != nil {
		return "", fmt.Errorf("error bundling with Metro: %v", err)
	}
	utils.S.Printf("Metro bundled successfully to %s for package %s@%s\n", bundledOutputFileName, packageName, packageVersion)

	return bundledOutputFileName, nil
}

// step 2: use hermes to compile the bundle
func UseHermesToCompile(fileToCompilePath, hermesCompilerPath, reactNativeWorkingDirectoryPath, version string, baseline bool) (string, string, error) {
	hermesVersionOutput, err := exec.Command(hermesCompilerPath, "-version").Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting Hermes version: %v", err)
	}

	versionOutput := strings.TrimSpace(string(hermesVersionOutput))
	re, err := regexp.Compile(`HBC bytecode version: (\d+)`)
	if err != nil {
		return "", "", fmt.Errorf("error compiling regex: %v", err)
	}

	matches := re.FindStringSubmatch(versionOutput)
	var bytecodeVersion string
	if len(matches) > 1 {
		bytecodeVersion = matches[1]
	}

	utils.L.Printf("Using Hermes version %s to compile with RN version %s\n", bytecodeVersion, version)

	outputFileName := ""
	if baseline {
		outputFileName = fmt.Sprintf("baseline-%s.hbc", version)
	} else {
		outputFileName = fmt.Sprintf("package-entry-%s.hbc", version)
	}
	hermesCompileArgs := []string{hermesCompilerPath, "-O", "-emit-binary", "-out", outputFileName, fileToCompilePath}

	hermesCompileCommand := exec.Command(hermesCompileArgs[0], hermesCompileArgs[1:]...)
	hermesCompileCommand.Dir = reactNativeWorkingDirectoryPath
	if global.Verbose {
		hermesCompileCommand.Stdout = os.Stdout
		hermesCompileCommand.Stderr = os.Stderr
	}

	err = hermesCompileCommand.Run()
	if err != nil {
		return "", "", fmt.Errorf("error compiling with Hermes: %v", err)
	}

	utils.S.Println("Hermes compiled successfully to " + outputFileName)

	return outputFileName, bytecodeVersion, nil
}

// step 3: use hermes-decompiler to disassemble the hermes bytecode into our IR
func UseHermesDecompilerToDisassemble(fileToDisassemblePath string) ([]*types.FunctionObject, error) {
	inputFile, err := os.Open(fileToDisassemblePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file to disassemble: %v", err)
	}
	defer inputFile.Close()

	utils.L.Println("Disassembling Hermes bytecode file " + fileToDisassemblePath)

	hbcReader := hbc.NewHBCReader()

	err = hbcReader.ReadWholeFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing file to disassemble: %v", err)
	}

	utils.L.Println("Creating function objects from Hermes bytecode file")

	fois, err := hbc.CreateFunctionObjects(hbcReader)
	if err != nil {
		return nil, fmt.Errorf("error creating function objects: %v", err)
	}

	return fois, nil
}

// step 4: create database entry (baseline or package entry)
func CreateBaselineDatabaseEntry(fois []*types.FunctionObject, reactNativeVersion, hermesVersion string) *models.BaselineModel {
	utils.L.Println("Creating baseline database entry")

	hashes := make([]models.Hash, 0)

	for _, foi := range fois {
		structuralRaw, content1Raw, content2Raw := foi.ToIR()

		var structuralHash, content1Hash, content2Hash string
		structuralHash = fmt.Sprintf("%x", sha256.Sum256([]byte(structuralRaw)))
		if content1Raw != "" {
			content1Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content1Raw)))
		}
		if content2Raw != "" {
			content2Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content2Raw)))
		}

		hash := models.Hash{
			RelativeFunctionIndex: foi.Metadata.Index,
			StructuralRaw:         structuralRaw,
			ContentIR1Raw:         content1Raw,
			ContentIR2Raw:         content2Raw,
			StructuralHash:        structuralHash,
			ContentIR1Hash:        content1Hash,
			ContentIR2Hash:        content2Hash,
		}
		hashes = append(hashes, hash)
	}

	baselineEntry := &models.BaselineModel{
		ID:                    primitive.NewObjectID(),
		ReactNativeVersion:    reactNativeVersion,
		HermesBytecodeVersion: hermesVersion,
		Hashes:                hashes,
	}

	return baselineEntry
}

func CreatePackageHashesDatabaseEntry(fois []*types.FunctionObject, reactNativeVersion, hermesVersion string, pkg models.Package) *models.PackageHash {
	utils.L.Println("Creating package hashes database entry")

	hashes := make([]models.Hash, 0)

	for _, foi := range fois {
		structuralRaw, content1Raw, content2Raw := foi.ToIR()

		var structuralHash, content1Hash, content2Hash string
		structuralHash = fmt.Sprintf("%x", sha256.Sum256([]byte(structuralRaw)))
		if content1Raw != "" {
			content1Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content1Raw)))
		}
		if content2Raw != "" {
			content2Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content2Raw)))
		}

		hash := models.Hash{
			RelativeFunctionIndex: foi.Metadata.Index,
			StructuralRaw:         structuralRaw,
			ContentIR1Raw:         content1Raw,
			ContentIR2Raw:         content2Raw,
			StructuralHash:        structuralHash,
			ContentIR1Hash:        content1Hash,
			ContentIR2Hash:        content2Hash,
		}
		hashes = append(hashes, hash)
	}

	packageHashesEntry := &models.PackageHash{
		ID:                 primitive.NewObjectID(),
		PackageID:          pkg.ID,
		ReactNativeVersion: reactNativeVersion,
		Hashes:             hashes,
	}

	return packageHashesEntry
}

// step 5: add database entry to database (baseline or package entry)
func AddBaselineDatabaseEntryToDatabase(entry *models.BaselineModel) error {
	utils.L.Println("Adding baseline database entry to database")

	_, err := db.InsertOne("baselines_v3", entry)
	if err != nil {
		return fmt.Errorf("error adding baseline database entry to database: %v", err)
	}

	utils.S.Println("Baseline database entry added to database")

	return nil
}
