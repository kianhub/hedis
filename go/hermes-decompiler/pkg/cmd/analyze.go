package cmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"hermes-decompiler/pkg/database"
	"hermes-decompiler/pkg/database/models"
	"hermes-decompiler/pkg/hbc"
	"hermes-decompiler/pkg/utils"
	"io"
	"os"
	"slices"
	"sync"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HashMatch represents a hash match with information about its source
type HashMatch struct {
	Hash                  models.Hash
	PackageHashID         *primitive.ObjectID `json:"package_hash_id,omitempty"`
	ReactNativeVersion    string              `json:"react_native_version,omitempty"`
	BaselineID            *primitive.ObjectID `json:"baseline_id,omitempty"`
	HermesBytecodeVersion string              `json:"hermes_bytecode_version,omitempty"`
	Source                string              `json:"source"` // "package" or "baseline"
}

// SimilarityMatch represents a fuzzy match with confidence scoring
type SimilarityMatch struct {
	HashMatch
	Confidence    float64 `json:"confidence"`     // 0.0 to 1.0
	MatchType     string  `json:"match_type"`     // "exact" or "fuzzy"
	MatchedString string  `json:"matched_string"` // The raw IR string that matched
	InputString   string  `json:"input_string"`   // The input IR string being searched
}

type Content struct {
	StructuralHashes []string
	ContentIR1Hashes []string
	ContentIR2Hashes []string
	StructuralIRs    []string
	ContentIR1s      []string
	ContentIR2s      []string
}

type FuzzyResults struct {
	Structural *FuzzyContent
	ContentIR1 *FuzzyContent
	ContentIR2 *FuzzyContent
}

type FuzzyContent struct {
	Matched    int
	Eligible   int
	MatchedIRs []string
}

var (
	ipaFilePath         string
	fuzzyMatching       bool
	confidenceThreshold float64

	// Fuzzy matching performance controls
	maxFuzzyDocs    int     = 5000 // Maximum documents to check for fuzzy matching
	lengthTolerance float64 = 0.2  // ±20% length tolerance for IR string filtering

	// specifc analysis flags
	specificAnalysisFlag           bool
	specificPackageFlag            string
	compareToSpecificAppBundleFlag string

	analyzeCmd = &cobra.Command{
		Use:   "analyze",
		Short: "Analyzes an IPA file to find Hermes bytecode and check for hash matches in the database",
		Long:  "Opens an IPA file, searches for Hermes bytecode files, disassembles them to extract function hashes, and queries the database for matching hashes.",
		Run: func(cmd *cobra.Command, args []string) {

			if specificAnalysisFlag {

				if specificPackageFlag == "" {
					utils.F.Printf("specific package flag is set, but no package is set\n")
					return
				}

				if compareToSpecificAppBundleFlag == "" {
					utils.F.Printf("specific app bundle flag is set, but no app bundle is set\n")
					return
				}

				utils.L.Printf("Analyzing specific package %s against specific app bundle %s\n", specificPackageFlag, compareToSpecificAppBundleFlag)

				fileContent, err := openFileFromPath(compareToSpecificAppBundleFlag)
				if err != nil {
					utils.F.Printf("failed to open HBC file: %v\n", err)
					return
				}

				// app bundle hbc file
				appBundleContent, err := analyzeHBCFile(fileContent)
				if err != nil {
					utils.F.Printf("failed to analyze app HBC file: %v\n", err)
					return
				}

				utils.S.Printf("App bundle HBC file analyzed successfully.")

				// package hbc file
				packageFileContent, err := openFileFromPath(specificPackageFlag)
				if err != nil {
					utils.F.Printf("failed to open package HBC file: %v\n", err)
					return
				}

				packageContent, err := analyzeHBCFile(packageFileContent)
				if err != nil {
					utils.F.Printf("failed to analyze package HBC file: %v\n", err)
					return
				}

				utils.S.Printf("Package HBC file analyzed successfully.")

				// compare the hashes
				utils.L.Printf("Comparing hashes...")
				structuralFuzzyMatches, contentIR1FuzzyMatches, contentIR2FuzzyMatches, exactMatchesMessage, fuzzyMatchesMessage := compareContents(appBundleContent, packageContent)

				utils.L.Println("Writing fuzzy matches to file")
				err = writeFuzzyMatchesToFile(structuralFuzzyMatches, contentIR1FuzzyMatches, contentIR2FuzzyMatches, exactMatchesMessage, fuzzyMatchesMessage)
				if err != nil {
					utils.F.Printf("failed to write fuzzy matches to file: %v\n", err)
				}

				utils.S.Printf("All comparisons complete.")
				os.Exit(0)

				return
			}

			// Initialize database connection
			database.Init()

		},
	}
)

