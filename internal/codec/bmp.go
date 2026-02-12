package codec

import (
	"image"
	"io"

	"golang.org/x/image/bmp"
)

type bmpDecoder struct{}
type bmpEncoder struct{}

func (d *bmpDecoder) Decode(r io.ReadSeeker) (image.Image, error) {
	return bmp.Decode(r)
}

func (d *bmpDecoder) Format() Format { return BMP }

func (e *bmpEncoder) Encode(w io.Writer, img image.Image, _ int) error {
	return bmp.Encode(w, img)
}

func (e *bmpEncoder) Format() Format { return BMP }

func registerBMP(r *Registry) {
	r.RegisterDecoder(&bmpDecoder{})
	r.RegisterEncoder(&bmpEncoder{})
}
