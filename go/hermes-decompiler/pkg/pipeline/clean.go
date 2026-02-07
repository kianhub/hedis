package pipeline

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hermes-decompiler/pkg/database"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/global"
	"hermes-decompiler/pkg/hbc"
	"hermes-decompiler/pkg/hbc/types"
	"hermes-decompiler/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.philip.id/mongopiet/pkg/db"
)

var (
	rnFolderVersions = []string{"rn069", "rn070", "rn071", "rn072", "rn073", "rn074", "rn075", "rn076", "rn077", "rn078", "rn079"}
	rnBaselines      = make(map[string]*models.BaselineModel)
)

func GetRNBaselines() error {
	dbRnBaselines, err := db.Find[models.BaselineModel]("baselines", bson.M{})
	if err != nil {
		return fmt.Errorf("error getting rn baselines: %v", err)
	}

	for _, baseline := range dbRnBaselines {
		rnBaselines[baseline.ReactNativeVersion] = baseline
	}

	return nil
}

func CreateHashesForPackage(pkg *models.PackageWithVulnerabilitiesModel) error {
	// first, we check if the package already has hashes
	// if it does, we skip it
	// we do this by checking the package entry in the database, the hashes are stored
	// in the hashes collection, so we need to check if there are any hash entries for the package and react native version

	// download all the baseline hashes from the database before we start so we have them in memory to compare against

	for _, rnVersion := range rnFolderVersions {
		err := runPipeline(pkg, rnVersion)
		if err != nil {
			utils.F.Printf("error running pipeline for package %s@%s for RN %s: %v\n", pkg.PackageName, pkg.PackageVersion, rnVersion, err)
			pkg.HasError = true
			if pkg.Error == nil {
				pkg.Error = make(map[string]string)
			}
			pkg.Error[rnVersion] = err.Error()
			err := database.UpdatePackageErrorLog(pkg)
			if err != nil {
				utils.F.Printf("failed to update package error log for package %s@%s for RN %s: %v\n", pkg.PackageName, pkg.PackageVersion, rnVersion, err)
			}
		}
	}

	return nil
}

func runPipeline(pkg *models.PackageWithVulnerabilitiesModel, rnFolderVersion string) error {

	rnVersionParsed := extractNumericVersion(rnFolderVersion)
	if rnVersionParsed == "" {
		return fmt.Errorf("invalid react native version: %s", rnFolderVersion)
	}

	utils.L.Printf("Processing package %s@%s for RN %s\n", pkg.PackageName, pkg.PackageVersion, rnVersionParsed)

	// CRAZY SHIT NOW: REBUILD DATABASE
	if pkg.HasError && pkg.Error[rnFolderVersion] != "" && !strings.Contains(pkg.Error[rnFolderVersion], "must provide at least one element in input slice") {
		return fmt.Errorf("package %s@%s for RN %s can't be processed (historically), skipping", pkg.PackageName, pkg.PackageVersion, rnFolderVersion)
	}

	// first, we check if the package already has hashes
	hashExists, err := database.CheckIfPackageHasHashes(pkg, rnVersionParsed)
	if err != nil {
		return fmt.Errorf("error checking if package has hashes: %v", err)
	}
	if hashExists {
		utils.S.Printf("Package %s@%s for RN %s already has hashes, skipping\n", pkg.PackageName, pkg.PackageVersion, rnVersionParsed)
		return nil
	}
	// if it doesn't, we start with installing the package in the react native directory

	// we need to check if the react native directory exists
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %v", err)
	}
	reactNativeWorkingDirPath := filepath.Join(cwd, "..", "..", "pipeline", "react-natives", rnFolderVersion)
	if !checkIfRNDirectoryExists(reactNativeWorkingDirPath) {
		return fmt.Errorf("react native directory does not exist: %s", reactNativeWorkingDirPath)
	}

	// we install the package in the react native directory
	err = installPackage(pkg, reactNativeWorkingDirPath, rnVersionParsed)
	if err != nil {
		return fmt.Errorf("error installing package: %v", err)
	}

	// we bundle the package
	bundleFilePath, err := bundlePackage(pkg, reactNativeWorkingDirPath, rnVersionParsed)
	if err != nil {
		// at least try to uninstall the package
		_ = uninstallPackage(pkg, reactNativeWorkingDirPath, rnVersionParsed)
		return fmt.Errorf("error bundling package: %v", err)
	}

	// after we have the bundle file, we can get rid of the package
	err = uninstallPackage(pkg, reactNativeWorkingDirPath, rnVersionParsed)
	if err != nil {
		return fmt.Errorf("error uninstalling package: %v", err)
	}

	// we compile the bundle using the hermes compiler that was shipped with the react native version
	hermesCompilerPath := filepath.Join(reactNativeWorkingDirPath, "node_modules", "react-native", "sdks", "hermesc", global.OSHermes, "hermesc")

	hermesOutputFileName, _, err := UseHermesToCompile(bundleFilePath, hermesCompilerPath, reactNativeWorkingDirPath, rnVersionParsed, false)
	if err != nil {
		return fmt.Errorf("error compiling with Hermes: %v", err)
	}

	// we disassemble the hermes bytecode
	hermesOutputFilePath := filepath.Join(reactNativeWorkingDirPath, hermesOutputFileName)
	functionObjects, err := UseHermesDecompilerToDisassemble(hermesOutputFilePath)
	if err != nil {
		return fmt.Errorf("error disassembling hermes bytecode: %v", err)
	}

	utils.L.Printf("Found %d function objects\n", len(functionObjects))

	// we now have the function objects that we want to hash and also check if they are already in the baselines database collection
	hashes := generateHashes(functionObjects, rnVersionParsed, pkg.ID)
	utils.L.Printf("Generated %d hashes\n", len(hashes))
	filteredHashes := compareHashesWithBaselines(hashes, rnVersionParsed)
	utils.L.Printf("Filtered %d hashes\n", len(filteredHashes))

	err = database.AddHashesToDatabase(filteredHashes)
	if err != nil {
		return fmt.Errorf("error adding hashes to database: %v", err)
	}

	return nil
}

