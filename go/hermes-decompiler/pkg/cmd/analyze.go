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
	"strings"
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

// TODO: use this result struct
type Results struct {
	ExactMatchesMessage              string
	FuzzyMatchesMessage              string
	StructuralFuzzyMatches           []string
	ContentIR1FuzzyMatches           []string
	ContentIR2FuzzyMatches           []string
	StructuralExactMatchesPercentage float64
	ContentIR1ExactMatchesPercentage float64
	ContentIR2ExactMatchesPercentage float64
	StructuralFuzzyMatchesPercentage float64
	ContentIR1FuzzyMatchesPercentage float64
	ContentIR2FuzzyMatchesPercentage float64
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

var specialHash = ""

var (
	ipaFilePath         string
	fuzzyMatching       bool
	confidenceThreshold float64
	/* batchSize           int = 2000 // Default batch size for hash queries */

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

			/* if ipaFilePath != "" {
				// Open and analyze the IPA file
				err := analyzeIPA(ipaFilePath)
				if err != nil {
					utils.F.Printf("error analyzing IPA file: %v\n", err)
					return
				}
			}

			if hbcFilePath != "" {
				fileContent, err := openFileFromPath(hbcFilePath)
				if err != nil {
					utils.F.Printf("failed to open HBC file: %v\n", err)
					return
				}

				err = analyzeHBCFile(fileContent)
				if err != nil {
					utils.F.Printf("error analyzing HBC file: %v\n", err)
					return
				}
			} */

		},
	}

	// Common file extensions that are definitely not HBC files
	/* blacklistedExtensions = []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp", // Images
		".mp4", ".mov", ".avi", ".mkv", ".webm", ".m4v", // Videos
		".mp3", ".wav", ".aac", ".m4a", ".flac", ".ogg", // Audio
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", // Documents
		".zip", ".tar", ".gz", ".7z", ".rar", ".bz2", // Archives
		".txt", ".md", ".rtf", ".csv", ".json", ".xml", ".yml", ".yaml", // Text files
		".html", ".css", ".js", ".ts", ".jsx", ".tsx", ".vue", ".svelte", // Web files (except .js bundles)
		".strings", ".storyboard", ".xib", ".nib", ".car", ".icns", // iOS specific
		".dylib", ".a", ".so", ".dll", // Binaries
		".plist", ".entitlements", ".mobileprovision", ".cer", ".p12", // iOS certificates/config
		".sqlite", ".db", ".realm", // Databases
		".log", ".tmp", ".cache", ".lock", // Temporary files
	} */
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

// analyzeIPA opens the IPA file and analyzes it for HBC content
/* func analyzeIPA(ipaPath string) error {
	utils.L.Printf("Opening IPA file: %s\n", ipaPath)

	// Open the IPA file as a ZIP archive
	reader, err := zip.OpenReader(ipaPath)
	if err != nil {
		return fmt.Errorf("failed to open IPA file as ZIP archive: %v", err)
	}
	defer reader.Close()

	utils.L.Printf("IPA file opened successfully, searching for HBC files...\n")

	// Search for HBC files in the archive
	hbcFile, err := findHBCFile(&reader.Reader)
	if err != nil {
		return fmt.Errorf("error searching for HBC files: %v", err)
	}

	if hbcFile == nil {
		utils.L.Println("No HBC files found in the IPA archive")
		return nil
	}

	utils.S.Printf("Found HBC file: %s\n", hbcFile.Name)

	// Disassemble the HBC file and extract hashes
	fileContent, err := openZipFile(hbcFile)
	if err != nil {
		return fmt.Errorf("failed to open HBC file: %v", err)
	}

	return analyzeHBCFile(fileContent)
} */

// findHBCFile searches through the ZIP archive for files with HBC magic number
/* func findHBCFile(reader *zip.Reader) (*zip.File, error) {
	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Check if file extension is blacklisted
		if isBlacklistedExtension(file.Name) {
			utils.D.Printf("Skipping blacklisted file: %s\n", file.Name)
			continue
		}

		utils.D.Printf("Checking file: %s\n", file.Name)

		// Open the file and check for HBC magic number
		rc, err := file.Open()
		if err != nil {
			utils.F.Printf("Failed to open file %s: %v\n", file.Name, err)
			continue
		}

		// Read first 8 bytes to check magic number
		magicBytes := make([]byte, 8)
		n, err := rc.Read(magicBytes)
		rc.Close()

		if err != nil || n < 8 {
			continue
		}

		// Check if the magic number matches HBC header magic
		magic := uint64(magicBytes[0]) | uint64(magicBytes[1])<<8 | uint64(magicBytes[2])<<16 | uint64(magicBytes[3])<<24 |
			uint64(magicBytes[4])<<32 | uint64(magicBytes[5])<<40 | uint64(magicBytes[6])<<48 | uint64(magicBytes[7])<<56

		if magic == hbc.HeaderMagic {
			utils.S.Printf("Found HBC file with valid magic number: %s\n", file.Name)
			return file, nil
		}
	}

	return nil, nil
} */

