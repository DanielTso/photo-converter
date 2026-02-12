package codec

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
)

// rawDecoder extracts the embedded JPEG preview from RAW camera files (CR2, NEF, DNG).
// This is what most quick converters do — it does not process raw sensor data.
type rawDecoder struct {
	format Format
}

func (d *rawDecoder) Decode(r io.ReadSeeker) (image.Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read raw file: %w", err)
	}

	jpegData := extractLargestJPEG(data)
	if jpegData == nil {
		return nil, fmt.Errorf("no embedded JPEG preview found in %s file", d.format)
	}

	img, err := jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return nil, fmt.Errorf("decode embedded JPEG preview: %w", err)
	}
	return img, nil
}

func (d *rawDecoder) Format() Format { return d.format }

// extractLargestJPEG scans binary data for embedded JPEG images (FF D8 ... FF D9)
// and returns the largest one, which is typically the full-size preview.
func extractLargestJPEG(data []byte) []byte {
	var best []byte

	for i := 0; i < len(data)-3; i++ {
		// Look for JPEG SOI marker: FF D8 FF
		if data[i] != 0xFF || data[i+1] != 0xD8 || data[i+2] != 0xFF {
			continue
		}

		// Scan forward for JPEG EOI marker: FF D9
		end := findJPEGEnd(data, i+2)
		if end < 0 {
			continue
		}

		segment := data[i : end+2]
		if len(segment) > len(best) {
			best = segment
		}
	}

	return best
}

// findJPEGEnd scans for the FF D9 (EOI) marker starting from offset.
// It tries to follow JPEG segment structure for accuracy.
func findJPEGEnd(data []byte, offset int) int {
	i := offset
	for i < len(data)-1 {
		if data[i] != 0xFF {
			i++
			continue
		}
		marker := data[i+1]

		// EOI marker found
		if marker == 0xD9 {
			return i + 1
		}

		// SOS marker: scan data follows until next marker
		if marker == 0xDA {
			i += 2
			// Skip SOS header
			if i+1 < len(data) {
				segLen := int(data[i])<<8 | int(data[i+1])
				i += segLen
			}
			// Scan entropy-coded data for next marker
			for i < len(data)-1 {
				if data[i] == 0xFF && data[i+1] != 0x00 && data[i+1] != 0xFF {
					break
				}
				i++
			}
			continue
		}

		// Skip padding bytes (FF FF)
		if marker == 0xFF || marker == 0x00 {
			i++
			continue
		}

		// Regular marker with length field
		if i+3 < len(data) {
			segLen := int(data[i+2])<<8 | int(data[i+3])
			i += 2 + segLen
		} else {
			break
		}
	}
	return -1
}

func registerRAW(r *Registry) {
	// Register decoder for each RAW format. RAW formats are decode-only.
	r.RegisterDecoder(&rawDecoder{format: CR2})
	r.RegisterDecoder(&rawDecoder{format: NEF})
	r.RegisterDecoder(&rawDecoder{format: DNG})
}
