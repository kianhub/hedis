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

type MinHasher struct {
	NumHashes int      // number of hash functions (signature size)
	Seeds     []uint64 // seeds for each hash function
	A         []uint64 // hash parameters
	B         []uint64 // hash parameters
	Prime     uint64   // large prime number
}

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

// ComputeSignature computes MinHash signature for a slice of tokens
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

// JaccardSimilarity computes Jaccard similarity between two MinHash signatures
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

// LSH computes LHS band hashe for a MinHash signature
func (mh *MinHasher) ComputeLSHBand(signature []uint64, bandNum int) string {
	start := bandNum * LSH_ROWS
	end := min(start+LSH_ROWS, len(signature)) // ensure we don't go out of bounds of the signature

	bandSig := signature[start:end]
	return fmt.Sprintf("%d:%x", bandNum, bandSig)

}
