package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	utilsDir           string
	defVersionsDir     string
	pkgDir             string
	hbcTypesOpcodesDir string
)

func initPaths() {
	var err error
	utilsDir, err = filepath.Abs("./pkg/utils")
	if err != nil {
		panic(err)
	}
	defVersionsDir = filepath.Join(utilsDir, "defversions")
	pkgDir = filepath.Join(utilsDir, "..")
	hbcTypesOpcodesDir = filepath.Join(pkgDir, "hbc", "types", "opcodes")
}

// GenerateBytecodeDefinitionFiles generates Go source files containing version-specific
// opcode definitions from Hermes bytecode definition (.def) files. It scans the
// defversions/ directory for BytecodeList, Builtins, and BytecodeVersion source files,
// extracts the bytecode version number, and emits a Go package under
// pkg/hbc/types/opcodes/bcvNN/ with instruction tables (OpcodeToInstruction,
// NameToInstruction) and BuiltinFunctionNames. Existing output files are skipped.
//
// The update parameter is reserved for automatic invocation of download_all.sh to
// fetch new Hermes definitions before generation (currently unimplemented).
//
// Prerequisites: run "sh pkg/utils/download_all.sh" to download Hermes definition
// files into defversions/ before calling this function. After generation, register
// new versions in pkg/hbc/bytecode_parser.go's GetParser method.
//
// This function is invoked by the "genopcodes" CLI command. It panics on file I/O
// errors or if a BytecodeVersion header cannot be parsed.
func GenerateBytecodeDefinitionFiles(update bool) {
	initPaths()

	if update {
		// TODO: implement automatic invocation of download_all.sh
	}
	files, err := os.ReadDir(filepath.Join(defVersionsDir, "function_builtins_src"))
	if err != nil {
		panic(err)
	}

	var tags []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".def") {
			parts := strings.Split(file.Name(), "-")
			if len(parts) > 1 {
				tag := strings.Split(parts[1], ".def")[0]
				tags = append(tags, tag)
			}
		}
	}
	sort.Strings(tags)

	for _, tag := range tags {
		// define file paths
		inputFileName := filepath.Join(defVersionsDir, "hermes_bytecode_src", fmt.Sprintf("BytecodeList-%s.def", tag))
		inputBuiltinsFileName := filepath.Join(defVersionsDir, "function_builtins_src", fmt.Sprintf("Builtins-%s.def", tag))
		inputVersionFileName := filepath.Join(defVersionsDir, "hermes_bytecode_src", fmt.Sprintf("BytecodeVersion-%s.h", tag))

		// get bytecode version
		bytecodeVersion := getBytecodeVersion(inputVersionFileName)
		// set output file dir & name
		outputFileName := filepath.Join(hbcTypesOpcodesDir, fmt.Sprintf("bcv%d/hbc%d.go", bytecodeVersion, bytecodeVersion))

		// skip if file already exists
		if _, err := os.Stat(outputFileName); err == nil {
			fmt.Printf("[hermes-decompiler] Skipping existing file => %s\n", outputFileName)
			continue
		}

		// get builtin function names
		builtinFunctionNames := getBuiltinFunctionNames(inputBuiltinsFileName)
		// create output source data buffer
		outSource := generateHeader(bytecodeVersion)

		// process bytecode definition file

		opcodeCount := 0
		inputSource := readInputFile(inputFileName)

		// backport operand declarations - if needed
		inputSource = backportFunctionIDs(inputSource)

		lines := strings.Split(inputSource, "\n")
		for _, line := range lines {
			opcodeCount = processLine(line, outSource, opcodeCount)
		}

		// add final data to output source data buffer
		addFinalData(outSource, builtinFunctionNames)

		// write output file from data buffer
		err := os.MkdirAll(filepath.Dir(outputFileName), 0755)
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(outputFileName, []byte(outSource.String()), 0644)
		if err != nil {
			panic(err)
		}

		fmt.Printf("[hermes-decompiler] Wrote Def File => %s\n", outputFileName)
	}
}

func getBytecodeVersion(versionFileName string) int {
	data, err := os.ReadFile(versionFileName)
	if err != nil {
		panic(err)
	}

	regBcv := regexp.MustCompile(`BYTECODE_VERSION = (\d+)`)
	matches := regBcv.FindSubmatch(data)
	if len(matches) < 2 {
		panic("could not find BYTECODE_VERSION in " + versionFileName)
	}

	version, err := strconv.Atoi(string(matches[1]))
	if err != nil {
		panic(err)
	}

	return version
}

func getBuiltinFunctionNames(builtinsFileName string) []string {
	file, err := os.Open(builtinsFileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var names []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "(")
		if len(parts) > 1 && (parts[0] == "BUILTIN_METHOD" ||
			parts[0] == "PRIVATE_BUILTIN" ||
			parts[0] == "JS_BUILTIN") {
			sp := strings.Split(parts[1], ")")[0]
			operands := strings.Split(sp, ", ")
			names = append(names, strings.Join(operands, "."))
		}
	}

	return names
}

