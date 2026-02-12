package codec

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// DetectFormat reads the first bytes of a file to identify its format by magic bytes.
// Falls back to file extension if magic bytes are inconclusive.
func DetectFormat(r io.ReadSeeker, filename string) (Format, error) {
	buf := make([]byte, 32)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read header: %w", err)
	}
	buf = buf[:n]

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek: %w", err)
	}

	if f, ok := detectByMagic(buf, filename); ok {
		return f, nil
	}

	if f, ok := detectByExtension(filename); ok {
		return f, nil
	}

	return "", fmt.Errorf("unable to detect format for %q", filename)
}

func detectByMagic(buf []byte, filename string) (Format, bool) {
	if len(buf) < 4 {
		return "", false
	}

	// JPEG: FF D8 FF
	if buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
		return JPEG, true
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
		return PNG, true
	}

	// GIF: GIF87a or GIF89a
	if buf[0] == 'G' && buf[1] == 'I' && buf[2] == 'F' && buf[3] == '8' {
		return GIF, true
	}

	// WebP: RIFF....WEBP
	if len(buf) >= 12 &&
		buf[0] == 'R' && buf[1] == 'I' && buf[2] == 'F' && buf[3] == 'F' &&
		buf[8] == 'W' && buf[9] == 'E' && buf[10] == 'B' && buf[11] == 'P' {
		return WebP, true
	}

	// BMP: BM
	if buf[0] == 'B' && buf[1] == 'M' {
		return BMP, true
	}

	// TIFF-based formats: II (little-endian) or MM (big-endian)
	if (buf[0] == 'I' && buf[1] == 'I') || (buf[0] == 'M' && buf[1] == 'M') {
		return detectTIFFVariant(buf, filename)
	}

	// HEIC/AVIF: ISO BMFF with ftyp box
	if len(buf) >= 12 && buf[4] == 'f' && buf[5] == 't' && buf[6] == 'y' && buf[7] == 'p' {
		brand := string(buf[8:12])
		switch {
		case brand == "heic" || brand == "heix" || brand == "mif1":
			return HEIC, true
		case brand == "avif" || brand == "avis":
			return AVIF, true
		}
	}

	return "", false
}

func detectTIFFVariant(buf []byte, filename string) (Format, bool) {
	// CR2: TIFF header with "CR" signature at offset 8-9
	if len(buf) >= 10 && buf[8] == 'C' && buf[9] == 'R' {
		return CR2, true
	}

	// NEF and DNG have TIFF-like headers; use extension as tiebreaker.
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".nef":
		return NEF, true
	case ".dng":
		return DNG, true
	case ".cr2":
		return CR2, true
	}

	return TIFF, true
}

func detectByExtension(filename string) (Format, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return JPEG, true
	case ".png":
		return PNG, true
	case ".gif":
		return GIF, true
	case ".webp":
		return WebP, true
	case ".tiff", ".tif":
		return TIFF, true
	case ".bmp":
		return BMP, true
	case ".heic", ".heif":
		return HEIC, true
	case ".avif":
		return AVIF, true
	case ".cr2":
		return CR2, true
	case ".nef":
		return NEF, true
	case ".dng":
		return DNG, true
	}
	return "", false
}
