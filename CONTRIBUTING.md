# Contributing to Pixshift

## Adding a New Format

Adding a new image format requires a single file and one line of registration:

### 1. Create the codec file

Create `internal/codec/yourformat.go`:

```go
package codec

import (
    "image"
    "io"

    "example.com/yourlib"
)

type yourDecoder struct{}
type yourEncoder struct{}

func (d *yourDecoder) Decode(r io.ReadSeeker) (image.Image, error) {
    return yourlib.Decode(r)
}

func (d *yourDecoder) Format() Format { return YourFormat }

func (e *yourEncoder) Encode(w io.Writer, img image.Image, quality int) error {
    return yourlib.Encode(w, img, quality)
}

func (e *yourEncoder) Format() Format { return YourFormat }

func registerYourFormat(r *Registry) {
    r.RegisterDecoder(&yourDecoder{})
    r.RegisterEncoder(&yourEncoder{})
}
```

### 2. Add the format constant

In `codec.go`, add your format constant:

```go
const (
    // ...existing formats...
    YourFormat Format = "yourformat"
)
```

### 3. Register the codec

In `registry.go`, add to `DefaultRegistry()`:

```go
func DefaultRegistry() *Registry {
    r := NewRegistry()
    // ...existing registrations...
    registerYourFormat(r)
    return r
}
```

### 4. Add magic bytes detection

In `detect.go`, add detection rules for your format's magic bytes.

### 5. Update ParseFormat

In `registry.go`, add your format to `ParseFormat()`.

### 6. Update DefaultExtension and IsSupportedExtension

In `codec.go`, add your format's extension.

## Development

```bash
make build    # Build
make test     # Run tests
make lint     # Run linter
```

## Pull Request Guidelines

- One feature/fix per PR
- Include tests for new codecs
- Run `make test` and `make lint` before submitting
- Follow existing code patterns
