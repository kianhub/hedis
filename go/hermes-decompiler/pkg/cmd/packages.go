package cmd

import (
	"encoding/json"
	"fmt"
	"hermes-decompiler/pkg/database"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/global"
	"hermes-decompiler/pkg/normhash"
	"hermes-decompiler/pkg/pipeline"
	"hermes-decompiler/pkg/utils"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	updateDatabaseFlag      bool
	createHashesFlag        bool
	analyzeWithDatabaseFlag bool
	legacyMatchFlag         bool
	forceRNVersionFlag      string

	packagesCmd = &cobra.Command{
		Use:   "packages",
		Short: "Create package models and hash them",
		Long:  "Takes the direct-vulnerabilities.json file, creates package models, pushes them into the database and also creates hashes for them if you want to.",
		Run: func(cmd *cobra.Command, args []string) {
			global.Verbose = verboseFlag

			utils.L.Println("Connecting to database...")
			dbName := os.Getenv("HEDIS_PACKAGES_DB") // override for local/eval DBs; default keeps original "clean"
			if dbName == "" {
				dbName = "clean"
			}
			database.InitWithDatabaseName(dbName)
			utils.L.Println("Database connected")

			global.OSHermes = os.Getenv("OS_HERMES")
			utils.L.Printf("OS_HERMES: %s\n", global.OSHermes)

			if updateDatabaseFlag {
				cwd, err := os.Getwd()
				if err != nil {
					utils.F.Printf("failed to get working directory: %v\n", err)
					return
				}

				// direct-vulnerabilities.json path
				directVulnerabilitiesPath := filepath.Join(cwd, "direct-vulnerabilities.json")

				packages, err := extractPackagesFromJson(directVulnerabilitiesPath)
				if err != nil {
					utils.F.Printf("failed to extract packages from json: %v\n", err)
					return
				}

				utils.L.Printf("Found %d packages in direct-vulnerabilities.json\n", amountOfUniquePackages(packages))
				utils.L.Printf("Total of %d package versions\n", len(packages))
				utils.L.Println("Updating packages in database...")
				for _, pkg := range packages {
					err := database.AddPackageToDatabase(&pkg)
					if err != nil {
						utils.F.Printf("failed to add package to database: %v\n", err)
						return
					}
				}
			}

			if createHashesFlag {
				utils.L.Println("Creating hashes for packages...")

				if err := pipeline.GetRNBaselines(); err != nil {
					utils.F.Printf("failed to get rn baselines: %v\n", err)
					return
				}

				packages, err := database.GetPackages()
				if err != nil {
					utils.F.Printf("failed to get packages: %v\n", err)
					return
				}

				utils.L.Printf("Found %d packages in database\n", len(packages))

				for _, pkg := range packages {
					pipeline.CreateHashesForPackage(pkg)
				}

				// for each package version, go to every react native version, install the package with the version, bundle the package, disassemble the bundle, create the hashes and add them to the database, if there is an error, add it to the error log of the package entry and continue with the next package version
				// it might be a good idea to parallelize the react native versions? so we can process multiple react native versions at the same time? which means the loop over the package versions waits for all react native versions to be done before moving on to the next package version
				// also, before saving the hashes, check the baseline collection to see if there are any matches, if there are, we delete those hashes from the package entry
			}

			if specificAnalysisFlag {
				if specificPackageFlag == "" {
					utils.F.Printf("specific package flag is not set\n")
					return
				}

				if compareToSpecificAppBundleFlag == "" {
					utils.F.Printf("compare to specific app bundle flag is not set\n")
					return
				}

				if outputFilePath == "" {
					utils.F.Printf("output file path is not set\n")
					return
				}

				utils.L.Println("Getting RN baselines...")
				if err := pipeline.GetRNBaselines(); err != nil {
					utils.F.Printf("failed to get rn baselines: %v\n", err)
					return
				}

				utils.L.Println("Extracting HBC versions...")
				appFos, appHBCVersion, err := pipeline.UseHermesDecompilerToDisassembleWithVersion(compareToSpecificAppBundleFlag)
				if err != nil {
					utils.F.Printf("failed to disassemble app bundle: %v\n", err)
					return
				}
				utils.S.Printf("App bundle HBC version: %s\n", appHBCVersion)
				utils.S.Printf("Found %d function objects in app bundle\n", len(appFos))
				packageFos, packageHBCVersion, err := pipeline.UseHermesDecompilerToDisassembleWithVersion(specificPackageFlag)
				if err != nil {
					utils.F.Printf("failed to disassemble package: %v\n", err)
					return
				}
				utils.S.Printf("Package HBC version: %s\n", packageHBCVersion)
				utils.S.Printf("Found %d function objects in package\n", len(packageFos))

				utils.L.Println("Generating hashes for app bundle...")
				appHashes := pipeline.GenerateHashesForSpecific(appFos)
				utils.L.Printf("Generated %d hashes for app bundle\n", len(appHashes))

				uniqueAppHashes := pipeline.CheckForDuplicates(appHashes)
				utils.L.Printf("Found %d unique hashes in app bundle\n", len(uniqueAppHashes))

				utils.L.Println("Generating hashes for package...")
				packageHashes := pipeline.GenerateHashesForSpecific(packageFos)
				utils.L.Printf("Generated %d hashes for package\n", len(packageHashes))

				utils.L.Println("Matching hashes with RN baselines...")

				filteredAppHashes, bestMatchRNVersion := pipeline.FindRNBaselineCandidates(appHashes, appHBCVersion)
				utils.L.Printf("Found %d hashes for app bundle that matched the best RN baseline: %s\n", len(filteredAppHashes), bestMatchRNVersion)

				/* filteredPackageHashes, bestMatchRNVersion := pipeline.FindRNBaselineCandidates(packageHashes, packageHBCVersion)
				utils.L.Printf("Found %d hashes for package that matched the best RN baseline: %s\n", len(filteredPackageHashes), bestMatchRNVersion) */

				utils.L.Printf("Comparing %d app bundle hashes with %d package hashes...\n", len(filteredAppHashes), len(packageHashes))

				matchResults := pipeline.CompareAppWithPackage(filteredAppHashes, packageHashes)
				c1Amount, c2Amount := pipeline.CountPackageMetadata(packageHashes)

				utils.S.Printf("Structural: %d/%d package hashes matched (%f%%)\n", matchResults.StructuralMatches, matchResults.TotalPackageHashes, matchResults.StructuralMatchPercentage)
				utils.S.Printf("ContentIR1: %d/%d package hashes matched (%f%%)\n", matchResults.ContentIR1Matches, c1Amount, matchResults.ContentIR1MatchPercentage)
				utils.S.Printf("ContentIR2: %d/%d package hashes matched (%f%%)\n", matchResults.ContentIR2Matches, c2Amount, matchResults.ContentIR2MatchPercentage)

				utils.S.Printf("Combined: %d/%d package hashes matched\n", matchResults.CombinedMatches, matchResults.TotalPackageHashes)

				utils.S.Printf("Total: %d/%d package hashes matched (%f%%)\n", matchResults.TotalAmountOfMatches, matchResults.TotalAppHashes, matchResults.TotalMatchPercentage)
			}

			if analyzeWithDatabaseFlag && fromDumpFlag != "" && normMatchFlag != "" {
				// match a --dump-app file (header "# rn X hbc Y", columns S C1 C2 NS NC) when the .hbc is unavailable
				tStart := time.Now()
				if err := pipeline.GetRNBaselines(); err != nil {
					utils.F.Printf("baselines: %v\n", err)
					return
				}
				b, err := os.ReadFile(fromDumpFlag)
				if err != nil {
					utils.F.Println(err)
					return
				}
				lines := strings.Split(string(b), "\n")
				hdr := strings.Fields(lines[0])
				rn, hbcV := hdr[2], hdr[4]
				var pairs [][2]string
				for _, l := range lines[1:] {
					// split on single spaces: empty C1/C2 columns must keep their position (strings.Fields shifted NS/NC)
					f := strings.Split(strings.TrimRight(l, "\r"), " ")
					if len(f) >= 4 && !strings.HasPrefix(l, "#") {
						nc := ""
						if len(f) >= 5 {
							nc = f[4]
						}
						pairs = append(pairs, [2]string{f[3], nc})
					}
				}
				cfg := pipeline.NormMatchConfig{KeyMode: normMatchFlag, TieBreak: tieBreakFlag, Thresholds: map[string]float64{"_global": 0}, LowUniq: lowUniqFlag, LowDf: lowDfFlag, Dep: depFlag, RootVer: rootVerFlag}
				if depsFlag != "" {
					if db, err := os.ReadFile(depsFlag); err == nil {
						json.Unmarshal(db, &cfg.Deps)
					}
				}
				if thresholdsFlag != "" {
					if tb, err := os.ReadFile(thresholdsFlag); err == nil {
						json.Unmarshal(tb, &cfg.Thresholds)
					}
				}
				rs, err := pipeline.NormMatchPairs(pairs, rn, cfg, nil)
				if err != nil {
					utils.F.Println(err)
					return
				}
				if err := writeMatchResultsJSON(rs, hbcV, rn, len(pairs), len(pairs), map[string]float64{"total_s": time.Since(tStart).Seconds()}); err != nil {
					utils.F.Println(err)
				}
				utils.S.Printf("norm match (dump): %d packages predicted\n", len(rs))
				return
			}

			if analyzeWithDatabaseFlag {
				if compareToSpecificAppBundleFlag == "" {
					utils.F.Printf("compare to specific app bundle flag is not set\n")
					return
				}

				tStart := time.Now()
				timings := map[string]float64{}
				utils.L.Println("Getting RN baselines...")
				err := pipeline.GetRNBaselines()
				if err != nil {
					utils.F.Printf("failed to get rn baselines: %v\n", err)
					return
				}

				utils.L.Println("Extracting HBC versions...")
				appFos, appHBCVersion, err := pipeline.UseHermesDecompilerToDisassembleWithVersion(compareToSpecificAppBundleFlag)
				if err != nil {
					utils.F.Printf("failed to disassemble app bundle: %v\n", err)
					return
				}
				utils.S.Printf("App bundle HBC version: %s\n", appHBCVersion)
				utils.S.Printf("Found %d function objects in app bundle\n", len(appFos))

				appHashes := pipeline.GenerateHashesForSpecific(appFos)
				utils.L.Printf("Generated %d hashes for app bundle\n", len(appHashes))

				utils.L.Println("Matching hashes with RN baselines...")
				tBase := time.Now()
				var filteredAppHashes []*models.FunctionObjectHashModel
				var bestMatchRNVersion string
				if legacyMatchFlag {
					filteredAppHashes, bestMatchRNVersion = pipeline.FindRNBaselineCandidates(appHashes, appHBCVersion)
				} else {
					filteredAppHashes, bestMatchRNVersion = pipeline.FindRNBaselineCandidatesFast(appHashes, appHBCVersion)
				}
				if forceRNVersionFlag != "" {
					bestMatchRNVersion = forceRNVersionFlag
				}
				timings["baseline_filter_s"] = time.Since(tBase).Seconds()
				if dumpAppFlag != "" {
					// one line per filtered app function: "S C1 C2" (for offline matcher experiments)
					if f, err := os.Create(dumpAppFlag); err == nil {
						fmt.Fprintf(f, "# rn %s hbc %s\n", bestMatchRNVersion, appHBCVersion)
						for _, h := range filteredAppHashes {
							ns, nc := normhash.Hashes(h.Structural.Raw, h.ContentAR1.Raw, h.ContentAR2.Raw)
							fmt.Fprintf(f, "%s %s %s %s %s\n", h.Structural.Hash, h.ContentAR1.Hash, h.ContentAR2.Hash, ns, nc)
						}
						f.Close()
						if os.Getenv("HEDIS_DUMP_RAW") != "" {
							if g, err := os.Create(dumpAppFlag + ".raw"); err == nil {
								for _, h := range filteredAppHashes {
									fmt.Fprintf(g, "%s\t%s\t%s\n", h.Structural.Raw, strings.ReplaceAll(h.ContentAR1.Raw, "\n", " "), strings.ReplaceAll(h.ContentAR2.Raw, "\n", " "))
								}
								g.Close()
							}
						}
					}
					if dumpOnlyFlag {
						return
					}
				}
				utils.L.Printf("Found %d hashes for app bundle that matched the best RN baseline: %s\n", len(filteredAppHashes), bestMatchRNVersion)

				if normMatchFlag != "" {
					cfg := pipeline.NormMatchConfig{KeyMode: normMatchFlag, TieBreak: tieBreakFlag, Thresholds: map[string]float64{"_global": 0}, LowUniq: lowUniqFlag, LowDf: lowDfFlag, Dep: depFlag, RootVer: rootVerFlag}
					if depsFlag != "" {
						if db, err := os.ReadFile(depsFlag); err == nil {
							json.Unmarshal(db, &cfg.Deps)
						}
					}
					if thresholdsFlag != "" {
						if b, err := os.ReadFile(thresholdsFlag); err == nil {
							json.Unmarshal(b, &cfg.Thresholds)
						}
					}
					tN := time.Now()
					rs, err := pipeline.NormMatch(appHashes, bestMatchRNVersion, cfg)
					if err != nil {
						utils.F.Printf("norm match failed: %v\n", err)
						return
					}
					timings["norm_match_s"] = time.Since(tN).Seconds()
					timings["total_s"] = time.Since(tStart).Seconds()
					if err := writeMatchResultsJSON(rs, appHBCVersion, bestMatchRNVersion, len(appHashes), len(filteredAppHashes), timings); err != nil {
						utils.F.Printf("failed to write json results: %v\n", err)
					}
					utils.S.Printf("norm match: %d packages predicted\n", len(rs))
					return
				}
				tDB := time.Now()
				utils.L.Println("Getting hashes from database...")
				var databasePackageHashes []*pipeline.HashGroup
				if legacyMatchFlag {
					databasePackageHashes, err = pipeline.GetHashesFromDatabaseWithRNVersion(bestMatchRNVersion)
				} else {
					databasePackageHashes, err = pipeline.GetHashesFromDatabaseFast(bestMatchRNVersion)
				}
				if err != nil {
					utils.F.Printf("failed to get hashes from database: %v\n", err)
					return
				}

				totalAmountOfHashesFromDb := 0
				for _, dbPkgHashes := range databasePackageHashes {
					totalAmountOfHashesFromDb += len(dbPkgHashes.Hashes)
				}

				utils.L.Printf("Found %d packages with a total of %d hashes in database for RN version %s\n", len(databasePackageHashes), totalAmountOfHashesFromDb, bestMatchRNVersion)

				timings["db_load_s"] = time.Since(tDB).Seconds()
				utils.L.Println("Comparing app bundle hashes with database hashes...")
				tCmp := time.Now()
				matchResults := make([]*pipeline.MatchResults, len(databasePackageHashes))
				appSet := pipeline.AppHashSet(filteredAppHashes)
				for i, dbHashGroup := range databasePackageHashes {
					if legacyMatchFlag {
						matchResults[i] = pipeline.CompareAppWithPackage(filteredAppHashes, dbHashGroup.Hashes)
					} else {
						matchResults[i] = pipeline.CompareAppWithPackageFast(appSet, len(filteredAppHashes), dbHashGroup.Hashes)
					}
					matchResults[i].PackageInfo = dbHashGroup.Result
				}
				timings["compare_s"] = time.Since(tCmp).Seconds()
				timings["total_s"] = time.Since(tStart).Seconds()
				if err := writeMatchResultsJSON(matchResults, appHBCVersion, bestMatchRNVersion, len(appHashes), len(filteredAppHashes), timings); err != nil {
					utils.F.Printf("failed to write json results: %v\n", err)
				}

				slices.SortFunc(matchResults, func(a, b *pipeline.MatchResults) int {
					return b.CombinedMatches - a.CombinedMatches
				})

				// write all combined matches/total and their percentage + package name + package version to a file, one line per matchresult
				if err := writeMatchResultsToFile(matchResults); err != nil {
					utils.F.Printf("failed to write match results to file: %v\n", err)
					return
				}

				utils.L.Println("Top 10 matches:")
				for i := 0; i < 10 && i < len(matchResults); i++ {
					utils.S.Printf("%d. %s@%s (%s)\n", i+1, matchResults[i].PackageInfo.PackageName, matchResults[i].PackageInfo.PackageVersion, matchResults[i].PackageInfo.ID.Hex())
					utils.S.Printf("Combined: %d/%d (%f%%) package hashes matched\n", matchResults[i].CombinedMatches, matchResults[i].TotalPackageHashes, matchResults[i].TotalMatchPercentage)
				}
			}
		},
	}
)

