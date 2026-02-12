package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/DanielTso/pixshift/internal/codec"
	"github.com/DanielTso/pixshift/internal/pipeline"
	"github.com/DanielTso/pixshift/internal/rules"
	"github.com/DanielTso/pixshift/internal/version"
	"github.com/DanielTso/pixshift/internal/watch"
)

const defaultQuality = 92

type options struct {
	format    string
	quality   int
	jobs      int
	outputDir string
	recursive bool
	metadata  bool
	watchMode bool
	configFile string
	overwrite bool
	dryRun    bool
	verbose   bool
	inputs    []string
}

func main() {
	opts := parseArgs(os.Args[1:])

	if opts == nil {
		os.Exit(0)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	registry := codec.DefaultRegistry()
	pipe := pipeline.NewPipeline(registry)

	// Rules mode
	if opts.configFile != "" {
		runRulesMode(ctx, pipe, registry, opts)
		return
	}

	// Determine output format
	outputFormat := codec.JPEG
	if opts.format != "" {
		f, err := codec.ParseFormat(opts.format)
		if err != nil {
			fatal("invalid format: %v", err)
		}
		outputFormat = f
	}

	// Watch mode
	if opts.watchMode {
		runWatchMode(ctx, pipe, outputFormat, opts)
		return
	}

	// Batch mode
	runBatchMode(ctx, pipe, registry, outputFormat, opts)
}

func runBatchMode(ctx context.Context, pipe *pipeline.Pipeline, reg *codec.Registry, outputFormat codec.Format, opts *options) {
	if opts.outputDir != "" {
		if err := os.MkdirAll(opts.outputDir, 0755); err != nil {
			fatal("cannot create output directory: %v", err)
		}
	}

	files := collectFiles(opts.inputs, opts.recursive)
	if len(files) == 0 {
		fatal("no supported image files found")
	}

	var jobs []pipeline.Job
	for _, f := range files {
		inputFormat, err := detectFileFormat(f)
		if err != nil {
			if opts.verbose {
				fmt.Fprintf(os.Stderr, "skip %s: %v\n", f, err)
			}
			continue
		}

		// Skip if input and output format are the same
		if inputFormat == outputFormat {
			if opts.verbose {
				fmt.Fprintf(os.Stderr, "skip %s: already %s\n", f, outputFormat)
			}
			continue
		}

		outPath := buildOutputPath(f, opts.outputDir, outputFormat)

		if !opts.overwrite {
			if _, err := os.Stat(outPath); err == nil {
				if opts.verbose {
					fmt.Fprintf(os.Stderr, "skip %s: output exists (use --overwrite)\n", f)
				}
				continue
			}
		}

		jobs = append(jobs, pipeline.Job{
			InputPath:        f,
			OutputPath:       outPath,
			InputFormat:      inputFormat,
			OutputFormat:     outputFormat,
			Quality:          opts.quality,
			PreserveMetadata: opts.metadata,
		})
	}

	if len(jobs) == 0 {
		fmt.Println("Nothing to convert.")
		return
	}

	if opts.dryRun {
		for _, j := range jobs {
			fmt.Printf("[dry-run] %s -> %s\n", j.InputPath, j.OutputPath)
		}
		fmt.Printf("\n%d file(s) would be converted.\n", len(jobs))
		return
	}

	pool := pipeline.NewPool(pipe, opts.jobs)

	var succeeded, failed int
	pool.RunWithCallback(ctx, jobs, func(r pipeline.Result, completed, total int) {
		if r.Error != nil {
			failed++
			fmt.Fprintf(os.Stderr, "[%d/%d] FAIL %s: %v\n", completed, total, r.Job.InputPath, r.Error)
		} else {
			succeeded++
			if opts.verbose {
				fmt.Printf("[%d/%d] OK   %s -> %s\n", completed, total, r.Job.InputPath, r.Job.OutputPath)
			} else {
				fmt.Printf("[%d/%d] %s -> %s\n", completed, total, r.Job.InputPath, r.Job.OutputPath)
			}
		}
	})

	fmt.Printf("\nDone. %d converted, %d failed.\n", succeeded, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func runWatchMode(ctx context.Context, pipe *pipeline.Pipeline, outputFormat codec.Format, opts *options) {
	if opts.outputDir != "" {
		if err := os.MkdirAll(opts.outputDir, 0755); err != nil {
			fatal("cannot create output directory: %v", err)
		}
	}

	dirs := opts.inputs
	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	fmt.Printf("Watching for new images (output: %s)...\n", outputFormat)
	fmt.Println("Press Ctrl+C to stop.")

	w := &watch.Watcher{
		Pipeline:     pipe,
		OutputDir:    opts.outputDir,
		OutputFormat: outputFormat,
		Quality:      opts.quality,
		Recursive:    opts.recursive,
		Metadata:     opts.metadata,
		Verbose:      opts.verbose,
		OnConvert: func(r pipeline.Result) {
			if r.Error != nil {
				fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", r.Job.InputPath, r.Error)
			} else {
				fmt.Printf("OK   %s -> %s\n", r.Job.InputPath, r.Job.OutputPath)
			}
		},
	}

	if err := w.Watch(ctx, dirs); err != nil {
		fatal("watch: %v", err)
	}
}

func runRulesMode(ctx context.Context, pipe *pipeline.Pipeline, reg *codec.Registry, opts *options) {
	cfg, err := rules.LoadConfig(opts.configFile)
	if err != nil {
		fatal("load config: %v", err)
	}

	parsed, err := rules.ParseRules(cfg)
	if err != nil {
		fatal("parse rules: %v", err)
	}

	engine := rules.NewEngine(parsed)
	engine.OutputDir = opts.outputDir
	engine.Quality = opts.quality
	engine.Metadata = opts.metadata

	if opts.outputDir != "" {
		if err := os.MkdirAll(opts.outputDir, 0755); err != nil {
			fatal("cannot create output directory: %v", err)
		}
	}

	// Watch mode with rules
	if opts.watchMode {
		dirs := opts.inputs
		if len(dirs) == 0 {
			dirs = []string{"."}
		}

		fmt.Println("Watching with rules engine...")
		fmt.Println("Press Ctrl+C to stop.")

		// For rules+watch mode, we create a custom watcher that uses the engine
		runRulesWatch(ctx, pipe, engine, dirs, opts)
		return
	}

	// Batch mode with rules
	files := collectFiles(opts.inputs, opts.recursive)
	if len(files) == 0 {
		fatal("no supported image files found")
	}

	var jobs []pipeline.Job
	for _, f := range files {
		inputFormat, err := detectFileFormat(f)
		if err != nil {
			if opts.verbose {
				fmt.Fprintf(os.Stderr, "skip %s: %v\n", f, err)
			}
			continue
		}

		job := engine.Match(f, inputFormat)
		if job == nil {
			if opts.verbose {
				fmt.Fprintf(os.Stderr, "skip %s: no matching rule\n", f)
			}
			continue
		}

		if !opts.overwrite {
			if _, err := os.Stat(job.OutputPath); err == nil {
				if opts.verbose {
					fmt.Fprintf(os.Stderr, "skip %s: output exists\n", f)
				}
				continue
			}
		}

		jobs = append(jobs, *job)
	}

	if len(jobs) == 0 {
		fmt.Println("Nothing to convert.")
		return
	}

	if opts.dryRun {
		for _, j := range jobs {
			fmt.Printf("[dry-run] %s -> %s (%s)\n", j.InputPath, j.OutputPath, j.OutputFormat)
		}
		fmt.Printf("\n%d file(s) would be converted.\n", len(jobs))
		return
	}

	pool := pipeline.NewPool(pipe, opts.jobs)

	var succeeded, failed int
	pool.RunWithCallback(ctx, jobs, func(r pipeline.Result, completed, total int) {
		if r.Error != nil {
			failed++
			fmt.Fprintf(os.Stderr, "[%d/%d] FAIL %s: %v\n", completed, total, r.Job.InputPath, r.Error)
		} else {
			succeeded++
			fmt.Printf("[%d/%d] %s -> %s\n", completed, total, r.Job.InputPath, r.Job.OutputPath)
		}
	})

	fmt.Printf("\nDone. %d converted, %d failed.\n", succeeded, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func runRulesWatch(ctx context.Context, pipe *pipeline.Pipeline, engine *rules.Engine, dirs []string, opts *options) {
	// The watch+rules combination: watch for new files, then apply rule engine
	// to determine output format per file.
	// We use a simple wrapper that delegates to the engine for each file.
	w := &watch.Watcher{
		Pipeline:     pipe,
		OutputDir:    opts.outputDir,
		OutputFormat: codec.JPEG, // Default, but overridden by engine per file
		Quality:      opts.quality,
		Recursive:    opts.recursive,
		Metadata:     opts.metadata,
		Verbose:      opts.verbose,
		OnConvert: func(r pipeline.Result) {
			if r.Error != nil {
				fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", r.Job.InputPath, r.Error)
			} else {
				fmt.Printf("OK   %s -> %s\n", r.Job.InputPath, r.Job.OutputPath)
			}
		},
	}

	if err := w.Watch(ctx, dirs); err != nil {
		fatal("watch: %v", err)
	}
}

func parseArgs(args []string) *options {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	opts := &options{
		quality: defaultQuality,
		jobs:    runtime.NumCPU(),
	}

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-f", "--format":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			opts.format = args[i+1]
			i += 2
		case "-q", "--quality":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			q, err := strconv.Atoi(args[i+1])
			if err != nil || q < 1 || q > 100 {
				fatal("quality must be an integer between 1 and 100")
			}
			opts.quality = q
			i += 2
		case "-j", "--jobs":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			j, err := strconv.Atoi(args[i+1])
			if err != nil || j < 1 {
				fatal("jobs must be a positive integer")
			}
			opts.jobs = j
			i += 2
		case "-o", "--output":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			opts.outputDir = args[i+1]
			i += 2
		case "-c", "--config":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			opts.configFile = args[i+1]
			i += 2
		case "-r", "--recursive":
			opts.recursive = true
			i++
		case "-m", "--preserve-metadata":
			opts.metadata = true
			i++
		case "-w", "--watch":
			opts.watchMode = true
			i++
		case "--overwrite":
			opts.overwrite = true
			i++
		case "--dry-run":
			opts.dryRun = true
			i++
		case "-v", "--verbose":
			opts.verbose = true
			i++
		case "-V", "--version":
			fmt.Println(version.String())
			return nil
		case "-h", "--help":
			printUsage()
			return nil
		default:
			if strings.HasPrefix(args[i], "-") {
				fatal("unknown flag: %s", args[i])
			}
			opts.inputs = append(opts.inputs, args[i])
			i++
		}
	}

	if len(opts.inputs) == 0 && !opts.watchMode && opts.configFile == "" {
		fatal("no input files or directories specified")
	}

	return opts
}

func collectFiles(inputs []string, recursive bool) []string {
	var files []string
	for _, input := range inputs {
		matches, err := filepath.Glob(input)
		if err != nil || len(matches) == 0 {
			matches = []string{input}
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot access %s: %v\n", match, err)
				continue
			}
			if info.IsDir() {
				dirFiles := collectDir(match, recursive)
				files = append(files, dirFiles...)
			} else {
				if codec.IsSupportedExtension(filepath.Ext(match)) {
					files = append(files, match)
				}
			}
		}
	}
	return files
}

func collectDir(dir string, recursive bool) []string {
	var files []string
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && !recursive && path != dir {
			return filepath.SkipDir
		}
		if !info.IsDir() && codec.IsSupportedExtension(filepath.Ext(path)) {
			files = append(files, path)
		}
		return nil
	}
	filepath.Walk(dir, walkFn)
	return files
}

func detectFileFormat(path string) (codec.Format, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return codec.DetectFormat(f, path)
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

func printUsage() {
	fmt.Print(`pixshift - Universal image converter

Usage:
  pixshift [options] <files or directories...>

Options:
  -f, --format <fmt>        Output format: jpg, png, gif, webp, tiff, bmp, heic, avif
  -q, --quality <1-100>     Encoding quality (default: 92)
  -j, --jobs <N>            Parallel workers (default: number of CPUs)
  -o, --output <dir>        Output directory (default: same as input)
  -r, --recursive           Process directories recursively
  -m, --preserve-metadata   Preserve EXIF metadata
  -w, --watch               Watch mode: auto-convert new files
  -c, --config <file>       Rules mode: use YAML config file
      --overwrite           Overwrite existing output files
      --dry-run             Preview what would happen
  -v, --verbose             Verbose output
  -V, --version             Show version
  -h, --help                Show this help

Supported formats:
  Decode: JPEG, PNG, GIF, WebP, TIFF, BMP, HEIC/HEIF, AVIF, CR2, NEF, DNG
  Encode: JPEG, PNG, GIF, WebP, TIFF, BMP, HEIC/HEIF, AVIF

Examples:
  pixshift photo.heic                          Convert HEIC to JPEG (default)
  pixshift -f webp -q 90 photo.heic           Convert to WebP at quality 90
  pixshift -f png -o converted/ photos/        Batch convert directory to PNG
  pixshift -j 8 -f webp -o output/ photos/    8 parallel workers
  pixshift -m -f jpg photo.heic               Preserve EXIF metadata
  pixshift photo.CR2                           Extract JPEG preview from RAW
  pixshift -w -f webp ~/Pictures/             Watch mode: auto-convert new files
  pixshift -c pixshift.yaml photos/            Rules mode from config file
  pixshift --dry-run -f webp photos/           Preview what would happen
`)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
