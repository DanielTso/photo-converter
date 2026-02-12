package metadata

import (
	"bytes"
	"fmt"
	"io"

	"github.com/DanielTso/pixshift/internal/codec"
)

// Extract reads EXIF metadata from a source image file.
func Extract(r io.ReadSeeker, format codec.Format) (*Metadata, error) {
	switch format {
	case codec.JPEG:
		return extractFromJPEG(r)
	case codec.HEIC:
		return extractFromHEIC(r)
	case codec.TIFF, codec.CR2, codec.NEF, codec.DNG:
		return extractFromTIFF(r)
	default:
		return nil, fmt.Errorf("EXIF extraction not supported for %s", format)
	}
}

// extractFromJPEG finds the APP1 EXIF segment in a JPEG file.
func extractFromJPEG(r io.ReadSeeker) (*Metadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	exifData := findJPEGExifSegment(data)
	if exifData == nil {
		return nil, fmt.Errorf("no EXIF data found in JPEG")
	}
	return &Metadata{EXIFRaw: exifData}, nil
}

// findJPEGExifSegment scans JPEG data for an APP1 marker containing EXIF data.
// Returns the full APP1 payload (including "Exif\0\0" prefix).
func findJPEGExifSegment(data []byte) []byte {
	exifHeader := []byte("Exif\x00\x00")

	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil
	}

	i := 2
	for i < len(data)-3 {
		if data[i] != 0xFF {
			break
		}
		marker := data[i+1]

		// APP1 marker
		if marker == 0xE1 {
			segLen := int(data[i+2])<<8 | int(data[i+3])
			segData := data[i+4 : i+2+segLen]
			if bytes.HasPrefix(segData, exifHeader) {
				return segData
			}
		}

		// SOS marker means we've reached image data
		if marker == 0xDA {
			break
		}

		// Skip to next marker
		if i+3 < len(data) {
			segLen := int(data[i+2])<<8 | int(data[i+3])
			i += 2 + segLen
		} else {
			break
		}
	}
	return nil
}

// extractFromHEIC extracts EXIF from a HEIC file.
// HEIC stores EXIF in an "Exif" box within the ISO BMFF container.
func extractFromHEIC(r io.ReadSeeker) (*Metadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	// Search for the "Exif" box in the ISOBMFF structure.
	// The Exif box contains: 4-byte offset + TIFF header (starting with "MM" or "II").
	exifMarker := []byte("Exif")
	idx := bytes.Index(data, exifMarker)
	if idx < 4 {
		return nil, fmt.Errorf("no EXIF box found in HEIC")
	}

	// Read the box size from 4 bytes before the "Exif" marker
	boxStart := idx - 4
	boxSize := int(data[boxStart])<<24 | int(data[boxStart+1])<<16 |
		int(data[boxStart+2])<<8 | int(data[boxStart+3])

	if boxSize <= 8 || boxStart+boxSize > len(data) {
		return nil, fmt.Errorf("invalid EXIF box size in HEIC")
	}

	// The EXIF payload starts after the box header (8 bytes) + 4-byte TIFF offset
	payloadStart := idx + 4 + 4 // "Exif" + 4-byte offset
	if payloadStart >= boxStart+boxSize {
		return nil, fmt.Errorf("EXIF payload too small in HEIC")
	}

	exifPayload := data[payloadStart : boxStart+boxSize]

	// Wrap as APP1 segment format: "Exif\0\0" + TIFF data
	wrapped := append([]byte("Exif\x00\x00"), exifPayload...)
	return &Metadata{EXIFRaw: wrapped}, nil
}

// extractFromTIFF extracts EXIF from TIFF-based files (TIFF, CR2, NEF, DNG).
// These formats store EXIF in IFD entries within the TIFF structure.
func extractFromTIFF(r io.ReadSeeker) (*Metadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	if len(data) < 8 {
		return nil, fmt.Errorf("file too small for TIFF")
	}

	// For TIFF-based files, the entire file header IS the EXIF structure.
	// We wrap it in APP1 format for consistent injection.
	wrapped := append([]byte("Exif\x00\x00"), data...)
	return &Metadata{EXIFRaw: wrapped}, nil
}
