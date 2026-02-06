# Get Vulnerable Packages Script

This script identifies React Native packages with known security vulnerabilities.

It fetches a list of curated React Native packages, cross-references them with the GitHub Advisories Database, and checks for vulnerable versions using the NPM registry.

## Prerequisites

- Node.js (v18 or higher recommended)
- A GitHub Personal Access Token (PAT) to avoid rate limiting by the GitHub API.

## Setup

1.  **Clone the repository.**
2.  **Install dependencies:**
    ```bash
    npm install
    ```
3.  **Set up your GitHub Token:**

    Create a `.env` file in the root of the project and add your GitHub token:

    ```
    GITHUB_TOKEN=your_github_personal_access_token
    ```

    The script will work without a token, but you might encounter rate-limiting from the GitHub API, which could cause the script to fail.

## Running the Script

To run the script, use the following command:

```bash
npm start
```

The script will perform the following steps:

1.  Fetch all React Native packages from the `reactnative.directory`.
2.  Fetch all `npm` ecosystem advisories from the GitHub Advisory Database.
3.  Identify which of the React Native packages have vulnerabilities listed.
4.  For each vulnerable package, it will fetch all available versions from the NPM registry.
5.  It will determine the exact versions that are affected by the vulnerabilities.
6.  The results will be saved in a `vulnerable-packages.json` file in the project root.

## Output

The output is a `vulnerable-packages.json` file with the following structure:

```json
{
  "vulnerable_packages": [
    {
      "name": "some-package-name",
      "advisories": [
        {
          "ghsa_id": "GHSA-...",
          "cve_id": "CVE-...",
          "url": "https://github.com/advisories/...",
          "vulnerable_version_range": ">= 1.0.0 < 1.0.5",
          "vulnerable_versions": ["1.0.0", "1.0.1", "1.0.2", "1.0.3", "1.0.4"]
        }
      ]
    }
  ]
}
```
