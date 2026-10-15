package hbc

import (
	"encoding/binary"
	"fmt"
	"io"
)

// HBC 98 (Hermes 250829098, RN 0.85) function header layout (BytecodeFileFormat.h):
// 12-byte SmallFuncHeader:
//
//	w1: Offset 25 | ParamCount 5 | LoopDepth 2
//	w2: BytecodeSizeInBytes 14 | FunctionName 8 | NumberRegCount 5 | NonPtrRegCount 5
//	b8 FrameSize, b9 ReadCacheSize, b10 WriteCacheSize 6 | NumCacheNewObject 1 | PrivateNameCacheSize 1, b11 flags
//	flags: ProhibitInvoke 2 | StrictMode 1 | HasExceptionHandler 1 | HasDebugInfo 1 | Overflowed 1 | Kind 2
//
// Overflowed: large header at (FunctionName<<24)|Offset, 37 packed bytes (8 x uint32, 4 x uint8, flags),
// followed by the (4-aligned) exception table and debug offsets. Small headers carry neither.
const smallFuncHeaderSizeV98 = 12
const largeFuncHeaderSizeV98 = 37

func decodeFlagsV98(f uint8) (prohibit uint8, strict, exc, dbg, over bool, kind uint8) {
	return f & 0x3, (f>>2)&1 == 1, (f>>3)&1 == 1, (f>>4)&1 == 1, (f>>5)&1 == 1, (f >> 6) & 0x3
}

// decodeSmallHeaderV98 decodes one 12-byte bcv98 small function header.
func decodeSmallHeaderV98(b []byte) *SmallFunctionHeader {
	w1 := binary.LittleEndian.Uint32(b[0:4])
	w2 := binary.LittleEndian.Uint32(b[4:8])
	sh := &SmallFunctionHeader{
		Offset:                 w1 & 0x1FFFFFF,
		ParamCount:             uint8((w1 >> 25) & 0x1F),
		BytecodeSizeInBytes:    uint16(w2 & 0x3FFF),
		FunctionName:           (w2 >> 14) & 0xFF,
		FrameSize:              b[8],
		HighestReadCacheIndex:  b[9],
		HighestWriteCacheIndex: b[10] & 0x3F,
	}
	sh.ProhibitInvoke, sh.StrictMode, sh.HasExceptionHandler, sh.HasDebugInfo, sh.Overflowed, sh.Unused = decodeFlagsV98(b[11])
	return sh
}

// decodeLargeHeaderV98 decodes the 37-byte bcv98 FunctionHeader.
func decodeLargeHeaderV98(b []byte) *LargeFunctionHeader {
	u := func(i int) uint32 { return binary.LittleEndian.Uint32(b[4*i : 4*i+4]) }
	// u(2) LoopDepth, u(5) NumberRegCount, u(6) NonPtrRegCount are not used by the disassembler.
	lh := &LargeFunctionHeader{
		Offset: u(0), ParamCount: u(1), BytecodeSizeInBytes: u(3), FunctionName: u(4), FrameSize: u(7),
		HighestReadCacheIndex: b[32], HighestWriteCacheIndex: b[33],
	}
	lh.ProhibitInvoke, lh.StrictMode, lh.HasExceptionHandler, lh.HasDebugInfo, lh.Overflowed, lh.Unused = decodeFlagsV98(b[36])
	return lh
}

func (h *HBCReader) readFunctionsV98() error {
	h.FunctionHeaders = make([]any, 0, h.Header.FunctionCount)
	h.FunctionIDToExcHandlers = make(map[int][]ExceptionHandlerInfo)
	h.FunctionIDToDebugOffsets = make(map[int]DebugOffsets)
	h.alignOverPadding(4)
	for i := 0; i < int(h.Header.FunctionCount); i++ {
		b := make([]byte, smallFuncHeaderSizeV98)
		if _, err := io.ReadFull(h.FileBuffer, b); err != nil {
			return fmt.Errorf("bcv98: read small header %d: %v", i, err)
		}
		sh := decodeSmallHeaderV98(b)
		if !sh.Overflowed {
			h.FunctionHeaders = append(h.FunctionHeaders, sh)
			continue
		}
		back, _ := h.FileBuffer.Seek(0, io.SeekCurrent)
		off := int64(sh.FunctionName)<<24 | int64(sh.Offset)
		if _, err := h.FileBuffer.Seek(off, io.SeekStart); err != nil {
			return fmt.Errorf("bcv98: seek large header %d: %v", i, err)
		}
		lb := make([]byte, largeFuncHeaderSizeV98)
		if _, err := io.ReadFull(h.FileBuffer, lb); err != nil {
			return fmt.Errorf("bcv98: read large header %d: %v", i, err)
		}
		lh := decodeLargeHeaderV98(lb)
		h.FunctionHeaders = append(h.FunctionHeaders, lh)
		if lh.HasExceptionHandler {
			h.alignOverPadding(4)
			var n uint32
			if err := binary.Read(h.FileBuffer, binary.LittleEndian, &n); err != nil {
				return err
			}
			hs := make([]ExceptionHandlerInfo, n)
			for j := range hs {
				var t [3]uint32
				if err := binary.Read(h.FileBuffer, binary.LittleEndian, &t); err != nil {
					return err
				}
				hs[j] = ExceptionHandlerInfo{Start: t[0], End: t[1], Target: t[2]}
			}
			h.FunctionIDToExcHandlers[i] = hs
		}
		if lh.HasDebugInfo {
			h.alignOverPadding(4)
			var t [3]uint32
			if err := binary.Read(h.FileBuffer, binary.LittleEndian, &t); err != nil {
				return err
			}
			h.FunctionIDToDebugOffsets[i] = DebugOffsets{SourceLocations: t[0], ScopeDescData: t[1], TextifiedCallees: t[2]}
		}
		if _, err := h.FileBuffer.Seek(back, io.SeekStart); err != nil {
			return err
		}
	}
	return nil
}