// writeMatchResultsJSON writes every (package, version) score plus run metadata next to the text output.
func writeMatchResultsJSON(rs []*pipeline.MatchResults, hbcVersion, rnVersion string, nApp, nFiltered int, timings map[string]float64) error {
	type row struct {
		Package string  `json:"package"`
		Version string  `json:"version"`
		Matched int     `json:"matched_fos"`
		Total   int     `json:"total_fos"`
		Score   float64 `json:"score"`
	}
	out := struct {
		HBC      string             `json:"hbc_version"`
		RN       string             `json:"rn_version"`
		AppFOs   int                `json:"app_fos"`
		Filtered int                `json:"app_fos_after_baseline"`
		Timings  map[string]float64 `json:"timings"`
		Results  []row              `json:"results"`
	}{hbcVersion, rnVersion, nApp, nFiltered, timings, nil}
	for _, r := range rs {
		out.Results = append(out.Results, row{r.PackageInfo.PackageName, r.PackageInfo.PackageVersion, r.CombinedMatches, r.TotalPackageHashes, r.TotalMatchPercentage / 100})
	}
	p := outputFilePath
	if p == "" {
		p = filepath.Base(compareToSpecificAppBundleFlag) + "_match_results.txt"
	}
	b, _ := json.MarshalIndent(out, "", " ")
	return os.WriteFile(p+".json", b, 0644)
}

