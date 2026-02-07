package analyzer

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/cespare/xxhash"
)

const (
	LSH_BANDS = 32 // Number of LSH bands
	LSH_ROWS  = 4  // MINHASH_SIZE / LSH_BANDS (128/32 = 4)
)

// MinHasher estimates Jaccard similarity between token sets using locality-sensitive
// hashing (LSH). It maintains NumHashes independent universal hash functions of the
// form h(x) = (a*x + b) % p, where a and b are random coefficients and p is a
// Mersenne prime (2^31 − 1). Each call to ComputeSignature produces a fixed-length
// signature by keeping the minimum hash value per function across all input tokens
// (hashed via xxhash). Signatures can be compared with JaccardSimilarity for an
// approximate similarity score, or partitioned into LSH bands with ComputeLSHBand
// for efficient candidate-pair identification in large corpora.
type MinHasher struct {
	NumHashes int      // number of hash functions (signature size)
	Seeds     []uint64 // seeds for each hash function
	A         []uint64 // hash parameters
	B         []uint64 // hash parameters
	Prime     uint64   // large prime number
}

// NewMinHasher creates a MinHasher with numHashes independent universal hash
// functions for computing MinHash signatures. Each hash function h(x) = (a*x + b) % p
// uses randomly generated coefficients a (in [1, p)) and b (in [0, p)), where p is the
// Mersenne prime 2^31 − 1. The numHashes parameter controls the signature length and
// determines the accuracy/speed trade-off: higher values produce more accurate Jaccard
// similarity estimates at the cost of larger signatures and longer computation. A
// typical value is 128 (MINHASH_SIZE). Seeds are generated from the current time, so
// each MinHasher instance produces distinct signatures.
func NewMinHasher(numHashes int) *MinHasher {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// large prime
	prime := uint64(2147483647) // 2^31 - 1

	seeds := make([]uint64, numHashes)
	a := make([]uint64, numHashes)
	b := make([]uint64, numHashes)

	for i := range numHashes {
		seeds[i] = rand.Uint64()
		a[i] = rand.Uint64()%prime + 1 // to ensure that a > 0
		b[i] = rand.Uint64() % prime   // b can be 0
	}

	return &MinHasher{
		NumHashes: numHashes,
		Seeds:     seeds,
		A:         a,
		B:         b,
		Prime:     prime,
	}
}

// ComputeSignature computes a fixed-length MinHash signature for a set of string
// tokens. Each token is base-hashed via xxhash, then passed through all NumHashes
// universal hash functions h(x) = (a*x + b) % p. The signature retains the minimum
// hash value seen per function across all input tokens, producing a []uint64 of
// length NumHashes. For empty input, every position is set to math.MaxUint64
// (indicating no tokens were observed). The resulting signature can be compared
// with JaccardSimilarity for approximate set similarity, or partitioned into bands
// with ComputeLSHBand for efficient candidate-pair retrieval.
func (mh *MinHasher) ComputeSignature(tokens []string) []uint64 {
	if len(tokens) == 0 {
		// return empty signature of max values
		signature := make([]uint64, mh.NumHashes)
		for i := range signature {
			signature[i] = math.MaxUint64
		}
		return signature
	}

	// inititalize with max values
	signature := make([]uint64, mh.NumHashes)
	for i := range signature {
		signature[i] = math.MaxUint64
	}

	// for each token, compute hash values and update signature
	for _, token := range tokens {
		tokenBytes := []byte(token)
		baseHash := xxhash.Sum64(tokenBytes)

		// compute hash for each hash function
		for i := 0; i < mh.NumHashes; i++ {
			// hash function: h_i(x) = ((a_i * x + b_i) % p)
			hashValue := (mh.A[i]*baseHash + mh.B[i]) % mh.Prime

			// keep minimum hash value
			if hashValue < signature[i] {
				signature[i] = hashValue
			}
		}
	}

	return signature
}

// JaccardSimilarity estimates the Jaccard similarity between two token sets by
// comparing their MinHash signatures element-wise. It counts the fraction of
// positions where sig1[i] == sig2[i], returning a float64 in [0.0, 1.0] that
// approximates the ratio |A ∩ B| / |A ∪ B| of the original sets. Both signatures
// must have the same length (typically NumHashes); mismatched or zero-length
// signatures return 0.0. Signatures produced by ComputeSignature with
// math.MaxUint64 sentinel values (empty input) will match each other but not
// non-empty signatures, yielding correct boundary behavior.
func (mh *MinHasher) JaccardSimilarity(sig1, sig2 []uint64) float64 {
	if len(sig1) != len(sig2) || len(sig1) == 0 {
		return 0.0
	}

	matches := 0
	for i := range sig1 {
		if sig1[i] == sig2[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(sig1))
}

// ComputeLSHBand extracts a single band from a MinHash signature and returns a
// hex-encoded hash string for locality-sensitive hashing (LSH) candidate-pair
// identification. It slices LSH_ROWS consecutive values (default 4) starting at
// bandNum * LSH_ROWS from the signature, and formats them as "bandNum:hexDigest".
// Two signatures that share the same band string for any bandNum are candidate
// similar pairs. Typical usage partitions a full signature into LSH_BANDS bands
// (default 32), each stored in an index for fast lookup. The bandNum parameter
// must be in [0, LSH_BANDS); out-of-range values that exceed the signature length
// are clamped to avoid a panic.
func (mh *MinHasher) ComputeLSHBand(signature []uint64, bandNum int) string {
	start := bandNum * LSH_ROWS
	end := min(start+LSH_ROWS, len(signature)) // ensure we don't go out of bounds of the signature

	bandSig := signature[start:end]
	return fmt.Sprintf("%d:%x", bandNum, bandSig)

}
