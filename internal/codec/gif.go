package codec

import (
	"image"
	"image/gif"
	"io"
)

type gifDecoder struct{}
type gifEncoder struct{}

func (d *gifDecoder) Decode(r io.ReadSeeker) (image.Image, error) {
	return gif.Decode(r)
}

func (d *gifDecoder) Format() Format { return GIF }

func (e *gifEncoder) Encode(w io.Writer, img image.Image, _ int) error {
	return gif.Encode(w, img, nil)
}

func (e *gifEncoder) Format() Format { return GIF }

func registerGIF(r *Registry) {
	r.RegisterDecoder(&gifDecoder{})
	r.RegisterEncoder(&gifEncoder{})
}
