package types

// defstructs.go is inspired by
// [P1sec/hermes-dec]
// https://github.com/P1sec/hermes-dec/blob/main/src/parsers/hbc_opcodes/def_classes.py

import (
	"encoding/binary"
	"math"
)

// OperandType holds the name and the corresponding C++ type
type OperandType struct {
	Name string
	Size int
	Kind string
}

// NewOperandType constructs an OperandType
func NewOperandType(name string, cType string) OperandType {
	var size int
	switch cType {
	case "int8_t", "uint8_t":
		size = 1
	case "int16_t", "uint16_t":
		size = 2
	case "int32_t", "uint32_t":
		size = 4
	case "double":
		size = 8
	default:
		panic("unknown C type: " + cType)
	}
	return OperandType{
		Name: name,
		Size: size,
		Kind: cType,
	}
}

// OperandMeaning represents different meanings of an operand
type OperandMeaning int

const (
	BigIntID OperandMeaning = iota + 1
	FunctionID
	StringID
)

// Operand holds an OperandType and optionally its meaning
type Operand struct {
	Type    OperandType
	Meaning *OperandMeaning // optional - can be nil
}

// Instruction represents a Hermes runtime instruction with its operands
type Instruction struct {
	Name         string
	Opcode       int
	Operands     []Operand
	StructSize   int // size of all operands combined
	BinarySize   int // Opcode byte + operands size
	HasRetTarget bool
}

// NewInstruction creates a new instruction with the given name, opcode and operand types
func NewInstruction(name string, opcode int, args []OperandType) *Instruction {
	operands := make([]Operand, len(args))
	structSize := 0

	for i, arg := range args {
		operands[i] = Operand{
			Type:    arg,
			Meaning: nil,
		}
		structSize += arg.Size
	}

	inst := &Instruction{
		Name:         name,
		Opcode:       opcode,
		Operands:     operands,
		StructSize:   structSize,
		BinarySize:   1 + structSize, // 1 byte for opcode + size of all operands
		HasRetTarget: false,
	}

	return inst
}

// EncodeWithArgs is a utility function to write an instruction to a byte slice
func (inst *Instruction) EncodeWithArgs(args []any) []byte {
	// create buffer
	buf := make([]byte, inst.BinarySize)
	// write opcode
	buf[0] = byte(inst.Opcode)
	// current offset in buffer
	offset := 1

	// write each arg
	for i, arg := range args {
		if i >= len(inst.Operands) {
			panic("too many arguments for instruction:" + inst.Name)
		}

		operand := inst.Operands[i]
		switch operand.Type.Kind {
		case "int8_t", "uint8_t":
			buf[offset] = arg.(byte)
			offset += 1
		case "int16_t", "uint16_t":
			binary.LittleEndian.PutUint16(buf[offset:], arg.(uint16))
			offset += 2
		case "int32_t", "uint32_t":
			binary.LittleEndian.PutUint32(buf[offset:], arg.(uint32))
			offset += 4
		case "double":
			binary.LittleEndian.PutUint64(buf[offset:], math.Float64bits(arg.(float64)))
			offset += 8
		}
	}

	return buf
}
