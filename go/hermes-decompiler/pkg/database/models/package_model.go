package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Packages []Package

type PackageWithVulnerabilitiesModel struct {
	ID             primitive.ObjectID `bson:"_id"`
	PackageName    string             `bson:"package_name"`
	PackageVersion string             `bson:"package_version"`
	AdvisoryUrls   []string           `bson:"advisory_urls"`
	HasError       bool               `bson:"has_error"`
	Error          map[string]string  `bson:"error"`
}

// NPM Package Model
type Package struct {
	ID              primitive.ObjectID `bson:"_id"`
	PackageName     string             `bson:"package_name"`
	PackageVersion  string             `bson:"package_version"`
	PackageUniqueId string             `bson:"package_unique_id"` // package_name@package_version
	ErrorLog        ErrorLog           `bson:"error_log"`
	Metadata        Metadata           `bson:"metadata"`
}

// NPM Package Metadata Model
type Metadata struct {
	PackageAuthor           string            `bson:"package_author"`
	PackageHomepage         string            `bson:"package_homepage"`
	PackageRepository       string            `bson:"package_repository"`
	PackageDependencies     map[string]string `bson:"package_dependencies"`
	PackageDevDependencies  map[string]string `bson:"package_dev_dependencies"`
	PackagePeerDependencies map[string]string `bson:"package_peer_dependencies"`
	PackageScripts          map[string]string `bson:"package_scripts"`
}

// NPM Package Security Data Model
type SecurityData struct {
	OpenSourceVulnerabilities OpenSourceVulnerabilities `bson:"open_source_vulnerabilities"`
	GithubVulnerabilities     GithubVulnerabilities     `bson:"github_vulnerabilities"`
	// might add more fields here later
}

type OpenSourceVulnerabilities struct {
	Vulnerabilities []OSVulnerability `bson:"vulnerabilities"`
}

type OSVulnerability struct {
	Id       string `bson:"id"`
	Modified string `bson:"modified"`
}

type GithubVulnerabilities struct {
	Advisories []GithubVulnerability `bson:"advisories"`
}

type GithubVulnerability struct {
	Id                    primitive.ObjectID `bson:"id"`
	GhsaId                *string            `bson:"ghsa_id" json:"ghsa_id,omitempty"`
	CveId                 *string            `bson:"cve_id" json:"cve_id,omitempty"`
	HtmlUrl               *string            `bson:"html_url" json:"html_url,omitempty"`
	Summary               *string            `bson:"summary" json:"summary,omitempty"`
	Type                  *string            `bson:"type" json:"type,omitempty"`
	Severity              *string            `bson:"severity" json:"severity,omitempty"`
	RepositoryAdvisoryUrl *string            `bson:"repository_advisory_url" json:"repository_advisory_url,omitempty"`
	Identifiers           *[]Identifier      `bson:"identifiers" json:"identifiers,omitempty"`
	References            *[]string          `bson:"references" json:"references,omitempty"`
	PublishedAt           *string            `bson:"published_at" json:"published_at,omitempty"`
	UpdatedAt             *string            `bson:"updated_at" json:"updated_at,omitempty"`
	NvdPublishedAt        *string            `bson:"nvd_published_at" json:"nvd_published_at,omitempty"`
	WithdrawnAt           *string            `bson:"withdrawn_at" json:"withdrawn_at,omitempty"`
	Vulnerabilities       *[]Vulnerability   `bson:"vulnerabilities" json:"vulnerabilities,omitempty"`
	Cvss                  *Cvss              `bson:"cvss" json:"cvss,omitempty"`
	Cwes                  *[]Cwe             `bson:"cwes" json:"cwes,omitempty"`
}

type Identifier struct {
	Type  *string `bson:"type" json:"type,omitempty"`
	Value *string `bson:"value" json:"value,omitempty"`
}

type Vulnerability struct {
	PackageName            *PackageInfo `bson:"package" json:"package,omitempty"`
	VulnerableVersionRange *string      `bson:"vulnerable_version_range" json:"vulnerable_version_range,omitempty"`
	FirstPatchedVersion    *string      `bson:"first_patched_version" json:"first_patched_version,omitempty"`
	VulnerableFunctions    *[]string    `bson:"vulnerable_functions" json:"vulnerable_functions,omitempty"`
}

type Cvss struct {
	VectorString *string  `bson:"vector_string" json:"vector_string,omitempty"`
	Score        *float64 `bson:"score" json:"score,omitempty"`
}

type Cwe struct {
	CweId *string `bson:"cwe_id" json:"cwe_id,omitempty"`
	Name  *string `bson:"name" json:"name,omitempty"`
}

type PackageInfo struct {
	Name *string `json:"name,omitempty"`
}

type ErrorLog struct {
	HasError bool   `bson:"has_error"`
	Error    string `bson:"error"`
}

func (p Packages) Len() int {
	return len(p)
}

func (p Packages) Less(i, j int) bool {
	return p[i].PackageUniqueId < p[j].PackageUniqueId
}

func (p Packages) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
