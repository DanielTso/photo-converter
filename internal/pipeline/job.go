package pipeline

import "github.com/DanielTso/pixshift/internal/codec"

// Job describes a single conversion task.
type Job struct {
	InputPath        string
	OutputPath       string
	InputFormat      codec.Format
	OutputFormat     codec.Format
	Quality          int
	PreserveMetadata bool
}

// Result holds the outcome of a conversion job.
type Result struct {
	Job   Job
	Error error
}