func writeMatchResultsToFile(matchResults []*pipeline.MatchResults) error {
	if outputFilePath == "" {
		outputFilePath = filepath.Base(compareToSpecificAppBundleFlag) + "_match_results.txt"
	}

	matchResultsFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create match results file: %v", err)
	}
	defer matchResultsFile.Close()

	for _, matchResult := range matchResults {
		fmt.Fprintf(matchResultsFile, "%d/%d (%f%%) %s@%s\n", matchResult.CombinedMatches, matchResult.TotalPackageHashes, matchResult.TotalMatchPercentage, matchResult.PackageInfo.PackageName, matchResult.PackageInfo.PackageVersion)
	}

	fmt.Fprintf(matchResultsFile, "\n\n")
	fmt.Fprintf(matchResultsFile, "Package versions with the highest combined matches:\n")

	// map by package name and show the version with the highest combined matches
	packageNameToMatchResults := make(map[string][]*pipeline.MatchResults)
	for _, matchResult := range matchResults {
		packageNameToMatchResults[matchResult.PackageInfo.PackageName] = append(packageNameToMatchResults[matchResult.PackageInfo.PackageName], matchResult)
	}
	// Create a slice to hold the best match result for each package
	var bestMatchResults []*pipeline.MatchResults
	for _, matchResults := range packageNameToMatchResults {
		slices.SortFunc(matchResults, func(a, b *pipeline.MatchResults) int {
			return int(b.TotalMatchPercentage*100) - int(a.TotalMatchPercentage*100)
		})
		bestMatchResults = append(bestMatchResults, matchResults[0])
	}

	// Sort packages by their highest total match percentage
	slices.SortFunc(bestMatchResults, func(a, b *pipeline.MatchResults) int {
		return int(b.TotalMatchPercentage*100) - int(a.TotalMatchPercentage*100)
	})

	// Print the sorted results
	for _, matchResult := range bestMatchResults {
		fmt.Fprintf(matchResultsFile, "%s: %s@%s (%d/%d (%f%%))\n", matchResult.PackageInfo.PackageName, matchResult.PackageInfo.PackageName, matchResult.PackageInfo.PackageVersion, matchResult.CombinedMatches, matchResult.TotalPackageHashes, matchResult.TotalMatchPercentage)
	}

	return nil
}

