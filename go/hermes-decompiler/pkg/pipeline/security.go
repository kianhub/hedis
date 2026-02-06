package pipeline

import (
	"encoding/json"
	"fmt"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/utils"
	"net/http"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.philip.id/mongopiet/pkg/bulk"
	"go.philip.id/mongopiet/pkg/db"
)

func GetPurlsFromFromReactNativeDirectoryJson(reactNativeDirectoryJsonPath string) ([]string, error) {
	packages, err := GetPackagesFromReactNativeDirectoryJson(reactNativeDirectoryJsonPath)
	if err != nil {
		return nil, err
	}

	purls := make([]string, 0, len(packages))

	for _, pkg := range packages {
		purls = append(purls, fmt.Sprintf("pkg:npm/%s", pkg.PackageName))
	}

	return purls, nil
}

func GetPackageNamesFromReactNativeDirectoryJson(reactNativeDirectoryJsonPath string) ([]string, error) {
	packages, err := GetPackagesFromReactNativeDirectoryJson(reactNativeDirectoryJsonPath)
	if err != nil {
		return nil, err
	}

	groups := GroupPackagesByName(packages)

	packageNames := make([]string, 0, len(groups))

	for packageName := range groups {
		packageNames = append(packageNames, packageName)
	}

	return packageNames, nil
}

func GetGithubSecurityAdvisories(packageNames []string) ([]models.GithubVulnerability, error) {
	foundAdvisories := make([]models.GithubVulnerability, 0, len(packageNames))

	// Get the GitHub token from environment variable
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	// Process packages in batches of 100 (GitHub API limit)
	for i := 0; i < len(packageNames); i += 100 {
		end := i + 100
		if end > len(packageNames) {
			end = len(packageNames)
		}

		batch := packageNames[i:end]
		packageQuery := strings.Join(batch, ",")

		// Build the initial URL
		url := fmt.Sprintf("https://api.github.com/advisories?ecosystem=npm&per_page=100&affects=%s", packageQuery)

		// Process all pages for this batch
		for url != "" {
			utils.L.Printf("Fetching batch %d of %d\n", i/100+1, len(packageNames)/100)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %v", err)
			}

			// Add required headers
			req.Header.Set("Authorization", "Bearer "+githubToken)
			req.Header.Set("Accept", "application/vnd.github+json")

			// Make the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
			}

			// Parse the response
			var advisories []models.GithubVulnerability
			if err := json.NewDecoder(resp.Body).Decode(&advisories); err != nil {
				return nil, fmt.Errorf("failed to decode response: %v", err)
			}

			foundAdvisories = append(foundAdvisories, advisories...)

			// Check for pagination in the Link header
			url = ""
			linkHeader := resp.Header.Get("Link")
			if linkHeader != "" {
				utils.L.Printf("Paginated response, extracting next page URL\n")
				url = extractNextPageURL(linkHeader)
			}
		}
	}

	return foundAdvisories, nil
}

func GetAllGithubSecurityAdvisories() ([]models.GithubVulnerability, error) {
	// we grab all advisories from github with the ecosystem npm
	// we then paginate through all the pages
	// we then return all the advisories

	// the endpoint is https://api.github.com/advisories?ecosystem=npm&per_page=100

	foundAdvisories := make([]models.GithubVulnerability, 0)

	// Get the GitHub token from environment variable
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}

	// Start with the base URL for all npm advisories
	url := "https://api.github.com/advisories?ecosystem=npm&per_page=100"
	pageNum := 1

	// Process all pages
	for url != "" {
		utils.L.Printf("Fetching all npm advisories page %d\n", pageNum)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		// Add required headers
		req.Header.Set("Authorization", "Bearer "+githubToken)
		req.Header.Set("Accept", "application/vnd.github+json")

		// Make the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
		}

		// Parse the response
		var advisories []models.GithubVulnerability
		if err := json.NewDecoder(resp.Body).Decode(&advisories); err != nil {
			return nil, fmt.Errorf("failed to decode response: %v", err)
		}

		utils.L.Printf("Fetched %d advisories from page %d\n", len(advisories), pageNum)
		foundAdvisories = append(foundAdvisories, advisories...)

		// Check for pagination in the Link header
		url = ""
		linkHeader := resp.Header.Get("Link")
		if linkHeader != "" {
			utils.L.Printf("Paginated response, extracting next page URL\n")
			url = extractNextPageURL(linkHeader)
		}

		pageNum++
	}

	utils.L.Printf("Total advisories fetched: %d\n", len(foundAdvisories))
	return foundAdvisories, nil
}

