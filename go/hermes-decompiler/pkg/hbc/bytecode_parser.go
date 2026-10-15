package hbc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hermes-decompiler/pkg/hbc/types"
	"hermes-decompiler/pkg/hbc/types/opcodes/bcv84"
	"hermes-decompiler/pkg/hbc/types/opcodes/bcv85"
	"hermes-decompiler/pkg/hbc/types/opcodes/bcv89"
	"hermes-decompiler/pkg/hbc/types/opcodes/bcv90"
	"hermes-decompiler/pkg/hbc/types/opcodes/bcv94"
	"hermes-decompiler/pkg/hbc/types/opcodes/bcv96"
	"hermes-decompiler/pkg/hbc/types/opcodes/bcv98"
	"io"
	"log"
	"strings"
)

// ParsedInstruction represents a single decoded Hermes bytecode instruction
// produced by ParseHBCBytecode. It pairs the static instruction definition
// (Inst) with the concrete operand values (Args) read from the bytecode
// stream. OriginalPos and NextPos record the instruction's byte offset within
// the function's bytecode buffer (not the file), enabling address resolution
// for relative jumps. For SwitchImm instructions, SwitchJumpTable holds the
// resolved jump target offsets from the 4-byte-aligned jump table that follows
// the instruction. The HBCReader back-reference allows the String() method to
// resolve semantic operand meanings (StringID, FunctionID, BigIntID) and
// inline literal values, array contents, and object key-value pairs as
// human-readable comments in the disassembly output.
type ParsedInstruction struct {
	Inst            *types.Instruction
	Args            []any
	SwitchJumpTable []int
	OriginalPos     int
	NextPos         int
	HBCReader       *HBCReader
}

