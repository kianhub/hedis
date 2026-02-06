# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**hedis** (Hermes Decompiler/Disassembler) is a Go tool for analyzing React Native mobile applications for supply chain security risks. It decompiles Hermes bytecode (.hbc files), extracts function fingerprints, and matches them against a database of known vulnerable npm packages. Inspired by [P1Sec/hermes-dec](https://github.com/P1Sec/hermes-dec).

## Repository Layout

```
go/hermes-decompiler/    # Main Go CLI application (entry: main.go)
js/dependency-resolver/  # Node.js utility for resolving npm dependency trees
js/get-packages/         # Node.js scripts to fetch vulnerable packages from GitHub Security Advisories
pipeline/react-natives/  # 11 React Native environments (rn069–rn079) used for fingerprinting
pipeline/*.sh            # Setup scripts for Hermes compilers and RN environments
```

## Build & Run

All Go commands run from `go/hermes-decompiler/`:

```bash
# Build
go build -o hermes-decompiler .

# Run directly
go run main.go <command> [flags]

# Run tests
go test ./...

# Run a single test
go test ./pkg/analyzer/ -run TestMinHasher
```

## Environment Variables

A `.env` file in `go/hermes-decompiler/` is auto-loaded via `godotenv/autoload`. Required variables:

| Variable | Description |
|---|---|
| `MONGO_CONNECTION_STRING` | MongoDB connection URI |
| `MONGO_DB_NAME` | MongoDB database name (default: `hedis`) |
| `OS_HERMES` | Hermes binary platform: `osx-bin` (macOS) or `linux64-bin` (Linux) |
| `GITHUB_TOKEN` | GitHub PAT for Security Advisory API queries |

## CLI Commands

All commands are built with **cobra**. Root command: `hermes-decompiler`.

- **`disassemble`** — Disassemble .hbc files to human-readable output
  - `-i/--hbc` input file, `-o/--output` output file
  - `-s/--strings`, `-f/--functions`, `-j/--objects`, `-z/--function-objects`, `-r/--function-objects-ir`
  - `-n/--normalization` (0=none, 1=IR1, 2=IR2)

- **`analyze`** — Analyze .ipa or .hbc files against the fingerprint database
  - `-i/--ipa` or `-b/--hbc-file` for input
  - `-f/--fuzzy-matching`, `-c/--confidence-threshold` (default 0.8)
  - `-s/--specific-analysis` with `-p/--specific-package` and `-a/--compare-to-app-bundle` for direct comparison

- **`maintain-database`** — Build/update the fingerprint database
  - `-b/--baseline` — Generate baselines for each RN version
  - `-p/--packages` — Process packages (parallel pipeline across RN versions)
  - `-s/--update-security-advisories`, `-g/--download-github-advisories-and-packages`

- **`packages`** — Package management utilities (update DB, create hashes, analyze)

- **`genopcodes`** — Generate Go opcode definitions from Hermes source files
  - Run `sh pkg/utils/download_all.sh` first to download Hermes definitions

## Architecture

### HBC Reader (`pkg/hbc/`)
Parses Hermes bytecode files (magic: `0x1F1903C103BC1FC6`). Supports **27 bytecode versions** (v61–v96), each with generated opcode definitions in `pkg/hbc/types/opcodes/bcvXX/`. The reader extracts:
- Function headers and bytecode instructions (`HBCReader.go`, `bytecode_parser.go`)
- Function objects with IR generation (`types/functionobject.go`)
- Normalized representations via `normalizer.go`

### Fingerprinting (`pkg/hbc/types/functionobject.go`, `pkg/cmd/analyze.go`)
Three complementary hash types per function:
1. **Structural Hash** — SHA256 of instruction bigrams (opcode sequences)
2. **Content IR1 Hash** — SHA256 of non-identifier strings (with trigram shingling for fuzzy matching)
3. **Content IR2 Hash** — SHA256 of identifiers and object references

### Pipeline (`pkg/pipeline/`)
Processes npm packages across all 11 RN environments in parallel:
1. `npm install` package → Metro bundle → Hermes compile → disassemble → hash → store in MongoDB
2. `rnprocessor.go` — Parallel RN version processing (semaphore-limited goroutines)
3. `batcher.go` — Batched database writes (100 ops/batch)
4. `progress.go` — JSON-based resume capability (`pipeline_progress.json`)
5. `security.go` — GitHub Security Advisory GraphQL integration
6. `packages.go` — Package grouping and batch management

### Database (`pkg/database/`)
MongoDB via `mongopiet`. Database name: `hedis` (configurable via `MONGO_DB_NAME`; or `clean` for packages command). Key collections:
- `packages` — npm package metadata
- `hashes` / `hashes_ghsa` — function fingerprints per package per RN version
- `baselines_v3` — empty RN app fingerprints (used to filter framework functions)

### Analyzer (`pkg/analyzer/`)
Fuzzy matching via MinHash and Levenshtein distance. Length-based pre-filtering (±20% tolerance) reduces comparison space.

## Pipeline React Native Environments

Each `pipeline/react-natives/rnXXX/` directory is a standalone React Native project at that version. The pipeline installs packages into these environments one at a time, bundles with Metro, compiles with the version-specific Hermes compiler (`node_modules/hermes-engine/{osx-bin,linux64-bin}/hermesc`), then cleans up. These environments must **not** be modified or duplicated — the pipeline expects them in place with their `node_modules` intact.

## Opcode Generation

To add support for a new Hermes bytecode version:
1. Download definitions: `sh pkg/utils/download_all.sh`
2. Run: `go run main.go genopcodes`
3. Register the new version in `pkg/hbc/bytecode_parser.go` `GetParser` method
