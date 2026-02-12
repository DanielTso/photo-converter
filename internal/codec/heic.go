package codec

import (
	"image"
	"io"

	heif "github.com/vegidio/heif-go"
)

type heicDecoder struct{}
type heicEncoder struct{}

func (d *heicDecoder) Decode(r io.ReadSeeker) (image.Image, error) {
	return heif.Decode(r)
}

func (d *heicDecoder) Format() Format { return HEIC }

func (e *heicEncoder) Encode(w io.Writer, img image.Image, quality int) error {
	return heif.Encode(w, img, &heif.Options{Quality: quality})
}

func (e *heicEncoder) Format() Format { return HEIC }

func registerHEIC(r *Registry) {
	r.RegisterDecoder(&heicDecoder{})
	r.RegisterEncoder(&heicEncoder{})
}
