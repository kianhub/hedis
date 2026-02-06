package analyzer

import (
	"crypto/sha256"
	"fmt"
	"hermes-decompiler/pkg/hbc/types"
)

func (minHasher *MinHasher) ComputeFunctionSignature(fo *types.FunctionObject) *FunctionSignature {

	// compute exact match hashes from IRs
	structuralIR, contentIR1, contentIR2 := fo.ToIR()
	var structuralHash, contentIR1Hash, contentIR2Hash, combinedHash string

	if len(structuralIR) >= 10 {
		structuralHash = computeSHA256Hash(structuralIR)
	}
	if contentIR1 != "" && len(contentIR1) >= 10 {
		contentIR1Hash = computeSHA256Hash(contentIR1)
	}
	if contentIR2 != "" && len(contentIR2) >= 10 {
		contentIR2Hash = computeSHA256Hash(contentIR2)
	}
	if contentIR1 != "" && contentIR2 != "" && len(contentIR1) >= 10 && len(contentIR2) >= 10 {
		combinedHash = computeSHA256Hash(structuralIR + contentIR1 + contentIR2)
	}

	// tokenize structural IR
	structuralTokens := fo.TokenizeStructuralIR()

	// tokenize content IRs
	nonIdentifierTokens, identifierAndObjectTokens := fo.TokenizeContentIRs()

	// compute signatures
	structuralSig := minHasher.ComputeSignature(structuralTokens)
	contentIR1Sig := minHasher.ComputeSignature(nonIdentifierTokens)
	contentIR2Sig := minHasher.ComputeSignature(identifierAndObjectTokens)

	// combined signature - all tokens concatenated with prefixes
	combinedTokens := make([]string, 0)
	for _, token := range structuralTokens {
		combinedTokens = append(combinedTokens, "s:"+token)
	}
	for _, token := range nonIdentifierTokens {
		combinedTokens = append(combinedTokens, "c1:"+token)
	}
	for _, token := range identifierAndObjectTokens {
		combinedTokens = append(combinedTokens, "c2:"+token)
	}

	combinedSig := minHasher.ComputeSignature(combinedTokens)

	// compute LSH bands for combined signature
	lshBands := make([]string, LSH_BANDS)
	for i := range LSH_BANDS {
		lshBands[i] = minHasher.ComputeLSHBand(combinedSig, i)
	}

	return &FunctionSignature{
		StructuralHashSHA256: structuralHash,
		ContentIR1HashSHA256: contentIR1Hash,
		ContentIR2HashSHA256: contentIR2Hash,
		CombinedHashSHA256:   combinedHash,
		StructuralHash:       structuralSig,
		ContentIR1Hash:       contentIR1Sig,
		ContentIR2Hash:       contentIR2Sig,
		LSHBands:             lshBands,
	}

}

func computeSHA256Hash(input string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(input)))
}