func readInputFile(fileName string) string {
	data, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func backportFunctionIDs(inputSource string) string {
	if !strings.Contains(inputSource, "OPERAND_FUNCTION_ID(CallDirect, 3)") {
		inputSource += `
OPERAND_FUNCTION_ID(CallDirect, 3)
OPERAND_FUNCTION_ID(CreateClosure, 3)
OPERAND_FUNCTION_ID(CreateClosureLongIndex, 3)`
	}

	if strings.Contains(inputSource, "CreateGeneratorClosure") {
		inputSource += `
OPERAND_FUNCTION_ID(CreateGeneratorClosure, 3)
OPERAND_FUNCTION_ID(CreateGeneratorClosureLongIndex, 3)`
	}

	if strings.Contains(inputSource, "CreateGenerator") {
		inputSource += `
OPERAND_FUNCTION_ID(CreateGenerator, 3)
OPERAND_FUNCTION_ID(CreateGeneratorLongIndex, 3)`
	}

	if strings.Contains(inputSource, "CreateAsyncClosure") {
		inputSource += `
OPERAND_FUNCTION_ID(CreateAsyncClosure, 3)
OPERAND_FUNCTION_ID(CreateAsyncClosureLongIndex, 3)`
	}

	return inputSource
}

func generateHeader(bytecodeVersion int) *strings.Builder {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(` // Code automatically generated by "utils/bcdefparser.go". DO NOT EDIT.
package bcv%d

import (
	"hermes-decompiler/pkg/hbc/types"
)

var instructions []*types.Instruction
var OpcodeToInstruction map[int]*types.Instruction
var NameToInstruction map[string]*types.Instruction
var BuiltinFunctionNames []string

func init() {

`, bytecodeVersion))

	return &sb
}

func processLine(line string, sb *strings.Builder, opcodeCount int) int {
	regDefOp := regexp.MustCompile(`^((?:DEFINE|OPERAND)[^(]+?)\((.+?)\)`)
	matches := regDefOp.FindStringSubmatch(line)

	if len(matches) < 3 {
		return opcodeCount
	}

	directive := matches[1]
	args := strings.Split(matches[2], ", ")

	switch {
	case strings.HasPrefix(directive, "DEFINE_OPERAND_TYPE"):
		sb.WriteString(fmt.Sprintf("\t%s := types.NewOperandType(\"%s\",\"%s\")\n", args[0], args[0], args[1]))

	case strings.HasPrefix(directive, "DEFINE_OPCODE"):
		argsList := strings.Join(args[1:], ", ")
		sb.WriteString(fmt.Sprintf("\t%s := types.NewInstruction(\"%s\",%d,[]types.OperandType{%s})\n", args[0], args[0], opcodeCount, argsList))
		sb.WriteString(fmt.Sprintf("\tinstructions = append(instructions, %s)\n", args[0]))
		opcodeCount++

	case strings.HasPrefix(directive, "DEFINE_JUMP"):
		var jumpArgs []string
		switch directive {
		case "DEFINE_JUMP_1":
			jumpArgs = []string{"Addr8"}
		case "DEFINE_JUMP_2":
			jumpArgs = []string{"Addr8", "Reg8"}
		case "DEFINE_JUMP_3":
			jumpArgs = []string{"Addr8", "Reg8", "Reg8"}
		}
		args = append(args, jumpArgs...)
		argsList := strings.Join(args[1:], ", ")
		sb.WriteString(fmt.Sprintf("\t%s := types.NewInstruction(\"%s\",%d,[]types.OperandType{%s})\n", args[0], args[0], opcodeCount, argsList))
		sb.WriteString(fmt.Sprintf("\tinstructions = append(instructions, %s)\n", args[0]))
		longArgs := append([]string{"Addr32"}, args[2:]...)
		longArgsList := strings.Join(longArgs, ", ")
		sb.WriteString(fmt.Sprintf("\t%sLong := types.NewInstruction(\"%sLong\",%d,[]types.OperandType{%s})\n", args[0], args[0], opcodeCount+1, longArgsList))
		sb.WriteString(fmt.Sprintf("\tinstructions = append(instructions, %sLong)\n", args[0]))
		opcodeCount += 2

	case strings.HasPrefix(directive, "OPERAND_"):
		var meaningType string
		switch directive {
		case "OPERAND_BIGINT_ID":
			meaningType = "types.BigIntID"
		case "OPERAND_FUNCTION_ID":
			meaningType = "types.FunctionID"
		case "OPERAND_STRING_ID":
			meaningType = "types.StringID"
		}
		operandIndex, err := strconv.Atoi(args[1])
		if err != nil {
			panic(err)
		}
		sb.WriteString(fmt.Sprintf("\t%s.Operands[%d].Meaning = meaningPtr(%s)\n", args[0], operandIndex-1, meaningType))
	}

	return opcodeCount
}

func addMeaningPtrFunction(sb *strings.Builder) {
	sb.WriteString("\n\tfunc meaningPtr(m types.OperandMeaning) *types.OperandMeaning {\n")
	sb.WriteString("\t	return &m\n")
	sb.WriteString("\t}\n")
}

func addFinalData(sb *strings.Builder, builtinFunctionNames []string) {
	sb.WriteString("\n\tOpcodeToInstruction = make(map[int]*types.Instruction)\n")
	sb.WriteString("\tNameToInstruction = make(map[string]*types.Instruction)\n")
	sb.WriteString("\tfor _, inst := range instructions {\n")
	sb.WriteString("\t\tOpcodeToInstruction[inst.Opcode] = inst\n")
	sb.WriteString("\t\tNameToInstruction[inst.Name] = inst\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\tBuiltinFunctionNames = []string{\n")
	for _, name := range builtinFunctionNames {
		sb.WriteString(fmt.Sprintf("\t\t\"%s\",\n", name))
	}
	sb.WriteString("\t}\n")

	sb.WriteString("}\n")

	addMeaningPtrFunction(sb)
}
