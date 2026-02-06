package cmd

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"hermes-decompiler/pkg/hbc"
	"hermes-decompiler/pkg/hbc/types"
	"hermes-decompiler/pkg/utils"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	hbcFilePath    string
	outputFilePath string

	// bool flags
	stringsFlag           bool
	functionsFlag         bool
	objectsFlag           bool
	functionObjectsFlag   bool
	functionObjectsIRFlag bool
	// normalization level flags - these flags will have their effect in 'pkg/hbc/bytecode_parser.go'
	normalizationFlag int

	disassembleHbcFileCmd = &cobra.Command{
		Use:   "disassemble",
		Short: "Reads and Disassembles an HBC (Hermes Bytecode) file to an output file",
		Long:  "Reads and Disassembles an HBC (Hermes Bytecode) file and creates a new output file containing a parsed version of the HBC contents.",
		Run: func(cmd *cobra.Command, args []string) {
			inputFile, err := os.Open(hbcFilePath)
			if err != nil {
				panic(err)
			}
			defer inputFile.Close()

			hbcReader := hbc.NewHBCReader()

			err = hbcReader.ReadWholeFile(inputFile)
			if err != nil {
				utils.F.Printf("error parsing file: %v\n", err)
				os.Exit(1)
			}

			if functionObjectsIRFlag {
				fois, err := hbc.CreateFunctionObjects(hbcReader)
				if err != nil {
					utils.F.Printf("error creating function objects: %v\n", err)
					os.Exit(1)
				}
				for _, foi := range fois {
					logFunctionObjectIR(foi)
				}
			}

			if outputFilePath == "" {
				os.Exit(0)
				return
			}

			utils.L.Println("Writing File Content to " + outputFilePath)
			err = writeFileContent(hbcReader, outputFilePath)
			if err != nil {
				utils.F.Printf("error writing output file: %v\n", err)
				os.Exit(1)
			}

			utils.S.Printf("Disassembled and wrote parsed file: %s\n", outputFilePath)
		},
	}
)

func init() {
	disassembleHbcFileCmd.Flags().StringVarP(&hbcFilePath, "hbc", "i", "", "Path to input HBC (Hermes Bytecode) file")
	disassembleHbcFileCmd.MarkFlagRequired("hbc")
	disassembleHbcFileCmd.Flags().StringVarP(&outputFilePath, "output", "o", "", "Path to output file")

	// bool flags
	disassembleHbcFileCmd.Flags().BoolVarP(&stringsFlag, "strings", "s", false, "Include Strings in disassembled file output (default false)")
	disassembleHbcFileCmd.Flags().BoolVarP(&functionsFlag, "functions", "f", true, "Include Functions in disassembled file output")
	disassembleHbcFileCmd.Flags().BoolVarP(&objectsFlag, "objects", "j", false, "Include Objects in disassembled file output (default false)")
	disassembleHbcFileCmd.Flags().BoolVarP(&functionObjectsFlag, "function-objects", "z", false, "Include Function Objects in disassembled file output (default false)")
	disassembleHbcFileCmd.Flags().BoolVarP(&functionObjectsIRFlag, "function-objects-ir", "r", false, "Include Function Objects IR in disassembled file output (function-objects must be true) (default false)")
	// normalization level flags - these flags will have their effect in 'pkg/hbc/bytecode_parser.go'
	disassembleHbcFileCmd.Flags().IntVarP(&normalizationFlag, "normalization", "n", 0, "Normalize output (Level 0: No Normalization (default), Level 1: IR1, Level 2: IR2)")
}

// writeFileContent writes the HBC reader content to the output file using buffered I/O for performance
func writeFileContent(hbcReader *hbc.HBCReader, outputPath string) error {
	err := os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write each section to the file

	if err := writeHeader(writer, hbcReader); err != nil {
		return err
	}

	if stringsFlag {
		if err := writeStrings(writer, hbcReader); err != nil {
			return err
		}
	}

	if functionsFlag {
		if err := writeFunctions(writer, hbcReader); err != nil {
			return err
		}
	}

	if objectsFlag {
		if err := writeObjects(writer, hbcReader); err != nil {
			return err
		}
	}

	if functionObjectsFlag {
		fois, err := hbc.CreateFunctionObjects(hbcReader)
		if err != nil {
			return err
		}
		for _, foi := range fois {
			if _, err := fmt.Fprintf(writer, "%s\n", foi.String()); err != nil {
				return err
			}
		}
	}

	return nil
}

