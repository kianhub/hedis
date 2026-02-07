package hbc

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"hermes-decompiler/pkg/utils"
	"io"
	"strings"
)

const (
	HeaderMagic           uint64 = 0x1F1903C103BC1FC6 // Unicode 'Ἑρμῆ' encoded as UTF-16BE
	LatestBytecodeVersion uint32 = 96
	SHA1NumBytes          int    = 20
)

type ProhibitInvoke int

const (
	ProhibitCall ProhibitInvoke = iota
	ProhibitConstruct
	ProhibitNone
)

type StringKind int

const (
	String StringKind = iota
	Identifier
	Predefined // unused since v0.3.0 - merged with 'Identifier'
)

// HBCHeader represents the file header of a Hermes bytecode (.hbc) file.
// It contains metadata about the file's contents including counts and sizes
// of all sections (functions, strings, arrays, RegExp, BigInts, CJS modules,
// debug info, etc.) that guide subsequent parsing.
//
// The header layout is version-dependent:
//   - BigIntCount and BigIntStorageSize are only present in bytecode version >= 87.
//   - FunctionSourceCount is only present in bytecode version >= 84.
//   - SegmentID was called CjsModuleOffset before bytecode version 78.
//
// The trailing flags byte packs three boolean fields (StaticBuiltins,
// CjsModulesStaticallyResolved, HasAsync) into bits 0-2. After the flags
// byte, the reader aligns to a 32-byte boundary before the first data section.
type HBCHeader struct {
	Magic                        uint64
	Version                      uint32
	SourceHash                   [SHA1NumBytes]byte
	FileLength                   uint32
	GlobalCodeIndex              uint32
	FunctionCount                uint32
	StringKindCount              uint32
	IdentifierCount              uint32
	StringCount                  uint32
	OverflowStringCount          uint32
	StringStorageSize            uint32
	BigIntCount                  uint32 // added in v0.12.0 - bcv87
	BigIntStorageSize            uint32 // added in v0.12.0 - bcv87
	RegExpCount                  uint32
	RegExpStorageSize            uint32
	ArrayBufferSize              uint32
	ObjKeyBufferSize             uint32
	ObjValueBufferSize           uint32
	SegmentID                    uint32 // was called 'CjsModuleOffset' before bcv78
	CjsModuleCount               uint32
	FunctionSourceCount          uint32 // added in v0.8.1 - bcv84
	DebugInfoOffset              uint32
	StaticBuiltins               bool
	CjsModulesStaticallyResolved bool
	HasAsync                     bool
}

// PrintHBCHeaderTable is just a nice print function to log the Hermes Header in the cli
func (header *HBCHeader) PrintHBCHeaderTable() {
	// Calculate column widths
	nameWidth := 40
	valueWidth := 40

	// Print table header
	fmt.Printf("+-%s-+-%s-+\n",
		strings.Repeat("-", nameWidth),
		strings.Repeat("-", valueWidth))

	fmt.Printf("| %-*s | %-*s |\n",
		nameWidth, "Hermes File Header",
		valueWidth, "Value")

	fmt.Printf("+-%s-+-%s-+\n",
		strings.Repeat("-", nameWidth),
		strings.Repeat("-", valueWidth))

	// Helper function to print a row
	printRow := func(name, value string) {
		fmt.Printf("| %-*s | %-*s |\n",
			nameWidth, name,
			valueWidth, value)
	}

	// Print each field
	printRow("Magic", fmt.Sprintf("%d", header.Magic))
	printRow("Version", fmt.Sprintf("%d", header.Version))
	printRow("SourceHash", fmt.Sprintf("%x", header.SourceHash))
	printRow("FileLength", fmt.Sprintf("%d", header.FileLength))
	printRow("GlobalCodeIndex", fmt.Sprintf("%d", header.GlobalCodeIndex))
	printRow("FunctionCount", fmt.Sprintf("%d", header.FunctionCount))
	printRow("StringKindCount", fmt.Sprintf("%d", header.StringKindCount))
	printRow("IdentifierCount", fmt.Sprintf("%d", header.IdentifierCount))
	printRow("StringCount", fmt.Sprintf("%d", header.StringCount))
	printRow("OverflowStringCount", fmt.Sprintf("%d", header.OverflowStringCount))
	printRow("StringStorageSize", fmt.Sprintf("%d", header.StringStorageSize))
	printRow("BigIntCount", fmt.Sprintf("%d", header.BigIntCount))
	printRow("BigIntStorageSize", fmt.Sprintf("%d", header.BigIntStorageSize))
	printRow("RegExpCount", fmt.Sprintf("%d", header.RegExpCount))
	printRow("RegExpStorageSize", fmt.Sprintf("%d", header.RegExpStorageSize))
	printRow("ArrayBufferSize", fmt.Sprintf("%d", header.ArrayBufferSize))
	printRow("ObjKeyBufferSize", fmt.Sprintf("%d", header.ObjKeyBufferSize))
	printRow("ObjValueBufferSize", fmt.Sprintf("%d", header.ObjValueBufferSize))
	printRow("SegmentID", fmt.Sprintf("%d", header.SegmentID))
	printRow("CjsModuleCount", fmt.Sprintf("%d", header.CjsModuleCount))
	printRow("FunctionSourceCount", fmt.Sprintf("%d", header.FunctionSourceCount))
	printRow("DebugInfoOffset", fmt.Sprintf("%d", header.DebugInfoOffset))
	printRow("StaticBuiltins", fmt.Sprintf("%t", header.StaticBuiltins))
	printRow("CjsModulesStaticallyResolved", fmt.Sprintf("%t", header.CjsModulesStaticallyResolved))
	printRow("HasAsync", fmt.Sprintf("%t", header.HasAsync))

	// Print table footer
	fmt.Printf("+-%s-+-%s-+\n",
		strings.Repeat("-", nameWidth),
		strings.Repeat("-", valueWidth))
}

type SmallFunctionHeader struct {
	Offset                 uint32 // 25 bits
	ParamCount             uint8  // 7 bits
	BytecodeSizeInBytes    uint16 // 15 bits
	FunctionName           uint32 // 17 bits
	InfoOffset             uint32 // 25 bits
	FrameSize              uint8  // 7 bits
	EnvironmentState       uint8
	HighestReadCacheIndex  uint8
	HighestWriteCacheIndex uint8
	ProhibitInvoke         uint8 // 2 bits
	StrictMode             bool  // 1 bit
	HasExceptionHandler    bool  // 1 bit
	HasDebugInfo           bool  // 1 bit
	Overflowed             bool  // 1 bit
	Unused                 uint8 // 2 bits
}