//func CompareHashesWithUnknownBaseline()

func compareHashesWithBaselines(hashes []*models.FunctionObjectHashModel, rnVersionParsed string) []*models.FunctionObjectHashModel {
	filteredHashes := make([]*models.FunctionObjectHashModel, 0)
	var counter = 0
	for _, pkgHash := range hashes {
		matchFound := false
		for _, rnBaselineHash := range rnBaselines[rnVersionParsed].Hashes {
			if pkgHash.Structural.Hash == rnBaselineHash.StructuralHash && pkgHash.ContentAR1.Hash == rnBaselineHash.ContentIR1Hash && pkgHash.ContentAR2.Hash == rnBaselineHash.ContentIR2Hash {
				counter++
				matchFound = true
				break
			}
		}
		if !matchFound {
			filteredHashes = append(filteredHashes, pkgHash)
		}
	}

	utils.L.Printf("Found %d matches\n", counter)

	return filteredHashes
}

func extractNumericVersion(rnVersion string) string {
	digits := strings.TrimPrefix(rnVersion, "rn")
	num, err := strconv.Atoi(digits)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%.2f", float64(num)/100)
}

func generateHashes(functionObjects []*types.FunctionObject, rnVersionParsed string, pkgId primitive.ObjectID) []*models.FunctionObjectHashModel {
	hashes := make([]*models.FunctionObjectHashModel, 0)
	for _, functionObject := range functionObjects {
		structuralRaw, content1Raw, content2Raw := functionObject.ToIR()

		var structuralHash, content1Hash, content2Hash string
		structuralHash = fmt.Sprintf("%x", sha256.Sum256([]byte(structuralRaw)))
		if content1Raw != "" {
			content1Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content1Raw)))
		}
		if content2Raw != "" {
			content2Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content2Raw)))
		}

		hash := &models.FunctionObjectHashModel{
			PackageID:          pkgId,
			ReactNativeVersion: rnVersionParsed,
			Structural: models.FunctionObjectDataModel{
				Raw:  structuralRaw,
				Hash: structuralHash,
			},
			ContentAR1: models.FunctionObjectDataModel{
				Raw:  content1Raw,
				Hash: content1Hash,
			},
			ContentAR2: models.FunctionObjectDataModel{
				Raw:  content2Raw,
				Hash: content2Hash,
			},
		}
		hashes = append(hashes, hash)
	}

	return hashes
}