func (p *ParsedInstruction) String() string {
	operands := make([]string, 0, len(p.Inst.Operands))

	for i, operand := range p.Inst.Operands {
		var name string
		if operand.Meaning != nil {
			name = operandMeaningName(*operand.Meaning)
		} else {
			name = operand.Type.Name
		}
		operands = append(operands, fmt.Sprintf("%s: %v", name, p.Args[i]))
	}

	comment := ""
	for i, operand := range p.Inst.Operands {
		operandValue := p.Args[i]

		if operand.Meaning != nil {
			var idx int

			idxRaw, ok := operandValue.(uint)
			if ok {
				idx = int(idxRaw)
			}
			switch *operand.Meaning {
			case types.StringID:
				if ok && idx >= 0 && idx < len(p.HBCReader.Strings) {
					stringVal := p.HBCReader.Strings[idx]
					if idx < len(p.HBCReader.StringKinds) {
						comment += fmt.Sprintf("  # String %q (%s)", stringVal, StringKindName(p.HBCReader.StringKinds[idx]))
					} else {
						comment += fmt.Sprintf("  # String %q", stringVal)
					}
				}
			case types.BigIntID:
				if ok && idx >= 0 && idx < len(p.HBCReader.BigIntValues) {
					comment += fmt.Sprintf("  # BigInt %d", p.HBCReader.BigIntValues[idx])
				}
			case types.FunctionID:
				if ok && idx >= 0 && idx < len(p.HBCReader.FunctionHeaders) {
					funcHeader := p.HBCReader.FunctionHeaders[idx]
					var funcName string
					var bytecodeSize, paramCount, offset int

					// extract values based on header type
					switch header := funcHeader.(type) {
					case *SmallFunctionHeader:
						if int(header.FunctionName) < len(p.HBCReader.Strings) {
							funcName = p.HBCReader.Strings[header.FunctionName]
						}
						bytecodeSize = int(header.BytecodeSizeInBytes)
						paramCount = int(header.ParamCount)
						offset = int(header.Offset)
					case *LargeFunctionHeader:
						if int(header.FunctionName) < len(p.HBCReader.Strings) {
							funcName = p.HBCReader.Strings[header.FunctionName]
						}
						bytecodeSize = int(header.BytecodeSizeInBytes)
						paramCount = int(header.ParamCount)
						offset = int(header.Offset)
					}

					comment += fmt.Sprintf("  # Function: [#%d '%s' of %d bytes]: %d params @ offset 0x%08x", idx, funcName, bytecodeSize, paramCount, offset)
				}
			}
		} else if operand.Type.Name == "Addr8" || operand.Type.Name == "Addr32" {
			if val, ok := operandValue.(int); ok {
				comment += fmt.Sprintf("  # Address: %08x", p.OriginalPos+val)
			}
		}
	}

	// handle special instruction types
	switch p.Inst.Name {
	case "NewArrayWithBuffer", "NewArrayWithBufferLong":
		if len(p.Args) >= 4 {
			var arrayIdx, arraySize int

			arrayIdxRaw, ok1 := p.Args[3].(uint)
			if ok1 {
				arrayIdx = int(arrayIdxRaw)
			}
			arraySizeRaw, ok2 := p.Args[2].(uint)
			if ok2 {
				arraySize = int(arraySizeRaw)
			}

			if ok1 && ok2 && arrayIdx < len(p.HBCReader.Arrays) {
				array, err := types.UnpackSLPArray(p.HBCReader.Arrays[arrayIdx:], arraySize)
				if err != nil {
					panic(fmt.Errorf("failed to unpack SLPArray (arr): %v", err))
				}
				arrayStrings := array.ToStrings(p.HBCReader.Strings)
				comment += fmt.Sprintf("  # Array [%s]", strings.Join(arrayStrings, ", "))
			}
		}
	case "NewObjectWithBuffer", "NewObjectWithBufferLong":
		if len(p.Args) >= 5 {
			var keyIdx, valueIdx, objSize int

			keyIdxRaw, ok1 := p.Args[3].(uint)
			if ok1 {
				keyIdx = int(keyIdxRaw)
			}
			valueIdxRaw, ok2 := p.Args[4].(uint)
			if ok2 {
				valueIdx = int(valueIdxRaw)
			}
			objSizeRaw, ok3 := p.Args[2].(uint)
			if ok3 {
				objSize = int(objSizeRaw)
			}

			if ok1 && ok2 && ok3 && keyIdx < len(p.HBCReader.ObjectKeys) && valueIdx < len(p.HBCReader.ObjectValues) {
				keys, err := types.UnpackSLPArray(p.HBCReader.ObjectKeys[keyIdx:], objSize)
				if err != nil {
					panic(fmt.Errorf("failed to unpack SLPArray (objk): %v", err))
				}
				values, err := types.UnpackSLPArray(p.HBCReader.ObjectValues[valueIdx:], objSize)
				if err != nil {
					panic(fmt.Errorf("failed to unpack SLPArray (objv): %v", err))
				}

				keyStrings := keys.ToStrings(p.HBCReader.Strings)
				valueStrings := values.ToStrings(p.HBCReader.Strings)

				pairs := make([]string, 0, len(keyStrings))
				for i := 0; i < len(keyStrings) && i < len(valueStrings); i++ {
					pairs = append(pairs, fmt.Sprintf("%s: %s", keyStrings[i], valueStrings[i]))
				}
				comment += fmt.Sprintf("  # Object {%s}", strings.Join(pairs, ", "))
			}
		}
	case "CallBuiltin", "CallBuiltinLong", "GetBuiltinClosure":
		if len(p.Args) >= 2 {
			if builtinNumRaw, ok := p.Args[1].(uint); ok {
				builtinNum := int(builtinNumRaw)
				builtinFunctions := GetBuiltinFunctions(p.HBCReader.ParserModule)
				if builtinNum >= 0 && builtinNum < len(builtinFunctions) {
					comment += fmt.Sprintf("  # Builtin function [#%d %s]", builtinNum, builtinFunctions[builtinNum])
				}
			}
		}
	case "SwitchImm":
		if p.SwitchJumpTable != nil {
			jumpAddresses := make([]string, 0, len(p.SwitchJumpTable))
			for _, addr := range p.SwitchJumpTable {
				jumpAddresses = append(jumpAddresses, fmt.Sprintf("%08x", addr))
			}
			comment += fmt.Sprintf("  # Jump table [%s]", strings.Join(jumpAddresses, ", "))
		}
	}

	return fmt.Sprintf("%08x: <%s>: <%s>%s", p.OriginalPos, p.Inst.Name, strings.Join(operands, ", "), comment)
}

func StringKindName(kind StringKind) string {
	switch kind {
	case String:
		return "String"
	case Identifier:
		return "Identifier"
	case Predefined:
		return "Predefined"
	default:
		return "Unknown"
	}
}

