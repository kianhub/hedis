const fs = require("fs/promises");
const path = require("path");
const semver = require("semver");
require("dotenv").config();

const GITHUB_TOKEN = process.env.GITHUB_TOKEN;

async function fetchWithPagination(url, headers = {}, getNextPage) {
    let items = [];
    let nextPageUrl = url;

    while (nextPageUrl) {
        console.log(`Fetching from ${nextPageUrl}`);
        try {
            const response = await fetch(nextPageUrl, { headers });
            if (!response.ok) {
                throw new Error(
                    `HTTP error! status: ${response.status} ${response.statusText}`
                );
            }
            const data = await response.json();
            const newItems = Array.isArray(data) ? data : data.libraries || [];
            items = items.concat(newItems);
            nextPageUrl = getNextPage(response, data);
        } catch (error) {
            console.error("Error fetching data:", error);
            break;
        }
    }

    return items;
}

async function getReactNativePackages() {
    console.log("Fetching React Native packages...");
    const baseUrl =
        "https://reactnative.directory/api/libraries?ios=true&skipTools=true&skipTemplates=true";
    const packages = await fetchWithPagination(
        baseUrl,
        {},
        (response, data) => {
            const currentOffset =
                new URL(response.url).searchParams.get("offset") || 0;
            const newOffset = parseInt(currentOffset, 10) + 30;
            return data.libraries && data.libraries.length > 0
                ? `${baseUrl}&offset=${newOffset}`
                : null;
        }
    );
    return packages.map((pkg) => pkg.npmPkg);
}

