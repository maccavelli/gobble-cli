package builtin

// EXIF orientation, read as Pi reads it in exif-orientation.ts: a JPEG APP1
// segment or a WebP EXIF chunk, TIFF tag 0x0112. A missing or out-of-range
// value is 1. No extra module: the records name golang.org/x/image only.

func exifOrientation(b []byte) int {
	off := tiffOffset(b)
	if off < 0 {
		return 1
	}
	return orientationAt(b, off)
}

func tiffOffset(b []byte) int {
	switch {
	case len(b) >= 2 && b[0] == 0xff && b[1] == 0xd8:
		return jpegTIFF(b)
	case len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return webpTIFF(b)
	default:
		return -1
	}
}

func jpegTIFF(b []byte) int {
	offset := 2
	for offset < len(b)-1 {
		if b[offset] != 0xff {
			return -1
		}
		marker := b[offset+1]
		if marker == 0xff {
			offset++
			continue
		}
		if marker == 0xe1 {
			if offset+4 >= len(b) {
				return -1
			}
			start := offset + 4
			if start+6 <= len(b) && hasExif(b, start) {
				return start + 6
			}
			return -1
		}
		if offset+4 > len(b) {
			return -1
		}
		length := int(b[offset+2])<<8 | int(b[offset+3])
		if length < 2 {
			return -1
		}
		offset += 2 + length
	}
	return -1
}

func webpTIFF(b []byte) int {
	offset := 12
	for offset+8 <= len(b) {
		id := string(b[offset : offset+4])
		size := int(b[offset+4]) | int(b[offset+5])<<8 | int(b[offset+6])<<16 | int(b[offset+7])<<24
		start := offset + 8
		if size < 0 || start+size > len(b) {
			return -1
		}
		if id == "EXIF" {
			if size >= 6 && hasExif(b, start) {
				return start + 6
			}
			return start
		}
		offset = start + size + size%2
	}
	return -1
}

func hasExif(b []byte, i int) bool {
	return i+6 <= len(b) && string(b[i:i+6]) == "Exif\x00\x00"
}

func orientationAt(b []byte, tiff int) int {
	if tiff+8 > len(b) {
		return 1
	}
	le := b[tiff] == 'I' && b[tiff+1] == 'I'
	if !le && (b[tiff] != 'M' || b[tiff+1] != 'M') {
		return 1
	}
	u16 := func(p int) int {
		if p+2 > len(b) {
			return 0
		}
		if le {
			return int(b[p]) | int(b[p+1])<<8
		}
		return int(b[p])<<8 | int(b[p+1])
	}
	u32 := func(p int) int {
		if p+4 > len(b) {
			return 0
		}
		if le {
			return int(b[p]) | int(b[p+1])<<8 | int(b[p+2])<<16 | int(b[p+3])<<24
		}
		return int(b[p])<<24 | int(b[p+1])<<16 | int(b[p+2])<<8 | int(b[p+3])
	}
	ifd := tiff + u32(tiff+4)
	if ifd < 0 || ifd+2 > len(b) {
		return 1
	}
	n := u16(ifd)
	for i := range n {
		entry := ifd + 2 + i*12
		if entry+12 > len(b) {
			return 1
		}
		if u16(entry) == 0x0112 {
			v := u16(entry + 8)
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}