// isBlacklistedExtension checks if a file extension is in the blacklist
/* func isBlacklistedExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, blacklistedExt := range blacklistedExtensions {
		if ext == blacklistedExt {
			return true
		}
	}
	return false
} */

/* func openZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", file.Name, err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
} */

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
			/* if i%1000 == 0 {
				utils.L.Printf("Checking app IR %d of %d\n", i, len(longAppIRs))
			} */
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
				if h == specialHash {
					utils.S.Printf("Found special hash as exact match: %s\n", h)
				}
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
			if strings.Contains(contentIR1, "document_picker_canceled") {
				utils.S.Printf("Found document_picker_canceled in contentIR1: %s\n", contentIR1)
				specialHash = contentIR1Hash
			}
		}

		if contentIR2 != "" && len(contentIR2) >= 30 {
			contentIR2Hash := fmt.Sprintf("%x", sha256.Sum256([]byte(contentIR2)))
			contentIR2Hashes = append(contentIR2Hashes, contentIR2Hash)
			contentIR2s = append(contentIR2s, contentIR2)
		}
	}

	utils.L.Printf("Extracted %d structural hashes, %d content IR1 hashes, %d content IR2 hashes\n",
		len(structuralHashes), len(contentIR1Hashes), len(contentIR2Hashes))

	// Query database for matches (exact and/or fuzzy)
	/* return queryDatabaseForMatches(structuralHashes, contentIR1Hashes, contentIR2Hashes, structuralIRs, contentIR1s, contentIR2s) */

	return Content{
		StructuralHashes: structuralHashes,
		ContentIR1Hashes: contentIR1Hashes,
		ContentIR2Hashes: contentIR2Hashes,
		StructuralIRs:    structuralIRs,
		ContentIR1s:      contentIR1s,
		ContentIR2s:      contentIR2s,
	}, nil
}

// queryDatabaseForMatches performs bulk database queries to find matching hashes
/* func queryDatabaseForMatches(structuralHashes, contentIR1Hashes, contentIR2Hashes []string, structuralIRs, contentIR1s, contentIR2s []string) error {
	utils.L.Println("Querying database for hash matches...")

	totalMatches := 0

	// Query for structural hash matches
	if len(structuralHashes) > 0 {
		utils.L.Printf("Searching for %d structural hash matches...\n", len(structuralHashes))
		matches, err := queryHashMatches("structural_hash", structuralHashes, structuralIRs)
		if err != nil {
			utils.F.Printf("Error querying structural hashes: %v\n", err)
		} else {
			utils.S.Printf("Found %d structural hash matches\n", len(matches))
			totalMatches += len(matches)
			for _, match := range matches {
				logHashMatch("Structural", match)
			}
		}
	}

	// Query for content IR1 hash matches
	if len(contentIR1Hashes) > 0 {
		utils.L.Printf("Searching for %d content IR1 hash matches...\n", len(contentIR1Hashes))
		matches, err := queryHashMatches("content_ir1_hash", contentIR1Hashes, contentIR1s)
		if err != nil {
			utils.F.Printf("Error querying content IR1 hashes: %v\n", err)
		} else {
			utils.S.Printf("Found %d content IR1 hash matches\n", len(matches))
			totalMatches += len(matches)
			for _, match := range matches {
				logHashMatch("Content IR1", match)
			}
		}
	}

	// Query for content IR2 hash matches
	if len(contentIR2Hashes) > 0 {
		utils.L.Printf("Searching for %d content IR2 hash matches...\n", len(contentIR2Hashes))
		matches, err := queryHashMatches("content_ir2_hash", contentIR2Hashes, contentIR2s)
		if err != nil {
			utils.F.Printf("Error querying content IR2 hashes: %v\n", err)
		} else {
			utils.S.Printf("Found %d content IR2 hash matches\n", len(matches))
			totalMatches += len(matches)
			for _, match := range matches {
				logHashMatch("Content IR2", match)
			}
		}
	}

	utils.S.Printf("Analysis complete. Total hash matches found: %d\n", totalMatches)
	return nil
} */

