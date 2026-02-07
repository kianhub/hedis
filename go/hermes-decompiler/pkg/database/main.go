package database

import (
	"hermes-decompiler/pkg/utils"
	"os"

	"go.philip.id/mongopiet"
)

// Init establishes the global MongoDB connection used by all database operations.
// It reads MONGO_CONNECTION_STRING from the environment (required) and MONGO_DB_NAME
// (optional, defaults to "hedis"). The connection is registered via mongopiet.NewClient
// and remains open for the lifetime of the process. Init terminates the process with
// os.Exit(1) if MONGO_CONNECTION_STRING is unset, and panics if the connection fails.
func Init() {
	mongoConnectionString := os.Getenv("MONGO_CONNECTION_STRING")
	if mongoConnectionString == "" {
		utils.F.Println("MONGO_CONNECTION_STRING is not set")
		os.Exit(1)
	}

	dbName := os.Getenv("MONGO_DB_NAME")
	if dbName == "" {
		dbName = "hedis"
	}

	if err := mongopiet.NewClient(mongoConnectionString, dbName); err != nil {
		utils.F.Println("Failed to connect to MongoDB")
		panic(err)
	}
}

func InitWithDatabaseName(databaseName string) {
	mongoConnectionString := os.Getenv("MONGO_CONNECTION_STRING")
	if mongoConnectionString == "" {
		utils.F.Println("MONGO_CONNECTION_STRING is not set")
		os.Exit(1)
	}

	if err := mongopiet.NewClient(mongoConnectionString, databaseName); err != nil {
		utils.F.Println("Failed to connect to MongoDB")
		panic(err)
	}
}