func operandMeaningName(meaning types.OperandMeaning) string {
	switch meaning {
	case types.BigIntID:
		return "bigint_id"
	case types.FunctionID:
		return "function_id"
	case types.StringID:
		return "string_id"
	default:
		return "unknown"
	}
}

// ParserModule holds the opcode-to-instruction lookup tables for a specific
// Hermes bytecode version. OpcodeToInstruction maps numeric opcodes to their
// instruction definitions for decoding, NameToInstruction provides reverse
// lookup by mnemonic name, and BuiltinFunctionNames lists the built-in
// function names defined in that bytecode version (used to annotate
// CallBuiltin/GetBuiltinClosure instructions in disassembly output).
// Each supported bytecode version (bcv84, bcv85, ..., bcv96) has a
// pre-generated ParserModule produced by the genopcodes command. GetParser
// selects the closest compatible module for a given bytecode version.
type ParserModule struct {
	OpcodeToInstruction  map[int]*types.Instruction
	NameToInstruction    map[string]*types.Instruction
	BuiltinFunctionNames []string
}

// GetParser returns the ParserModule for the given Hermes bytecode version
// (bcv). It selects the highest available parser version that does not exceed
// bcv, so newer files automatically fall back to the closest older opcode set
// (e.g., bcv 92 uses the bcv90 parser). The supported parser versions are 84,
// 85, 89, 90, 94, and 96. Returns nil if bcv is below all available parsers
// (bcv < 84). Logs a warning for ancient formats (bcv < 72) and future formats
// (bcv > 98, i.e., above LatestBytecodeVersion).
func GetParser(bcv int) *ParserModule {
	parserModuleTable := map[int]*ParserModule{
		// For each bcv there should be a definition generated in `hbc/types/opcodes` from
		// the `utils/bcdefparser.go` script which generates the opcodes
		84: {bcv84.OpcodeToInstruction, bcv84.NameToInstruction, bcv84.BuiltinFunctionNames},
		85: {bcv85.OpcodeToInstruction, bcv85.NameToInstruction, bcv85.BuiltinFunctionNames},
		89: {bcv89.OpcodeToInstruction, bcv89.NameToInstruction, bcv89.BuiltinFunctionNames},
		90: {bcv90.OpcodeToInstruction, bcv90.NameToInstruction, bcv90.BuiltinFunctionNames},
		94: {bcv94.OpcodeToInstruction, bcv94.NameToInstruction, bcv94.BuiltinFunctionNames},
		96: {bcv96.OpcodeToInstruction, bcv96.NameToInstruction, bcv96.BuiltinFunctionNames},
		98: {bcv98.OpcodeToInstruction, bcv98.NameToInstruction, bcv98.BuiltinFunctionNames},
	}

	if bcv < 72 {
		log.Printf("Warning: This file uses an ancient Hermes bytecode format, which is not supported [%d].", bcv)
	} else if bcv > 98 {
		log.Printf("Warning: This file uses a future Hermes bytecode format, which is not yet supported [%d].", bcv)
	}

	var parser *ParserModule
	var maxVersion int
	for version, module := range parserModuleTable {
		if version <= bcv && version > maxVersion {
			maxVersion = version
			parser = module
		}
	}

	return parser
}

func GetBuiltinFunctions(module *ParserModule) []string {
	if module == nil {
		return []string{}
	}

	return module.BuiltinFunctionNames
}

