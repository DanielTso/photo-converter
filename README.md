# Pixshift

Universal image converter CLI. Convert between JPEG, PNG, GIF, WebP, TIFF, BMP, HEIC/HEIF, AVIF, and RAW camera formats (CR2, NEF, DNG).

## Features

- **Any-to-any conversion** between 10+ image formats
- **Smart format detection** via magic bytes (not file extensions)
- **Parallel processing** with configurable worker pool
- **Metadata preservation** — keep EXIF data across conversions
- **Watch mode** — auto-convert new files dropped into a directory
- **Rules engine** — YAML config for batch conversion with per-format rules
- **RAW support** — extract JPEG previews from CR2, NEF, DNG files
- **Single binary** — no runtime dependencies

## Format Support

| Format | Decode | Encode | Notes |
|--------|--------|--------|-------|
| JPEG | Yes | Yes | stdlib |
| PNG | Yes | Yes | stdlib |
| GIF | Yes | Yes | stdlib |
| WebP | Yes | Yes | CGO for encode |
| TIFF | Yes | Yes | |
| BMP | Yes | Yes | |
| HEIC/HEIF | Yes | Yes | CGO |
| AVIF | Yes | Yes | CGO |
| CR2 | Yes | - | Extracts embedded preview |
| NEF | Yes | - | Extracts embedded preview |
| DNG | Yes | - | Extracts embedded preview |

## Install

Download a binary from [Releases](https://github.com/DanielTso/pixshift/releases), or build from source:

```bash
git clone https://github.com/DanielTso/pixshift.git
cd pixshift
make build
```

## Usage

```bash
# Basic conversion (auto-detects input format)
pixshift photo.heic                          # -> photo.jpg

# Specify output format and quality
pixshift -f webp -q 90 photo.heic           # -> photo.webp at quality 90

# Batch convert a directory
pixshift -f png -o converted/ photos/

# Parallel workers
pixshift -j 8 -f webp -o output/ photos/

# Preserve EXIF metadata
pixshift -m -f jpg photo.heic

# Extract JPEG preview from RAW
pixshift photo.CR2

# Watch mode: auto-convert new files
pixshift -w -f webp ~/Pictures/

# Rules mode from config file
pixshift -c pixshift.yaml photos/

# Preview what would happen
pixshift --dry-run -f webp photos/
```

## Flags

| Flag | Description |
|------|-------------|
| `-f, --format` | Output format (jpg, png, gif, webp, tiff, bmp, heic, avif) |
| `-q, --quality` | Encoding quality 1-100 (default: 92) |
| `-j, --jobs` | Number of parallel workers (default: CPU count) |
| `-o, --output` | Output directory |
| `-r, --recursive` | Process directories recursively |
| `-m, --preserve-metadata` | Preserve EXIF metadata |
| `-w, --watch` | Watch mode |
| `-c, --config` | Rules config file |
| `--overwrite` | Overwrite existing files |
| `--dry-run` | Preview without converting |
| `-v, --verbose` | Verbose output |
| `-V, --version` | Show version |

## Rules Config

Create a `pixshift.yaml` to define per-format conversion rules:

```yaml
rules:
  - name: heic-to-webp
    format: heic
    output: webp
    quality: 90

  - name: raw-to-jpeg
    format: cr2
    output: jpg
    quality: 95

  - name: default
    output: jpg
    quality: 92
```

Rules are evaluated in order. First match wins. See [pixshift.yaml.example](pixshift.yaml.example) for more examples.

## Building

Requires Go 1.24+ and CGO (for HEIC, AVIF, and WebP encoding).

```bash
make build          # Build for current platform
make build-static   # Build with static linking (Linux)
make test           # Run tests
make lint           # Run linter
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
