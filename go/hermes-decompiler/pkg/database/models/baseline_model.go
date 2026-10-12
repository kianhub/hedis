package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// BaselineModel is a model for storing the baseline of a react native version (empty entry point)
type BaselineModel struct {
	ID                    primitive.ObjectID `bson:"_id"`
	ReactNativeVersion    string             `bson:"react_native_version"`
	HermesBytecodeVersion string             `bson:"hermes_bytecode_version"`
	Hashes                []Hash             `bson:"hashes"`
}

type Hash struct {
	RelativeFunctionIndex int    `bson:"relative_function_index"`
	StructuralRaw         string `bson:"structural_raw"`
	ContentIR1Raw         string `bson:"content_ir1_raw,omitempty"`
	ContentIR2Raw         string `bson:"content_ir2_raw,omitempty"`
	StructuralHash        string `bson:"structural_hash"`
	ContentIR1Hash        string `bson:"content_ir1_hash,omitempty"`
	ContentIR2Hash        string `bson:"content_ir2_hash,omitempty"`
	NormS                 string `bson:"ns,omitempty"`
	NormC                 string `bson:"nc,omitempty"`
}