func bundlePackage(pkg *models.PackageWithVulnerabilitiesModel, workingDir, rnVersion string) (string, error) {
	utils.L.Printf("Bundling package %s@%s for RN %s \n", pkg.PackageName, pkg.PackageVersion, rnVersion)

	// write the package entry file that will be used to bundle the package
	err := writePackageEntryFile(workingDir, pkg.PackageName)
	if err != nil {
		return "", fmt.Errorf("error writing package entry file: %v", err)
	}

	// write the bundle script file
	bundleFilePath, err := writeBundleScriptFile(workingDir, rnVersion, false)
	if err != nil {
		return "", fmt.Errorf("error writing bundle script file: %v", err)
	}

	// we run the bundle script file
	bundleCommand := exec.Command("node", "bundle-script.js")
	bundleCommand.Dir = workingDir
	if global.Verbose {
		bundleCommand.Stdout = os.Stdout
		bundleCommand.Stderr = os.Stderr
	}

	err = bundleCommand.Run()
	if err != nil {
		return "", err
	}

	utils.S.Printf("Package %s@%s for RN %s bundled successfully\n", pkg.PackageName, pkg.PackageVersion, rnVersion)
	return bundleFilePath, nil
}

func installPackage(pkg *models.PackageWithVulnerabilitiesModel, workingDir, rnVersion string) error {
	utils.L.Printf("Installing package %s@%s for RN %s \n", pkg.PackageName, pkg.PackageVersion, rnVersion)

	installPackageCommand := exec.Command("npm", "install", pkg.PackageName+"@"+pkg.PackageVersion, "--no-audit")
	installPackageCommand.Dir = workingDir
	if global.Verbose {
		installPackageCommand.Stdout = os.Stdout
		installPackageCommand.Stderr = os.Stderr
	}
	err := installPackageCommand.Run()
	if err != nil {
		return err
	}

	utils.S.Printf("Package %s@%s for RN %s installed successfully\n", pkg.PackageName, pkg.PackageVersion, rnVersion)
	return nil
}

func uninstallPackage(pkg *models.PackageWithVulnerabilitiesModel, workingDir, rnVersion string) error {
	utils.L.Printf("Uninstalling package %s@%s for RN %s \n", pkg.PackageName, pkg.PackageVersion, rnVersion)

	uninstallPackageCommand := exec.Command("npm", "uninstall", pkg.PackageName)
	uninstallPackageCommand.Dir = workingDir
	if global.Verbose {
		uninstallPackageCommand.Stdout = os.Stdout
		uninstallPackageCommand.Stderr = os.Stderr
	}

	err := uninstallPackageCommand.Run()
	if err != nil {
		return err
	}

	utils.S.Printf("Package %s@%s for RN %s uninstalled successfully\n", pkg.PackageName, pkg.PackageVersion, rnVersion)
	return nil
}

func checkIfRNDirectoryExists(path string) bool {
	rnDir, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	return len(rnDir) > 0
}

// UserHermesDecompilerToDisassembleWithVersion is for analyzing a specific app bundle pipeline
func UseHermesDecompilerToDisassembleWithVersion(fileToDisassemblePath string) (fos []*types.FunctionObject, hbVersion string, err error) {
	inputFile, err := os.Open(fileToDisassemblePath)
	if err != nil {
		return nil, "", fmt.Errorf("error opening file to disassemble: %v", err)
	}
	defer inputFile.Close()

	utils.L.Println("Disassembling Hermes bytecode file " + fileToDisassemblePath)

	hbcReader := hbc.NewHBCReader()

	err = hbcReader.ReadWholeFile(inputFile)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing file to disassemble: %v", err)
	}

	utils.L.Println("Creating function objects from Hermes bytecode file")

	fois, err := hbc.CreateFunctionObjects(hbcReader)
	if err != nil {
		return nil, "", fmt.Errorf("error creating function objects: %v", err)
	}

	return fois, fmt.Sprintf("%d", hbcReader.Header.Version), nil
}

