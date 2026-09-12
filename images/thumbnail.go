package images

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"

	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"

	"presenter/fsutil"
)

const thumbnailRatio = 8

// thumbName returns the thumbnail file name for name (always .png). The
// full original name is kept (not just its stem), so two uploads that
// share a basename but differ only in extension (e.g. foto.jpg and
// foto.png) get distinct thumbnails instead of colliding.
func thumbName(name string) string {
	return name + ".png"
}

func decodeByFormat(format string, data []byte) (image.Image, error) {
	r := bytes.NewReader(data)
	switch format {
	case "jpeg":
		return jpeg.Decode(r)
	case "png":
		return png.Decode(r)
	case "gif":
		return gif.Decode(r)
	case "bmp":
		return bmp.Decode(r)
	case "webp":
		return webp.Decode(r)
	default:
		return nil, fmt.Errorf("no decoder for format %q", format)
	}
}

// saveThumbnail decodes data as format, scales it down by thumbnailRatio,
// and writes the result as a PNG under thumbsDir()/thumbName(name).
func saveThumbnail(name, format string, data []byte) error {
	src, err := decodeByFormat(format, data)
	if err != nil {
		return fmt.Errorf("decoding image: %w", err)
	}
	bounds := src.Bounds()
	w, h := bounds.Dx()/thumbnailRatio, bounds.Dy()/thumbnailRatio
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	if _, err := fsutil.CreateFolder(thumbsDir()); err != nil {
		return fmt.Errorf("creating thumbs dir: %w", err)
	}
	out, err := os.Create(thumbsDir() + thumbName(name))
	if err != nil {
		return fmt.Errorf("creating thumbnail file: %w", err)
	}
	defer out.Close()
	return png.Encode(out, dst)
}

// ThumbPath returns the on-disk path of name's thumbnail (always .png), or
// ErrNotFound if it doesn't exist.
func ThumbPath(name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	path := thumbsDir() + thumbName(name)
	exists, err := fsutil.PathExists(path)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrNotFound
	}
	return path, nil
}
