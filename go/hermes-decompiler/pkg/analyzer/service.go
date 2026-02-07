package analyzer

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoMinHashService provides MinHash-based fuzzy similarity search backed by MongoDB.
// It maintains two collections: "signatures" for storing FunctionSignature fingerprints
// and "lsh_bands" for locality-sensitive hashing band indices that enable fast candidate
// retrieval without exhaustive pairwise comparison. The embedded MinHasher is initialized
// with MINHASH_SIZE (128) hash functions for computing and comparing MinHash signatures.
type MongoMinHashService struct {
	client               *mongo.Client
	database             *mongo.Database
	signaturesCollection *mongo.Collection
	lshBandsCollection   *mongo.Collection
	hasher               *MinHasher
}

// MINHASH_SIZE is the number of universal hash functions used to compute each MinHash
// signature. Higher values increase similarity estimation accuracy at the cost of memory
// and computation time. This value also determines the LSH band geometry: with LSH_BANDS=32,
// each band contains MINHASH_SIZE/LSH_BANDS = 4 rows (LSH_ROWS). All signatures stored in
// MongoDB and compared during analysis use this fixed size.
const (
	MINHASH_SIZE = 128

	HIGH_SIMILARITY_THRESHOLD   = 0.8
	MEDIUM_SIMILARITY_THRESHOLD = 0.6
)

func NewMongoMinHashService(mongoURI, dbName string) (*MongoMinHashService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}

	database := client.Database(dbName)

	service := &MongoMinHashService{
		client:               client,
		database:             database,
		signaturesCollection: database.Collection("signatures"),
		lshBandsCollection:   database.Collection("lsh_bands"),
		hasher:               NewMinHasher(MINHASH_SIZE),
	}

	return service, nil
}
