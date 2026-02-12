package metadata

// Metadata holds raw EXIF data extracted from a source image.
type Metadata struct {
	// EXIFRaw contains the raw EXIF bytes (APP1 payload after "Exif\0\0" prefix).
	EXIFRaw []byte
}

// HasEXIF returns true if EXIF data is present.
func (m *Metadata) HasEXIF() bool {
	return m != nil && len(m.EXIFRaw) > 0
}