func init() {
	analyzeCmd.Flags().StringVarP(&ipaFilePath, "ipa", "i", "", "Path to the IPA file to analyze")
	analyzeCmd.Flags().StringVarP(&hbcFilePath, "hbc-file", "b", "", "Path to the HBC file to analyze")
	analyzeCmd.Flags().BoolVarP(&fuzzyMatching, "fuzzy-matching", "f", false, "Enable fuzzy matching with confidence scores")
	analyzeCmd.Flags().Float64VarP(&confidenceThreshold, "confidence-threshold", "c", 0.8, "Minimum confidence threshold for fuzzy matches (0.0-1.0)")
	analyzeCmd.Flags().IntVar(&maxFuzzyDocs, "max-fuzzy-docs", 5000, "Maximum number of documents to check for fuzzy matching")
	analyzeCmd.Flags().Float64Var(&lengthTolerance, "length-tolerance", 0.2, "Length tolerance for fuzzy matching (0.2 = ±20%)")
	analyzeCmd.Flags().BoolVarP(&specificAnalysisFlag, "specific-analysis", "s", false, "Enable specific analysis")
	analyzeCmd.Flags().StringVarP(&specificPackageFlag, "specific-package", "p", "", "Path to the package HBC file to analyze")
	analyzeCmd.Flags().StringVarP(&compareToSpecificAppBundleFlag, "compare-to-app-bundle", "a", "", "Path to the app bundle HBC file to compare to")
	analyzeCmd.Flags().StringVarP(&outputFilePath, "output-file", "o", "", "Path to the output file to write fuzzy matches to")
}