type LargeFunctionHeader struct {
	Offset                 uint32
	ParamCount             uint32
	BytecodeSizeInBytes    uint32
	FunctionName           uint32
	InfoOffset             uint32
	FrameSize              uint32
	EnvironmentSize        uint32
	HighestReadCacheIndex  uint8
	HighestWriteCacheIndex uint8
	ProhibitInvoke         uint8 // 2 bits
	StrictMode             bool  // 1 bit
	HasExceptionHandler    bool  // 1 bit
	HasDebugInfo           bool  // 1 bit
	Overflowed             bool  // 1 bit
	Unused                 uint8 // 2 bits
}

type SmallStringTableEntry struct {
	IsUTF16      bool
	IsIdentifier bool // removed because "only used for asserts"
	Offset       uint32
	Length       uint8
}

type OffsetLengthPair struct {
	Offset uint32
	Length uint32
}

type ExceptionHandlerInfo struct {
	Start  uint32
	End    uint32
	Target uint32
}

type DebugOffsets struct {
	SourceLocations  uint32
	ScopeDescData    uint32 // 'LexicalDataOffset' before up to bcv91 and in bcv93
	TextifiedCallees uint32 // Only present in >= bcv91
}

type DebugInfoHeader struct {
	FilenameCount       uint32
	FilenameStorageSize uint32
	FileRegionCount     uint32
	ScopeDescDataOffset uint32
	TextifiedDataOffset uint32 // Only present in >= bcv91
	StringTableOffset   uint32 // Only present in >= bcv91
	DebugDataSize       uint32
}

type DebugFileRegion struct {
	FromAddress     uint32
	FilenameID      uint32
	SourceMappingID uint32
}

type SymbolOffsetPair struct {
	SymbolID uint32
	Offset   uint32
}

type FunctionSourceEntry struct {
	FunctionID uint32
	StringID   uint32
}

// HBCReader holds all parsed data from a Hermes bytecode (.hbc) file.
// It is the central structure produced by reading an HBC file and serves
// as input to the bytecode parser, normalizer, and fingerprinting pipeline.
//
// Create an HBCReader with NewHBCReader, then call ReadWholeFile to populate
// all fields from an io.ReadSeeker. ReadWholeFile parses sections in the order
// defined by the HBC format: header, functions, string kinds, identifier
// hashes, string tables, string storage, arrays, BigInts (bcv >= 87),
// RegExp, CJS modules, function sources (bcv >= 84), and debug info.
//
// FunctionHeaders contains either SmallFunctionHeader or LargeFunctionHeader
// values depending on whether the compact representation overflowed.
// CjsModules similarly holds either uint32 or SymbolOffsetPair values
// depending on bytecode version (< 77 uses pairs, >= 77 uses uint32).
//
// The ParserModule field is initialized by InitParserModule after the header
// is read and provides version-specific opcode definitions for disassembly.
type HBCReader struct {
	Header                   HBCHeader
	FunctionHeaders          []any // can be {Small,Large}FunctionHeader
	FunctionIDToExcHandlers  map[int][]ExceptionHandlerInfo
	FunctionIDToDebugOffsets map[int]DebugOffsets
	StringKinds              []StringKind
	IdentifierHashes         []uint32

	SmallStringTable    []SmallStringTableEntry
	OverflowStringTable []OffsetLengthPair
	Strings             []string

	Arrays       []byte
	ObjectKeys   []byte
	ObjectValues []byte
	BigIntValues []int64

	RegExpTable   []OffsetLengthPair
	RegExpStorage *bytes.Reader

	CjsModules      []any // can be {uint32, SymbolOffsetPair}
	FunctionSources []FunctionSourceEntry

	DebugInfoHeader      DebugInfoHeader
	DebugStringTable     []OffsetLengthPair
	DebugStringStorage   *bytes.Reader
	DebugFileRegions     []DebugFileRegion
	SourcesDataStorage   *bytes.Reader
	ScopeDescDataStorage *bytes.Reader
	TextifiedDataStorage *bytes.Reader
	StringTableStorage   *bytes.Reader

	FileBuffer *bytes.Reader

	ParserModule *ParserModule // contains instruction defs for bcv
}

func NewHBCReader() *HBCReader {
	return &HBCReader{
		FunctionIDToExcHandlers:  make(map[int][]ExceptionHandlerInfo),
		FunctionIDToDebugOffsets: make(map[int]DebugOffsets),
		ParserModule:             nil,
	}
}

func (h *HBCReader) InitParserModule() error {
	if h.Header.Version == 0 {
		return fmt.Errorf("header must be read before init parser moduler")
	}

	h.ParserModule = GetParser(int(h.Header.Version))
	if h.ParserModule == nil {
		utils.F.Println(int(h.Header.Version))
		return fmt.Errorf("no parser module available for bcv %d", h.Header.Version)
	}

	return nil
}

// alignOverPadding
func (h *HBCReader) alignOverPadding(amount int) {
	currentPos, err := h.FileBuffer.Seek(0, io.SeekCurrent)
	if err != nil {
		utils.F.Printf("WARN: alignOverPadding failed to get current position: %v\n", err)
		return
	}

	remainder := currentPos % int64(amount)
	if remainder != 0 {
		paddingNeeded := int64(amount) - remainder
		_, err := h.FileBuffer.Seek(paddingNeeded, io.SeekCurrent)
		if err != nil {
			utils.F.Printf("WARN: alignOverPadding failed to seek %d bytes: %v\n", paddingNeeded, err)
		}
	}
}