func logFunctionObjectIR(foi *types.FunctionObject) {
	structuralIR, contentIR1, contentIR2 := foi.ToIR()
	fmt.Println("Structural IR:", structuralIR)
	fmt.Println("Content IR1:", contentIR1)
	fmt.Println("Content IR2:", contentIR2)
	fmt.Println("--------------------------------")
	fmt.Printf("SHA256-StructuralIR: %x\n", sha256.Sum256([]byte(structuralIR)))
	if contentIR1 != "" {
		fmt.Printf("SHA256-ContentIR1: %x\n", sha256.Sum256([]byte(contentIR1)))
	} else {
		fmt.Println("SHA256-ContentIR1: N/A")
	}
	if contentIR2 != "" {
		fmt.Printf("SHA256-ContentIR2: %x\n", sha256.Sum256([]byte(contentIR2)))
	} else {
		fmt.Println("SHA256-ContentIR2: N/A")
	}
	fmt.Println("--------------------------------")
}

// writeHeader writes the header information to the output writer
func writeHeader(w *bufio.Writer, hbcReader *hbc.HBCReader) error {
	_, err := fmt.Fprintf(w, "HBC Hermes Header: \nHermes Magic: 0x%016x\nVersion: %d\nFunction Count: %d\n\n",
		hbcReader.Header.Magic, hbcReader.Header.Version, hbcReader.Header.FunctionCount)
	return err
}

// writeStrings writes the strings table to the output writer
func writeStrings(w *bufio.Writer, hbcReader *hbc.HBCReader) error {
	if _, err := fmt.Fprintln(w, "\nStrings:"); err != nil {
		return err
	}

	for i, str := range hbcReader.Strings {
		if i < len(hbcReader.StringKinds) {
			if _, err := fmt.Fprintf(w, "\t=> %s: %q\n", hbc.StringKindName(hbcReader.StringKinds[i]), str); err != nil {
				return err
			}
		}
	}
	return nil
}

type ExtractedObject struct {
	SourceFunctionId   int
	SourceFunctionName string
	InstName           string
	Content            map[string]any
}

func extractObject(p *hbc.ParsedInstruction) *ExtractedObject {
	var extractedObject = &ExtractedObject{InstName: p.Inst.Name}

	if len(p.Args) >= 5 {
		var keyIdx, valueIdx, objSize int
		if keyIdxRaw, ok1 := p.Args[3].(uint); ok1 {
			keyIdx = int(keyIdxRaw)
		}
		if valueIdxRaw, ok2 := p.Args[4].(uint); ok2 {
			valueIdx = int(valueIdxRaw)
		}
		if objSizeRaw, ok3 := p.Args[2].(uint); ok3 {
			objSize = int(objSizeRaw)
		}

		if keyIdx < len(p.HBCReader.ObjectKeys) && valueIdx < len(p.HBCReader.ObjectValues) {
			keys, err := types.UnpackSLPArray(p.HBCReader.ObjectKeys[keyIdx:], objSize)
			if err != nil {
				panic(fmt.Errorf("object extraction failed to unpack SLPArray (objk): %v", err))
			}
			values, err := types.UnpackSLPArray(p.HBCReader.ObjectValues[valueIdx:], objSize)
			if err != nil {
				panic(fmt.Errorf("object extraction failed to unpack SLPArray (objv): %v", err))
			}
			keyStrings := keys.ToStrings(p.HBCReader.Strings)
			valueStrings := values.ToStrings(p.HBCReader.Strings)

			extractedObject.Content = make(map[string]any)
			for i := 0; i < len(keyStrings) && i < len(valueStrings); i++ {
				extractedObject.Content[keyStrings[i]] = valueStrings[i]
			}
		}

		return extractedObject
	}

	return nil
}