func writeFuzzyMatchesToFile(structuralFuzzyMatches, contentIR1FuzzyMatches, contentIR2FuzzyMatches []string, exactMatchesMessage, fuzzyMatchesMessage string) error {
	if outputFilePath == "" {
		utils.F.Printf("output file path is not set\n")
		return fmt.Errorf("output file path is not set")
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		utils.F.Printf("failed to create output file: %v\n", err)
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	fmt.Fprintln(writer, exactMatchesMessage)
	fmt.Fprintln(writer, fuzzyMatchesMessage)
	fmt.Fprintln(writer, "--------------------------------")
	fmt.Fprintln(writer)

	fmt.Fprintln(writer, "Structural Fuzzy Matches:")
	for _, match := range structuralFuzzyMatches {
		fmt.Fprintln(writer, "\t", match)
	}

	fmt.Fprintln(writer, "\n\nContent IR1 Fuzzy Matches:")
	for _, match := range contentIR1FuzzyMatches {
		fmt.Fprintln(writer, "\t", match)
	}

	fmt.Fprintln(writer, "\n\nContent IR2 Fuzzy Matches:")
	for _, match := range contentIR2FuzzyMatches {
		fmt.Fprintln(writer, "\t", match)
	}

	utils.S.Printf("Fuzzy matches written to file: %s\n", outputFilePath)
	return nil
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix to store distances
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// levenshteinSimilarity calculates similarity as a ratio between 0.0 and 1.0
func levenshteinSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 && len(s2) == 0 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	distance := levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))
	return 1.0 - (float64(distance) / float64(maxLen))
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// chunkStrings splits a slice of strings into chunks of specified size
func chunkStrings(slice []string, chunkSize int) [][]string {
	if len(slice) == 0 {
		return nil
	}

	var chunks [][]string
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// isLengthCompatible checks if two strings have compatible lengths based on tolerance
func isLengthCompatible(s1, s2 string, tolerance float64) bool {
	len1, len2 := float64(len(s1)), float64(len(s2))
	minLen, maxLen := len1, len2
	if len2 < len1 {
		minLen, maxLen = len2, len1
	}

	// Check if the difference is within tolerance
	return (maxLen-minLen)/maxLen <= tolerance
}

// buildLengthFilter creates a MongoDB filter for IR strings within length tolerance
func buildLengthFilter(inputLength int, tolerance float64, hashField string) bson.M {
	// Create filter for the raw IR field within the hashes array
	rawFieldName := ""
	switch hashField {
	case "structural_hash":
		rawFieldName = "structural_raw"
	case "content_ir1_hash":
		rawFieldName = "content_ir1_raw"
	case "content_ir2_hash":
		rawFieldName = "content_ir2_raw"
	}

	// Use direct field access since hash is a single object, not an array
	// We'll do the actual length filtering in the application code with isLengthCompatible
	fieldPath := fmt.Sprintf("hash.%s", rawFieldName)
	return bson.M{
		fieldPath: bson.M{
			"$exists": true,
			"$ne":     "",
			"$type":   "string", // Ensure it's actually a string
		},
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

func fuzzyMatch(packageIRs, appIRs []string, exactMatchedIndices []int, resultsContent *FuzzyContent, wg *sync.WaitGroup) {
	defer wg.Done()

	var matches, eligible int
	var matchedIRs []string
	var longAppIRs []string // pre-filter app IRs to avoid checking all of them
	for _, s := range appIRs {
		if len(s) >= 30 {
			longAppIRs = append(longAppIRs, s)
		}
	}

	for i, pkgIR := range packageIRs { // package IRs to check
		if len(pkgIR) < 30 || slices.Contains(exactMatchedIndices, i) {
			continue // skip short strings and exact matches
		}
		eligible++

		for _, appIR := range longAppIRs { // app IRs to check
			// quick length compatibility check to avoid expensive distance when clearly different
			if !isLengthCompatible(pkgIR, appIR, lengthTolerance) {
				continue
			}
			sim := levenshteinSimilarity(pkgIR, appIR) // similarity score
			if sim >= confidenceThreshold {
				matches++
				matchedIRs = append(matchedIRs, appIR)
				break
			}
		}
	}

	resultsContent.Eligible = eligible
	resultsContent.Matched = matches
	resultsContent.MatchedIRs = matchedIRs
}

func compareContents(appBundleContent, packageContent Content) (structualFuzzyMatches, contentIR1FuzzyMatches, contentIR2FuzzyMatches []string, exactMatchesMessage, fuzzyMatchesMessage string) {
	// The goal is to make an educated guess whether a given package is present in
	// the provided application bundle by comparing:
	// 1. Exact hash equality (structural/content-IR hashes)
	// 2. Fuzzy string similarity (structural/content-IR raw strings)
	//
	// A match is only considered for strings that are at least 30 characters
	// long. Fuzzy similarity uses the levenshteinSimilarity with confidenceThreshold.

	utils.L.Printf("\n────────────────────────────────────────\n")
	utils.L.Printf("Starting content comparison (confidence ≥ %.2f)\n", confidenceThreshold)

	// Exact hash comparison

	// Helper to build a quick lookup map
	buildSet := func(arr []string) map[string]struct{} {
		m := make(map[string]struct{}, len(arr))
		for _, v := range arr {
			m[v] = struct{}{}
		}
		return m
	}

	exactMatch := func(packageHashes, appHashes []string) (indices []int, matches int) {
		appSet := buildSet(appHashes)
		for i, h := range packageHashes {
			if _, ok := appSet[h]; ok {
				indices = append(indices, i)
				matches++
			}
		}
		return
	}

	structuralExactIndices, structuralExact := exactMatch(packageContent.StructuralHashes, appBundleContent.StructuralHashes)
	contentIR1ExactIndices, contentIR1Exact := exactMatch(packageContent.ContentIR1Hashes, appBundleContent.ContentIR1Hashes)
	contentIR2ExactIndices, contentIR2Exact := exactMatch(packageContent.ContentIR2Hashes, appBundleContent.ContentIR2Hashes)

	totalPackageHashes := len(packageContent.StructuralHashes) + len(packageContent.ContentIR1Hashes) + len(packageContent.ContentIR2Hashes)
	totalExactMatches := structuralExact + contentIR1Exact + contentIR2Exact

	pct := func(matches, total int) float64 {
		if total == 0 {
			return 0.0
		}
		return (float64(matches) / float64(total)) * 100.0
	}

	exactMatchesMessage = fmt.Sprintf("Exact hash matches:\n  • Structural : %d/%d (%.2f%%)\n  • Content IR1: %d/%d (%.2f%%)\n  • Content IR2: %d/%d (%.2f%%)\n  ⇒ Overall    : %d/%d (%.2f%%)",
		structuralExact, len(packageContent.StructuralHashes), pct(structuralExact, len(packageContent.StructuralHashes)),
		contentIR1Exact, len(packageContent.ContentIR1Hashes), pct(contentIR1Exact, len(packageContent.ContentIR1Hashes)),
		contentIR2Exact, len(packageContent.ContentIR2Hashes), pct(contentIR2Exact, len(packageContent.ContentIR2Hashes)),
		totalExactMatches, totalPackageHashes, pct(totalExactMatches, totalPackageHashes))

	utils.S.Println(exactMatchesMessage)

	// Fuzzy IR comparison (Levenshtein) - parallelized
	var wg sync.WaitGroup

	wg.Add(3)

	results := &FuzzyResults{
		Structural: &FuzzyContent{},
		ContentIR1: &FuzzyContent{},
		ContentIR2: &FuzzyContent{},
	}

	go fuzzyMatch(packageContent.StructuralIRs, appBundleContent.StructuralIRs, structuralExactIndices, results.Structural, &wg)

	go fuzzyMatch(packageContent.ContentIR1s, appBundleContent.ContentIR1s, contentIR1ExactIndices, results.ContentIR1, &wg)

	go fuzzyMatch(packageContent.ContentIR2s, appBundleContent.ContentIR2s, contentIR2ExactIndices, results.ContentIR2, &wg)

	wg.Wait()

	totalEligible := results.Structural.Eligible + results.ContentIR1.Eligible + results.ContentIR2.Eligible
	totalFuzzyMatches := results.Structural.Matched + results.ContentIR1.Matched + results.ContentIR2.Matched

	fuzzyMatchesMessage = fmt.Sprintf("Fuzzy IR matches (≥%.2f similarity):\n  • Structural : %d/%d (%.2f%%)\n  • Content IR1: %d/%d (%.2f%%)\n  • Content IR2: %d/%d (%.2f%%)\n  ⇒ Overall    : %d/%d (%.2f%%)",
		confidenceThreshold,
		results.Structural.Matched, results.Structural.Eligible, pct(results.Structural.Matched, results.Structural.Eligible),
		results.ContentIR1.Matched, results.ContentIR1.Eligible, pct(results.ContentIR1.Matched, results.ContentIR1.Eligible),
		results.ContentIR2.Matched, results.ContentIR2.Eligible, pct(results.ContentIR2.Matched, results.ContentIR2.Eligible),
		totalFuzzyMatches, totalEligible, pct(totalFuzzyMatches, totalEligible))

	utils.S.Println(fuzzyMatchesMessage)

	return results.Structural.MatchedIRs, results.ContentIR1.MatchedIRs, results.ContentIR2.MatchedIRs, exactMatchesMessage, fuzzyMatchesMessage
}

// analyzeHBCFile disassembles the HBC file and queries the database for hash matches
func analyzeHBCFile(fileContent []byte) (Content, error) {
	utils.L.Printf("Disassembling HBC file")

	// Create a reader from the file content
	contentReader := bytes.NewReader(fileContent)

	// Create HBC reader and parse the file
	hbcReader := hbc.NewHBCReader()
	err := hbcReader.ReadWholeFile(contentReader)
	if err != nil {
		return Content{}, fmt.Errorf("failed to parse HBC file: %v", err)
	}

	utils.S.Printf("HBC file parsed successfully. Hermes version: %d, Function count: %d\n",
		hbcReader.Header.Version, hbcReader.Header.FunctionCount)

	// Create function objects and extract IRs
	fois, err := hbc.CreateFunctionObjects(hbcReader)
	if err != nil {
		return Content{}, fmt.Errorf("failed to create function objects: %v", err)
	}

	utils.L.Printf("Created %d function objects, extracting hashes and IR strings...\n", len(fois))

	var structuralHashes []string
	var contentIR1Hashes []string
	var contentIR2Hashes []string
	var structuralIRs []string
	var contentIR1s []string
	var contentIR2s []string

	for _, foi := range fois {
		structuralIR, contentIR1, contentIR2 := foi.ToIR()

		// Calculate hashes and store raw IRs
		if len(structuralIR) >= 30 {
			structuralHash := fmt.Sprintf("%x", sha256.Sum256([]byte(structuralIR)))
			structuralHashes = append(structuralHashes, structuralHash)
			structuralIRs = append(structuralIRs, structuralIR)
		}

		if contentIR1 != "" && len(contentIR1) >= 30 {
			contentIR1Hash := fmt.Sprintf("%x", sha256.Sum256([]byte(contentIR1)))
			contentIR1Hashes = append(contentIR1Hashes, contentIR1Hash)
			contentIR1s = append(contentIR1s, contentIR1)
		}

		if contentIR2 != "" && len(contentIR2) >= 30 {
			contentIR2Hash := fmt.Sprintf("%x", sha256.Sum256([]byte(contentIR2)))
			contentIR2Hashes = append(contentIR2Hashes, contentIR2Hash)
			contentIR2s = append(contentIR2s, contentIR2)
		}
	}

	utils.L.Printf("Extracted %d structural hashes, %d content IR1 hashes, %d content IR2 hashes\n",
		len(structuralHashes), len(contentIR1Hashes), len(contentIR2Hashes))

	return Content{
		StructuralHashes: structuralHashes,
		ContentIR1Hashes: contentIR1Hashes,
		ContentIR2Hashes: contentIR2Hashes,
		StructuralIRs:    structuralIRs,
		ContentIR1s:      contentIR1s,
		ContentIR2s:      contentIR2s,
	}, nil
}
