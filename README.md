# HEDIS

**Hermes Decompiler/Disassembler** — A Go tool for analyzing React Native mobile applications for supply chain security risks.

Hedis decompiles Hermes bytecode (`.hbc` files), extracts function fingerprints, and matches them against a database of known vulnerable npm packages. It helps security researchers identify whether a React Native app includes dependencies with known vulnerabilities.

Inspired by [P1Sec/hermes-dec](https://github.com/P1Sec/hermes-dec).

> **Note:** This is primarily also a test to use AI agents. We used Opus 4.6 and [Maestro](https://runmaestro.ai) to refactor, document and update the original codebase, in order to make it more accessible to the public. If we believe this approach didn’t yield satisfactory results, we will revert to the original code and manually recreate the repository to adhere to the code used in the thesis paper.

## Research

This tool was developed as part of my bachelor thesis at TU Darmstadt titled **"Evaluating Dependency Vulnerabilities in React Native Applications"**. The thesis presents a novel static analysis pipeline for detecting vulnerable JavaScript packages within production React Native applications. The approach works by constructing a reference database of npm package fingerprints extracted from disassembled Hermes bytecode, then matching these fingerprints against real-world application bundles obtained from the iOS App Store. Validation was performed on open-source apps (Rainbow, Mattermost) and production apps (Discord, Microsoft Outlook), demonstrating the feasibility of coarse-grained dependency detection across different React Native versions and build configurations.

The full thesis is available at: [doi.org/10.26083/tuprints-00031438](https://doi.org/10.26083/tuprints-00031438)

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

### `genopcodes`

Generate Go opcode definition packages from downloaded Hermes source files. See [Hermes Bytecode Versions & Opcode Generation](#hermes-bytecode-versions--opcode-generation) for the full workflow.

```bash
# Download Hermes definitions first
sh pkg/utils/download_all.sh

# Generate Go opcode packages
hermes-decompiler genopcodes
```

### `packages`

Package management utilities for updating the database and creating hashes.

```bash
hermes-decompiler packages -u                # Update database
hermes-decompiler packages --create-hashes   # Create hashes
hermes-decompiler packages -d -a <app.hbc>   # Analyze app against database
```

## Hermes Bytecode Versions & Opcode Generation

Hermes bytecode files embed a version number that determines the instruction set used to encode the JavaScript. Each version of the Hermes engine may introduce, remove, or reorder opcodes, so the decompiler needs version-specific instruction tables to correctly parse any given `.hbc` file.

Hedis ships with pre-generated opcode definitions for **30 bytecode versions** (v61–v96), covering Hermes releases from v0.1.0 through v0.14.0+. These definitions live in `go/hermes-decompiler/pkg/hbc/types/opcodes/bcvNN/` as auto-generated Go packages.

### When you need to regenerate or add opcodes

- **Analyzing apps built with a newer Hermes version** — If you encounter an `.hbc` file whose bytecode version exceeds v96, you need to add support for that version.
- **Updating after a new Hermes release** — When Facebook ships a new Hermes version with bytecode changes, the opcode tables must be regenerated to include the new instruction set.

### How opcode generation works

1. **Download Hermes definitions** from the official [facebook/hermes](https://github.com/nicolo-ribaudo/hermes) repository. This fetches the C++ instruction definitions (`BytecodeList.def`), version headers (`BytecodeVersion.h`), and builtin function lists (`Builtins.def`) for each tagged release:

   ```bash
   cd go/hermes-decompiler
   sh pkg/utils/download_all.sh
   ```

   The downloaded files are stored in `pkg/utils/defversions/`.

2. **Generate Go opcode packages** from the downloaded definitions. This parses each version's `.def` files and produces a Go package with instruction lookup tables:

   ```bash
   go run main.go genopcodes
   ```

   Each generated package (e.g., `pkg/hbc/types/opcodes/bcv96/hbc96.go`) contains:
   - An ordered list of all instructions with their operand types and sizes
   - Opcode-to-instruction and name-to-instruction lookup maps
   - Builtin function name lists

3. **Register the new version in the parser.** Generated opcode packages are not automatically used — you must register them in `pkg/hbc/bytecode_parser.go`'s `GetParser()` method by adding the new version to the `parserModuleTable` map and importing the generated package.

### Parser version fallback

Not every bytecode version requires its own parser entry. The parser uses a fallback strategy: it selects the highest registered version that does not exceed the file's bytecode version. For example, a v92 file will use the v90 parser if v92 is not explicitly registered. Only versions that introduce meaningful instruction set changes need their own entry.

## Repository Layout

```
go/hermes-decompiler/    # Main Go CLI application
js/dependency-resolver/  # Node.js utility for resolving npm dependency trees
js/get-packages/         # Scripts to fetch vulnerable packages from GitHub Security Advisories
pipeline/react-natives/  # 11 React Native environments (rn069–rn079) for fingerprinting
pipeline/*.sh            # Setup scripts for Hermes compilers and RN environments
```
