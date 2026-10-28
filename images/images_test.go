package images

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

func TestSniffFormat(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}, "jpeg"},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00}, "png"},
		{"gif87a", append([]byte("GIF87a"), 0x00), "gif"},
		{"gif89a", append([]byte("GIF89a"), 0x00), "gif"},
		{"bmp", append([]byte("BM"), 0x00, 0x00), "bmp"},
		{"webp", append([]byte("RIFF\x00\x00\x00\x00WEBP"), 0x00), "webp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sniffFormat(tc.data)
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
	_, err := sniffFormat([]byte("not an image, just text"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestValidateNameRejectsPathTraversal(t *testing.T) {
	for _, name := range []string{"", "../etc/passwd", "a/b.png", `a\b.png`, "..png"} {
		if err := validateName(name); err != ErrInvalidName {
			t.Errorf("validateName(%q) = %v, want ErrInvalidName", name, err)
		}
	}
}

func TestValidateNameAcceptsPlainName(t *testing.T) {
	if err := validateName("foto.png"); err != nil {
		t.Errorf("validateName(\"foto.png\") = %v, want nil", err)
	}
}

func pngBytes() []byte {
	return []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x00}
}

func TestSaveWritesFileAndReturnsName(t *testing.T) {
	withTempMediaDir(t)

	saved, err := Save("foto.png", bytes.NewReader(pngBytes()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved != "foto.png" {
		t.Errorf("saved name = %q, want %q", saved, "foto.png")
	}
	path := filepath.Join(storage.BasePath(), "images", "foto.png")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s: %v", path, err)
	}
}

func TestSaveOverwritesExisting(t *testing.T) {
	withTempMediaDir(t)

	if _, err := Save("foto.png", bytes.NewReader(pngBytes())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	newContent := append(pngBytes(), 0xFF, 0xFF)
	if _, err := Save("foto.png", bytes.NewReader(newContent)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(storage.BasePath(), "images", "foto.png")
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

	_, err := Save("nota.txt", strings.NewReader("just some text"))
	if err != ErrUnsupportedFormat {
		t.Fatalf("error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestSaveRejectsTooLarge(t *testing.T) {
	withTempMediaDir(t)

	oversized := make([]byte, MaxUploadSize+1)
	copy(oversized, pngBytes())
	_, err := Save("grande.png", bytes.NewReader(oversized))
	if err != ErrTooLarge {
		t.Fatalf("error = %v, want ErrTooLarge", err)
	}
}

func TestSaveRejectsInvalidName(t *testing.T) {
	withTempMediaDir(t)

	_, err := Save("../escape.png", bytes.NewReader(pngBytes()))
	if err != ErrInvalidName {
		t.Fatalf("error = %v, want ErrInvalidName", err)
	}
}

func TestListReturnsSortedNames(t *testing.T) {
	withTempMediaDir(t)

	for _, name := range []string{"zebra.png", "abacaxi.png"} {
		if _, err := Save(name, bytes.NewReader(pngBytes())); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	got, err := List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"abacaxi.png", "zebra.png"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("List() = %v, want %v", got, want)
	}
}

func TestListOnEmptyDirReturnsEmpty(t *testing.T) {
	withTempMediaDir(t)

	got, err := List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() = %v, want empty", got)
	}
}

func TestContentPathFoundAndNotFound(t *testing.T) {
	withTempMediaDir(t)

	if _, err := Save("foto.png", bytes.NewReader(pngBytes())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path, err := ContentPath("foto.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("ContentPath returned a path that doesn't exist: %v", err)
	}

	if _, err := ContentPath("nao-existe.png"); err != ErrNotFound {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestContentPathRejectsInvalidName(t *testing.T) {
	withTempMediaDir(t)

	if _, err := ContentPath("../escape.png"); err != ErrInvalidName {
		t.Fatalf("error = %v, want ErrInvalidName", err)
	}
}
