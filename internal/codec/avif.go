package codec

import (
	"image"
	"io"

	avif "github.com/vegidio/avif-go"
)

type avifDecoder struct{}
type avifEncoder struct{}

func (d *avifDecoder) Decode(r io.ReadSeeker) (image.Image, error) {
	return avif.Decode(r)
}

func (d *avifDecoder) Format() Format { return AVIF }

func (e *avifEncoder) Encode(w io.Writer, img image.Image, quality int) error {
	return avif.Encode(w, img, &avif.Options{ColorQuality: quality, AlphaQuality: quality})
}

func (e *avifEncoder) Format() Format { return AVIF }

func registerAVIF(r *Registry) {
	r.RegisterDecoder(&avifDecoder{})
	r.RegisterEncoder(&avifEncoder{})
}