// GenerateHashesForSpecific is for analyzing a specific app bundle pipeline
func GenerateHashesForSpecific(functionObjects []*types.FunctionObject) []*models.FunctionObjectHashModel {
	hashes := make([]*models.FunctionObjectHashModel, 0)
	for _, functionObject := range functionObjects {
		structuralRaw, content1Raw, content2Raw := functionObject.ToIR()

		var structuralHash, content1Hash, content2Hash string
		structuralHash = fmt.Sprintf("%x", sha256.Sum256([]byte(structuralRaw)))
		if content1Raw != "" {
			content1Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content1Raw)))
		}
		if content2Raw != "" {
			content2Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(content2Raw)))
		}

		hash := &models.FunctionObjectHashModel{
			Structural: models.FunctionObjectDataModel{
				Raw:  structuralRaw,
				Hash: structuralHash,
			},
			ContentAR1: models.FunctionObjectDataModel{
				Raw:  content1Raw,
				Hash: content1Hash,
			},
			ContentAR2: models.FunctionObjectDataModel{
				Raw:  content2Raw,
				Hash: content2Hash,
			},
		}
		hashes = append(hashes, hash)
	}

	return hashes
}

// FindRNBaselineCandidates is checking the rnBaselines map for all HBC version that match and then looks which one of the baselines has the most matches and return the filtered set of hashes with the react native version of the baseline that has the most matches
func FindRNBaselineCandidates(hashes []*models.FunctionObjectHashModel, hbcVersion string) ([]*models.FunctionObjectHashModel, string) {
	filteredHashes := make([]*models.FunctionObjectHashModel, 0)
	//bestMatchCount := 0
	bestMatchRNVersion := ""
	bestCombinedMatchCount := 0

	baselineMatches := make(map[string]struct {
		Structural int
		Content1   int
		Content2   int
		Combined   int
	})

	for rnVersion, baseline := range rnBaselines {
		if baseline.HermesBytecodeVersion != hbcVersion {
			continue
		}
		baselineMatches[rnVersion] = struct {
			Structural int
			Content1   int
			Content2   int
			Combined   int
		}{
			Structural: 0,
			Content1:   0,
			Content2:   0,
			Combined:   0,
		}
		for _, pkgHash := range hashes {
			for _, rnBaselineHash := range baseline.Hashes {
				matches := baselineMatches[rnVersion]

				if pkgHash.ContentAR1.Hash == rnBaselineHash.ContentIR1Hash && pkgHash.ContentAR2.Hash == rnBaselineHash.ContentIR2Hash && pkgHash.Structural.Hash == rnBaselineHash.StructuralHash {
					matches.Combined++
					baselineMatches[rnVersion] = matches
					break // continue
				}

				if pkgHash.Structural.Hash == rnBaselineHash.StructuralHash {
					matches.Structural++
					baselineMatches[rnVersion] = matches
					break
				}

				if rnBaselineHash.ContentIR1Hash != "" {
					if pkgHash.ContentAR1.Hash != "" && pkgHash.ContentAR1.Hash == rnBaselineHash.ContentIR1Hash {
						matches.Content1++
						baselineMatches[rnVersion] = matches
						break
					}

					if rnBaselineHash.ContentIR2Hash == "" {
						continue
					}
				}

				if rnBaselineHash.ContentIR2Hash != "" {
					if pkgHash.ContentAR2.Hash != "" && pkgHash.ContentAR2.Hash == rnBaselineHash.ContentIR2Hash {
						matches.Content2++
						baselineMatches[rnVersion] = matches
					}
				}

				/* if pkgHash.ContentAR1.Hash != "" && pkgHash.ContentAR1.Hash == rnBaselineHash.ContentIR1Hash {
					matches.Content1++
					baselineMatches[rnVersion] = matches
				}
				if pkgHash.ContentAR2.Hash != "" && pkgHash.ContentAR2.Hash == rnBaselineHash.ContentIR2Hash {
					matches.Content2++
					baselineMatches[rnVersion] = matches
				} */

				/* if pkgHash.ContentAR1.Hash == rnBaselineHash.ContentIR1Hash {
					matches.Content1++
				}
				if pkgHash.ContentAR2.Hash == rnBaselineHash.ContentIR2Hash {
					matches.Content2++
				}
				baselineMatches[rnVersion] = matches */
			}
		}
	}
	spew.Dump(baselineMatches)

	// now for every element in the baselineMatches map, we need to calculate the total matches and then find the react native version with the most matches
	for rnVersion, matches := range baselineMatches {
		/* totalMatches := matches.Structural + matches.Content1 + matches.Content2 + matches.Combined
		if totalMatches > bestMatchCount {
			bestMatchCount = totalMatches
			bestMatchRNVersion = rnVersion
		} */
		if matches.Combined > bestCombinedMatchCount {
			bestCombinedMatchCount = matches.Combined
			bestMatchRNVersion = rnVersion
		}
	}

	if bestMatchRNVersion == "" {
		// if we didn't find a match with the combined hashes, we need to find the best structural match
		bestStructural := 0
		for rnVersion, matches := range baselineMatches {
			if matches.Structural > bestStructural {
				bestStructural = matches.Structural
				bestMatchRNVersion = rnVersion
			}
		}
	}

	// remove all the hashes that matched the best match RN version
	for _, hashToFilter := range hashes {
		matchFound := false
		for _, rnBaselineHash := range rnBaselines[bestMatchRNVersion].Hashes {

			if hashToFilter.ContentAR1.Hash == rnBaselineHash.ContentIR1Hash && hashToFilter.ContentAR2.Hash == rnBaselineHash.ContentIR2Hash && hashToFilter.Structural.Hash == rnBaselineHash.StructuralHash {
				matchFound = true
				break
			}

			/* if hashToFilter.Structural.Hash == rnBaselineHash.StructuralHash {
				matchFound = true
				break
			}
			if hashToFilter.ContentAR1.Hash != "" && rnBaselineHash.ContentIR1Hash != "" && hashToFilter.ContentAR1.Hash == rnBaselineHash.ContentIR1Hash {
				matchFound = true
				break
			}
			if hashToFilter.ContentAR2.Hash != "" && rnBaselineHash.ContentIR2Hash != "" && hashToFilter.ContentAR2.Hash == rnBaselineHash.ContentIR2Hash {
				matchFound = true
				break
			} */
		}
		if !matchFound {
			filteredHashes = append(filteredHashes, hashToFilter)
		}
	}

	// we now have the filtered hashes which are pruned from the hashes that matched the best match RN version
	return filteredHashes, bestMatchRNVersion
}

