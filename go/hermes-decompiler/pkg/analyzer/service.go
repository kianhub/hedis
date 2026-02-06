package analyzer

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoMinHashService struct {
	client               *mongo.Client
	database             *mongo.Database
	signaturesCollection *mongo.Collection
	lshBandsCollection   *mongo.Collection
	hasher               *MinHasher
}

const (
	MINHASH_SIZE = 128 // Single hash size for all signatures

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
