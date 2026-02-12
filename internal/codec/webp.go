package codec

import (
	"image"
	"io"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	xwebp "golang.org/x/image/webp"
)

type webpDecoder struct{}
type webpEncoder struct{}

func (d *webpDecoder) Decode(r io.ReadSeeker) (image.Image, error) {
	return xwebp.Decode(r)
}

func (d *webpDecoder) Format() Format { return WebP }

func (e *webpEncoder) Encode(w io.Writer, img image.Image, quality int) error {
	opts, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, float32(quality))
	if err != nil {
		return err
	}
	return webp.Encode(w, img, opts)
}

func (e *webpEncoder) Format() Format { return WebP }

func registerWebP(r *Registry) {
	r.RegisterDecoder(&webpDecoder{})
	r.RegisterEncoder(&webpEncoder{})
}
