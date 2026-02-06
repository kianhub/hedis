package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"hermes-decompiler/pkg/database"
	"hermes-decompiler/pkg/global"
	"hermes-decompiler/pkg/pipeline"
	"hermes-decompiler/pkg/utils"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.philip.id/mongopiet/pkg/db"

	"hermes-decompiler/pkg/database/models"
)

var (
	baselineFlag                            bool
	packagesFlag                            bool
	verboseFlag                             bool
	updateSecurityAdvisoriesFlag            bool
	downloadGithubAdvisoriesAndPackagesFlag bool
	changeCollectionFlag                    bool

	maintainDatabaseCmd = &cobra.Command{
		Use:   "maintain-database",
		Short: "Maintain the database",
		Long:  "Maintain the database",
		Run: func(cmd *cobra.Command, args []string) {
			global.Verbose = verboseFlag

			utils.L.Println("Connecting to database...")
			database.Init()
			utils.L.Println("Database connected")

			osHermes := os.Getenv("OS_HERMES")
			utils.L.Printf("OS_HERMES: %s\n", osHermes)

			if baselineFlag {
				utils.L.Println("Checking for new React Native Versions to create a baseline for...")

				// Determine the absolute path to the react-natives directory.
				cwd, err := os.Getwd()
				if err != nil {
					utils.F.Printf("failed to get working directory: %v\n", err)
					return
				}
				// project layout of `code/pipeline/react-natives` - binary is executed from inside the
				// `go/hermes-decompiler` directory (two levels below the repository root).
				rnRoot := filepath.Join(cwd, "..", "..", "pipeline", "react-natives")

				entries, err := os.ReadDir(rnRoot)
				if err != nil {
					utils.F.Printf("failed to read react-natives directory: %v\n", err)
					return
				}

				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}

					name := entry.Name()
					if !strings.HasPrefix(name, "rn") {
						continue
					}

					// Extract numeric part and convert it into a React Native version string.
					digits := strings.TrimPrefix(name, "rn")
					num, err := strconv.Atoi(digits)
					if err != nil {
						utils.D.Printf("skipping folder %s: cannot parse version\n", name)
						continue
					}

					version := fmt.Sprintf("%.2f", float64(num)/100)

					// Check if a baseline entry already exists in the database.
					_, err = db.FindOne[models.BaselineModel]("baselines_v3", bson.M{"react_native_version": version})
					if err != nil {
						if err == mongo.ErrNoDocuments {
							utils.L.Printf("React Native %s missing baseline -> running pipeline\n", version)

							reactNativeWorkingDirectoryPath := filepath.Join(rnRoot, name)

							hermesCompilerPath := filepath.Join(reactNativeWorkingDirectoryPath, "node_modules", "react-native", "sdks", "hermesc", osHermes, "hermesc")
							err = pipeline.RunBaselinePipeline(reactNativeWorkingDirectoryPath, hermesCompilerPath, version)
							if err != nil {
								utils.F.Printf("error running baseline pipeline for version %s: %v\n", version, err)
							}
						} else {
							utils.F.Printf("error querying baseline for version %s: %v\n", version, err)
						}
					} else {
						utils.D.Printf("Baseline for React Native %s already exists\n", version)
					}
				}
			}

			if packagesFlag {
				utils.L.Println("Checking for new packages to create a baseline for...")

				// Determine the absolute path to the react-natives directory.
				cwd, err := os.Getwd()
				if err != nil {
					utils.F.Printf("failed to get working directory: %v\n", err)
					return
				}

				rnRoot := filepath.Join(cwd, "..", "..", "pipeline", "react-natives")
				reactNatives, err := os.ReadDir(rnRoot)
				if err != nil {
					utils.F.Printf("failed to read react-natives directory: %v\n", err)
					return
				}

				directoryJsonPath := filepath.Join(cwd, "react-native-directory-packages.json")

				if changeCollectionFlag {
					pipeline.CHANGE_COLLECTION = true
					directoryJsonPath = filepath.Join(cwd, "security-advisories-packages.json")
				}

				packages, err := pipeline.GetPackagesFromReactNativeDirectoryJson(directoryJsonPath)
				if err != nil {
					utils.F.Printf("failed to get packages from packages directory json: %v\n", err)
					return
				}

				utils.S.Printf("Found %d packages in packages directory json\n", len(packages))
				sort.Sort(packages)

				// PHASE 4: Load existing progress
				globalProgress := pipeline.NewGlobalProgress("pipeline_progress.json")
				err = globalProgress.LoadProgress()
				if err != nil {
					utils.F.Printf("Error loading progress: %v\n", err)
					return
				}

				// Filter valid React Native directories
				var validRNDirs []os.DirEntry
				for _, reactNative := range reactNatives {
					if !reactNative.IsDir() {
						continue
					}

					name := reactNative.Name()
					if !strings.HasPrefix(name, "rn") {
						continue
					}

					// Verify we can parse the version
					digits := strings.TrimPrefix(name, "rn")
					_, err := strconv.Atoi(digits)
					if err != nil {
						utils.D.Printf("skipping folder %s: cannot parse version\n", name)
						continue
					}

					validRNDirs = append(validRNDirs, reactNative)
				}

				// PHASE 4: Filter out completed RN environments
				pendingRNVersions := pipeline.FilterPendingRNVersions(validRNDirs, globalProgress)
				globalProgress.TotalRNVersions = len(validRNDirs)

				utils.L.Printf("Total RN versions: %d, Pending: %d, Completed: %d\n",
					len(validRNDirs), len(pendingRNVersions), globalProgress.CompletedRNVersions)

				if len(pendingRNVersions) == 0 {
					utils.S.Println("All RN versions already completed!")
					return
				}

				// Parallel processing of pending React Native versions
				var wg sync.WaitGroup
				results := make(chan []*models.PackageHash, len(pendingRNVersions))
				semaphore := make(chan struct{}, 4) // Limit to 4 concurrent RN environments

				for _, reactNative := range pendingRNVersions {
					wg.Add(1)
					go func(rnDir os.DirEntry) {
						defer wg.Done()
						semaphore <- struct{}{}        // Acquire semaphore
						defer func() { <-semaphore }() // Release semaphore

						// PHASE 4: Get or create progress for this RN version
						progress := globalProgress.GetOrCreateRNProgress(rnDir.Name())

						utils.L.Printf("Starting parallel processing for RN version: %s\n", rnDir.Name())
						rnHashes, err := pipeline.ProcessPackagesForRNVersionWithProgress(rnDir, packages, rnRoot, progress, osHermes)
						if err != nil {
							utils.F.Printf("Error processing RN version %s: %v\n", rnDir.Name(), err)
							results <- []*models.PackageHash{} // Send empty result
						} else {
							results <- rnHashes
						}

						// PHASE 4: Mark RN version as completed and save progress
						globalProgress.MarkRNVersionComplete(rnDir.Name())
						err = globalProgress.SaveProgress()
						if err != nil {
							utils.F.Printf("Error saving progress: %v\n", err)
						}

						utils.L.Printf("Completed parallel processing for RN version: %s\n", rnDir.Name())
					}(reactNative)
				}

				// Wait for all workers to complete and close results channel
				go func() {
					wg.Wait()
					close(results)
				}()

				// Collect all results
				var allPackageHashes []*models.PackageHash
				for rnHashes := range results {
					allPackageHashes = append(allPackageHashes, rnHashes...)
				}

				utils.S.Printf("--- Parallel processing completed ---\n")
				utils.S.Printf("Total RN versions processed: %d\n", len(pendingRNVersions))
				utils.S.Printf("Total package hashes generated: %d\n", len(allPackageHashes))
				utils.S.Println("All package processing completed successfully!")
			}

			if updateSecurityAdvisoriesFlag {
				utils.L.Println("Updating security advisories...")

				cwd, err := os.Getwd()
				if err != nil {
					utils.F.Printf("failed to get working directory: %v\n", err)
					return
				}
				reactNativeDirectoryJsonPath := filepath.Join(cwd, "react-native-directory-packages.json")

				packageNames, err := pipeline.GetPackageNamesFromReactNativeDirectoryJson(reactNativeDirectoryJsonPath)
				if err != nil {
					utils.F.Printf("failed to create purls from react-natives directory: %v\n", err)
					return
				}

				utils.S.Printf("Found %d package names\n", len(packageNames))
				utils.L.Println("Getting security advisories from GitHub...")

				advisories, err := pipeline.GetGithubSecurityAdvisories(packageNames)
				if err != nil {
					utils.F.Printf("failed to get security advisories from GitHub: %v\n", err)
					return
				}

				utils.S.Printf("Found %d security advisories\n", len(advisories))

				err = pipeline.AddGithubSecurityAdvisoriesToDatabase(advisories)
				if err != nil {
					utils.F.Printf("failed to add security advisories to database: %v\n", err)
					return
				}

				utils.S.Println("Security advisories added to database successfully")
			}

			if downloadGithubAdvisoriesAndPackagesFlag {
				utils.L.Println("Downloading github advisories and packages...")

				advisories, err := pipeline.GetAllGithubSecurityAdvisories()
				if err != nil {
					utils.F.Printf("failed to get all github security advisories: %v\n", err)
					return
				}

				utils.S.Printf("Found %d security advisories\n", len(advisories))

				err = pipeline.AddGithubSecurityAdvisoriesToDatabase(advisories)
				if err != nil {
					utils.F.Printf("failed to add security advisories to database: %v\n", err)
					return
				}

				/* err = pipeline.CreateDirectoryJsonFromGithubSecurityAdvisories(advisories)
				if err != nil {
					utils.F.Printf("failed to create directory json from github security advisories: %v\n", err)
					return
				} */

				utils.S.Println("Github advisories and packages downloaded successfully")
			}
		},
	}
)

func init() {
	maintainDatabaseCmd.Flags().BoolVarP(&baselineFlag, "baseline", "b", false, "Create or update a React Native Version baseline (empty entry point) in the database")
	maintainDatabaseCmd.Flags().BoolVarP(&packagesFlag, "packages", "p", false, "Create or update a package baseline in the database")
	maintainDatabaseCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose output")
	maintainDatabaseCmd.Flags().BoolVarP(&updateSecurityAdvisoriesFlag, "update-security-advisories", "s", false, "Update security advisories")
	maintainDatabaseCmd.Flags().BoolVarP(&downloadGithubAdvisoriesAndPackagesFlag, "download-github-advisories-and-packages", "g", false, "Download github advisories and packages")
	maintainDatabaseCmd.Flags().BoolVarP(&changeCollectionFlag, "change-collection", "c", false, "Change the collection name")
}
