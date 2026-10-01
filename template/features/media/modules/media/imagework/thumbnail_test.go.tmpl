package imagework

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestThumbnailBounds(t *testing.T) {
	var input bytes.Buffer
	_ = png.Encode(&input, image.NewRGBA(image.Rect(0, 0, 20, 10)))
	out, err := Thumbnail(input.Bytes(), 10000, 200, 10)
	if err != nil {
		t.Fatal(err)
	}
	size, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil || size.Width != 10 || size.Height != 5 {
		t.Fatalf("dimensions: %v %v", size, err)
	}
	for _, test := range []struct {
		bytes  int
		pixels int64
		width  int
	}{{1, 200, 10}, {10000, 199, 10}, {10000, 200, 2049}} {
		if _, err := Thumbnail(input.Bytes(), test.bytes, test.pixels, test.width); err == nil {
			t.Fatal("resource limit ignored")
		}
	}
	if _, err := Thumbnail([]byte("not an image"), 10000, 200, 10); err == nil {
		t.Fatal("invalid image accepted")
	}
}
