package pipeline

import (
	"fmt"
	"os"

	"github.com/DanielTso/pixshift/internal/codec"
	"github.com/DanielTso/pixshift/internal/metadata"
)

// Pipeline executes the detect -> decode -> encode -> metadata inject flow.
type Pipeline struct {
	Registry *codec.Registry
}

// NewPipeline creates a pipeline with the given codec registry.
func NewPipeline(reg *codec.Registry) *Pipeline {
	return &Pipeline{Registry: reg}
}

// Execute runs a single conversion job.
func (p *Pipeline) Execute(job Job) error {
	// Open input file
	f, err := os.Open(job.InputPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", job.InputPath, err)
	}
	defer f.Close()

	// Detect input format if not set
	inputFormat := job.InputFormat
	if inputFormat == "" {
		inputFormat, err = codec.DetectFormat(f, job.InputPath)
		if err != nil {
			return fmt.Errorf("detect format: %w", err)
		}
	}

	// Extract metadata before decoding (if requested)
	var meta *metadata.Metadata
	if job.PreserveMetadata {
		meta, err = metadata.Extract(f, inputFormat)
		if err != nil {
			// Non-fatal: warn but continue without metadata
			meta = nil
		}
		// Reset file position after metadata extraction
		if _, err := f.Seek(0, 0); err != nil {
			return fmt.Errorf("seek: %w", err)
		}
	}

	// Get decoder
	dec, err := p.Registry.Decoder(inputFormat)
	if err != nil {
		return err
	}

	// Decode
	img, err := dec.Decode(f)
	if err != nil {
		return fmt.Errorf("decode %s: %w", inputFormat, err)
	}

	// Get encoder
	enc, err := p.Registry.Encoder(job.OutputFormat)
	if err != nil {
		return err
	}

	// Create output file
	out, err := os.Create(job.OutputPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", job.OutputPath, err)
	}
	defer out.Close()

	// Encode
	if err := enc.Encode(out, img, job.Quality); err != nil {
		os.Remove(job.OutputPath)
		return fmt.Errorf("encode %s: %w", job.OutputFormat, err)
	}

	// Close the file before metadata injection (needs to re-read/write)
	out.Close()

	// Inject metadata if available
	if meta.HasEXIF() {
		if err := metadata.Inject(job.OutputPath, job.OutputFormat, meta); err != nil {
			// Non-fatal: file was converted but metadata not preserved
			return fmt.Errorf("metadata inject (file converted OK): %w", err)
		}
	}

	return nil
}