func (h *HBCReader) ReadHeader() error {
	h.FileBuffer.Seek(0, io.SeekStart)

	// read magic number
	err := binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.Magic)
	if err != nil {
		return fmt.Errorf("this file is not a Hermes Binary file")
	}

	// read version
	err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.Version)
	if err != nil {
		return fmt.Errorf("failed to read version: %v", err)
	}

	// read source hash
	_, err = io.ReadFull(h.FileBuffer, h.Header.SourceHash[:])
	if err != nil {
		return fmt.Errorf("failed to read source hash: %v", err)
	}

	// read file length
	err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.FileLength)
	if err != nil {
		return fmt.Errorf("failed to read file length: %v", err)
	}

	// verify sha footer if bcv >= 75
	if h.Header.Version >= 75 {
		h.FileBuffer.Seek(0, io.SeekStart)
		fileData := make([]byte, h.Header.FileLength)
		_, err = io.ReadFull(h.FileBuffer, fileData)
		if err != nil {
			return fmt.Errorf("failed to read file data: %v", err)
		}

		// calculate sha and verify
		hash := sha1.Sum(fileData[:len(fileData)-SHA1NumBytes])
		if !bytes.Equal(hash[:], fileData[len(fileData)-SHA1NumBytes:]) {
			return fmt.Errorf("SHA1 hash verification failed")
		}
	}

	// read the rest of the header based on bcv
	h.FileBuffer.Seek(0, io.SeekStart)

	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.Magic); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.Version); err != nil {
		return err
	}
	if _, err = io.ReadFull(h.FileBuffer, h.Header.SourceHash[:]); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.FileLength); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.GlobalCodeIndex); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.FunctionCount); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.StringKindCount); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.IdentifierCount); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.StringCount); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.OverflowStringCount); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.StringStorageSize); err != nil {
		return err
	}

	if h.Header.Version >= 87 {
		if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.BigIntCount); err != nil {
			return err
		}
		if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.BigIntStorageSize); err != nil {
			return err
		}
	} else {
		h.Header.BigIntCount = 0
		h.Header.BigIntStorageSize = 0
	}

	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.RegExpCount); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.RegExpStorageSize); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.ArrayBufferSize); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.ObjKeyBufferSize); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.ObjValueBufferSize); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.SegmentID); err != nil {
		return err
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.CjsModuleCount); err != nil {
		return err
	}

	if h.Header.Version >= 84 {
		if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.FunctionSourceCount); err != nil {
			return err
		}
	} else {
		h.Header.FunctionSourceCount = 0
	}
	if err = binary.Read(h.FileBuffer, binary.LittleEndian, &h.Header.DebugInfoOffset); err != nil {
		return err
	}

	// read single byte containing packed bools
	flagsByteData := make([]byte, 1)
	if _, err := io.ReadFull(h.FileBuffer, flagsByteData); err != nil {
		return fmt.Errorf("failed to read flags byte: %w", err)
	}
	flagsByte := flagsByteData[0]
	h.Header.StaticBuiltins = (flagsByte & 0x01) != 0               // bit 0
	h.Header.CjsModulesStaticallyResolved = (flagsByte & 0x02) != 0 // bit 1
	h.Header.HasAsync = (flagsByte & 0x04) != 0                     // bit 2

	// Perform alignment *after* reading all header fields including the flags byte
	h.alignOverPadding(32) // Align to 32 bytes after header

	// align over padding
	//h.alignOverPadding(32)

	return nil
}

func (h *HBCReader) ReadWholeFile(file io.ReadSeeker) error {
	// read entire file into memory TODO: might need to optimize this
	fileData, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	h.FileBuffer = bytes.NewReader(fileData)

	// read all sections in order
	if err := h.ReadHeader(); err != nil {
		return err
	}

	h.Header.PrintHBCHeaderTable()

	// init parser module based on bcv
	if err := h.InitParserModule(); err != nil {
		return err
	}

	utils.L.Println("Reading Functions...")
	if err := h.ReadFunctions(); err != nil {
		return err
	}
	utils.L.Println("Reading String Kinds...")
	if err := h.ReadStringKinds(); err != nil {
		return err
	}
	utils.L.Println("Reading Identifier Hashes...")
	if err := h.ReadIdentifierHashes(); err != nil {
		return err
	}
	utils.L.Println("Reading Small String Table...")
	if err := h.ReadSmallStringTable(); err != nil {
		return err
	}
	utils.L.Println("Reading Overflow String Table...")
	if err := h.ReadOverflowStringTable(); err != nil {
		return err
	}
	utils.L.Println("Reading String Storage...")
	if err := h.ReadStringStorage(); err != nil {
		return err
	}
	utils.L.Println("Reading Arrays...")
	if err := h.ReadArrays(); err != nil {
		return err
	}

	// >= bcv87 has BigInts field
	if h.Header.Version >= 87 {
		utils.L.Println("Reading Big Ints...")
		if err := h.ReadBigInts(); err != nil {
			return err
		}
	}

	utils.L.Println("Reading Regular Expressions...")
	if err := h.ReadRegExp(); err != nil {
		return err
	}
	utils.L.Println("Reading CJS Modules...")
	if err := h.ReadCjsModules(); err != nil {
		return err
	}

	// >= bcv84 has FunctionSources field
	if h.Header.Version >= 84 {
		utils.L.Println("Reading Function Sources...")
		if err := h.ReadFunctionSources(); err != nil {
			return err
		}
	}

	utils.L.Println("Reading Debug Info...")
	if err := h.ReadDebugInfo(); err != nil {
		return err
	}

	return nil
}

