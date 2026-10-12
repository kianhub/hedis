package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type PackageHashes []PackageHash

type PackageHashModel struct {
	ID                 primitive.ObjectID `bson:"_id"`
	PackageID          primitive.ObjectID `bson:"package_id"`
	ReactNativeVersion string             `bson:"react_native_version"`
	Hash               Hash               `bson:"hash"`
}

// PackageHash is a model for storing the hashes of a npm package
type PackageHash struct {
	ID                 primitive.ObjectID `bson:"_id"`
	PackageID          primitive.ObjectID `bson:"package_id"`
	ReactNativeVersion string             `bson:"react_native_version"`
	Hashes             []Hash             `bson:"hashes"`
}

// FunctionObjectHashModel represents a single function's fingerprint hashes for a specific
// npm package compiled under a specific React Native version. Each document is stored in the
// "hashes" (or "hashes_ghsa") MongoDB collection and links back to its parent package via
// PackageID. The three IR fields hold the raw IR string and its SHA256 hex digest: Structural
// captures opcode-sequence bigrams, ContentAR1 captures non-identifier string literals, and
// ContentAR2 captures identifiers and object references. These hashes are produced by the
// pipeline's generateHashes function (via FunctionObject.ToIR + SHA256) and consumed by
// CheckIfPackageHasHashes for duplicate detection and by the analyzer for exact-match lookups.
// IR1/IR2 fields use omitempty and may be empty when the function lacks corresponding tokens.
type FunctionObjectHashModel struct {
	ID                 primitive.ObjectID      `bson:"_id"`
	PackageID          primitive.ObjectID      `bson:"package_id"`
	ReactNativeVersion string                  `bson:"react_native_version"`
	Structural         FunctionObjectDataModel `bson:"structural"`
	ContentAR1         FunctionObjectDataModel `bson:"ar1"`
	ContentAR2         FunctionObjectDataModel `bson:"ar2"`
	// normalized, bundle-context-robust hashes (pkg/normhash); empty in thesis-era DBs
	NormS string `bson:"ns,omitempty"`
	Sub   bool   `bson:"sub,omitempty"` // from a subpath-entry bundle (e.g. zod/v4, crypto-js/aes)
	NormC string `bson:"nc,omitempty"`
}

// FunctionObjectDataModel holds a single IR representation of a function as both its raw
// pipe-delimited string (Raw) and its SHA256 hex digest (Hash). Each FunctionObjectHashModel
// contains three instances of this struct—Structural, ContentAR1, and ContentAR2—corresponding
// to the three IR fingerprints produced by FunctionObject.ToIR. Both fields use omitempty and
// will be empty when the source function yields no tokens for that IR level (e.g. a function
// with no string literals produces an empty ContentAR1).
type FunctionObjectDataModel struct {
	Raw  string `bson:"raw,omitempty"`
	Hash string `bson:"hash,omitempty"`
}
