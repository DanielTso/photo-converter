# HEIC to JPEG Converter

Portable HEIC/HEIF to JPEG converter — no installation required. Two options:

1. **Web Converter** — a single HTML file you open in any browser
2. **CLI Tool** — a single binary for fast batch conversion

## Web Converter (Recommended for locked-down machines)

The web converter is a single `index.html` file. Open it in any browser (Chrome, Edge, Firefox) and convert HEIC files by dragging and dropping. Everything runs locally — no files are uploaded anywhere.

### How to use

1. Get `web/dist/index.html` (or download from [Releases](../../releases))
2. Copy it to a USB drive or your desktop
3. Double-click to open in your browser
4. Drag and drop `.heic` files (or click "Choose Files")
5. Adjust JPEG quality if needed
6. Click "Download" on each image, or "Download All as ZIP"

### Features

- Drag-and-drop or file picker
- Adjustable JPEG quality (1-100%)
- Batch conversion
- Preview thumbnails
- Download individual files or all as ZIP
- Works completely offline
- No installation, no admin privileges, no server

### Build from source

```bash
cd web
npm ci
npx webpack --config webpack.config.js
# Output: web/dist/index.html
```

## CLI Tool (For batch/power-user workflows)

A single native binary that converts HEIC files from the command line. Fast — processes files in well under a second each. When given a directory, it recursively scans all subdirectories for HEIC/HEIF files.

### How to use

```bash
# Convert a single file
./heic2jpg photo.heic

# Convert all HEIC files in a directory (recursively)
./heic2jpg photos/

# Set JPEG quality (default: 92)
./heic2jpg -q 85 photo.heic

# Output to a specific directory
./heic2jpg -o converted/ photos/

# Convert to PNG instead of JPEG
./heic2jpg -f png photos/

# Use 4 parallel workers for faster batch conversion
./heic2jpg -j 4 -o converted/ photos/
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `-q, --quality <1-100>` | JPEG quality (ignored for PNG) | 92 |
| `-f, --format <jpg\|png>` | Output format | jpg |
| `-j, --jobs <N>` | Parallel workers | Number of CPUs |
| `-o, --output <dir>` | Output directory | Same as input file |
| `-h, --help` | Show help | |

### Build from source

```bash
# Build for current platform
cd cli
go build -o heic2jpg .

# Build for Windows (requires mingw-w64 on Linux, or build on Windows directly)
CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -o heic2jpg.exe .
```

### Pre-built binaries

Pre-built binaries for Windows, macOS, and Linux are available from [Releases](../../releases). Tag a version (`git tag v1.0.0 && git push --tags`) to trigger automated builds via GitHub Actions.

## Project Structure

```
photo-converter/
├── web/                  # Browser-based converter
│   ├── src/
│   │   ├── index.html    # HTML template
│   │   ├── index.js      # Converter logic + UI
│   │   └── style.css     # Styles
│   ├── dist/
│   │   └── index.html    # Built single-file converter (1.5 MB)
│   ├── webpack.config.js
│   └── package.json
├── cli/                  # Command-line converter
│   ├── main.go           # CLI source
│   └── go.mod
├── .github/workflows/
│   └── build.yml         # CI: builds all platforms + creates releases
├── Makefile
└── README.md
```

## Which one should I use?

| Scenario | Use |
|----------|-----|
| Work laptop, can't install anything | **Web converter** — just open the HTML file |
| Need to convert a few photos | **Web converter** |
| Need to batch-convert hundreds of files | **CLI tool** |
| Want maximum speed | **CLI tool** (~0.5s/image vs ~5s/image) |
| Not comfortable with command line | **Web converter** |
