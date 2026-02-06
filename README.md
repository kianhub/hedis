# HEDIS

**Hermes Decompiler/Disassembler** — A Go tool for analyzing React Native mobile applications for supply chain security risks.

Hedis decompiles Hermes bytecode (`.hbc` files), extracts function fingerprints, and matches them against a database of known vulnerable npm packages. It helps security researchers identify whether a React Native app includes dependencies with known vulnerabilities.

Inspired by [P1Sec/hermes-dec](https://github.com/P1Sec/hermes-dec).

> **Note:** This is primarily also a test to use AI agents. We used Opus 4.6 and [Maestro](https://runmaestro.ai) to refactor, document and update the original codebase in order to make this more accessible to the public. If we think this didn't go well, we will rollback and manually remake this repository to stay true to the original code that has been used in the thesis paper.

## Prerequisites

- **Go** 1.23+
- **MongoDB** — Used to store function fingerprints and package metadata
- **Node.js** — Required for JS utilities (`js/dependency-resolver/`, `js/get-packages/`)

### Database Collections

| Collection | Purpose |
|---|---|
| `packages` | npm package metadata |
| `hashes` / `hashes_ghsa` | Function fingerprints per package per RN version |
| `baselines_v3` | Empty RN app fingerprints (used to filter framework functions) |

## Build & Run

All commands are run from `go/hermes-decompiler/`:

```bash
# Build
go build -o hermes-decompiler .

# Run directly
go run main.go <command> [flags]

# Run tests
go test ./...
```

## Configuration

Create a `.env` file in `go/hermes-decompiler/` (see `.env.example`). Variables are auto-loaded via `godotenv`.

| Variable | Description |
|---|---|
| `MONGO_CONNECTION_STRING` | MongoDB connection URI |
| `MONGO_DB_NAME` | Database name (default: `hedis`) |
| `OS_HERMES` | Hermes binary platform: `osx-bin` (macOS) or `linux64-bin` (Linux) |
| `GITHUB_TOKEN` | GitHub PAT for Security Advisory API queries |

## Supported Formats

- **Hermes Bytecode:** Versions v61–v96 (30 versions)
- **Input Formats:** `.hbc` (Hermes bytecode), `.ipa` (iOS app archives)

## CLI Commands

### `disassemble`

Disassemble `.hbc` files to human-readable output.

```bash
hermes-decompiler disassemble -i <input.hbc> -o <output.txt>
```

Flags:
- `-i/--hbc` — Input HBC file
- `-o/--output` — Output file
- `-s/--strings` — Include strings
- `-f/--functions` — Include functions (default: true)
- `-j/--objects` — Include objects
- `-z/--function-objects` — Include function objects
- `-r/--function-objects-ir` — Include function objects IR
- `-n/--normalization` — Normalization level (0=none, 1=IR1, 2=IR2)

### `analyze`

Analyze `.ipa` or `.hbc` files against the fingerprint database for known vulnerable packages.

```bash
# Analyze an HBC file directly
hermes-decompiler analyze -b <bundle.hbc>

# With fuzzy matching
hermes-decompiler analyze -b <bundle.hbc> -f -c 0.8

# Compare a specific package to an app bundle
hermes-decompiler analyze -s -p <package.hbc> -a <app-bundle.hbc>
```

Flags:
- `-i/--ipa` — Input IPA file
- `-b/--hbc-file` — Input HBC file
- `-f/--fuzzy-matching` — Enable fuzzy matching
- `-c/--confidence-threshold` — Confidence threshold (default: 0.8)
- `-s/--specific-analysis` — Analyze a specific package
- `-p/--specific-package` — Package HBC file for specific analysis
- `-a/--compare-to-app-bundle` — App bundle HBC file to compare against
- `-o/--output-file` — Output results to file

### `maintain-database`

Build and update the fingerprint database.

```bash
# Generate baselines for each RN version
hermes-decompiler maintain-database -b

# Process packages (parallel pipeline across RN versions)
hermes-decompiler maintain-database -p

# Update security advisories
hermes-decompiler maintain-database -s

# Download GitHub advisories and packages
hermes-decompiler maintain-database -g
```

### `packages`

Package management utilities for updating the database and creating hashes.

```bash
hermes-decompiler packages -u                # Update database
hermes-decompiler packages --create-hashes   # Create hashes
hermes-decompiler packages -d -a <app.hbc>   # Analyze app against database
```

## Repository Layout

```
go/hermes-decompiler/    # Main Go CLI application
js/dependency-resolver/  # Node.js utility for resolving npm dependency trees
js/get-packages/         # Scripts to fetch vulnerable packages from GitHub Security Advisories
pipeline/react-natives/  # 11 React Native environments (rn069–rn079) for fingerprinting
pipeline/*.sh            # Setup scripts for Hermes compilers and RN environments
```
