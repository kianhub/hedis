package database

import (
	"hermes-decompiler/pkg/utils"
	"os"

	"go.philip.id/mongopiet"
)

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