func (h *HBCReader) ReadDebugInfo() error {

	// seek into debug info section
	_, err := h.FileBuffer.Seek(int64(h.Header.DebugInfoOffset), io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to debug info section: %v", err)
	}

	h.DebugInfoHeader = DebugInfoHeader{}

	filenameCountData := make([]byte, 4)
	_, err = io.ReadFull(h.FileBuffer, filenameCountData)
	if err != nil {
		return fmt.Errorf("failed to read debug info filename count: %v", err)
	}
	h.DebugInfoHeader.FilenameCount = binary.LittleEndian.Uint32(filenameCountData)

	filenameStorageSizeData := make([]byte, 4)
	_, err = io.ReadFull(h.FileBuffer, filenameStorageSizeData)
	if err != nil {
		return fmt.Errorf("failed to read debug info filename storage size: %v", err)
	}
	h.DebugInfoHeader.FilenameStorageSize = binary.LittleEndian.Uint32(filenameStorageSizeData)

	fileRegionCountData := make([]byte, 4)
	_, err = io.ReadFull(h.FileBuffer, fileRegionCountData)
	if err != nil {
		return fmt.Errorf("failed to read debug info file region count: %v", err)
	}
	h.DebugInfoHeader.FileRegionCount = binary.LittleEndian.Uint32(fileRegionCountData)

	scopeDescDataOffsetData := make([]byte, 4)
	_, err = io.ReadFull(h.FileBuffer, scopeDescDataOffsetData)
	if err != nil {
		return fmt.Errorf("failed to read debug info scope desc data offset: %v", err)
	}
	h.DebugInfoHeader.ScopeDescDataOffset = binary.LittleEndian.Uint32(scopeDescDataOffsetData)

	// bcv >= bcv91 read some more fields
	if h.Header.Version >= 91 {
		textifiedDataOffsetData := make([]byte, 4)
		_, err := io.ReadFull(h.FileBuffer, textifiedDataOffsetData)
		if err != nil {
			return fmt.Errorf("failed to read debug info textified data offset: %v", err)
		}
		h.DebugInfoHeader.TextifiedDataOffset = binary.LittleEndian.Uint32(textifiedDataOffsetData)

		stringTableOffsetData := make([]byte, 4)
		_, err = io.ReadFull(h.FileBuffer, stringTableOffsetData)
		if err != nil {
			return fmt.Errorf("failed to read debug info string table offset: %v", err)
		}
		h.DebugInfoHeader.StringTableOffset = binary.LittleEndian.Uint32(stringTableOffsetData)
	}

	debugDataSizeData := make([]byte, 4)
	_, err = io.ReadFull(h.FileBuffer, debugDataSizeData)
	if err != nil {
		return fmt.Errorf("failed to read debug info debug data size: %v", err)
	}
	h.DebugInfoHeader.DebugDataSize = binary.LittleEndian.Uint32(debugDataSizeData)

	// read debug string table
	h.DebugStringTable = make([]OffsetLengthPair, h.DebugInfoHeader.FilenameCount)
	for i := uint32(0); i < h.DebugInfoHeader.FilenameCount; i++ {
		offsetData := make([]byte, 4)
		lengthData := make([]byte, 4)
		_, err := io.ReadFull(h.FileBuffer, offsetData)
		if err != nil {
			return fmt.Errorf("failed to read debug info string table offset: %v", err)
		}
		_, err = io.ReadFull(h.FileBuffer, lengthData)
		if err != nil {
			return fmt.Errorf("failed to read debug info string table length: %v", err)
		}
		h.DebugStringTable[i] = OffsetLengthPair{
			Offset: binary.LittleEndian.Uint32(offsetData),
			Length: binary.LittleEndian.Uint32(lengthData),
		}
	}

	// read debug string storage
	debugStringStorage := make([]byte, h.DebugInfoHeader.FilenameStorageSize)
	_, err = io.ReadFull(h.FileBuffer, debugStringStorage)
	if err != nil {
		return fmt.Errorf("failed to read debug info string storage: %v", err)
	}
	h.DebugStringStorage = bytes.NewReader(debugStringStorage)

	// read debug file regions
	h.DebugFileRegions = make([]DebugFileRegion, h.DebugInfoHeader.FileRegionCount)
	for i := uint32(0); i < h.DebugInfoHeader.FileRegionCount; i++ {
		fromAddressData := make([]byte, 4)
		_, err = io.ReadFull(h.FileBuffer, fromAddressData)
		if err != nil {
			return fmt.Errorf("failed to read debug file region fromAddress: %v", err)
		}

		filenameIdData := make([]byte, 4)
		_, err = io.ReadFull(h.FileBuffer, filenameIdData)
		if err != nil {
			return fmt.Errorf("failed to read debug file region filenameId: %v", err)
		}

		sourceMappingIdData := make([]byte, 4)
		_, err = io.ReadFull(h.FileBuffer, sourceMappingIdData)
		if err != nil {
			return fmt.Errorf("failed to read debug file region sourceMappingId: %v", err)
		}

		h.DebugFileRegions[i] = DebugFileRegion{
			FromAddress:     binary.LittleEndian.Uint32(fromAddressData),
			FilenameID:      binary.LittleEndian.Uint32(filenameIdData),
			SourceMappingID: binary.LittleEndian.Uint32(sourceMappingIdData),
		}
	}

	// read remaining debug data based on bcv
	if h.Header.Version < 91 {
		sourcesDataSize := h.DebugInfoHeader.ScopeDescDataOffset
		scopeDescDataSize := h.DebugInfoHeader.DebugDataSize - h.DebugInfoHeader.ScopeDescDataOffset

		sourcesData := make([]byte, sourcesDataSize)
		_, err = io.ReadFull(h.FileBuffer, sourcesData)
		if err != nil {
			return fmt.Errorf("failed to read debug sources data storage: %v", err)
		}
		h.SourcesDataStorage = bytes.NewReader(sourcesData)

		scopeDescData := make([]byte, scopeDescDataSize)
		_, err = io.ReadFull(h.FileBuffer, scopeDescData)
		if err != nil {
			return fmt.Errorf("failed to read debug scope desc data storage: %v", err)
		}
		h.ScopeDescDataStorage = bytes.NewReader(scopeDescData)
	} else {
		sourcesDataSize := h.DebugInfoHeader.ScopeDescDataOffset
		scopeDescDataSize := h.DebugInfoHeader.TextifiedDataOffset - h.DebugInfoHeader.ScopeDescDataOffset
		textifiedDataSize := h.DebugInfoHeader.StringTableOffset - h.DebugInfoHeader.TextifiedDataOffset
		stringTableSize := h.DebugInfoHeader.DebugDataSize - h.DebugInfoHeader.StringTableOffset

		sourcesData := make([]byte, sourcesDataSize)
		_, err = io.ReadFull(h.FileBuffer, sourcesData)
		if err != nil {
			return fmt.Errorf("failed to read sources data storage: %v", err)
		}
		h.SourcesDataStorage = bytes.NewReader(sourcesData)

		scopeDescData := make([]byte, scopeDescDataSize)
		_, err = io.ReadFull(h.FileBuffer, scopeDescData)
		if err != nil {
			return fmt.Errorf("failed to read scope desc data storage: %v", err)
		}
		h.ScopeDescDataStorage = bytes.NewReader(scopeDescData)

		textifiedData := make([]byte, textifiedDataSize)
		_, err = io.ReadFull(h.FileBuffer, textifiedData)
		if err != nil {
			return fmt.Errorf("failed to read textified data storage: %v", err)
		}
		h.TextifiedDataStorage = bytes.NewReader(textifiedData)

		stringTableData := make([]byte, stringTableSize)
		_, err = io.ReadFull(h.FileBuffer, stringTableData)
		if err != nil {
			return fmt.Errorf("failed to read string table storage: %v", err)
		}
		h.StringTableStorage = bytes.NewReader(stringTableData)
	}

	return nil
}

