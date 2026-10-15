package hbc

import (
	"os"
	"testing"
)

// testdata/bcv98_sample.hbc: testdata/bcv98_sample.js compiled with hermesc from hermes-compiler
// 250829098.0.10 (RN 0.85), `hermesc -O -g -emit-binary`. Expected values from `hermesc -dump-bytecode`.
func TestReadHBC98(t *testing.T) {
	f, err := os.Open("testdata/bcv98_sample.hbc")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	h := NewHBCReader()
	if err := h.ReadWholeFile(f); err != nil {
		t.Fatalf("ReadWholeFile: %v", err)
	}
	if h.Header.Version != 98 || h.Header.FunctionCount != 11 {
		t.Fatalf("version %d funcs %d", h.Header.Version, h.Header.FunctionCount)
	}
	want := map[int][]string{ // function index -> first opcodes
		0: {"CreateFunctionEnvironment", "DeclareGlobalVar"},
		1: {"LoadParam", "LoadParam", "Add", "Ret"},
	}
	total := 0
	for i, fh := range h.FunctionHeaders {
		ins, err := ParseHBCBytecode(fh, h)
		if err != nil {
			t.Fatalf("func %d: %v", i, err)
		}
		total += len(ins)
		for j, name := range want[i] {
			if ins[j].Inst.Name != name {
				t.Errorf("func %d instr %d = %s, want %s", i, j, ins[j].Inst.Name, name)
			}
		}
	}
	if total != 307 {
		t.Errorf("total instructions %d, want 307", total)
	}
}

func TestDecodeSmallHeaderV98(t *testing.T) {
	w1 := uint32(0x123) | 3<<25 | 1<<30
	w2 := uint32(0x2A) | 7<<14 | 1<<22 | 2<<27
	b := []byte{byte(w1), byte(w1 >> 8), byte(w1 >> 16), byte(w1 >> 24), byte(w2), byte(w2 >> 8), byte(w2 >> 16), byte(w2 >> 24), 9, 4, 5 | 1<<6, 1<<2 | 1<<5}
	sh := decodeSmallHeaderV98(b)
	if sh.Offset != 0x123 || sh.ParamCount != 3 || sh.BytecodeSizeInBytes != 0x2A || sh.FunctionName != 7 || sh.FrameSize != 9 ||
		sh.HighestWriteCacheIndex != 5 || !sh.StrictMode || !sh.Overflowed || sh.HasDebugInfo {
		t.Fatalf("bad decode: %+v", sh)
	}
}
