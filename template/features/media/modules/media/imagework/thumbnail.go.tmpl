package imagework

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
)

var ErrInvalid = errors.New("unsupported or oversized image")

// Thumbnail validates dimensions before decode; output is metadata-free PNG.
func Thumbnail(data []byte, maxInput int, maxPixels int64, width int) ([]byte, error) {
	if len(data) > maxInput || maxInput <= 0 || maxPixels <= 0 || width < 1 || width > 2048 {
		return nil, ErrInvalid
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width) > maxPixels/int64(config.Height) {
		return nil, ErrInvalid
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrInvalid
	}
	targetWidth := width
	if targetWidth > config.Width {
		targetWidth = config.Width
	}
	height := int(int64(config.Height) * int64(targetWidth) / int64(config.Width))
	if height < 1 {
		height = 1
	}
	out := image.NewRGBA(image.Rect(0, 0, targetWidth, height))
	bounds := source.Bounds()
	if targetWidth == config.Width {
		draw.Draw(out, out.Bounds(), source, bounds.Min, draw.Src)
	} else {
		for y := 0; y < height; y++ {
			for x := 0; x < targetWidth; x++ {
				out.Set(x, y, source.At(bounds.Min.X+x*config.Width/targetWidth, bounds.Min.Y+y*config.Height/height))
			}
		}
	}
	var encoded bytes.Buffer
	if png.Encode(&encoded, out) != nil {
		return nil, ErrInvalid
	}
	return encoded.Bytes(), nil
}