type MatchResults struct {
	PackageInfo               *models.PackageWithVulnerabilitiesModel
	StructuralMatches         int
	ContentIR1Matches         int
	ContentIR2Matches         int
	CombinedMatches           int
	TotalAppHashes            int
	TotalPackageHashes        int
	TotalC1Hashes             int
	TotalC2Hashes             int
	StructuralMatchPercentage float64
	ContentIR1MatchPercentage float64
	ContentIR2MatchPercentage float64
	TotalAmountOfMatches      int
	TotalMatchPercentage      float64
}

// CompareAppWithPackage compares an app bundle's function fingerprints against a known package's
// fingerprints to determine whether the package is present in the app. It iterates over every
// package hash and searches for a matching app hash using a combined three-IR equality check
// (Structural + ContentAR1 + ContentAR2 must all match). The first match per package hash
// short-circuits to avoid double-counting.
//
// The returned MatchResults contains per-IR match counts, total hash counts (including
// ContentIR1/IR2 non-empty counts via CountPackageMetadata), per-IR match percentages,
// and an overall TotalMatchPercentage computed as CombinedMatches / TotalPackageHashes * 100.
// A high TotalMatchPercentage indicates the package's compiled functions are present in the app
// bundle.
//
// Parameters:
//   - appHashes: function fingerprints extracted from the target app's HBC file, typically
//     produced by GenerateHashesForSpecific after baseline filtering via FindRNBaselineCandidates.
//   - packageHashes: function fingerprints for the candidate package from the "hashes" or
//     "hashes_ghsa" MongoDB collection, grouped by GetHashesFromDatabaseWithRNVersion.
//
// Returns:
//
//	*MatchResults with match counts, hash totals, and percentage breakdowns. PackageInfo is
//	not populated by this function and must be set by the caller.
func CompareAppWithPackage(appHashes []*models.FunctionObjectHashModel, packageHashes []*models.FunctionObjectHashModel) *MatchResults {
	matchResults := &MatchResults{
		StructuralMatches:  0,
		ContentIR1Matches:  0,
		ContentIR2Matches:  0,
		CombinedMatches:    0,
		TotalAppHashes:     len(appHashes),
		TotalPackageHashes: len(packageHashes),
	}

	c1Amount, c2Amount := CountPackageMetadata(packageHashes)
	matchResults.TotalC1Hashes = c1Amount
	matchResults.TotalC2Hashes = c2Amount

	for _, packageHash := range packageHashes {
		for _, appHash := range appHashes {

			// Currently this is a pretty lossy approach because if any of our three hash types match, we say that we found a match and do not check the other hash type

			doBreak := false

			if packageHash.Structural.Hash == appHash.Structural.Hash && packageHash.ContentAR1.Hash == appHash.ContentAR1.Hash && packageHash.ContentAR2.Hash == appHash.ContentAR2.Hash {
				matchResults.CombinedMatches++
				// we break here because we definitely found a match
				doBreak = true
			}

			/* if appHash.Structural.Hash == packageHash.Structural.Hash {
				matchResults.StructuralMatches++
				// we break here because we definitely found a match
				doBreak = true
			}
			if packageHash.ContentAR1.Hash != "" && appHash.ContentAR1.Hash == packageHash.ContentAR1.Hash {
				//utils.D.Println(packageHash.ContentAR1.Raw)
				matchResults.ContentIR1Matches++
				// we break here because we definitely found a match
				doBreak = true
			}
			if packageHash.ContentAR2.Hash != "" && appHash.ContentAR2.Hash == packageHash.ContentAR2.Hash {
				//utils.D.Println(packageHash.ContentAR2.Raw)
				matchResults.ContentIR2Matches++
				// we break here because we definitely found a match
				doBreak = true
			} */

			if doBreak {
				break
			}
		}
	}

	matchResults.StructuralMatchPercentage = float64(matchResults.StructuralMatches) / float64(matchResults.TotalPackageHashes)
	matchResults.ContentIR1MatchPercentage = float64(matchResults.ContentIR1Matches) / float64(matchResults.TotalC1Hashes)
	matchResults.ContentIR2MatchPercentage = float64(matchResults.ContentIR2Matches) / float64(matchResults.TotalC2Hashes)

	matchResults.TotalAmountOfMatches = matchResults.StructuralMatches + matchResults.ContentIR1Matches + matchResults.ContentIR2Matches + matchResults.CombinedMatches

	matchResults.TotalMatchPercentage = float64(matchResults.CombinedMatches) / float64(matchResults.TotalPackageHashes) * 100

	return matchResults
}

