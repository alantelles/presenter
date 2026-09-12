package images

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/bmp"
)

// sampleImage returns a small solid-color image to encode into test fixtures.
func sampleImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 50, B: 50, A: 255})
		}
	}
	return img
}

func encodeJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, sampleImage(), nil); err != nil {
		t.Fatalf("encoding jpeg fixture: %v", err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, sampleImage()); err != nil {
		t.Fatalf("encoding png fixture: %v", err)
	}
	return buf.Bytes()
}

func encodeGIF(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gif.Encode(&buf, sampleImage(), nil); err != nil {
		t.Fatalf("encoding gif fixture: %v", err)
	}
	return buf.Bytes()
}

func encodeBMP(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, sampleImage()); err != nil {
		t.Fatalf("encoding bmp fixture: %v", err)
	}
	return buf.Bytes()
}

func readWebpFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "sample.webp"))
	if err != nil {
		t.Fatalf("reading webp fixture (see Task 2 Step 1 for how to create it): %v", err)
	}
	return data
}

func TestSaveGeneratesThumbnailForEveryFormat(t *testing.T) {
	cases := []struct {
		name string
		file string
		data func(t *testing.T) []byte
	}{
		{"jpeg", "foto.jpg", encodeJPEG},
		{"png", "foto.png", encodePNG},
		{"gif", "foto.gif", encodeGIF},
		{"bmp", "foto.bmp", encodeBMP},
		{"webp", "foto.webp", readWebpFixture},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTempMediaDir(t)

			if _, err := Save(tc.file, bytes.NewReader(tc.data(t))); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			thumbPath, err := ThumbPath(tc.file)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			thumbData, err := os.ReadFile(thumbPath)
			if err != nil {
				t.Fatalf("reading thumbnail: %v", err)
			}
			decoded, err := png.Decode(bytes.NewReader(thumbData))
			if err != nil {
				t.Fatalf("thumbnail is not a valid PNG: %v", err)
			}
			if decoded.Bounds().Dx() < 1 || decoded.Bounds().Dy() < 1 {
				t.Errorf("thumbnail has empty bounds: %v", decoded.Bounds())
			}
		})
	}
}

func TestThumbPathIsAlwaysPNGRegardlessOfSourceExtension(t *testing.T) {
	withTempMediaDir(t)

	if _, err := Save("foto.jpg", bytes.NewReader(encodeJPEG(t))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path, err := ThumbPath("foto.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Ext(path) != ".png" {
		t.Errorf("thumbnail path = %s, want a .png extension", path)
	}
}

func TestThumbNameDoesNotCollideAcrossExtensions(t *testing.T) {
	withTempMediaDir(t)

	if _, err := Save("evento.jpg", bytes.NewReader(encodeJPEG(t))); err != nil {
		t.Fatalf("unexpected error saving jpg: %v", err)
	}
	if _, err := Save("evento.png", bytes.NewReader(encodePNG(t))); err != nil {
		t.Fatalf("unexpected error saving png: %v", err)
	}

	jpgThumb, err := ThumbPath("evento.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pngThumb, err := ThumbPath("evento.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if jpgThumb == pngThumb {
		t.Fatalf("thumbnails for evento.jpg and evento.png collided at %s", jpgThumb)
	}

	jpgData, err := os.ReadFile(jpgThumb)
	if err != nil {
		t.Fatalf("reading jpg thumbnail: %v", err)
	}
	pngData, err := os.ReadFile(pngThumb)
	if err != nil {
		t.Fatalf("reading png thumbnail: %v", err)
	}

	if _, err := png.Decode(bytes.NewReader(jpgData)); err != nil {
		t.Fatalf("jpg-derived thumbnail is not a valid PNG: %v", err)
	}
	if _, err := png.Decode(bytes.NewReader(pngData)); err != nil {
		t.Fatalf("png-derived thumbnail is not a valid PNG: %v", err)
	}
}

func TestSaveToleratesThumbnailDecodeFailure(t *testing.T) {
	withTempMediaDir(t)

	// A PNG-signed file whose body isn't actually valid PNG data: passes
	// sniffFormat (checks only the signature) but fails to decode.
	corrupt := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, []byte("not really png data")...)

	saved, err := Save("corrompida.png", bytes.NewReader(corrupt))
	if err != nil {
		t.Fatalf("Save should not fail when only the thumbnail step fails: %v", err)
	}
	if saved != "corrompida.png" {
		t.Errorf("saved name = %q, want %q", saved, "corrompida.png")
	}

	// The original file must still be there...
	if _, err := ContentPath("corrompida.png"); err != nil {
		t.Errorf("original file should still exist: %v", err)
	}
	// ...but there is no usable thumbnail.
	if _, err := ThumbPath("corrompida.png"); err != ErrNotFound {
		t.Errorf("ThumbPath error = %v, want ErrNotFound", err)
	}
}

func TestThumbPathNotFound(t *testing.T) {
	withTempMediaDir(t)

	if _, err := ThumbPath("nao-existe.png"); err != ErrNotFound {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestThumbPathRejectsInvalidName(t *testing.T) {
	withTempMediaDir(t)

	if _, err := ThumbPath("../escape.png"); err != ErrInvalidName {
		t.Fatalf("error = %v, want ErrInvalidName", err)
	}
}