func CreateDirectoryJsonFromGithubSecurityAdvisories(advisories []models.GithubVulnerability) error {
	// for every advisory, we need to get the package name (only once, so we need to group by package name)
	// and then we send a request to the endpoint https://registry.npmjs.org/package-name
	// we then parse the response and create a models.Package object from the response
	// finally, we write a .json file with all the packages (one array of models.Package)

	// Extract unique package names from all advisories
	packageNames := extractUniquePackageNames(advisories)
	utils.L.Printf("Found %d unique packages in security advisories\n", len(packageNames))

	// Fetch package information from NPM registry
	packages, err := fetchPackageInfoFromNPM(packageNames)
	if err != nil {
		return fmt.Errorf("failed to fetch package info from NPM: %v", err)
	}

	// Convert to models.Package format
	modelPackages := convertNPMResponseToPackages(packages)
	utils.L.Printf("Created %d package models\n", len(modelPackages))

	// Write to JSON file
	err = writePackagesToJSONFile(modelPackages, "security-advisories-packages.json")
	if err != nil {
		return fmt.Errorf("failed to write packages to JSON file: %v", err)
	}

	utils.S.Printf("Successfully created security-advisories-packages.json with %d packages\n", len(modelPackages))
	return nil
}

// extractUniquePackageNames extracts unique package names from security advisories
func extractUniquePackageNames(advisories []models.GithubVulnerability) []string {
	packageNameSet := make(map[string]bool)

	for _, advisory := range advisories {
		if advisory.Vulnerabilities != nil {
			for _, vulnerability := range *advisory.Vulnerabilities {
				if vulnerability.PackageName != nil && vulnerability.PackageName.Name != nil {
					packageName := *vulnerability.PackageName.Name
					if packageName != "" {
						packageNameSet[packageName] = true
					}
				}
			}
		}
	}

	// Convert set to slice
	packageNames := make([]string, 0, len(packageNameSet))
	for packageName := range packageNameSet {
		packageNames = append(packageNames, packageName)
	}

	return packageNames
}

// NPMPackageResponse represents the response from NPM registry
type NPMPackageResponse struct {
	ID       string                     `json:"_id"`
	Rev      string                     `json:"_rev"`
	Name     string                     `json:"name"`
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]*NPMVersionInfo `json:"versions"`
}

type NPMVersionInfo struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Description      string            `json:"description,omitempty"`
	Homepage         string            `json:"homepage,omitempty"`
	Author           json.RawMessage   `json:"author,omitempty"`
	Repository       json.RawMessage   `json:"repository,omitempty"`
	Dependencies     map[string]string `json:"dependencies,omitempty"`
	DevDependencies  map[string]string `json:"devDependencies,omitempty"`
	PeerDependencies map[string]string `json:"peerDependencies,omitempty"`
	Scripts          map[string]string `json:"scripts,omitempty"`
}

type NPMRepository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type NPMAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// fetchPackageInfoFromNPM fetches package information from NPM registry
func fetchPackageInfoFromNPM(packageNames []string) ([]*NPMPackageResponse, error) {
	var packages []*NPMPackageResponse
	client := &http.Client{}

	for i, packageName := range packageNames {
		utils.L.Printf("Fetching package %d/%d: %s\n", i+1, len(packageNames), packageName)

		url := fmt.Sprintf("https://registry.npmjs.org/%s", packageName)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			utils.F.Printf("Failed to create request for package %s: %v\n", packageName, err)
			continue
		}

		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			utils.F.Printf("Failed to fetch package %s: %v\n", packageName, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			utils.F.Printf("NPM registry returned status %d for package %s\n", resp.StatusCode, packageName)
			continue
		}

		var npmPackage NPMPackageResponse
		if err := json.NewDecoder(resp.Body).Decode(&npmPackage); err != nil {
			utils.F.Printf("Failed to decode response for package %s: %v\n", packageName, err)
			continue
		}

		packages = append(packages, &npmPackage)
	}

	utils.L.Printf("Successfully fetched %d packages from NPM registry\n", len(packages))
	return packages, nil
}

// parseRepositoryURL extracts the repository URL from either string or object format
func parseRepositoryURL(repositoryRaw json.RawMessage) string {
	if len(repositoryRaw) == 0 {
		return ""
	}

	// Try to parse as object first
	var repoObj NPMRepository
	if err := json.Unmarshal(repositoryRaw, &repoObj); err == nil && repoObj.URL != "" {
		return repoObj.URL
	}

	// Try to parse as string
	var repoStr string
	if err := json.Unmarshal(repositoryRaw, &repoStr); err == nil {
		return repoStr
	}

	return ""
}

