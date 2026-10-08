package builtin

import (
	"bytes"
	"encoding/binary"
	"slices"
)

// sniffBytes is how much of a file decides what it is.
const sniffBytes = 4096

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// imageType is the MIME type of an image read can name, by its magic bytes
// (the signatures Pi's and opencode's read tools check), or "" for anything
// else.
func imageType(b []byte) string {
	b = b[:min(len(b), sniffBytes)]
	switch {
	case bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}):
		return "image/jpeg"
	case bytes.HasPrefix(b, pngSignature) && isPNG(b):
		return "image/png"
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")):
		return "image/gif"
	case bytes.HasPrefix(b, []byte("RIFF")) && len(b) >= 12 && string(b[8:12]) == "WEBP":
		return "image/webp"
	case bytes.HasPrefix(b, []byte("BM")) && isBMP(b):
		return "image/bmp"
	}
	return ""
}

func isPNG(b []byte) bool {
	return len(b) >= 16 && binary.BigEndian.Uint32(b[8:]) == 13 && string(b[12:16]) == "IHDR"
}

// isBMP checks a BMP header's sizes, offsets, planes and depth, so text that
// happens to start "BM" is not an image.
func isBMP(b []byte) bool {
	if len(b) < 30 {
		return false
	}
	fileSize := binary.LittleEndian.Uint32(b[2:])
	pixelOffset := binary.LittleEndian.Uint32(b[10:])
	dibSize := binary.LittleEndian.Uint32(b[14:])
	if fileSize != 0 && (fileSize < 26 || pixelOffset >= fileSize) {
		return false
	}
	if uint64(pixelOffset) < 14+uint64(dibSize) {
		return false
	}
	var planes, bpp uint16
	switch {
	case dibSize == 12:
		planes, bpp = binary.LittleEndian.Uint16(b[22:]), binary.LittleEndian.Uint16(b[24:])
	case dibSize >= 40 && dibSize <= 124:
		planes, bpp = binary.LittleEndian.Uint16(b[26:]), binary.LittleEndian.Uint16(b[28:])
	default:
		return false
	}
	return planes == 1 && slices.Contains([]uint16{1, 4, 8, 16, 24, 32}, bpp)
}

// binaryContent reports a file read cannot show as text, as opencode's read
// decides it: a NUL byte, or more than 30% control bytes other than tab,
// newline, form feed and carriage return, in the first sniffBytes bytes. A
// UTF-16 or UTF-32 byte-order mark is text with NULs in it, and is binary
// here too, since read shows UTF-8.
func binaryContent(b []byte) bool {
	b = b[:min(len(b), sniffBytes)]
	if len(b) == 0 {
		return false
	}
	if bytes.IndexByte(b, 0) >= 0 {
		return true
	}
	control := 0
	for _, c := range b {
		if c < 9 || (c > 13 && c < 32) {
			control++
		}
	}
	return control*10 > len(b)*3
}
