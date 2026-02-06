package analyzer

import (
	"fmt"
	"hermes-decompiler/pkg/hbc/types"
	"strings"
	"testing"
)

func TestMinHasher(t *testing.T) {
	minHasher := NewMinHasher(128)

	fo1 := &types.FunctionObject{
		Metadata: &types.FunctionObjectMetadata{
			ParamCount: 1,
		},
		Instructions: []*types.FunctionObjectInstruction{
			{
				Name: "TryGetById",
				Operands: []string{
					"Reg8",
					"Reg8",
					"UInt8",
					"STRING",
				},
				ResolvedRichData: []types.FunctionResolvedRichData{
					{
						Type:         "STRING",
						Value:        "__d",
						IsIdentifier: true,
					},
				},
			},
			{
				Name: "LoadConst",
				Operands: []string{
					"Reg8",
					"Reg8",
					"UInt8",
					"STRING",
				},
				ResolvedRichData: []types.FunctionResolvedRichData{
					{
						Type:         "STRING",
						Value:        "__d",
						IsIdentifier: true,
					},
				},
			},
			{
				Name: "GetGlobalObject",
				Operands: []string{
					"Reg8",
					"Reg8",
					"UInt8",
					"STRING",
				},
				ResolvedRichData: []types.FunctionResolvedRichData{
					{
						Type:         "STRING",
						Value:        "__d",
						IsIdentifier: true,
					},
				},
			},
		},
	}

	fo2 := &types.FunctionObject{
		Metadata: &types.FunctionObjectMetadata{
			ParamCount: 1,
		},
		Instructions: []*types.FunctionObjectInstruction{
			{
				Name: "TryGetById",
				Operands: []string{
					"Reg8",
					"Reg8",
					"UInt8",
					"STRING",
				},
				ResolvedRichData: []types.FunctionResolvedRichData{
					{
						Type:         "STRING",
						Value:        "__request",
						IsIdentifier: true,
					},
				},
			},
			{
				Name: "LoadConst",
				Operands: []string{
					"Reg8",
					"Reg8",
					"UInt8",
					"STRING",
				},
				ResolvedRichData: []types.FunctionResolvedRichData{
					{
						Type:         "STRING",
						Value:        "__d",
						IsIdentifier: true,
					},
				},
			},
		},
	}

	fo1Signature := minHasher.ComputeFunctionSignature(fo1)
	fo2Signature := minHasher.ComputeFunctionSignature(fo2)

	similarity := minHasher.JaccardSimilarity(fo1Signature.StructuralHash, fo2Signature.StructuralHash)
	fmt.Println(similarity)

	fmt.Println("fo1 structural bands:")
	fmt.Println(fo1Signature.LSHBands)
	fmt.Println("fo2 structural bands:")
	fmt.Println(fo2Signature.LSHBands)
}

func TestMinHasher2(t *testing.T) {
	minHasher := NewMinHasher(128)

	string1 := "pc=3|GetEnvironment|LoadFromEnvironment|GetByIdShort|LoadParam|Call2|GetByIdShort|LoadParam|Call2|GetByIdShort|Call3|JmpTrueLong|LoadConstZero|JLess|GetGlobalObject|TryGetById|GetByIdShort|Call2|JStrictNotEqual|LoadConstDouble|JGreater|TryGetById|GetByIdShort|Call2|LoadConstUInt8|JGreater|JGreaterEqual|Negate|RShift|Jmp|LShift|Ret|LoadFromEnvironment|GetByIdShort|Ret|LoadFromEnvironment|GetByIdShort|Ret|LoadFromEnvironment|GetByIdShort|Ret|LoadFromEnvironment|GetByIdShort|Ret|LoadFromEnvironment|GetByIdShort|Ret|"

	string2 := "pc=3|GetEnvironment|LoadFromEnvironment|GetByIdShort|LoadParam|Call2|GetByIdShort|LoadParam|Call2|GetByIdShort|Call3|JmpTrueLong|LoadConstZero|JLess|GetGlobalObject|TryGetById|GetByIdShort|Call2|JStrictNotEqual|LoadConstDouble|JGreater|TryGetById|GetByIdShort|Call2|LoadConstUInt8|JGreater|JGreaterEqual|Negate|RShift|Jmp|LShift|Ret|LoadFromEnvironment|GetByIdShort|Ret|LoadFromEnvironment|GetByIdShort|Ret|LoadFromEnvironment|GetByIdShort|Ret|LoadFromEnvironment|GetByIdShort|Ret|LoadFromEnvironment|GetByIdShort|GetByIdShort|Ret|"

	/* string2 := "pc=3|LoadParam|LoadParam|GetEnvironment|LoadFromEnvironment|LoadConstUndefined|Call2|JmpTrue|GetEnvironment|LoadFromEnvironment|GetByIdShort|Ret|GetByIdShort|Call1|GetByIdShort|LoadConstUInt8|JStrictNotEqual|GetByIdShort|LoadConstZero|LoadConstUInt8|Call3|LoadConstString|JStrictEqualLong|GetGlobalObject|TryGetById|LoadConstUInt8|Call3|GetByIdShort|LoadConstUInt8|Call2|JStrictEqual|TryGetById|Call2|JmpTrue|LoadConstZero|JLess|TryGetById|GetByIdShort|Call2|GetByIdShort|JGreaterEqual|GetEnvironment|LoadFromEnvironment|GetByIdShort|Jmp|TryGetById|GetByIdShort|Sub|LoadConstString|Call3|Add|Ret|GetEnvironment|LoadFromEnvironment|GetByIdShort|Ret|GetEnvironment|LoadFromEnvironment|GetByIdShort|Ret|Ret|GetGlobalObject|TryGetById|GetByIdShort|Call2|LoadConstUInt8|Call3|LoadConstDouble|Add|GetByIdShort|LoadConstUInt8|Call2|Ret|" */

	tokens1 := tokenize(string1)
	tokens2 := tokenize(string2)

	signature1 := minHasher.ComputeSignature(tokens1)
	signature2 := minHasher.ComputeSignature(tokens2)

	similarity := minHasher.JaccardSimilarity(signature1, signature2)
	fmt.Println(similarity)
}

func tokenize(input string) []string {
	parts := strings.Split(input, "|")
	if len(parts) == 0 {
		return []string{}
	}

	instructions := parts[1:] // Skip param count
	tokens := make([]string, 0)

	// Individual instructions
	for _, inst := range instructions {
		if inst != "" {
			tokens = append(tokens, inst)
		}
	}

	// Instruction bigrams
	for i := 0; i < len(instructions)-1; i++ {
		if instructions[i] != "" && instructions[i+1] != "" {
			tokens = append(tokens, instructions[i]+"→"+instructions[i+1])
		}
	}

	return tokens
}