func (h *HBCReader) ReadFunctionSources() error {
	h.FunctionSources = make([]FunctionSourceEntry, 0, h.Header.FunctionSourceCount)

	h.alignOverPadding(4)

	functionSources := make([]FunctionSourceEntry, 0, h.Header.FunctionSourceCount)
	for i := uint32(0); i < h.Header.FunctionSourceCount; i++ {
		functionIdData := make([]byte, 4)
		_, err := io.ReadFull(h.FileBuffer, functionIdData)
		if err != nil {
			return fmt.Errorf("failed to read function source id: %v", err)
		}
		functionId := binary.LittleEndian.Uint32(functionIdData)

		stringIdData := make([]byte, 4)
		_, err = io.ReadFull(h.FileBuffer, stringIdData)
		if err != nil {
			return fmt.Errorf("failed to read function source string id: %v", err)
		}
		stringId := binary.LittleEndian.Uint32(stringIdData)

		functionSources = append(functionSources, FunctionSourceEntry{
			FunctionID: functionId,
			StringID:   stringId,
		})
	}

	h.FunctionSources = functionSources

	return nil
}

func (h *HBCReader) ReadCjsModules() error {
	// bcv >= 77 handles CjsModules differently so we have to handle it
	if h.Header.Version < 77 && h.Header.CjsModulesStaticallyResolved {
		modulesUint32 := make([]uint32, h.Header.CjsModuleCount)

		for i := uint32(0); i < h.Header.CjsModuleCount; i++ {
			data := make([]byte, 4)
			_, err := io.ReadFull(h.FileBuffer, data)
			if err != nil {
				return fmt.Errorf("failed to read CJS module: %v", err)
			}

			modulesUint32[i] = binary.LittleEndian.Uint32(data)
		}

		h.CjsModules = make([]any, h.Header.CjsModuleCount)
		for i, module := range modulesUint32 {
			h.CjsModules[i] = module
		}
	} else {
		// for newer versions, CjsModules are stored as symbol-offset pairs
		symbolOffsetPairs := make([]SymbolOffsetPair, h.Header.CjsModuleCount)

		for i := uint32(0); i < h.Header.CjsModuleCount; i++ {
			symbolData := make([]byte, 4)
			_, err := io.ReadFull(h.FileBuffer, symbolData)
			if err != nil {
				return fmt.Errorf("failed to read CJS module symbol id: %v", err)
			}

			offsetData := make([]byte, 4)
			_, err = io.ReadFull(h.FileBuffer, offsetData)
			if err != nil {
				return fmt.Errorf("failed to read CJS module offset: %v", err)
			}

			symbolID := binary.LittleEndian.Uint32(symbolData)
			offset := binary.LittleEndian.Uint32(offsetData)

			symbolOffsetPairs[i] = SymbolOffsetPair{
				SymbolID: symbolID,
				Offset:   offset,
			}
		}

		h.CjsModules = make([]any, h.Header.CjsModuleCount)
		for i, pair := range symbolOffsetPairs {
			h.CjsModules[i] = pair
		}
	}

	return nil
}

func (h *HBCReader) ReadRegExp() error {
	if h.Header.RegExpCount == 0 {
		return nil
	}

	h.RegExpTable = make([]OffsetLengthPair, h.Header.RegExpCount)

	h.alignOverPadding(4)

	for i := uint32(0); i < h.Header.RegExpCount; i++ {
		offsetData := make([]byte, 4)
		_, err := io.ReadFull(h.FileBuffer, offsetData)
		if err != nil {
			return fmt.Errorf("failed to read RegExp table offset: %v", err)
		}

		lengthData := make([]byte, 4)
		_, err = io.ReadFull(h.FileBuffer, lengthData)
		if err != nil {
			return fmt.Errorf("failed to read RegExp table length: %v", err)
		}

		offset := binary.LittleEndian.Uint32(offsetData)
		length := binary.LittleEndian.Uint32(lengthData)

		h.RegExpTable[i] = OffsetLengthPair{Offset: offset, Length: length}
	}

	// read the RegExp storage data
	h.alignOverPadding(4)

	regExpData := make([]byte, h.Header.RegExpStorageSize)
	_, err := io.ReadFull(h.FileBuffer, regExpData)
	if err != nil {
		return fmt.Errorf("failed to read RegExpStorage: %v", err)
	}

	h.RegExpStorage = bytes.NewReader(regExpData)

	// just like [p1sec/hermes-dec] we parse and decompile regexp seperately (for now)
	// TODO: Implement regexp parsing and decompilation

	return nil
}

func (h *HBCReader) ReadBigInts() error {
	// BigInts were introduced in bcv87
	if h.Header.Version < 87 {
		h.BigIntValues = []int64{}
		return nil
	}

	// Align over padding before reading the table
	h.alignOverPadding(4)

	// Read BigInt table entries (offset/length pairs)
	bigIntTable := make([]OffsetLengthPair, h.Header.BigIntCount)
	for i := uint32(0); i < h.Header.BigIntCount; i++ {
		offsetData := make([]byte, 4)
		_, err := io.ReadFull(h.FileBuffer, offsetData)
		if err != nil {
			return fmt.Errorf("failed to read BigInt table offset: %v", err)
		}

		lengthData := make([]byte, 4)
		_, err = io.ReadFull(h.FileBuffer, lengthData)
		if err != nil {
			return fmt.Errorf("failed to read BigInt table length: %v", err)
		}

		offset := binary.LittleEndian.Uint32(offsetData)
		length := binary.LittleEndian.Uint32(lengthData)

		bigIntTable[i] = OffsetLengthPair{
			Offset: offset,
			Length: length,
		}
	}

	// Align over padding before reading the storage data
	h.alignOverPadding(4)

	// Read the entire BigInt storage data
	bigIntData := make([]byte, h.Header.BigIntStorageSize)
	_, err := io.ReadFull(h.FileBuffer, bigIntData)
	if err != nil {
		return fmt.Errorf("failed to read BigInt storage data: %v", err)
	}

	// Process each BigInt value from the storage
	h.BigIntValues = make([]int64, len(bigIntTable))
	for i, entry := range bigIntTable {
		// Handle invalid offsets (sentinel values)
		if entry.Offset == 0xFFFFFFFF || entry.Length == 0 {
			h.BigIntValues[i] = 0
			continue
		}

		// Safety check to prevent slice bounds errors
		if entry.Offset+entry.Length > uint32(len(bigIntData)) {
			return fmt.Errorf("BigInt entry %d exceeds storage bounds: offset=%d, length=%d, storageSize=%d",
				i, entry.Offset, entry.Length, len(bigIntData))
		}

		// Extract the bytes for this BigInt
		bigIntBytes := bigIntData[entry.Offset : entry.Offset+entry.Length]

		// Convert bytes to int64, similar to Python's int.from_bytes
		// We can only handle up to 8 bytes in an int64
		if entry.Length <= 8 {
			// For values that fit in int64
			var value int64

			// Create padded buffer to ensure we have exactly 8 bytes
			paddedBytes := make([]byte, 8)
			copy(paddedBytes, bigIntBytes)

			// Convert to uint64 first (Go doesn't have direct little-endian int64 conversion)
			unsignedVal := binary.LittleEndian.Uint64(paddedBytes)

			// Handle sign extension for smaller values when highest bit is set
			if entry.Length < 8 && (bigIntBytes[entry.Length-1]&0x80) != 0 {
				// Apply sign extension mask
				mask := ^uint64(0) << (entry.Length * 8)
				unsignedVal |= mask
			}

			// Convert to signed int64
			value = int64(unsignedVal)
			h.BigIntValues[i] = value
		} else {
			// For values larger than 8 bytes, truncate to 8 bytes (64 bits)
			// This is a limitation compared to Python which can handle arbitrary precision
			paddedBytes := make([]byte, 8)
			copy(paddedBytes, bigIntBytes[:8]) // Take only the first 8 bytes
			unsignedVal := binary.LittleEndian.Uint64(paddedBytes)
			h.BigIntValues[i] = int64(unsignedVal)
		}
	}

	return nil
}