// ParseHBCBytecode disassembles the bytecode of a single Hermes function into
// a sequence of decoded instructions. It accepts either a *SmallFunctionHeader
// or *LargeFunctionHeader to locate the function's bytecode within the HBC
// file, reads the raw bytes from hbcReader.FileBuffer, then iterates through
// the bytecode stream decoding each opcode and its typed operands (int8 through
// uint32 and double) using the version-specific ParserModule. For SwitchImm
// instructions, it additionally reads the 4-byte-aligned jump table that
// follows the instruction and populates SwitchJumpTable with resolved target
// offsets. Returns an error if the function header type is unrecognized, the
// bytecode seek or read fails, an opcode has no definition in the current
// parser, or an operand has an unsupported type kind.
func ParseHBCBytecode(functionHeader any, hbcReader *HBCReader) ([]*ParsedInstruction, error) {
	// extract function metadata based on {Small,Large}FunctionHeader
	var offset, bytecodeSizeInBytes int

	switch header := functionHeader.(type) {
	case *SmallFunctionHeader:
		offset = int(header.Offset)
		bytecodeSizeInBytes = int(header.BytecodeSizeInBytes)
	case *LargeFunctionHeader:
		offset = int(header.Offset)
		bytecodeSizeInBytes = int(header.BytecodeSizeInBytes)
	default:
		return nil, fmt.Errorf("unsupported function header type (this should not happen)")
	}

	// seek to function bytecode
	_, err := hbcReader.FileBuffer.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to function bytecode: %w", err)
	}

	// read function bytecode
	bytecode := make([]byte, bytecodeSizeInBytes)
	_, err = io.ReadFull(hbcReader.FileBuffer, bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to read function bytecode: %w", err)
	}

	buf := bytes.NewReader(bytecode)
	var instructions []*ParsedInstruction
	for {
		originalPos := buf.Size() - int64(buf.Len())

		// read opcode
		var opcode byte
		err := binary.Read(buf, binary.LittleEndian, &opcode)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read opcode: %w", err)
		}

		// find instruction
		inst, ok := hbcReader.ParserModule.OpcodeToInstruction[int(opcode)]
		if !ok {
			return nil, fmt.Errorf("opcode 0x%02x is not implemented", opcode)
		}

		// read operands
		args := make([]any, len(inst.Operands))
		for i, operand := range inst.Operands {
			switch operand.Type.Kind {
			case "int8_t":
				var val int8
				binary.Read(buf, binary.LittleEndian, &val)
				args[i] = int(val)
			case "uint8_t":
				var val uint8
				binary.Read(buf, binary.LittleEndian, &val)
				args[i] = uint(val)
			case "int16_t":
				var val int16
				binary.Read(buf, binary.LittleEndian, &val)
				args[i] = int(val)
			case "uint16_t":
				var val uint16
				binary.Read(buf, binary.LittleEndian, &val)
				args[i] = uint(val)
			case "int32_t":
				var val int32
				binary.Read(buf, binary.LittleEndian, &val)
				args[i] = int(val)
			case "uint32_t":
				var val uint32
				binary.Read(buf, binary.LittleEndian, &val)
				args[i] = uint(val)
			case "double":
				var val float64
				binary.Read(buf, binary.LittleEndian, &val)
				args[i] = val
			default:
				return nil, fmt.Errorf("unkown operand type %s", operand.Type.Kind)
			}
		}

		result := &ParsedInstruction{
			Inst:        inst,
			Args:        args,
			OriginalPos: int(originalPos),
			NextPos:     int(originalPos) + inst.BinarySize,
			HBCReader:   hbcReader,
		}

		// special treatment for 'SwitchImm' because of jump table
		// and because it's so special✨
		if inst.Name == "SwitchImm" && len(args) >= 5 {
			result.SwitchJumpTable = make([]int, 0)

			// seek to jump table
			var argOffset int
			switch v := args[1].(type) {
			case int:
				argOffset = v
			case uint:
				argOffset = int(v)
			default:
				return nil, fmt.Errorf("unexpected type for jump table offset: %T", args[1])
			}
			jumpTableOffset := offset + int(originalPos) + argOffset
			_, err := hbcReader.FileBuffer.Seek(int64(jumpTableOffset), io.SeekStart)
			if err != nil {
				return nil, fmt.Errorf("failed to seek to jump table: %w", err)
			}

			hbcReader.alignOverPadding(4)

			// Handle both int and uint for min value
			var min int
			switch v := args[3].(type) {
			case int:
				min = v
			case uint:
				min = int(v)
			default:
				return nil, fmt.Errorf("unexpected type for min value: %T", args[3])
			}

			// Handle both int and uint for max value
			var max int
			switch v := args[4].(type) {
			case int:
				max = v
			case uint:
				max = int(v)
			default:
				return nil, fmt.Errorf("unexpected type for max value: %T", args[4])
			}
			for i := min; i <= max; i++ {
				var offset uint32
				err := binary.Read(hbcReader.FileBuffer, binary.LittleEndian, &offset)
				if err != nil {
					return nil, fmt.Errorf("failed to read jump table entry: %w", err)
				}
				result.SwitchJumpTable = append(result.SwitchJumpTable, int(originalPos)+int(offset))
			}
		}

		instructions = append(instructions, result)
	}

	return instructions, nil
}
