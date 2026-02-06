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

type FunctionObjectHashModel struct {
	ID                 primitive.ObjectID      `bson:"_id"`
	PackageID          primitive.ObjectID      `bson:"package_id"`
	ReactNativeVersion string                  `bson:"react_native_version"`
	Structural         FunctionObjectDataModel `bson:"structural"`
	ContentAR1         FunctionObjectDataModel `bson:"ar1"`
	ContentAR2         FunctionObjectDataModel `bson:"ar2"`
}

type FunctionObjectDataModel struct {
	Raw  string `bson:"raw,omitempty"`
	Hash string `bson:"hash,omitempty"`
}
