package metadata

import (
	"bytes"
	"fmt"
	"os"

	"github.com/DanielTso/pixshift/internal/codec"
)

// Inject writes EXIF metadata into an output image file.
// Currently supports JPEG and TIFF outputs.
func Inject(outputPath string, format codec.Format, meta *Metadata) error {
	if !meta.HasEXIF() {
		return nil
	}

	switch format {
	case codec.JPEG:
		return injectIntoJPEG(outputPath, meta)
	case codec.TIFF:
		// TIFF files already contain EXIF in their IFD structure;
		// injection would require full TIFF rewrite. Skip for v1.
		return nil
	default:
		return fmt.Errorf("EXIF injection not supported for %s", format)
	}
}

// injectIntoJPEG inserts an APP1 EXIF segment into a JPEG file.
func injectIntoJPEG(path string, meta *Metadata) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read output JPEG: %w", err)
	}

	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return fmt.Errorf("output is not a valid JPEG")
	}

	// Strip any existing APP1 EXIF segment from the output
	cleaned := stripExistingExif(data)

	// Build new APP1 segment
	app1 := buildAPP1Segment(meta.EXIFRaw)

	// Insert APP1 right after SOI marker (FF D8)
	var buf bytes.Buffer
	buf.Write(cleaned[:2])  // SOI
	buf.Write(app1)         // New APP1 with EXIF
	buf.Write(cleaned[2:])  // Rest of JPEG

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// stripExistingExif removes any existing APP1 EXIF segments from JPEG data.
func stripExistingExif(data []byte) []byte {
	exifHeader := []byte("Exif\x00\x00")

	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return data
	}

	var result bytes.Buffer
	result.Write(data[:2]) // SOI

	i := 2
	for i < len(data)-3 {
		if data[i] != 0xFF {
			// Reached image data, copy rest
			result.Write(data[i:])
			break
		}
		marker := data[i+1]

		// SOS marker: copy everything from here onward
		if marker == 0xDA {
			result.Write(data[i:])
			break
		}

		segLen := int(data[i+2])<<8 | int(data[i+3])
		segEnd := i + 2 + segLen

		if segEnd > len(data) {
			result.Write(data[i:])
			break
		}

		// Skip APP1 EXIF segments, keep everything else
		if marker == 0xE1 && segLen >= 6 {
			segPayload := data[i+4 : segEnd]
			if bytes.HasPrefix(segPayload, exifHeader) {
				i = segEnd
				continue
			}
		}

		result.Write(data[i:segEnd])
		i = segEnd
	}

	return result.Bytes()
}

// buildAPP1Segment constructs an APP1 marker segment from raw EXIF data.
func buildAPP1Segment(exifRaw []byte) []byte {
	// APP1 format: FF E1 <length-2-bytes> <payload>
	// Length includes the 2 length bytes but not the marker bytes.
	length := len(exifRaw) + 2
	return append(
		[]byte{0xFF, 0xE1, byte(length >> 8), byte(length & 0xFF)},
		exifRaw...,
	)
}
