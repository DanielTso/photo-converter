package codec

import (
	"image"
	"io"
	"strings"
)

// Format represents a supported image format.
type Format string

const (
	JPEG Format = "jpeg"
	PNG  Format = "png"
	GIF  Format = "gif"
	WebP Format = "webp"
	TIFF Format = "tiff"
	BMP  Format = "bmp"
	HEIC Format = "heic"
	AVIF Format = "avif"
	CR2  Format = "cr2"
	NEF  Format = "nef"
	DNG  Format = "dng"
)

// Decoder can decode an image from a reader.
type Decoder interface {
	Decode(r io.ReadSeeker) (image.Image, error)
	Format() Format
}

// Encoder can encode an image to a writer.
type Encoder interface {
	Encode(w io.Writer, img image.Image, quality int) error
	Format() Format
}

// IsRAW returns true if the format is a RAW camera format.
func IsRAW(f Format) bool {
	switch f {
	case CR2, NEF, DNG:
		return true
	}
	return false
}

// DefaultExtension returns the primary file extension for a format.
func DefaultExtension(f Format) string {
	switch f {
	case JPEG:
		return ".jpg"
	case PNG:
		return ".png"
	case GIF:
		return ".gif"
	case WebP:
		return ".webp"
	case TIFF:
		return ".tiff"
	case BMP:
		return ".bmp"
	case HEIC:
		return ".heic"
	case AVIF:
		return ".avif"
	case CR2:
		return ".cr2"
	case NEF:
		return ".nef"
	case DNG:
		return ".dng"
	}
	return "." + string(f)
}

// IsSupportedExtension checks if a file extension belongs to any known format.
func IsSupportedExtension(ext string) bool {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".tiff", ".tif",
		".bmp", ".heic", ".heif", ".avif", ".cr2", ".nef", ".dng":
		return true
	}
	return false
}
