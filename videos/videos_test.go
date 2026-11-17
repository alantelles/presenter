package videos

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"presenter/storage"
)

func withTempMediaDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	storage.SetBasePath(dir)
	t.Cleanup(func() { storage.SetBasePath("") })
}

func mp4Bytes() []byte {
	// 4 bytes size + "ftyp" + "isom"
	return []byte{0x00, 0x00, 0x00, 0x14, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0x00, 0x00, 0x02, 0x00}
}

func webmBytes() []byte {
	return []byte{0x1A, 0x45, 0xDF, 0xA3, 0x9F, 0x42, 0x86, 0x81}
}

func oggBytes() []byte {
	return []byte{'O', 'g', 'g', 'S', 0x00, 0x02, 0x00, 0x00}
}

func TestSniffFormat(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		data     []byte
		want     string
	}{
		{"mp4 magic", "video.bin", mp4Bytes(), "mp4"},
		{"webm magic", "video.bin", webmBytes(), "webm"},
		{"ogg magic", "video.bin", oggBytes(), "ogg"},
		{"mp4 by ext", "clip.mp4", []byte("arbitrary-video-data"), "mp4"},
		{"webm by ext", "clip.webm", []byte("arbitrary-video-data"), "webm"},
		{"mov by ext", "clip.mov", []byte("arbitrary-video-data"), "mov"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sniffFormat(tc.data, tc.filename)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("sniffFormat = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSniffFormatRejectsUnknown(t *testing.T) {
	_, err := sniffFormat([]byte("just plain text"), "document.txt")
	if err != ErrUnsupportedFormat {
		t.Fatalf("error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestValidateName(t *testing.T) {
	for _, name := range []string{"", "../clip.mp4", "a/b.mp4", `a\b.mp4`, "..mp4"} {
		if err := validateName(name); err != ErrInvalidName {
			t.Errorf("validateName(%q) = %v, want ErrInvalidName", name, err)
		}
	}
	if err := validateName("video.mp4"); err != nil {
		t.Errorf("validateName(\"video.mp4\") = %v, want nil", err)
	}
}

func TestSaveWritesFileAndReturnsName(t *testing.T) {
	withTempMediaDir(t)

	saved, err := Save("video.mp4", bytes.NewReader(mp4Bytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved != "video.mp4" {
		t.Errorf("saved name = %q, want %q", saved, "video.mp4")
	}
	path := filepath.Join(storage.BasePath(), "videos", "video.mp4")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s: %v", path, err)
	}
}

func TestSaveOverwritesExisting(t *testing.T) {
	withTempMediaDir(t)

	if _, err := Save("video.mp4", bytes.NewReader(mp4Bytes())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	newContent := append(mp4Bytes(), 0xAA, 0xBB, 0xCC)
	if _, err := Save("video.mp4", bytes.NewReader(newContent)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(storage.BasePath(), "videos", "video.mp4")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, newContent) {
		t.Error("expected the second upload to overwrite the first")
	}
}

func TestSaveRejectsUnsupportedFormat(t *testing.T) {
	withTempMediaDir(t)

	_, err := Save("document.pdf", strings.NewReader("%PDF-1.4 sample"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestSaveRejectsInvalidName(t *testing.T) {
	withTempMediaDir(t)

	_, err := Save("../escape.mp4", bytes.NewReader(mp4Bytes()))
	if err != ErrInvalidName {
		t.Fatalf("error = %v, want ErrInvalidName", err)
	}
}

func TestListReturnsSortedNames(t *testing.T) {
	withTempMediaDir(t)

	for _, name := range []string{"zebra.mp4", "abacaxi.mp4"} {
		if _, err := Save(name, bytes.NewReader(mp4Bytes())); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	got, err := List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"abacaxi.mp4", "zebra.mp4"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("List() = %v, want %v", got, want)
	}
}

func TestContentPathAndNotFound(t *testing.T) {
	withTempMediaDir(t)

	if _, err := Save("video.mp4", bytes.NewReader(mp4Bytes())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path, err := ContentPath("video.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("ContentPath returned a path that doesn't exist: %v", err)
	}

	if _, err := ContentPath("nao-existe.mp4"); err != ErrNotFound {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestDeleteVideo(t *testing.T) {
	withTempMediaDir(t)

	if _, err := Save("video.mp4", bytes.NewReader(mp4Bytes())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := Delete("video.mp4"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := ContentPath("video.mp4"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}
}