async function getGithubAdvisories() {
    console.log("Fetching GitHub advisories...");
    const url = "https://api.github.com/advisories?ecosystem=npm&per_page=100";
    const headers = {
        Accept: "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    };
    if (GITHUB_TOKEN) {
        headers["Authorization"] = `Bearer ${GITHUB_TOKEN}`;
    } else {
        console.warn("GITHUB_TOKEN not set. You may face rate-limiting.");
    }

    const advisories = await fetchWithPagination(url, headers, (response) => {
        const linkHeader = response.headers.get("link");
        if (linkHeader) {
            const match = linkHeader.match(/<([^>]+)>;\s*rel="next"/);
            return match ? match[1] : null;
        }
        return null;
    });

    const advisoriesByPackage = {};
    for (const advisory of advisories) {
        for (const vulnerability of advisory.vulnerabilities) {
            const pkgName = vulnerability.package.name;
            if (!advisoriesByPackage[pkgName]) {
                advisoriesByPackage[pkgName] = [];
            }
            advisoriesByPackage[pkgName].push({
                ghsa_id: advisory.ghsa_id,
                cve_id: advisory.cve_id,
                url: advisory.html_url,
                vulnerable_version_range:
                    vulnerability.vulnerable_version_range,
            });
        }
    }
    return advisoriesByPackage;
}

async function getNpmPackageVersions(packageName) {
    console.log(`Fetching versions for ${packageName}...`);
    try {
        const response = await fetch(
            `https://registry.npmjs.org/${packageName}`
        );
        if (!response.ok) {
            if (response.status === 404) {
                console.warn(`Package ${packageName} not found on npm.`);
                return { versions: [], versionData: {} };
            }
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        const versions = Object.keys(data.versions);
        const versionData = {};

        // Extract dependencies for each version
        for (const version of versions) {
            const versionInfo = data.versions[version];
            versionData[version] = {
                dependencies: versionInfo.dependencies || {},
            };
        }

        return { versions, versionData };
    } catch (error) {
        console.error(
            `Error fetching package versions for ${packageName}:`,
            error
        );
        return { versions: [], versionData: {} };
    }
}

async function main() {
    const rnPackages = await getReactNativePackages();
    const rnPackageSet = new Set(rnPackages);
    console.log(`Found ${rnPackages.length} React Native packages.`);

    const advisoriesByPackage = await getGithubAdvisories();
    console.log(
        `Found advisories for ${
            Object.keys(advisoriesByPackage).length
        } unique packages.`
    );

    const vulnerablePackages = [];

    for (const pkgName of rnPackages) {
        const { versions, versionData } = await getNpmPackageVersions(pkgName);
        if (versions.length === 0) continue;

        const vulnerablePackageData = {
            name: pkgName,
            direct_vulnerabilities: [],
            dependency_vulnerabilities: [],
        };

        // Check for direct vulnerabilities
        if (advisoriesByPackage[pkgName]) {
            for (const advisory of advisoriesByPackage[pkgName]) {
                const vulnerableVersions = versions.filter((version) =>
                    semver.satisfies(
                        version,
                        advisory.vulnerable_version_range.replace(",", ""),
                        { includePrerelease: false }
                    )
                );

                if (vulnerableVersions.length > 0) {
                    vulnerablePackageData.direct_vulnerabilities.push({
                        ...advisory,
                        vulnerable_versions: vulnerableVersions,
                    });
                }
            }
        }

        // Check for dependency vulnerabilities
        const dependencyVulnerabilities = new Map(); // Use Map to avoid duplicates

        for (const version of versions) {
            const deps = versionData[version];
            const allDependencies = {
                ...deps.dependencies,
                // Note: Excluding devDependencies as they don't affect production usage
            };

            for (const depName of Object.keys(allDependencies)) {
                if (advisoriesByPackage[depName]) {
                    const depVersion = allDependencies[depName];
                    if (
                        !semver.valid(depVersion) &&
                        !semver.validRange(depVersion)
                    ) {
                        continue;
                    }

                    for (const advisory of advisoriesByPackage[depName]) {
                        // Create a unique key for this vulnerability
                        const vulnKey = `${depName}-${advisory.ghsa_id}`;

                        if (!dependencyVulnerabilities.has(vulnKey)) {
                            dependencyVulnerabilities.set(vulnKey, {
                                dependency_name: depName,
                                dependency_version_range: depVersion,
                                affected_package_versions: [],
                                ...advisory,
                            });
                        }

                        if (
                            semver.satisfies(
                                depVersion,
                                advisory.vulnerable_version_range.replace(
                                    ",",
                                    ""
                                ),
                                { includePrerelease: false }
                            ) ||
                            semver.intersects(
                                depVersion,
                                advisory.vulnerable_version_range.replace(
                                    ",",
                                    ""
                                )
                            )
                        ) {
                            const vulnEntry =
                                dependencyVulnerabilities.get(vulnKey);
                            // Add this package version to the affected versions for this dependency vulnerability
                            if (
                                !vulnEntry.affected_package_versions.includes(
                                    version
                                )
                            ) {
                                vulnEntry.affected_package_versions.push(
                                    version
                                );
                            }
                        }
                    }
                }
            }
        }

        // Convert Map to array
        vulnerablePackageData.dependency_vulnerabilities = Array.from(
            dependencyVulnerabilities.values()
        );

        // Only include packages that have either direct or dependency vulnerabilities
        if (
            vulnerablePackageData.direct_vulnerabilities.length > 0 ||
            vulnerablePackageData.dependency_vulnerabilities.length > 0
        ) {
            vulnerablePackages.push(vulnerablePackageData);
        }
    }

    const results = {
        vulnerable_packages: vulnerablePackages,
        summary: {
            total_packages_checked: rnPackages.length,
            packages_with_direct_vulnerabilities: vulnerablePackages.filter(
                (p) => p.direct_vulnerabilities.length > 0
            ).length,
            packages_with_dependency_vulnerabilities: vulnerablePackages.filter(
                (p) => p.dependency_vulnerabilities.length > 0
            ).length,
            total_vulnerable_packages: vulnerablePackages.length,
        },
    };

    const outputPath = path.join(__dirname, "vulnerable-packages.json");
    await fs.writeFile(outputPath, JSON.stringify(results, null, 2));

    console.log(
        `\nProcessing complete. Found ${vulnerablePackages.length} vulnerable React Native packages.`
    );
    console.log(
        `- ${results.summary.packages_with_direct_vulnerabilities} packages with direct vulnerabilities`
    );
    console.log(
        `- ${results.summary.packages_with_dependency_vulnerabilities} packages with dependency vulnerabilities`
    );
    console.log(`Results saved to ${outputPath}`);
}

main().catch((error) => {
    console.error("An unexpected error occurred:", error);
    process.exit(1);
});