func (h *HBCReader) ReadArrays() error {

	h.alignOverPadding(4)
	arrayBuffer := make([]byte, h.Header.ArrayBufferSize)
	_, err := io.ReadFull(h.FileBuffer, arrayBuffer)
	if err != nil {
		return fmt.Errorf("failed to read array data: %v", err)
	}
	h.Arrays = arrayBuffer

	h.alignOverPadding(4)
	objectKeysBuffer := make([]byte, h.Header.ObjKeyBufferSize)
	_, err = io.ReadFull(h.FileBuffer, objectKeysBuffer)
	if err != nil {
		return fmt.Errorf("failed to read object key data: %v", err)
	}
	h.ObjectKeys = objectKeysBuffer

	h.alignOverPadding(4)
	objectValuesBuffer := make([]byte, h.Header.ObjValueBufferSize)
	_, err = io.ReadFull(h.FileBuffer, objectValuesBuffer)
	if err != nil {
		return fmt.Errorf("failed to read object value data: %v", err)
	}
	h.ObjectValues = objectValuesBuffer

	return nil
}

func (h *HBCReader) ReadStringStorage() error {
	h.Strings = make([]string, 0, h.Header.StringCount)

	h.alignOverPadding(4)

	stringStorageData := make([]byte, h.Header.StringStorageSize)
	n, err := io.ReadFull(h.FileBuffer, stringStorageData)
	if err != nil {
		utils.F.Printf("Read %d bytes out of %d requested\n", n, h.Header.StringStorageSize)
		return fmt.Errorf("failed to read string storage data: %v", err)
	}

	stringStorage := bytes.NewReader(stringStorageData)

	// process each string in the small string table
	for i := uint32(0); i < h.Header.StringCount; i++ {
		sste := h.SmallStringTable[i]
		isUTF16 := sste.IsUTF16
		var offset int64
		var length uint32

		// check if this is an overflow string
		if sste.Length == 0xff {
			// if length is 0xff, this is an overflow string, so the offset actually
			// points to an entry in the overflow table
			overflowIndex := int(sste.Offset)
			if overflowIndex >= len(h.OverflowStringTable) {
				return fmt.Errorf("overflow string table index %d out of bounds (size %d) for string idx %d", overflowIndex, len(h.OverflowStringTable), i)
			}

			overflowInfo := h.OverflowStringTable[overflowIndex]
			offset = int64(overflowInfo.Offset)
			length = overflowInfo.Length

			// if int(sste.Offset) >= len(h.OverflowStringTable) {
			// 	return fmt.Errorf("invalid overflow string tabel offset: %d", sste.Offset)
			// }

			// overflowInfo := h.OverflowStringTable[sste.Offset]
			// offset = int64(overflowInfo.Offset)
			// length = uint32(overflowInfo.Length)
		} else {
			offset = int64(sste.Offset)
			length = uint32(sste.Length)
		}

		// UTF-16 strings need x2 the bytes
		actualLength := length
		if isUTF16 {
			actualLength *= 2
		}

		if uint64(offset)+uint64(actualLength) > uint64(len(stringStorageData)) {
			return fmt.Errorf("string read out of bounds: index=%d, offset=%d, readLength=%d, storageSize=%d", i, offset, actualLength, len(stringStorageData))
		}

		stringBytes := make([]byte, actualLength)
		bytesRead, err := stringStorage.ReadAt(stringBytes, offset)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read string bytes at offset %d for length %d (string idx %d): %w", offset, length, i, err)
		}

		if uint32(bytesRead) != actualLength {
			return fmt.Errorf("incomplete string read at offset %d: expected %d bytes, got %d (string idx %d)", offset, length, bytesRead, i)
		}

		// decode string
		var decoded string
		if isUTF16 {
			uint16Data := make([]uint16, length)
			for j := uint32(0); j < length; j++ {
				// UTF-16LE
				uint16Data[j] = uint16(stringBytes[j*2]) | uint16(stringBytes[j*2+1])<<8
			}

			// convert UTF-16 code units to a Go string, handling surrogate pairs
			var sb strings.Builder
			for j := 0; j < len(uint16Data); j++ {
				r := rune(uint16Data[j])

				if r >= 0xD800 && r <= 0xDBFF && j+1 < len(uint16Data) {
					if surrogate := uint16Data[j+1]; surrogate >= 0xDC00 && surrogate <= 0xDFFF {
						// compute actual unicode code point
						r = (r-0xD800)<<10 | (rune(surrogate) - 0xDC00)
						// skip the low surrogate next time
						j++
					}
				}
				sb.WriteRune(r)
			}
			decoded = sb.String()
		} else {
			// for non UTF-16 strings convert each byte to rune and join them
			var sb strings.Builder
			for _, b := range stringBytes {
				sb.WriteRune(rune(b))
			}
			decoded = sb.String()
		}

		h.Strings = append(h.Strings, decoded)
	}

	return nil
}