// queryHashMatches queries the database for matching hashes of a specific type
/* func queryHashMatches(hashField string, hashes []string, irStrings []string) ([]SimilarityMatch, error) {
	var matches []SimilarityMatch

	// First, try exact hash matching
	utils.L.Printf("Performing exact hash matching for %s...\n", hashField)

	// Query baseline collection
	baselineMatches, err := queryBaselineHashes(hashField, hashes, irStrings)
	if err != nil {
		utils.F.Printf("Error querying baseline hashes: %v\n", err)
	} else {
		matches = append(matches, baselineMatches...)
	}

	// Query package hashes collection
	packageMatches, err := queryPackageHashes(hashField, hashes, irStrings)
	if err != nil {
		utils.F.Printf("Error querying package hashes: %v\n", err)
	} else {
		matches = append(matches, packageMatches...)
	}

	// If fuzzy matching is enabled, also perform fuzzy matching
	if fuzzyMatching {
		utils.L.Printf("Performing fuzzy matching for %s with confidence threshold %.2f...\n", hashField, confidenceThreshold)

		fuzzyPackageMatches, err := queryFuzzyPackageMatches(hashField, irStrings)
		if err != nil {
			utils.F.Printf("Error in fuzzy package matching: %v\n", err)
		} else {
			matches = append(matches, fuzzyPackageMatches...)
		}
	}

	return matches, nil
} */

// queryBaselineHashes queries the baseline collection for hash matches
/* func queryBaselineHashes(hashField string, hashes []string, irStrings []string) ([]SimilarityMatch, error) {
	var matches []SimilarityMatch

	// Split hashes into chunks to avoid query size limits
	hashChunks := chunkStrings(hashes, batchSize)

	utils.L.Printf("Querying baseline collection in %d batches of up to %d hashes each\n", len(hashChunks), batchSize)

	for batchNum, hashChunk := range hashChunks {
		utils.D.Printf("Processing baseline batch %d/%d (%d hashes)\n", batchNum+1, len(hashChunks), len(hashChunk))

		filter := bson.M{
			fmt.Sprintf("hashes.%s", hashField): bson.M{"$in": hashChunk},
		}

		baselines, err := db.Find[models.BaselineModel]("baselines", filter)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, fmt.Errorf("error in baseline batch %d: %v", batchNum+1, err)
		}

		// Extract matching hashes from baselines
		for _, baseline := range baselines {
			for _, hash := range baseline.Hashes {
				var hashValue string
				switch hashField {
				case "structural_hash":
					hashValue = hash.StructuralHash
				case "content_ir1_hash":
					hashValue = hash.ContentIR1Hash
				case "content_ir2_hash":
					hashValue = hash.ContentIR2Hash
				}

				// Check if this hash is in our search list (current chunk)
				for _, searchHash := range hashChunk {
					if hashValue == searchHash {
						match := SimilarityMatch{
							HashMatch: HashMatch{
								Hash:                  hash,
								BaselineID:            &baseline.ID,
								ReactNativeVersion:    baseline.ReactNativeVersion,
								HermesBytecodeVersion: baseline.HermesBytecodeVersion,
								Source:                "baseline",
							},
							Confidence:    1.0,
							MatchType:     "exact",
							MatchedString: hashValue,
							InputString:   hashValue,
						}
						matches = append(matches, match)
						break
					}
				}
			}
		}
	}

	utils.L.Printf("Baseline batched query complete: found %d matches\n", len(matches))
	return matches, nil
} */

// queryPackageHashes queries the package hashes collection for hash matches
/* func queryPackageHashes(hashField string, hashes []string, irStrings []string) ([]SimilarityMatch, error) {
	var matches []SimilarityMatch

	// Split hashes into chunks to avoid query size limits
	hashChunks := chunkStrings(hashes, batchSize)

	utils.L.Printf("Querying package hashes collection in %d batches of up to %d hashes each\n", len(hashChunks), batchSize)

	for batchNum, hashChunk := range hashChunks {
		utils.D.Printf("Processing package batch %d/%d (%d hashes)\n", batchNum+1, len(hashChunks), len(hashChunk))

		filter := bson.M{
			fmt.Sprintf("hash.%s", hashField): bson.M{"$in": hashChunk},
		}

		packageHashes, err := db.Find[models.PackageHashModel]("hashes_ghsa", filter)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, fmt.Errorf("error in package batch %d: %v", batchNum+1, err)
		}

		// Extract matching hashes from package hashes
		for _, pkgHash := range packageHashes {
			var hashValue string
			switch hashField {
			case "structural_hash":
				hashValue = pkgHash.Hash.StructuralHash
			case "content_ir1_hash":
				hashValue = pkgHash.Hash.ContentIR1Hash
			case "content_ir2_hash":
				hashValue = pkgHash.Hash.ContentIR2Hash
			}

			// Check if this hash is in our search list (current chunk)
			for _, searchHash := range hashChunk {
				if hashValue == searchHash {
					match := SimilarityMatch{
						HashMatch: HashMatch{
							Hash:               pkgHash.Hash,
							PackageHashID:      &pkgHash.ID,
							ReactNativeVersion: pkgHash.ReactNativeVersion,
							Source:             "package",
						},
						Confidence:    1.0,
						MatchType:     "exact",
						MatchedString: hashValue,
						InputString:   hashValue,
					}
					matches = append(matches, match)
					break
				}
			}
		}
	}

	utils.L.Printf("Package hashes batched query complete: found %d matches\n", len(matches))
	return matches, nil
} */