func writeObjects(w *bufio.Writer, hbcReader *hbc.HBCReader) error {
	if _, err := fmt.Fprintln(w, "\nObjects:"); err != nil {
		return err
	}

	extractedObjects := make([]*ExtractedObject, 0)

	for funcIdx, function := range hbcReader.FunctionHeaders {
		funcName := ""

		if smallFunc, ok := function.(*hbc.SmallFunctionHeader); ok {
			if int(smallFunc.FunctionName) < len(hbcReader.Strings) {
				funcName = hbcReader.Strings[smallFunc.FunctionName]
			}

			// Extract object references from small function
			instructions, err := hbc.ParseHBCBytecode(smallFunc, hbcReader)
			if err != nil {
				return err
			}
			for _, parsedInst := range instructions {
				if parsedInst.Inst.Name == "NewObjectWithBuffer" || parsedInst.Inst.Name == "NewObjectWithBufferLong" {
					o := extractObject(parsedInst)
					if o == nil {
						continue
					}
					o.SourceFunctionId = funcIdx
					o.SourceFunctionName = funcName
					extractedObjects = append(extractedObjects, o)
				}
			}
		} else if largeFunc, ok := function.(*hbc.LargeFunctionHeader); ok {
			if int(largeFunc.FunctionName) < len(hbcReader.Strings) {
				funcName = hbcReader.Strings[largeFunc.FunctionName]
			}

			// Extract object references from large function
			instructions, err := hbc.ParseHBCBytecode(largeFunc, hbcReader)
			if err != nil {
				return err
			}
			for _, parsedInst := range instructions {
				if parsedInst.Inst.Name == "NewObjectWithBuffer" || parsedInst.Inst.Name == "NewObjectWithBufferLong" {
					o := extractObject(parsedInst)
					if o == nil {
						continue
					}
					o.SourceFunctionId = funcIdx
					o.SourceFunctionName = funcName
					extractedObjects = append(extractedObjects, o)
				}
			}
		}
	}

	for _, o := range extractedObjects {
		if err := writeObject(w, o); err != nil {
			return err
		}
	}

	return nil
}

func writeObject(w *bufio.Writer, o *ExtractedObject) error {
	if _, err := fmt.Fprintf(w, "\t=> [Object from Function '%s' (id: %d) Inst: %s]\n", o.SourceFunctionName, o.SourceFunctionId, o.InstName); err != nil {
		return err
	}
	// now we need to write o.Content like a json object
	if _, err := fmt.Fprint(w, "\t\t{\n"); err != nil {
		return err
	}
	for k, v := range o.Content {
		if _, err := fmt.Fprintf(w, "\t\t\t'%s': %s,\n", k, v); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "\t\t}\n"); err != nil {
		return err
	}

	return nil
}

// writeFunctions writes all function information to the output writer
func writeFunctions(w *bufio.Writer, hbcReader *hbc.HBCReader) error {
	if _, err := fmt.Fprintln(w, "\nFunctions:"); err != nil {
		return err
	}

	for i, function := range hbcReader.FunctionHeaders {
		if err := writeFunction(w, hbcReader, i, function); err != nil {
			return err
		}
	}
	return nil
}

// writeFunction writes a single function's information to the output writer
func writeFunction(w *bufio.Writer, hbcReader *hbc.HBCReader, index int, function any) error {
	var instructions []*hbc.ParsedInstruction
	var err error

	if smallFunc, ok := function.(*hbc.SmallFunctionHeader); ok {
		funcName := ""
		if int(smallFunc.FunctionName) < len(hbcReader.Strings) {
			funcName = hbcReader.Strings[smallFunc.FunctionName]
		}

		if _, err := fmt.Fprintf(w, "\t=> [Function #%d '%s' of %d bytes]: %d params @ offset 0x%08x\n",
			index, funcName, smallFunc.BytecodeSizeInBytes, smallFunc.ParamCount, smallFunc.Offset); err != nil {
			return err
		}

		if instructions, err = hbc.ParseHBCBytecode(smallFunc, hbcReader); err != nil {
			return fmt.Errorf("there was an issue parsing HBC bytecode for small func: %v", err)
		}

		for _, inst := range instructions {
			if _, err := fmt.Fprintf(w, "\t\t %s \n", inst); err != nil {
				return err
			}
		}
	} else if largeFunc, ok := function.(*hbc.LargeFunctionHeader); ok {
		funcName := ""
		if int(largeFunc.FunctionName) < len(hbcReader.Strings) {
			funcName = hbcReader.Strings[largeFunc.FunctionName]
		}

		if _, err := fmt.Fprintf(w, "\t=> [Function #%d '%s' of %d bytes]: %d params @ offset 0x%08x\n",
			index, funcName, largeFunc.BytecodeSizeInBytes, largeFunc.ParamCount, largeFunc.Offset); err != nil {
			return err
		}

		if instructions, err = hbc.ParseHBCBytecode(largeFunc, hbcReader); err != nil {
			return fmt.Errorf("there was an issue parsing HBC bytecode for large func: %v", err)
		}

		for _, inst := range instructions {
			if _, err := fmt.Fprintf(w, "\t\t %s \n", inst); err != nil {
				return err
			}
		}
	}

	return nil
}
