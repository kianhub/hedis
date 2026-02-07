package hbc

import (
	"fmt"
	"hermes-decompiler/pkg/hbc/types"
	"strings"
)

// CreateFunctionObjects converts all parsed function headers in the HBCReader
// into normalized FunctionObject representations suitable for fingerprinting.
// For each function header (SmallFunctionHeader or LargeFunctionHeader), it
// resolves the function name from the string table (falling back to a synthetic
// "#func_<index>_<bytecodeSize>_<paramCount>" identifier for anonymous
// functions), disassembles the bytecode via ParseHBCBytecode, and resolves
// semantic operand meanings—mapping StringID, FunctionID, and BigIntID
// references to their concrete values. It also unpacks inline array and object
// literals (NewArrayWithBuffer, NewObjectWithBuffer), annotates CallBuiltin
// targets, and attaches SwitchImm jump table metadata. Returns nil and an
// error if bytecode parsing fails for any function.
func CreateFunctionObjects(hbcReader *HBCReader) ([]*types.FunctionObject, error) {
	var functionObjects []*types.FunctionObject

	for i, function := range hbcReader.FunctionHeaders {
		functionObject, err := createFunctionObjectFromFunctionHeader(hbcReader, i, function)
		if err != nil {
			return nil, fmt.Errorf("at function object creation, there was an issue parsing HBC bytecode for func: %v", err)
		}
		functionObjects = append(functionObjects, functionObject)
	}

	return functionObjects, nil
}