func (h *HBCReader) ReadOverflowStringTable() error {
	h.OverflowStringTable = make([]OffsetLengthPair, h.Header.OverflowStringCount)

	h.alignOverPadding(4)

	for i := uint32(0); i < h.Header.OverflowStringCount; i++ {
		offsetData := make([]byte, 4)
		lengthData := make([]byte, 4)

		_, err := io.ReadFull(h.FileBuffer, offsetData)
		if err != nil {
			return fmt.Errorf("failed to read overflow string table offset: %v", err)
		}
		_, err = io.ReadFull(h.FileBuffer, lengthData)
		if err != nil {
			return fmt.Errorf("failed to read overflow string table length: %v", err)
		}

		offset := binary.LittleEndian.Uint32(offsetData)
		length := binary.LittleEndian.Uint32(lengthData)

		h.OverflowStringTable[i] = OffsetLengthPair{
			Offset: offset,
			Length: length,
		}
	}

	return nil
}

func (h *HBCReader) ReadSmallStringTable() error {
	h.SmallStringTable = make([]SmallStringTableEntry, h.Header.StringCount)

	h.alignOverPadding(4)

	for i := uint32(0); i < h.Header.StringCount; i++ {
		data := make([]byte, 4)
		_, err := io.ReadFull(h.FileBuffer, data)
		if err != nil {
			return fmt.Errorf("failed to read small string table entry: %v", err)
		}

		w := binary.LittleEndian.Uint32(data)

		// extract fields based on bcv
		var isUTF16 bool
		var offset uint32
		var length uint8
		// isIdentifier for bcv < 56
		var isIdentifier bool

		if h.Header.Version >= 56 {
			// for bcv >= 56 its 1 bit x 23 bits x 8 bits
			isUTF16 = (w & 0x01) == 1
			offset = (w >> 1) & 0x7FFFFF
			length = uint8(w >> 24)
		} else {
			// for bcv < 56 its 1 bit x 1bit x 22 bits x 8
			isUTF16 = (w & 0x01) == 1
			// isIdentifier was removed but reads as 1 bit
			isIdentifier = (w & 0x02) == 2
			offset = (w >> 2) & 0x3FFFFF
			length = uint8(w >> 24)
		}

		h.SmallStringTable[i] = SmallStringTableEntry{
			IsUTF16:      isUTF16,
			Offset:       offset,
			IsIdentifier: isIdentifier,
			Length:       length,
		}
	}

	return nil
}

func (h *HBCReader) ReadIdentifierHashes() error {
	h.IdentifierHashes = make([]uint32, 0, h.Header.IdentifierCount)

	h.alignOverPadding(4)

	for i := uint32(0); i < h.Header.IdentifierCount; i++ {
		data := make([]byte, 4)
		if _, err := io.ReadFull(h.FileBuffer, data); err != nil {
			return fmt.Errorf("failed to read identifier hash entry: %v", err)
		}

		value := binary.LittleEndian.Uint32(data)

		h.IdentifierHashes = append(h.IdentifierHashes, value)
	}

	return nil
}

func (h *HBCReader) ReadStringKinds() error {
	h.StringKinds = make([]StringKind, 0, h.Header.StringKindCount)

	h.alignOverPadding(4)

	for i := uint32(0); i < h.Header.StringKindCount; i++ {
		// read raw 32 bit word
		data := make([]byte, 4)
		if _, err := io.ReadFull(h.FileBuffer, data); err != nil {
			return fmt.Errorf("failed to read string kind entry: %v", err)
		}

		value := binary.LittleEndian.Uint32(data)

		// extract fields based on bcv
		var count uint32
		var kind StringKind

		if h.Header.Version >= 71 {
			// for >= bcv71 (Hermes release 0.3.0 and later) count is 31 bits and kind is 1 bit
			count = value & 0x7FFFFFFF     // 31 bits
			kind = StringKind(value >> 31) // 1 bit
		} else {
			// for older bcv count is 30 bits and kind is 2 bits
			count = value & 0x3FFFFFFF             // 30 bits
			kind = StringKind((value >> 30) & 0x3) // 2 bits
		}

		// decode the run-length encoding by adding the kind value 'count' times
		for j := uint32(0); j < count; j++ {
			h.StringKinds = append(h.StringKinds, kind)
		}
	}

	return nil
}

