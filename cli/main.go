package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
			// Not a glob pattern or no matches — treat as literal path
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

	// Convert each file
	total := len(files)
	succeeded := 0
	failed := 0
	for i, f := range files {
		outPath := buildOutputPath(f, outputDir)
		fmt.Printf("[%d/%d] Converting: %s -> %s\n", i+1, total, f, outPath)
		if err := convertFile(f, outPath, quality); err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			failed++
		} else {
			succeeded++
		}
	}

	fmt.Printf("\nDone. %d converted, %d failed.\n", succeeded, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func convertFile(inputPath, outputPath string, quality int) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	img, err := heif.Decode(f)
	if err != nil {
		return fmt.Errorf("decode HEIC: %w", err)
	}

	return writeJpeg(img, outputPath, quality)
}

func writeJpeg(img image.Image, outputPath string, quality int) error {
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(out, img, opts); err != nil {
		return fmt.Errorf("encode JPEG: %w", err)
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

func buildOutputPath(inputPath, outputDir string) string {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext) + ".jpg"

	if outputDir != "" {
		return filepath.Join(outputDir, name)
	}
	return filepath.Join(filepath.Dir(inputPath), name)
}

func printUsage() {
	fmt.Println(`heic2jpg - Convert HEIC/HEIF images to JPEG

Usage:
  heic2jpg [options] <files or directories...>

Options:
  -q, --quality <1-100>   JPEG quality (default: 92)
  -o, --output <dir>      Output directory (default: same as input)
  -h, --help              Show this help

Examples:
  heic2jpg photo.heic
  heic2jpg -q 85 *.heic
  heic2jpg photos/ -o converted/
  heic2jpg -q 90 -o output/ photo1.heic photo2.heic`)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