func createFunctionObjectFromFunctionHeader(hbcReader *HBCReader, index int, function any) (*types.FunctionObject, error) {
	var instructions []*ParsedInstruction
	var err error

	var functionObject = &types.FunctionObject{
		Metadata: &types.FunctionObjectMetadata{
			Name:         "",
			BytecodeSize: 0,
			ParamCount:   0,
			Index:        index,
		},
		Instructions: make([]*types.FunctionObjectInstruction, 0),
	}

	if smallFunc, ok := function.(*SmallFunctionHeader); ok {
		funcName := retrieveFunctionName(smallFunc, hbcReader, index)
		functionObject.Metadata.Name = funcName
		functionObject.Metadata.BytecodeSize = int(smallFunc.BytecodeSizeInBytes)
		functionObject.Metadata.ParamCount = int(smallFunc.ParamCount)
		functionObject.Metadata.Index = index
	} else if largeFunc, ok := function.(*LargeFunctionHeader); ok {
		funcName := retrieveFunctionName(largeFunc, hbcReader, index)
		functionObject.Metadata.Name = funcName
		functionObject.Metadata.BytecodeSize = int(largeFunc.BytecodeSizeInBytes)
		functionObject.Metadata.ParamCount = int(largeFunc.ParamCount)
		functionObject.Metadata.Index = index
	}

	if instructions, err = ParseHBCBytecode(function, hbcReader); err != nil {
		return nil, fmt.Errorf("at function object creation, there was an issue parsing HBC bytecode for func: %v", err)
	}

	for _, inst := range instructions {
		foi := &types.FunctionObjectInstruction{
			Name:             inst.Inst.Name,
			Operands:         make([]string, 0),
			ResolvedRichData: make([]types.FunctionResolvedRichData, 0),
		}

		for i, operand := range inst.Inst.Operands {
			var name string
			var richData types.FunctionResolvedRichData

			if operand.Meaning != nil {
				var idx int

				idxRaw, ok := inst.Args[i].(uint)
				if ok {
					idx = int(idxRaw)
				}

				switch *operand.Meaning {
				// TODO: normalize here
				case types.BigIntID:
					name = "BIGINT"
					if ok && idx >= 0 && idx < len(hbcReader.BigIntValues) {
						richData.Type = "BIGINT"
						richData.Value = fmt.Sprintf("%d", hbcReader.BigIntValues[idx])
						richData.IsIdentifier = false
					}
				case types.FunctionID:
					name = "FUNCTION"
					if ok && idx >= 0 && idx < len(hbcReader.FunctionHeaders) {
						funcHeader := hbcReader.FunctionHeaders[idx]
						funcName := retrieveFunctionName(funcHeader, hbcReader, idx)
						richData.Type = "FUNCTION"
						richData.Value = funcName
						richData.IsIdentifier = false
					}
				case types.StringID:
					name = "STRING"
					if ok && idx >= 0 && idx < len(hbcReader.Strings) {
						stringVal := hbcReader.Strings[idx]
						richData.Type = "STRING"
						richData.Value = stringVal
						if idx < len(hbcReader.StringKinds) {
							if StringKindName(hbcReader.StringKinds[idx]) == "Identifier" {
								richData.IsIdentifier = true
							}
						}
					}
				default:
					name = "UNKNOWN"
				}
			} else {
				name = operand.Type.Name
			}
			foi.Operands = append(foi.Operands, name)
			if richData.Type != "" {
				foi.ResolvedRichData = append(foi.ResolvedRichData, richData)
			}
		}

		switch inst.Inst.Name {
		case "NewArrayWithBuffer", "NewArrayWithBufferLong":
			if len(inst.Args) >= 4 {
				var arrayIdx, arraySize int
				var ok1, ok2 bool
				if arrayIdxRaw, ok1 := inst.Args[3].(uint); ok1 {
					arrayIdx = int(arrayIdxRaw)
				}
				if arraySizeRaw, ok2 := inst.Args[2].(uint); ok2 {
					arraySize = int(arraySizeRaw)
				}

				if ok1 && ok2 && arrayIdx < len(hbcReader.Arrays) {
					array, err := types.UnpackSLPArray(hbcReader.Arrays[arrayIdx:], arraySize)
					if err != nil {
						panic(fmt.Errorf("failed to unpack SLPArray (arr): %v", err))
					}
					arrayStrings := array.ToStrings(hbcReader.Strings)
					foi.ResolvedRichData = append(foi.ResolvedRichData, types.FunctionResolvedRichData{
						Type:         "ARRAY",
						Value:        strings.Join(arrayStrings, ", "),
						IsIdentifier: false,
					})
				}
			}
		case "NewObjectWithBuffer", "NewObjectWithBufferLong":
			if len(inst.Args) >= 5 {
				var keyIdx, valueIdx, objSize int

				keyIdxRaw, ok1 := inst.Args[3].(uint)
				if ok1 {
					keyIdx = int(keyIdxRaw)
				}
				valueIdxRaw, ok2 := inst.Args[4].(uint)
				if ok2 {
					valueIdx = int(valueIdxRaw)
				}
				objSizeRaw, ok3 := inst.Args[2].(uint)
				if ok3 {
					objSize = int(objSizeRaw)
				}

				if ok1 && ok2 && ok3 && keyIdx < len(hbcReader.ObjectKeys) && valueIdx < len(hbcReader.ObjectValues) {
					keys, err := types.UnpackSLPArray(hbcReader.ObjectKeys[keyIdx:], objSize)
					if err != nil {
						panic(fmt.Errorf("failed to unpack SLPArray (objk): %v", err))
					}
					values, err := types.UnpackSLPArray(hbcReader.ObjectValues[valueIdx:], objSize)
					if err != nil {
						panic(fmt.Errorf("failed to unpack SLPArray (objv): %v", err))
					}

					keyStrings := keys.ToStrings(hbcReader.Strings)
					valueStrings := values.ToStrings(hbcReader.Strings)
					pairs := make([]string, 0, len(keyStrings))
					for i := 0; i < len(keyStrings) && i < len(valueStrings); i++ {
						pairs = append(pairs, fmt.Sprintf("%s: %s", keyStrings[i], valueStrings[i]))
					}

					foi.ResolvedRichData = append(foi.ResolvedRichData, types.FunctionResolvedRichData{
						Type:  "OBJECT",
						Value: fmt.Sprintf("{%s}", strings.Join(pairs, ", ")),
					})
				}
			}
		case "CallBuiltin", "CallBuiltinLong", "GetBuiltinClosure":
			if len(inst.Args) >= 2 {
				if builtinNumRaw, ok := inst.Args[1].(uint); ok {
					builtinNum := int(builtinNumRaw)
					builtinFunctions := GetBuiltinFunctions(hbcReader.ParserModule)
					if builtinNum >= 0 && builtinNum < len(builtinFunctions) {
						foi.ResolvedRichData = append(foi.ResolvedRichData, types.FunctionResolvedRichData{
							Type:  "BUILTIN",
							Value: builtinFunctions[builtinNum],
						})
					}
				}
			}
		case "SwitchImm":
			if inst.SwitchJumpTable != nil {
				foi.ResolvedRichData = append(foi.ResolvedRichData, types.FunctionResolvedRichData{
					Type: "SWITCH",
				})
			}
		}
		if len(foi.ResolvedRichData) == 0 {
			foi.ResolvedRichData = nil
		}
		functionObject.Instructions = append(functionObject.Instructions, foi)
	}

	return functionObject, nil
}

// retrieveFunctionName retrieves the function name from the header
// if the function name is empty, it creates an identifier name
// with the structure: "#func_<index>_<bytecode_size>_<param_count>"
func retrieveFunctionName(header any, hbcReader *HBCReader, idx int) string {
	var funcName string

	switch header := header.(type) {
	case *SmallFunctionHeader:
		if int(header.FunctionName) < len(hbcReader.Strings) {
			funcName = hbcReader.Strings[header.FunctionName]
		}
		if funcName == "" {
			funcName = fmt.Sprintf("#func_%d_%d_%d", idx, int(header.BytecodeSizeInBytes), int(header.ParamCount))
		}
	case *LargeFunctionHeader:
		if int(header.FunctionName) < len(hbcReader.Strings) {
			funcName = hbcReader.Strings[header.FunctionName]
		}
		if funcName == "" {
			funcName = fmt.Sprintf("#func_%d_%d_%d", idx, int(header.BytecodeSizeInBytes), int(header.ParamCount))
		}
	}

	return funcName
}