func (h *HBCReader) ReadFunctions() error {
	h.FunctionHeaders = make([]any, 0, h.Header.FunctionCount)
	h.FunctionIDToExcHandlers = make(map[int][]ExceptionHandlerInfo)
	h.FunctionIDToDebugOffsets = make(map[int]DebugOffsets)

	h.alignOverPadding(4)

	for functionCount := uint32(0); functionCount < h.Header.FunctionCount; functionCount++ {
		// first read function header as raw bytes to manually extract bit fields
		smallHeaderData := make([]byte, 16) // SmallFunctionHeader is 16 bytes
		_, err := io.ReadFull(h.FileBuffer, smallHeaderData)
		if err != nil {
			return err
		}

		// extract bit fields from the small Header
		smallHeader := &SmallFunctionHeader{}

		// Offset 25 bits and ParamCount 7 bits
		w1 := binary.LittleEndian.Uint32(smallHeaderData[0:4])
		smallHeader.Offset = w1 & 0x1FFFFFF      // 25 bits
		smallHeader.ParamCount = uint8(w1 >> 25) // 7 bits

		// ByteCodeSizeInBytes 15 bits and FunctionName 17 bits
		w2 := binary.LittleEndian.Uint32(smallHeaderData[4:8])
		smallHeader.BytecodeSizeInBytes = uint16(w2 & 0x7FFF)   // 15 bits
		smallHeader.FunctionName = uint32((w2 >> 15) & 0x1FFFF) // 17 bits

		// InfoOffset 25 bits and FrameSize 7 bits
		w3 := binary.LittleEndian.Uint32(smallHeaderData[8:12])
		smallHeader.InfoOffset = w3 & 0x1FFFFFF // 25 bits
		smallHeader.FrameSize = uint8(w3 >> 25) // 7 bits

		// EnvironmentsState, Highest{Read,Write}CacheIndex all fit 8 bits
		smallHeader.EnvironmentState = smallHeaderData[12]
		smallHeader.HighestReadCacheIndex = smallHeaderData[13]
		smallHeader.HighestWriteCacheIndex = smallHeaderData[14]

		// flags byte completes the rest of the small header 8 bits
		flagsByte := smallHeaderData[15]
		smallHeader.ProhibitInvoke = flagsByte & 0x3                    // 2 bits
		smallHeader.StrictMode = ((flagsByte >> 2) & 0x1) == 1          // bool bit
		smallHeader.HasExceptionHandler = ((flagsByte >> 3) & 0x1) == 1 // bool bit
		smallHeader.HasDebugInfo = ((flagsByte >> 4) & 0x1) == 1        // bool bit
		smallHeader.Overflowed = ((flagsByte >> 5) & 0x1) == 1          // bool bit
		smallHeader.Unused = (flagsByte >> 6) & 0x3                     // 2 bits

		// save current pos to return after reading extended information
		beforePos, err := h.FileBuffer.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("failed to save current pos: %v", err)
		}

		// check if reading large function header is needed (overflowed)
		var functionHeader any = smallHeader

		if smallHeader.Overflowed {
			// create extended offset from InfoOffset and Offset
			newOffset := (uint64(smallHeader.InfoOffset) << 16) | uint64(smallHeader.Offset)

			// seek to large function header
			_, err := h.FileBuffer.Seek(int64(newOffset), io.SeekStart)
			if err != nil {
				return fmt.Errorf("failed to seek large function header: %v", err)
			}

			// read large function header 7 uint32 x 3 bytes x 1 byte
			largeHeaderData := make([]byte, 32)
			_, err = io.ReadFull(h.FileBuffer, largeHeaderData)
			if err != nil {
				return fmt.Errorf("failed to read large function header: %v", err)
			}

			largeHeader := &LargeFunctionHeader{}

			// read full uint32 values
			largeHeader.Offset = binary.LittleEndian.Uint32(largeHeaderData[0:4])
			largeHeader.ParamCount = binary.LittleEndian.Uint32(largeHeaderData[4:8])
			largeHeader.BytecodeSizeInBytes = binary.LittleEndian.Uint32(largeHeaderData[8:12])
			largeHeader.FunctionName = binary.LittleEndian.Uint32(largeHeaderData[12:16])
			largeHeader.InfoOffset = binary.LittleEndian.Uint32(largeHeaderData[16:20])
			largeHeader.FrameSize = binary.LittleEndian.Uint32(largeHeaderData[20:24])
			largeHeader.EnvironmentSize = binary.LittleEndian.Uint32(largeHeaderData[24:28])

			// Highest{Read,Write}CacheIndex at 1 byte each
			largeHeader.HighestReadCacheIndex = largeHeaderData[28]
			largeHeader.HighestWriteCacheIndex = largeHeaderData[29]

			// flags byte completes the rest...
			flagsByte := largeHeaderData[30]
			largeHeader.ProhibitInvoke = flagsByte & 0x3                    // 2 bits
			largeHeader.StrictMode = ((flagsByte >> 2) & 0x1) == 1          // bool bit
			largeHeader.HasExceptionHandler = ((flagsByte >> 3) & 0x1) == 1 // bool bit
			largeHeader.HasDebugInfo = ((flagsByte >> 4) & 0x1) == 1        // bool bit
			largeHeader.Overflowed = ((flagsByte >> 5) & 0x1) == 1          // bool bit
			largeHeader.Unused = (flagsByte >> 6) & 0x3                     // 2 bits

			functionHeader = largeHeader
		} else {
			// seek to InfoOffset for excp handler and debug info
			_, err := h.FileBuffer.Seek(int64(smallHeader.InfoOffset), io.SeekStart)
			if err != nil {
				return fmt.Errorf("failed to seek to info offset: %v", err)
			}
		}

		h.FunctionHeaders = append(h.FunctionHeaders, functionHeader)

		// check if function has excp handlers
		hasExcpHandlers := false
		switch header := functionHeader.(type) {
		case *SmallFunctionHeader:
			hasExcpHandlers = header.HasExceptionHandler
		case *LargeFunctionHeader:
			hasExcpHandlers = header.HasExceptionHandler
		}

		// read excp handlers if available
		if hasExcpHandlers {
			h.alignOverPadding(4)

			// read count of excp handlers
			countData := make([]byte, 4)
			_, err := io.ReadFull(h.FileBuffer, countData)
			if err != nil {
				return fmt.Errorf("failed to read exception handler count: %v", err)
			}
			excpHeadersCount := binary.LittleEndian.Uint32(countData)

			// read all excp handlers
			excpHandlers := make([]ExceptionHandlerInfo, excpHeadersCount)
			for i := uint32(0); i < excpHeadersCount; i++ {
				handlerData := make([]byte, 12) // 3 x uint32 fields
				_, err := io.ReadFull(h.FileBuffer, handlerData)
				if err != nil {
					return fmt.Errorf("failed to read exception handler: %v", err)
				}

				excpHandlers[i] = ExceptionHandlerInfo{
					Start:  binary.LittleEndian.Uint32(handlerData[0:4]),
					End:    binary.LittleEndian.Uint32(handlerData[4:8]),
					Target: binary.LittleEndian.Uint32(handlerData[8:12]),
				}
			}

			h.FunctionIDToExcHandlers[int(functionCount)] = excpHandlers
		}

		// check if function has debug info
		hasDebugInfo := false
		switch header := functionHeader.(type) {
		case *SmallFunctionHeader:
			hasDebugInfo = header.HasDebugInfo
		case *LargeFunctionHeader:
			hasDebugInfo = header.HasDebugInfo
		}

		// read debug offsets if available
		if hasDebugInfo {
			h.alignOverPadding(4)

			// read debug offsets based on bcv
			debugOffsets := DebugOffsets{}

			// source locations
			sourceLocData := make([]byte, 4)
			_, err := io.ReadFull(h.FileBuffer, sourceLocData)
			if err != nil {
				return fmt.Errorf("failed to read source locations offset: %v", err)
			}
			debugOffsets.SourceLocations = binary.LittleEndian.Uint32(sourceLocData)

			// scope description
			scopeDescData := make([]byte, 4)
			_, err = io.ReadFull(h.FileBuffer, scopeDescData)
			if err != nil {
				return fmt.Errorf("failed to read scope description data offset: %v", err)
			}
			debugOffsets.ScopeDescData = binary.LittleEndian.Uint32(scopeDescData)

			// textified callees >= bcv91
			if h.Header.Version >= 91 {
				textifiedData := make([]byte, 4)
				_, err := io.ReadFull(h.FileBuffer, textifiedData)
				if err != nil {
					return fmt.Errorf("failed to read textified callees offset: %v", err)
				}
				debugOffsets.TextifiedCallees = binary.LittleEndian.Uint32(textifiedData)
			}

			h.FunctionIDToDebugOffsets[int(functionCount)] = debugOffsets
		}

		// return to position after original SMALL header
		_, err = h.FileBuffer.Seek(beforePos, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to restore file position: %v", err)
		}
	}

	return nil
}