// queryFuzzyBaselineMatches performs fuzzy matching against baseline collection
/* func queryFuzzyBaselineMatches(hashField string, irStrings []string) ([]SimilarityMatch, error) {
	var matches []SimilarityMatch

	utils.L.Printf("Fuzzy baseline matching: processing %d input IR strings with max %d docs per input\n", len(irStrings), maxFuzzyDocs)

	// Process each input IR string
	for i, inputIR := range irStrings {
		if len(inputIR) < 30 {
			continue // Skip short IR strings
		}

		// Build length-based filter for this input IR
		lengthFilter := buildLengthFilter(len(inputIR), lengthTolerance, hashField)

		// Query for documents with compatible IR string lengths
		baselines, err := db.Find[models.BaselineModel]("baselines", lengthFilter)
		if err != nil && err != mongo.ErrNoDocuments {
			utils.F.Printf("Error in fuzzy baseline query for input %d: %v\n", i+1, err)
			continue
		}

		// Limit the number of documents to check
		docsChecked := 0
		maxDocsForThisInput := maxFuzzyDocs

		utils.D.Printf("Fuzzy baseline: input %d/%d, found %d length-compatible docs, checking up to %d\n",
			i+1, len(irStrings), len(baselines), maxDocsForThisInput)

		// Check each document until we hit our limit
		for _, baseline := range baselines {
			if docsChecked >= maxDocsForThisInput {
				utils.D.Printf("Reached document limit (%d) for input %d\n", maxDocsForThisInput, i+1)
				break
			}

			for _, hash := range baseline.Hashes {
				var storedIR string
				switch hashField {
				case "structural_hash":
					storedIR = hash.StructuralRaw
				case "content_ir1_hash":
					storedIR = hash.ContentIR1Raw
				case "content_ir2_hash":
					storedIR = hash.ContentIR2Raw
				}

				if storedIR == "" || len(storedIR) < 30 {
					continue
				}

				// Double-check length compatibility (since MongoDB query might not be exact)
				if !isLengthCompatible(inputIR, storedIR, lengthTolerance) {
					continue
				}

				// Calculate similarity
				similarity := levenshteinSimilarity(inputIR, storedIR)

				if similarity >= confidenceThreshold && similarity < 1.0 { // Exclude exact matches
					match := SimilarityMatch{
						HashMatch: HashMatch{
							Hash:                  hash,
							BaselineID:            &baseline.ID,
							ReactNativeVersion:    baseline.ReactNativeVersion,
							HermesBytecodeVersion: baseline.HermesBytecodeVersion,
							Source:                "baseline",
						},
						Confidence:    similarity,
						MatchType:     "fuzzy",
						MatchedString: storedIR,
						InputString:   inputIR,
					}
					matches = append(matches, match)
				}

				docsChecked++
				if docsChecked >= maxDocsForThisInput {
					break
				}
			}
		}

		// Progress update every 10 inputs
		if (i+1)%10 == 0 || i+1 == len(irStrings) {
			utils.D.Printf("Fuzzy baseline progress: %d/%d input strings processed, %d matches so far\n",
				i+1, len(irStrings), len(matches))
		}
	}

	utils.L.Printf("Fuzzy baseline matching complete: found %d matches\n", len(matches))
	return matches, nil
}

// queryFuzzyPackageMatches performs fuzzy matching against package hashes collection
func queryFuzzyPackageMatches(hashField string, irStrings []string) ([]SimilarityMatch, error) {
	var matches []SimilarityMatch

	utils.L.Printf("Fuzzy package matching: processing %d input IR strings with max %d docs per input\n", len(irStrings), maxFuzzyDocs)

	// Process each input IR string
	for i, inputIR := range irStrings {
		if len(inputIR) < 30 {
			continue // Skip short IR strings
		}

		// Build length-based filter for this input IR
		lengthFilter := buildLengthFilter(len(inputIR), lengthTolerance, hashField)

		// Query for documents with compatible IR string lengths
		packageHashes, err := db.Find[models.PackageHashModel]("hashes_ghsa", lengthFilter)
		if err != nil && err != mongo.ErrNoDocuments {
			utils.F.Printf("Error in fuzzy package query for input %d: %v\n", i+1, err)
			continue
		}

		// Limit the number of documents to check
		docsChecked := 0
		maxDocsForThisInput := maxFuzzyDocs

		utils.D.Printf("Fuzzy package: input %d/%d, found %d length-compatible docs, checking up to %d\n",
			i+1, len(irStrings), len(packageHashes), maxDocsForThisInput)

		// Check each document until we hit our limit
		for _, pkgHash := range packageHashes {
			if docsChecked >= maxDocsForThisInput {
				utils.D.Printf("Reached document limit (%d) for input %d\n", maxDocsForThisInput, i+1)
				break
			}

			var storedIR string
			switch hashField {
			case "structural_hash":
				storedIR = pkgHash.Hash.StructuralRaw
			case "content_ir1_hash":
				storedIR = pkgHash.Hash.ContentIR1Raw
			case "content_ir2_hash":
				storedIR = pkgHash.Hash.ContentIR2Raw
			}

			if storedIR == "" || len(storedIR) < 30 {
				continue
			}

			// Double-check length compatibility (since MongoDB query might not be exact)
			if !isLengthCompatible(inputIR, storedIR, lengthTolerance) {
				continue
			}

			// Calculate similarity
			similarity := levenshteinSimilarity(inputIR, storedIR)

			if similarity >= confidenceThreshold && similarity < 1.0 { // Exclude exact matches
				match := SimilarityMatch{
					HashMatch: HashMatch{
						Hash:               pkgHash.Hash,
						PackageHashID:      &pkgHash.ID,
						ReactNativeVersion: pkgHash.ReactNativeVersion,
						Source:             "package",
					},
					Confidence:    similarity,
					MatchType:     "fuzzy",
					MatchedString: storedIR,
					InputString:   inputIR,
				}
				matches = append(matches, match)
			}

			docsChecked++
			if docsChecked >= maxDocsForThisInput {
				break
			}
		}

		// Progress update every 10 inputs
		if (i+1)%10 == 0 || i+1 == len(irStrings) {
			utils.D.Printf("Fuzzy package progress: %d/%d input strings processed, %d matches so far\n",
				i+1, len(irStrings), len(matches))
		}
	}

	utils.L.Printf("Fuzzy package matching complete: found %d matches\n", len(matches))
	return matches, nil
} */