func extractPackagesFromJson(jsonPath string) ([]models.PackageWithVulnerabilitiesModel, error) {
	packages, err := pipeline.GetPackagesFromVulnerabilitiesJson(jsonPath)
	if err != nil {
		return nil, err
	}

	return packages, nil
}

func amountOfUniquePackages(packages []models.PackageWithVulnerabilitiesModel) int {
	uniquePackages := make(map[string]bool)
	for _, pkg := range packages {
		uniquePackages[pkg.PackageName] = true
	}
	return len(uniquePackages)
}

var (
	fromDumpFlag   string
	normMatchFlag  string
	tieBreakFlag   bool
	thresholdsFlag string
	depsFlag       string
	depFlag        float64
	lowUniqFlag    int
	lowDfFlag      int
	rootVerFlag    bool
	dumpAppFlag    string
	dumpOnlyFlag   bool
)

func init() {
	packagesCmd.Flags().BoolVarP(&updateDatabaseFlag, "update-database", "u", false, "Update the database with the packages")
	packagesCmd.Flags().BoolVar(&createHashesFlag, "create-hashes", false, "Create hashes for the packages and adds them to the database")
	packagesCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")
	packagesCmd.Flags().BoolVarP(&specificAnalysisFlag, "specific-analysis", "s", false, "Enable specific analysis")
	packagesCmd.Flags().StringVarP(&specificPackageFlag, "specific-package", "p", "", "Path to the package HBC file to analyze")
	packagesCmd.Flags().StringVarP(&compareToSpecificAppBundleFlag, "compare-to-app-bundle", "a", "", "Path to the app bundle HBC file to compare to")
	packagesCmd.Flags().StringVarP(&outputFilePath, "output-file", "o", "", "Path to the output file to write fuzzy matches to")
	packagesCmd.Flags().BoolVar(&legacyMatchFlag, "legacy-match", false, "Use the original O(n*m) matching loops (for benchmarking)")
	packagesCmd.Flags().StringVar(&fromDumpFlag, "from-dump", "", "with -d --norm: read app NS/NC from a --dump-app file instead of an .hbc")
	packagesCmd.Flags().StringVar(&normMatchFlag, "norm", "", "normalized-hash matcher: nsnc or ns (needs a --release DB)")
	packagesCmd.Flags().BoolVar(&tieBreakFlag, "tiebreak", false, "with --norm: choose version by F1 of version keys vs app keys")
	packagesCmd.Flags().BoolVar(&rootVerFlag, "rootver", false, "with --norm: choose the version on root-entry keys only (ignore subpath rows)")
	packagesCmd.Flags().StringVar(&depsFlag, "deps", "", "with --norm: npm dependency table JSON {pkg: {version: [deps]}} for --dep")
	packagesCmd.Flags().Float64Var(&depFlag, "dep", 0, "with --norm: dependency implication min coverage of parent-shared keys (0 = off)")
	packagesCmd.Flags().IntVar(&lowUniqFlag, "lowuniq", 0, "with --norm: packages with <= N unique keys score on keys with df <= --lowdf")
	packagesCmd.Flags().IntVar(&lowDfFlag, "lowdf", 2, "with --norm --lowuniq: max package df of fallback keys")
	packagesCmd.Flags().StringVar(&thresholdsFlag, "thresholds", "", "with --norm: JSON {package: min score, _global: x} tuned on dev")
	packagesCmd.Flags().StringVar(&dumpAppFlag, "dump-app", "", "Write filtered app triples (S C1 C2 per line) to this file")
	packagesCmd.Flags().BoolVar(&dumpOnlyFlag, "dump-only", false, "With --dump-app: exit after dumping (skip DB match)")
	packagesCmd.Flags().StringVar(&forceRNVersionFlag, "rn-version", "", "Force the RN version (e.g. 0.74) instead of auto-detecting from baselines")
	packagesCmd.Flags().BoolVarP(&analyzeWithDatabaseFlag, "analyze-with-database", "d", false, "Analyze with the database")
}
