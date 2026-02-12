package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/DanielTso/pixshift/internal/codec"
	"github.com/DanielTso/pixshift/internal/pipeline"
	"github.com/fsnotify/fsnotify"
)

const debounceDelay = 500 * time.Millisecond

// Watcher monitors directories for new image files and converts them.
type Watcher struct {
	Pipeline     *pipeline.Pipeline
	OutputDir    string
	OutputFormat codec.Format
	Quality      int
	Recursive    bool
	Metadata     bool
	Verbose      bool
	OnConvert    func(pipeline.Result)
}

// Watch starts watching the given directories for new/modified image files.
// It blocks until the context is cancelled.
func (w *Watcher) Watch(ctx context.Context, dirs []string) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer fsw.Close()

	for _, dir := range dirs {
		if w.Recursive {
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return fsw.Add(path)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("watch %s: %w", dir, err)
			}
		} else {
			if err := fsw.Add(dir); err != nil {
				return fmt.Errorf("watch %s: %w", dir, err)
			}
		}
	}

	// Debounce map: track pending files to handle partial writes
	var mu sync.Mutex
	pending := make(map[string]*time.Timer)

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, t := range pending {
				t.Stop()
			}
			mu.Unlock()
			return nil

		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}

			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}

			path := event.Name
			ext := filepath.Ext(path)
			if !codec.IsSupportedExtension(ext) {
				continue
			}

			mu.Lock()
			if t, exists := pending[path]; exists {
				t.Stop()
			}
			pending[path] = time.AfterFunc(debounceDelay, func() {
				mu.Lock()
				delete(pending, path)
				mu.Unlock()
				w.processFile(path)
			})
			mu.Unlock()

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			if w.Verbose {
				fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
			}
		}
	}
}

func (w *Watcher) processFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		if w.OnConvert != nil {
			w.OnConvert(pipeline.Result{
				Job:   pipeline.Job{InputPath: path},
				Error: err,
			})
		}
		return
	}

	inputFormat, err := codec.DetectFormat(f, path)
	f.Close()
	if err != nil {
		if w.OnConvert != nil {
			w.OnConvert(pipeline.Result{
				Job:   pipeline.Job{InputPath: path},
				Error: err,
			})
		}
		return
	}

	// Don't convert if input matches output format
	if inputFormat == w.OutputFormat {
		return
	}

	outputPath := buildOutputPath(path, w.OutputDir, w.OutputFormat)

	job := pipeline.Job{
		InputPath:        path,
		OutputPath:       outputPath,
		InputFormat:      inputFormat,
		OutputFormat:     w.OutputFormat,
		Quality:          w.Quality,
		PreserveMetadata: w.Metadata,
	}

	err = w.Pipeline.Execute(job)
	if w.OnConvert != nil {
		w.OnConvert(pipeline.Result{Job: job, Error: err})
	}
}

func buildOutputPath(inputPath, outputDir string, format codec.Format) string {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)] + codec.DefaultExtension(format)

	if outputDir != "" {
		return filepath.Join(outputDir, name)
	}
	return filepath.Join(filepath.Dir(inputPath), name)
}