func CheckForDuplicates(hashes []*models.FunctionObjectHashModel) []string {
	uniqueHashes := make([]string, 0)
	for _, hash := range hashes {
		if !slices.Contains(uniqueHashes, hash.ContentAR2.Hash) {
			uniqueHashes = append(uniqueHashes, hash.ContentAR2.Hash)
		}
	}
	return uniqueHashes
}

func CountPackageMetadata(hashes []*models.FunctionObjectHashModel) (c1Amount, c2Amount int) {
	for _, hash := range hashes {
		if hash.ContentAR1.Hash != "" {
			c1Amount++
		}
		if hash.ContentAR2.Hash != "" {
			c2Amount++
		}
	}
	return
}

type HashGroup struct {
	PackageID primitive.ObjectID                      `bson:"_id"`
	Hashes    []*models.FunctionObjectHashModel       `bson:"hashes"`
	Result    *models.PackageWithVulnerabilitiesModel `bson:"result"`
}

func GetHashesFromDatabaseWithRNVersion(rnVersion string) ([]*HashGroup, error) {
	query := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "react_native_version", Value: rnVersion}}}},
		bson.D{
			{Key: "$group",
				Value: bson.D{
					{Key: "_id", Value: "$package_id"},
					{Key: "hashes", Value: bson.D{
						{Key: "$push", Value: bson.D{
							{Key: "package_id", Value: "$package_id"},
							{Key: "react_native_version", Value: "$react_native_version"},
							{Key: "structural", Value: bson.D{{Key: "hash", Value: "$structural.hash"}}},
							{Key: "ar1", Value: bson.D{{Key: "hash", Value: "$ar1.hash"}}},
							{Key: "ar2", Value: bson.D{{Key: "hash", Value: "$ar2.hash"}}},
						}},
					}},
				},
			},
		},
		bson.D{
			{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "packages"},
				{Key: "localField", Value: "_id"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "result"},
			}},
		},
		bson.D{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$result"}}}},
	}

	return db.AggregateCtx[HashGroup]("hashes", context.Background(), query)
}
