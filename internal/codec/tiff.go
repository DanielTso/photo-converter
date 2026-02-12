package codec

import (
	"image"
	"io"

	"golang.org/x/image/tiff"
)

type tiffDecoder struct{}
type tiffEncoder struct{}

func (d *tiffDecoder) Decode(r io.ReadSeeker) (image.Image, error) {
	return tiff.Decode(r)
}

func (d *tiffDecoder) Format() Format { return TIFF }

func (e *tiffEncoder) Encode(w io.Writer, img image.Image, _ int) error {
	return tiff.Encode(w, img, nil)
}

func (e *tiffEncoder) Format() Format { return TIFF }

func registerTIFF(r *Registry) {
	r.RegisterDecoder(&tiffDecoder{})
	r.RegisterEncoder(&tiffEncoder{})
}
