const fs = require("fs/promises");
const path = require("path");

async function readFile() {
    const file = await fs.readFile(
        path.join(__dirname, "vulnerable-packages.json"),
        "utf-8"
    );
    return JSON.parse(file);
}

async function main() {
    const data = await readFile();

    const directVulnerabilities = data.vulnerable_packages.filter(
        (pkg) => pkg.direct_vulnerabilities.length > 0
    );

    console.log(
        "Packages with direct vulnerabilities: " + directVulnerabilities.length
    );

    var totalAmountOfVulnerableVersions = 0;
    const dataToWrite = [];

    for (const pkg of directVulnerabilities) {
        const entry = {
            name: pkg.name,
            versions: [],
        };
        console.log(pkg.name);
        let versions = new Map();
        for (const vuln of pkg.direct_vulnerabilities) {
            const githubUrl = vuln.url;
            for (const version of vuln.vulnerable_versions) {
                if (!versions.has(version)) {
                    versions.set(version, []);
                }
                versions.get(version).push(githubUrl);
            }
        }

        for (const version of versions.keys()) {
            entry.versions.push({
                version: version,
                urls: versions.get(version),
            });
        }

        totalAmountOfVulnerableVersions += versions.size;
        console.log(versions);
        dataToWrite.push(entry);
    }

    console.log(
        "Total amount of vulnerable versions: " +
            totalAmountOfVulnerableVersions
    );

    await fs.writeFile(
        path.join(__dirname, "direct-vulnerabilities.json"),
        JSON.stringify(dataToWrite, null, 2)
    );
}

main();
