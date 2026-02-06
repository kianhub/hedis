package analyzer

import (
	"fmt"
	"hermes-decompiler/pkg/hbc/types"
	"io"
	"os"
	"testing"
)

const testFilePath = "testdata/Rainbow-1_9_70_app_test_bundle.hbc"

var functionObjects []*types.FunctionObject

/* func init() {
	utils.L.Println("Reading HBC file")
	content, err := openFileFromPath(testFilePath)
	if err != nil {
		utils.F.Printf("failed to open HBC file: %v\n", err)
		return
	}
	utils.L.Println("Disassembling HBC file")

	hbcReader := hbc.NewHBCReader()
	err = hbcReader.ReadWholeFile(bytes.NewReader(content))
	if err != nil {
		utils.F.Printf("failed to parse HBC file: %v\n", err)
		return
	}
	utils.L.Printf("HBC file parsed successfully. Hermes version: %d, Function count: %d\n",
		hbcReader.Header.Version, hbcReader.Header.FunctionCount)

	utils.L.Printf("Creating function objects")

	functionObjects, err = hbc.CreateFunctionObjects(hbcReader)
	if err != nil {
		utils.F.Printf("failed to create function objects: %v\n", err)
		return
	}
	utils.L.Printf("Created %d function objects\n", len(functionObjects))
} */

func BenchmarkComputeFunctionSignature(b *testing.B) {
	if len(functionObjects) == 0 {
		b.Skip("test fixture not available")
	}
	minHasher := NewMinHasher(MINHASH_SIZE)
	for i := 0; i < b.N; i++ {
		minHasher.ComputeFunctionSignature(functionObjects[i%len(functionObjects)])
	}
}

func openFileFromPath(path string) ([]byte, error) {
	rc, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", path, err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}
