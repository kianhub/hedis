package types

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// FunctionObject is the normalized intermediate representation of a single
// disassembled Hermes function, produced by CreateFunctionObjects. It pairs
// function-level metadata (name, parameter count, bytecode size, index) with
// a sequence of resolved instructions whose operands have been mapped to
// concrete string values, object references, and builtin annotations.
//
// FunctionObject supports three complementary fingerprint generation methods
// via ToIR: a structural IR (opcode bigrams), a content IR1 (sorted
// non-identifier string literals), and a content IR2 (sorted identifiers and
// object references). These IRs are hashed (SHA256 and MinHash) for
// similarity matching against the fingerprint database.
type FunctionObject struct {
	Metadata     *FunctionObjectMetadata
	Instructions []*FunctionObjectInstruction
}

type FunctionObjectInstruction struct {
	Name             string
	Operands         []string
	ResolvedRichData []FunctionResolvedRichData
}

type FunctionObjectMetadata struct {
	Name         string // original function name, usually not very useful
	BytecodeSize int
	ParamCount   int // number of parameters of the function, but ! this is the number of parameters for the hermes function (I think)
	Index        int // relative index of the function in the bytecode
}

type FunctionResolvedRichData struct {
	Type         string
	Value        string
	IsIdentifier bool
}

func (foi *FunctionObjectInstruction) String() string {
	jsonBytes, err := json.Marshal(foi)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

func (fo *FunctionObject) String() string {
	jsonBytes, err := json.MarshalIndent(fo, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

// TokenizeStructuralIR returns a slice of string tokens for the structural IR as bigrams (e.g. "push→push")
func (fo *FunctionObject) TokenizeStructuralIR() []string {
	tokens := make([]string, 0)

	// creating bigrams for structural IR
	for i := 0; i < len(fo.Instructions)-1; i++ {
		if fo.Instructions[i].Name != "" && fo.Instructions[i+1].Name != "" {
			tokens = append(tokens, fmt.Sprintf("%s→%s", fo.Instructions[i].Name, fo.Instructions[i+1].Name)) // e.g. "push→push"
		}
	}

	return tokens
}

// TokenizeContentIRs returns two slices of string tokens: non-identifier tokens (CIR1) and identifier-object tokens (CIR2)
// it also adds trigrams/shingles for partial matching for every token longer than 3 characters
func (fo *FunctionObject) TokenizeContentIRs() (cir1, cir2 []string) {
	nonIdentifierTokens := make([]string, 0)
	identifierAndObjectTokens := make([]string, 0)

	for _, inst := range fo.Instructions {
		resolvedRichData := inst.ResolvedRichData
		for _, richData := range resolvedRichData {
			richDataValue := strings.ToLower(richData.Value)
			richDataValue = strings.ReplaceAll(richDataValue, "|", "")
			if !richData.IsIdentifier && richData.Type == "STRING" {
				nonIdentifierTokens = append(nonIdentifierTokens, richDataValue)

				// add trigrams/shingles for partial matching
				if len(richDataValue) >= 3 {
					for i := 0; i < len(richDataValue)-3; i++ {
						nonIdentifierTokens = append(nonIdentifierTokens, richDataValue[i:i+3])
					}
				}
			}

			if richData.IsIdentifier && richData.Type == "STRING" {
				identifierAndObjectTokens = append(identifierAndObjectTokens, richDataValue)

				if len(richDataValue) >= 3 {
					for i := 0; i < len(richDataValue)-3; i++ {
						identifierAndObjectTokens = append(identifierAndObjectTokens, richDataValue[i:i+3])
					}
				}
			}
			if !richData.IsIdentifier && richData.Type == "OBJECT" {
				identifierAndObjectTokens = append(identifierAndObjectTokens, richDataValue)

				if len(richDataValue) >= 3 {
					for i := 0; i < len(richDataValue)-3; i++ {
						identifierAndObjectTokens = append(identifierAndObjectTokens, richDataValue[i:i+3])
					}
				}
			}
		}
	}

	return nonIdentifierTokens, identifierAndObjectTokens
}

func (fo *FunctionObject) ToIR() (structuralIR, contentIR1, contentIR2 string) {

	paramCountString := fmt.Sprintf("pc=%d|", fo.Metadata.ParamCount)

	for _, inst := range fo.Instructions {
		// structuralIR += fmt.Sprintf("%s_%s|", inst.Name, strings.Join(inst.Operands, "_"))
		structuralIR += fmt.Sprintf("%s|", inst.Name) // removed operands for now for performance and disk space savings
	}

	nonIndentifierStrings := make([]string, 0)
	identifierAndObjectStrings := make([]string, 0)
	for _, inst := range fo.Instructions {
		resolvedRichData := inst.ResolvedRichData
		for _, richData := range resolvedRichData {
			richDataValue := strings.ToLower(richData.Value)
			richDataValue = strings.ReplaceAll(richDataValue, "|", "")
			if !richData.IsIdentifier && richData.Type == "STRING" {
				nonIndentifierStrings = append(nonIndentifierStrings, richDataValue)
			}
			if richData.IsIdentifier && richData.Type == "STRING" {
				identifierAndObjectStrings = append(identifierAndObjectStrings, richDataValue)
			}
			if !richData.IsIdentifier && richData.Type == "OBJECT" {
				identifierAndObjectStrings = append(identifierAndObjectStrings, richDataValue)
			}
		}
	}

	slices.Sort(nonIndentifierStrings)
	slices.Sort(identifierAndObjectStrings)
	if len(nonIndentifierStrings) > 0 {
		contentIR1 = strings.Join(nonIndentifierStrings, "|")
	}
	if len(identifierAndObjectStrings) > 0 {
		contentIR2 = strings.Join(identifierAndObjectStrings, "|")
	}

	return paramCountString + structuralIR, contentIR1, contentIR2
}