// parseAuthorName extracts the author name from either string or object format
func parseAuthorName(authorRaw json.RawMessage) string {
	if len(authorRaw) == 0 {
		return ""
	}

	// Try to parse as object first
	var authorObj NPMAuthor
	if err := json.Unmarshal(authorRaw, &authorObj); err == nil && authorObj.Name != "" {
		return authorObj.Name
	}

	// Try to parse as string
	var authorStr string
	if err := json.Unmarshal(authorRaw, &authorStr); err == nil {
		return authorStr
	}

	return ""
}

// convertNPMResponseToPackages converts NPM registry responses to DirectoryPackageJson format
func convertNPMResponseToPackages(npmPackages []*NPMPackageResponse) []DirectoryPackageJson {
	var directoryPackages []DirectoryPackageJson

	for _, npmPkg := range npmPackages {
		// Extract versions
		versions := make([]string, 0, len(npmPkg.Versions))
		dependenciesMap := make(map[string]map[string]string)
		devDependenciesMap := make(map[string]map[string]string)
		peerDependenciesMap := make(map[string]map[string]string)
		scriptsMap := make(map[string]map[string]string)

		// Track package-level metadata from the latest or first version
		var packageAuthor string
		var packageHomepage string
		var packageRepository string

		for version, versionInfo := range npmPkg.Versions {
			versions = append(versions, version)

			// Store version-specific data
			if versionInfo.Dependencies != nil {
				dependenciesMap[version] = versionInfo.Dependencies
			}
			if versionInfo.DevDependencies != nil {
				devDependenciesMap[version] = versionInfo.DevDependencies
			}
			if versionInfo.PeerDependencies != nil {
				peerDependenciesMap[version] = versionInfo.PeerDependencies
			}
			if versionInfo.Scripts != nil {
				scriptsMap[version] = versionInfo.Scripts
			}

			// Use the first version's metadata as package-level metadata
			// (or prioritize if this is the latest version)
			if packageAuthor == "" {
				packageAuthor = parseAuthorName(versionInfo.Author)
			}
			if packageHomepage == "" && versionInfo.Homepage != "" {
				packageHomepage = versionInfo.Homepage
			}
			if packageRepository == "" {
				packageRepository = parseRepositoryURL(versionInfo.Repository)
			}
		}

		directoryPackage := DirectoryPackageJson{
			PackageName:             npmPkg.Name,
			PackageVersion:          versions,
			PackageAuthor:           packageAuthor,
			PackageHomepage:         packageHomepage,
			PackageRepository:       packageRepository,
			PackageDependencies:     dependenciesMap,
			PackageDevDependencies:  devDependenciesMap,
			PackagePeerDependencies: peerDependenciesMap,
			PackageScripts:          scriptsMap,
		}

		directoryPackages = append(directoryPackages, directoryPackage)
	}

	return directoryPackages
}

// writePackagesToJSONFile writes the packages to a JSON file
func writePackagesToJSONFile(packages []DirectoryPackageJson, filename string) error {
	jsonData, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal packages to JSON: %v", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	return nil
}

// extractNextPageURL parses the GitHub Link header to find the next page URL
func extractNextPageURL(linkHeader string) string {
	// GitHub Link header format: <https://api.github.com/path?page=2>; rel="next", <https://api.github.com/path?page=5>; rel="last"
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) == 2 {
			rel := strings.TrimSpace(parts[1])
			if rel == `rel="next"` {
				url := strings.TrimSpace(parts[0])
				// Remove the angle brackets
				if len(url) > 2 && url[0] == '<' && url[len(url)-1] == '>' {
					return url[1 : len(url)-1]
				}
			}
		}
	}
	return ""
}

func AddGithubSecurityAdvisoriesToDatabase(advisories []models.GithubVulnerability) error {
	if len(advisories) == 0 {
		return nil
	}

	write := []bulk.Write{}
	upsert := true

	for _, advisory := range advisories {
		advisory.Id = primitive.NewObjectID()
		write = append(write, &bulk.Replace{
			Filter:      bson.M{"id": advisory.Id, "ghsa_id": advisory.GhsaId},
			Upsert:      &upsert,
			Replacement: &advisory,
		})
	}

	if _, err := db.BulkWrite("gh_advisories", write); err != nil {
		return fmt.Errorf("failed to write advisories to database: %v", err)
	}

	return nil
}
