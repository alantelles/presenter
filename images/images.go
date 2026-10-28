// Package images handles validated upload, on-disk storage, listing and
// retrieval of presentation images (and, from Task 2 onward, their
// thumbnails), stored flat under storage.BasePath()+"images/".
package images

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"presenter/fsutil"
	"presenter/storage"
)

// MaxUploadSize is the largest accepted upload, in bytes.
const MaxUploadSize = 10 << 20 // 10MB

var (
	// ErrUnsupportedFormat is returned when the content's signature doesn't
	// match any of the accepted formats (JPEG, PNG, GIF, BMP, WEBP).
	ErrUnsupportedFormat = errors.New("unsupported image format")
	// ErrTooLarge is returned when the content exceeds MaxUploadSize.
	ErrTooLarge = errors.New("image exceeds the maximum upload size")
	// ErrInvalidName is returned when name contains a path separator or "..".
	ErrInvalidName = errors.New("invalid image name")
	// ErrNotFound is returned by ContentPath/ThumbPath when the file doesn't exist.
	ErrNotFound = errors.New("image not found")
)

func imagesDir() string { return storage.BasePath() + "images/" }
func thumbsDir() string { return imagesDir() + "thumbs/" }

// validateName rejects names that could escape the images directory.
func validateName(name string) error {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return ErrInvalidName
	}
	return nil
}

// sniffFormat inspects the first bytes of data and returns the detected
// format's canonical lowercase name ("jpeg", "png", "gif", "bmp", "webp"),
// or ErrUnsupportedFormat if none of the known signatures match.
func sniffFormat(data []byte) (string, error) {
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "jpeg", nil
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "png", nil
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		return "gif", nil
	case bytes.HasPrefix(data, []byte("BM")):
		return "bmp", nil
	case len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "webp", nil
	default:
		return "", ErrUnsupportedFormat
	}
}

// Save validates content (name, size, format) and writes it to disk under
// name, overwriting any existing file with the same name. It returns the
// saved name, or an error if content is invalid.
func Save(name string, content io.Reader) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(content, MaxUploadSize+1))
	if err != nil {
		return "", fmt.Errorf("reading upload: %w", err)
	}
	if len(data) > MaxUploadSize {
		return "", ErrTooLarge
	}
	format, err := sniffFormat(data)
	if err != nil {
		return "", err
	}
	if _, err := fsutil.CreateFolder(imagesDir()); err != nil {
		return "", fmt.Errorf("creating images dir: %w", err)
	}
	if err := os.WriteFile(imagesDir()+name, data, 0644); err != nil {
		return "", fmt.Errorf("saving image: %w", err)
	}
	if err := saveThumbnail(name, format, data); err != nil {
		log.Printf("images: failed to generate thumbnail for %s: %v", name, err)
	}
	return name, nil
}

// List returns the names of saved images (not thumbnails), sorted.
func List() ([]string, error) {
	entries, err := os.ReadDir(imagesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

// ContentPath returns the on-disk path of the original image named name,
// or ErrNotFound if it doesn't exist.
func ContentPath(name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	path := imagesDir() + name
	exists, err := fsutil.PathExists(path)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrNotFound
	}
	return path, nil
}
