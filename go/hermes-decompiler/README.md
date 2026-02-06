# Hermes Decompiler

This project is part of a larger effort to analyze mobile React Native applications for security risks stemming from potential vulnerabilities in their JavaScript dependencies. This Hermes Disassembler/Decompiler has been heavily inspired by the work of **P1Sec**'s [hermes-dec](https://github.com/P1Sec/hermes-dec).

The primary goal of this project is to provide a comprehensive and efficient Hermes bytecode decompiler written in Go.
Using this decompiler, we can analyze, manipulate and fingerprint Hermes bytecode files.

## Project Structure

```
hermes-decompiler
├── main.go        # Main executable entry point
├── input/         # Directory for input HBC files
├── output/        # Directory for decompiler output
└── pkg/
    ├── cmd/
    │   ├── disassembleHbcFile.go  # Logic for the disassembly command
    │   └── genopcodes.go          # Logic for generating opcode definitions from the downloaded Hermes definitions
    ├── hbc/
    │   ├── HBCReader.go           # Core HBC file reading logic
    │   ├── bytecode_parser.go     # Parsing bytecode instructions
    │   └── types/                 # HBC data structures (incl. opcodes)
    │
    └── utils/
        ├── bcdefparser.go         # Parser for bytecode definitions
        ├── defversions/           # Hermes definitions versioned data
        └── download_all.sh        # Script to download all necessary data from the Hermes repository
```

## Commands

### Gen Opcodes From Hermes Files

Download the necessary Hermes definitions and generate opcode definitions.
```sh
sh ./pkg/utils/download_all.sh
```
Generate the opcode defintions as Go files.
```sh
go run main.go genopcodes
```
Make sure to write the generated opcode definitions to the `pkg/hbc/bytecode_parser.go#GetParser` method.

### Disassemble HBC File

Disassemble an HBC file into human-readable bytecode instructions.
```sh
go run main.go disassemble --hbc <path/to/hbc/file> --output <path/to/output/file>
# or
go run main.go disassemble -i <path/to/hbc/file> -o <path/to/output/file>
```
