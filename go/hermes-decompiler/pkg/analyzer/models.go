package analyzer

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FunctionSignature holds the complete fingerprint for a single function extracted
// from an npm package compiled with Hermes. It pairs two complementary matching
// strategies: four SHA256 hex-digest hashes (StructuralHashSHA256, ContentIR1HashSHA256,
// ContentIR2HashSHA256, CombinedHashSHA256) for exact-match lookups, and four
// MinHash signature vectors (StructuralHash, ContentIR1Hash, ContentIR2Hash,
// CombinedHash) for approximate Jaccard-similarity comparison via MinHasher.
// LSHBands stores pre-computed locality-sensitive hash band strings produced by
// ComputeLSHBand over the CombinedHash signature, enabling fast candidate-pair
// retrieval in MongoDB without exhaustive pairwise comparison. The struct is
// produced by MinHasher.ComputeFunctionSignature and persisted in the "hashes"
// and "hashes_ghsa" MongoDB collections with BSON field tags. IR1 and IR2 fields
// are omitempty because functions may lack non-identifier strings or identifier
// references respectively, making those IRs empty.
type FunctionSignature struct {
	// metadata
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	PackageName    string             `bson:"package_name"`
	PackageVersion string             `bson:"package_version"`
	PackageID      string             `bson:"package_id"`

	// MinHash signatures as arrays of integers
	StructuralHash []uint64 `bson:"structural_hash"`
	ContentIR1Hash []uint64 `bson:"content_ir1_hash,omitempty"`
	ContentIR2Hash []uint64 `bson:"content_ir2_hash,omitempty"`
	CombinedHash   []uint64 `bson:"combined_hash"`

	// Exact match hashes using SHA256
	StructuralHashSHA256 string `bson:"structural_hash_sha256"`
	ContentIR1HashSHA256 string `bson:"content_ir1_hash_sha256,omitempty"`
	ContentIR2HashSHA256 string `bson:"content_ir2_hash_sha256,omitempty"`
	CombinedHashSHA256   string `bson:"combined_hash_sha256,omitempty"`

	// LSH band hashes for fast similarity search
	LSHBands []string `bson:"lsh_bands"`
}

func (fs *FunctionSignature) String() string {
	return fmt.Sprintf("{StructuralHash: %v, ContentIR1Hash: %v, ContentIR2Hash: %v, CombinedHash: %v, StructuralHashSHA256: %s, ContentIR1HashSHA256: %s, ContentIR2HashSHA256: %s, CombinedHashSHA256: %s, LSHBands: %v}",
		fs.StructuralHash, fs.ContentIR1Hash, fs.ContentIR2Hash, fs.CombinedHash,
		fs.StructuralHashSHA256, fs.ContentIR1HashSHA256, fs.ContentIR2HashSHA256, fs.CombinedHashSHA256, fs.LSHBands)
}

type LSHBand struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty"`
	BandHash    string               `bson:"band_hash"`
	BandNumber  int                  `bson:"band_number"`
	FunctionIDs []primitive.ObjectID `bson:"function_ids"`
}

// MatchResult holds the outcome of comparing a query function's fingerprint against
// a candidate FunctionSignature in the database. Score is the overall similarity
// (0.0–1.0), while StructuralSimilarity, ContentIR1Similarity, ContentIR2Similarity,
// and CombinedSimilarity break the comparison down by IR level — structural captures
// opcode-sequence similarity, content IR1 covers non-identifier strings, content IR2
// covers identifiers and object references, and combined merges all three. IR1 and
// IR2 similarity fields are omitempty because the corresponding IR may be empty for
// functions that lack string literals or identifier references. CandidateID links
// back to the matched FunctionSignature's MongoDB ObjectID.
type MatchResult struct {
	CandidateID          primitive.ObjectID `bson:"candidate_id"`
	PackageName          string             `bson:"package_name"`
	PackageVersion       string             `bson:"package_version"`
	Score                float64            `bson:"score"`
	StructuralSimilarity float64            `bson:"structural_similarity"`
	ContentIR1Similarity float64            `bson:"content_ir1_similarity,omitempty"`
	ContentIR2Similarity float64            `bson:"content_ir2_similarity,omitempty"`
	CombinedSimilarity   float64            `bson:"combined_similarity"`
}
