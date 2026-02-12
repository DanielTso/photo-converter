package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/vegidio/heif-go"
)

const defaultQuality = 92

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	quality := defaultQuality
	outputDir := ""
	format := "jpg"
	jobs := runtime.NumCPU()
	var inputs []string

	// Parse arguments
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-q", "--quality":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			q, err := strconv.Atoi(args[i+1])
			if err != nil || q < 1 || q > 100 {
				fatal("quality must be an integer between 1 and 100")
			}
			quality = q
			i += 2
		case "-o", "--output":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			outputDir = args[i+1]
			i += 2
		case "-f", "--format":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			f := strings.ToLower(args[i+1])
			if f != "jpg" && f != "png" {
				fatal("format must be 'jpg' or 'png'")
			}
			format = f
			i += 2
		case "-j", "--jobs":
			if i+1 >= len(args) {
				fatal("missing value for %s", args[i])
			}
			j, err := strconv.Atoi(args[i+1])
			if err != nil || j < 1 {
				fatal("jobs must be a positive integer")
			}
			jobs = j
			i += 2
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		default:
			inputs = append(inputs, args[i])
			i++
		}
	}

	if len(inputs) == 0 {
		fatal("no input files or directories specified")
	}

	// Create output directory if specified
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fatal("cannot create output directory: %v", err)
		}
	}

	// Expand globs and collect all HEIC files
	var files []string
	for _, input := range inputs {
		// Try glob expansion first (needed on Windows where the shell doesn't expand wildcards)
		matches, err := filepath.Glob(input)
		if err != nil || len(matches) == 0 {
			// Not a glob pattern or no matches -- treat as literal path
			matches = []string{input}
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot access %s: %v\n", match, err)
				continue
			}
			if info.IsDir() {
				dirFiles, err := collectHeicFiles(match)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: error reading directory %s: %v\n", match, err)
					continue
				}
				files = append(files, dirFiles...)
			} else {
				files = append(files, match)
			}
		}
	}

	if len(files) == 0 {
		fatal("no HEIC files found")
	}

	// Convert files using worker pool
	total := len(files)
	var succeeded int64
	var failed int64
	var mu sync.Mutex

	// Limit jobs to number of files
	if jobs > total {
		jobs = total
	}

	// Create work channel
	type workItem struct {
		index int
		path  string
	}
	work := make(chan workItem, total)
	for idx, f := range files {
		work <- workItem{index: idx, path: f}
	}
	close(work)

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < jobs; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range work {
				outPath := buildOutputPath(item.path, outputDir, format)

				mu.Lock()
				fmt.Printf("[%d/%d] Converting: %s -> %s\n", item.index+1, total, item.path, outPath)
				mu.Unlock()

				if err := convertFile(item.path, outPath, quality, format); err != nil {
					mu.Lock()
					fmt.Fprintf(os.Stderr, "  error: %v\n", err)
					mu.Unlock()
					atomic.AddInt64(&failed, 1)
				} else {
					atomic.AddInt64(&succeeded, 1)
				}
			}
		}()
	}

	wg.Wait()

	fmt.Printf("\nDone. %d converted, %d failed.\n", succeeded, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func convertFile(inputPath, outputPath string, quality int, format string) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	img, err := heif.Decode(f)
	if err != nil {
		return fmt.Errorf("decode HEIC: %w", err)
	}

	return writeImage(img, outputPath, quality, format)
}

func writeImage(img image.Image, outputPath string, quality int, format string) error {
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	switch format {
	case "png":
		if err := png.Encode(out, img); err != nil {
			return fmt.Errorf("encode PNG: %w", err)
		}
	default:
		opts := &jpeg.Options{Quality: quality}
		if err := jpeg.Encode(out, img, opts); err != nil {
			return fmt.Errorf("encode JPEG: %w", err)
		}
	}
	return nil
}

func collectHeicFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() && isHeicFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isHeicFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".heic" || ext == ".heif"
}

func buildOutputPath(inputPath, outputDir, format string) string {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext) + "." + format

	if outputDir != "" {
		return filepath.Join(outputDir, name)
	}
	return filepath.Join(filepath.Dir(inputPath), name)
}

func printUsage() {
	fmt.Println(`heic2jpg - Convert HEIC/HEIF images to JPEG or PNG

Usage:
  heic2jpg [options] <files or directories...>

Options:
  -q, --quality <1-100>   JPEG quality (default: 92, ignored for PNG)
  -f, --format <jpg|png>  Output format (default: jpg)
  -j, --jobs <N>          Number of parallel workers (default: number of CPUs)
  -o, --output <dir>      Output directory (default: same as input)
  -h, --help              Show this help

Note: EXIF metadata is not preserved during conversion. The underlying
HEIF decoding library does not expose EXIF data extraction.

Examples:
  heic2jpg photo.heic
  heic2jpg -q 85 *.heic
  heic2jpg -f png photos/ -o converted/
  heic2jpg -j 4 -q 90 -o output/ photo1.heic photo2.heic`)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
