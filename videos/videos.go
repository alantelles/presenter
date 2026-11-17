// Package videos handles validated upload, on-disk storage, listing,
// retrieval and deletion of presentation videos, stored flat under
// storage.BasePath()+"videos/".
package videos

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"presenter/fsutil"
	"presenter/storage"
)

// MaxUploadSize is the largest accepted video upload, in bytes (500MB).
const MaxUploadSize = 500 << 20

var (
	// ErrUnsupportedFormat is returned when the content's signature or extension
	// doesn't match any accepted video format (MP4, WebM, OGG, MOV).
	ErrUnsupportedFormat = errors.New("unsupported video format")
	// ErrTooLarge is returned when the content exceeds MaxUploadSize.
	ErrTooLarge = errors.New("video exceeds the maximum upload size")
	// ErrInvalidName is returned when name contains a path separator or "..".
	ErrInvalidName = errors.New("invalid video name")
	// ErrNotFound is returned by ContentPath when the video doesn't exist.
	ErrNotFound = errors.New("video not found")
)

func videosDir() string { return storage.BasePath() + "videos/" }

// validateName rejects names that could escape the videos directory.
func validateName(name string) error {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return ErrInvalidName
	}
	return nil
}

// sniffFormat inspects initial bytes and filename to verify if it is an accepted video format.
func sniffFormat(data []byte, filename string) (string, error) {
	// MP4 / M4V / QuickTime: "ftyp" or "moov" box in first 32 bytes
	if len(data) >= 8 {
		if bytes.Equal(data[4:8], []byte("ftyp")) || bytes.Equal(data[4:8], []byte("moov")) || bytes.Equal(data[4:8], []byte("wide")) {
			return "mp4", nil
		}
		if bytes.Contains(data[:min(len(data), 64)], []byte("ftyp")) {
			return "mp4", nil
		}
	}

	// WebM / MKV: EBML header \x1A\x45\xDF\xA3
	if bytes.HasPrefix(data, []byte{0x1A, 0x45, 0xDF, 0xA3}) {
		return "webm", nil
	}

	// Ogg: "OggS"
	if bytes.HasPrefix(data, []byte("OggS")) {
		return "ogg", nil
	}

	// Fallback to extension check if data is non-empty
	if len(data) > 0 {
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".mp4", ".m4v":
			return "mp4", nil
		case ".webm":
			return "webm", nil
		case ".ogg", ".ogv":
			return "ogg", nil
		case ".mov":
			return "mov", nil
		}
	}

	return "", ErrUnsupportedFormat
}

// Save validates content (name, format, size) and writes it to disk under name.
func Save(name string, content io.Reader) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}

	headerBuf := make([]byte, 512)
	n, err := io.ReadFull(content, headerBuf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("reading upload header: %w", err)
	}
	header := headerBuf[:n]
	if len(header) == 0 {
		return "", ErrUnsupportedFormat
	}

	if _, err := sniffFormat(header, name); err != nil {
		return "", err
	}

	if _, err := fsutil.CreateFolder(videosDir()); err != nil {
		return "", fmt.Errorf("creating videos dir: %w", err)
	}

	destPath := videosDir() + name
	tempPath := destPath + ".tmp"

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("creating video file: %w", err)
	}

	defer func() {
		file.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := file.Write(header); err != nil {
		return "", fmt.Errorf("writing video header: %w", err)
	}

	remainingLimit := int64(MaxUploadSize - len(header))
	limitedReader := io.LimitReader(content, remainingLimit+1)
	written, err := io.Copy(file, limitedReader)
	if err != nil {
		return "", fmt.Errorf("writing video content: %w", err)
	}

	if int64(len(header))+written > MaxUploadSize {
		return "", ErrTooLarge
	}

	if err := file.Close(); err != nil {
		return "", fmt.Errorf("closing video file: %w", err)
	}

	if err := os.Rename(tempPath, destPath); err != nil {
		return "", fmt.Errorf("finalizing video file: %w", err)
	}

	return name, nil
}

// List returns the names of saved videos, sorted alphabetically.
func List() ([]string, error) {
	entries, err := os.ReadDir(videosDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

// ContentPath returns the on-disk path of the video named name,
// or ErrNotFound if it doesn't exist.
func ContentPath(name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	path := videosDir() + name
	exists, err := fsutil.PathExists(path)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrNotFound
	}
	return path, nil
}

// Delete removes a saved video from disk.
func Delete(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	path := videosDir() + name
	exists, err := fsutil.PathExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return os.Remove(path)
}
