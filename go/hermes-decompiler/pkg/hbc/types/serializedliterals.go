package types

// serializedliterals.go is inspired by
// [P1sec/hermes-dec]
// https://github.com/P1sec/hermes-dec/blob/main/src/parsers/serialized_literal_parser.py

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

/*
SerializedLiterals format is used to store structured arrays of string references and literals within the Hermes array buffer,
object key buffer, and object value buffer.
*/

// TagType represents the type of a serialized literal value
type TagType int

const (
	NullTag TagType = iota
	TrueTag
	FalseTag
	NumberTag
	LongStringTag
	ShortStringTag
	ByteStringTag
	IntegerTag
)

// SLPValue represents a single serialized literal value with its tag type
type SLPValue struct {
	TagType TagType
	Value   any
}

// SLPArray represents an array of serialized literal values
type SLPArray struct {
	Items []SLPValue
}

// ToStrings converts the SLPArray to a list of readable strings
func (slpa *SLPArray) ToStrings(stringTable []string) []string {
	stringifiedItems := make([]string, 0, len(slpa.Items))

	for _, item := range slpa.Items {
		var str string

		switch item.TagType {
		case NullTag:
			str = "null"
		case TrueTag:
			str = "true"
		case FalseTag:
			str = "false"
		case NumberTag:
			str = fmt.Sprintf("%v", item.Value)
		case LongStringTag, ShortStringTag, ByteStringTag:
			idx := item.Value.(int)
			str = fmt.Sprintf("%q", stringTable[idx])
		case IntegerTag:
			str = fmt.Sprintf("%d", item.Value)
		default:
			panic("unknown tag type")
		}

		stringifiedItems = append(stringifiedItems, str)
	}

	return stringifiedItems
}

// UnpackSLPArray parses the serialized literal array data into an SLPArray
func UnpackSLPArray(data []byte, numItems int) (*SLPArray, error) {
	reader := bytes.NewReader(data)
	items := make([]SLPValue, 0, numItems)

	for len(items) < numItems {
		// read tag byte
		nextTag := make([]byte, 1)
		n, err := reader.Read(nextTag)
		if err != nil || n == 0 {
			break
		}

		// extract tag type and length
		tagType := TagType((nextTag[0] >> 4) & 0b111)
		var length int
		if nextTag[0]>>7 == 1 { // extended length flag
			lengthByte := make([]byte, 1)
			_, err := reader.Read(lengthByte)
			if err != nil {
				return nil, fmt.Errorf("failed to read extended length byte: %v", err)
			}
			length = int((int(nextTag[0])&0b1111)<<8 | int(lengthByte[0]))
		} else {
			length = int(nextTag[0] & 0b1111)
		}

		// read value
		values := make([]any, 0, length)
		for range length {
			var value any

			switch tagType {
			case NullTag:
				value = nil
			case TrueTag:
				value = true
			case FalseTag:
				value = false
			case NumberTag:
				var num float64
				err := binary.Read(reader, binary.LittleEndian, &num)
				if err != nil {
					return nil, fmt.Errorf("failed to read number: %v", err)
				}
				value = num
			case LongStringTag:
				var val uint32
				err := binary.Read(reader, binary.LittleEndian, &val)
				if err != nil {
					return nil, fmt.Errorf("failed to read long string index: %v", err)
				}
				value = int(val)
			case ShortStringTag:
				var val uint16
				err := binary.Read(reader, binary.LittleEndian, &val)
				if err != nil {
					return nil, fmt.Errorf("failed to read short string index: %v", err)
				}
				value = int(val)
			case ByteStringTag:
				var val uint8
				err := binary.Read(reader, binary.LittleEndian, &val)
				if err != nil {
					return nil, fmt.Errorf("failed to read byte string index: %v", err)
				}
				value = int(val)
			case IntegerTag:
				var val uint32
				err := binary.Read(reader, binary.LittleEndian, &val)
				if err != nil {
					return nil, fmt.Errorf("failed to read integer: %v", err)
				}
				value = int(val)
			default:
				return nil, fmt.Errorf("unknown tag type: %d", tagType)
			}

			values = append(values, value)
		}

		// add values to items
		for _, value := range values {
			items = append(items, SLPValue{
				TagType: tagType,
				Value:   value,
			})
		}
	}

	// trim to requested number of items if needed
	if len(items) > numItems {
		items = items[:numItems]
	}

	return &SLPArray{Items: items}, nil
}