// logHashMatch logs information about a hash match
/* func logHashMatch(hashType string, match SimilarityMatch) {
	var hashValue string
	switch hashType {
	case "Structural":
		hashValue = match.Hash.StructuralHash
	case "Content IR1":
		hashValue = match.Hash.ContentIR1Hash
	case "Content IR2":
		hashValue = match.Hash.ContentIR2Hash
	}

	// Log match information with confidence
	if match.MatchType == "exact" {
		utils.S.Printf("EXACT MATCH - %s Hash: %s (Function Index: %d)\n",
			hashType, hashValue, match.Hash.RelativeFunctionIndex)
	} else {
		utils.S.Printf("FUZZY MATCH - %s Hash: %s (Function Index: %d) [Confidence: %.3f]\n",
			hashType, hashValue, match.Hash.RelativeFunctionIndex, match.Confidence)
	}

	// Log source-specific information
	if match.Source == "package" && match.PackageHashID != nil {
		utils.S.Printf("  └─ Source: Package Hash ID: %s, React Native Version: %s\n",
			match.PackageHashID.Hex(), match.ReactNativeVersion)
	} else if match.Source == "baseline" && match.BaselineID != nil {
		utils.S.Printf("  └─ Source: Baseline ID: %s, React Native Version: %s, Hermes Version: %s\n",
			match.BaselineID.Hex(), match.ReactNativeVersion, match.HermesBytecodeVersion)
	}

	// For fuzzy matches, optionally show IR string comparison (truncated for readability)
	if match.MatchType == "fuzzy" {
		inputTruncated := truncateString(match.InputString, 100)
		matchedTruncated := truncateString(match.MatchedString, 100)
		utils.D.Printf("  └─ Input IR:   %s\n", inputTruncated)
		utils.D.Printf("  └─ Matched IR: %s\n", matchedTruncated)
	}
}

// truncateString truncates a string to a maximum length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
*/
