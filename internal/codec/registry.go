package codec

import (
	"fmt"
	"strings"
)

// Registry holds all registered codecs.
type Registry struct {
	decoders map[Format]Decoder
	encoders map[Format]Encoder
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		decoders: make(map[Format]Decoder),
		encoders: make(map[Format]Encoder),
	}
}

// RegisterDecoder registers a decoder for a format.
func (r *Registry) RegisterDecoder(d Decoder) {
	r.decoders[d.Format()] = d
}

// RegisterEncoder registers an encoder for a format.
func (r *Registry) RegisterEncoder(e Encoder) {
	r.encoders[e.Format()] = e
}

// Decoder returns the decoder for a format.
func (r *Registry) Decoder(f Format) (Decoder, error) {
	d, ok := r.decoders[f]
	if !ok {
		return nil, fmt.Errorf("no decoder registered for format %q", f)
	}
	return d, nil
}

// Encoder returns the encoder for a format.
func (r *Registry) Encoder(f Format) (Encoder, error) {
	e, ok := r.encoders[f]
	if !ok {
		return nil, fmt.Errorf("no encoder registered for format %q", f)
	}
	return e, nil
}

// SupportedDecoders returns all formats with registered decoders.
func (r *Registry) SupportedDecoders() []Format {
	formats := make([]Format, 0, len(r.decoders))
	for f := range r.decoders {
		formats = append(formats, f)
	}
	return formats
}

// SupportedEncoders returns all formats with registered encoders.
func (r *Registry) SupportedEncoders() []Format {
	formats := make([]Format, 0, len(r.encoders))
	for f := range r.encoders {
		formats = append(formats, f)
	}
	return formats
}

// ParseFormat parses a user-provided format string (case-insensitive).
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "jpeg", "jpg":
		return JPEG, nil
	case "png":
		return PNG, nil
	case "gif":
		return GIF, nil
	case "webp":
		return WebP, nil
	case "tiff", "tif":
		return TIFF, nil
	case "bmp":
		return BMP, nil
	case "heic", "heif":
		return HEIC, nil
	case "avif":
		return AVIF, nil
	case "cr2":
		return CR2, nil
	case "nef":
		return NEF, nil
	case "dng":
		return DNG, nil
	default:
		return "", fmt.Errorf("unsupported format: %q", s)
	}
}

// DefaultRegistry creates a registry with all built-in codecs.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	registerJPEG(r)
	registerPNG(r)
	registerGIF(r)
	registerHEIC(r)
	registerWebP(r)
	registerTIFF(r)
	registerBMP(r)
	registerAVIF(r)
	registerRAW(r)
	return r
}
