# Upload e Exibição de Imagens Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an operator upload images (JPEG/PNG/WEBP/GIF/BMP), pick one from a gallery of previously uploaded images, and have it appear on the live panel (`/live`) in the same area that shows text today.

**Architecture:** A new `presenter/images` submodule owns validation (content-sniffed format, 10MB size limit, name sanitization), on-disk storage, and thumbnail generation (always re-encoded to PNG, since there's no WEBP encoder available). The root module exposes 4 HTTP endpoints over that package and reuses the existing generic provider-content mechanism (`POST /api/content/set/main` with `type: 'IMAGE'`) to publish an image — no provider-specific plumbing needed beyond the panel learning to render `type === 'IMAGE'` as an `<img>`. A new controller page (`templates/controllers/images.html` + `static/js/images/`) provides the upload/gallery UI, kept separate from the already-large `songs.html`.

**Tech Stack:** Go 1.24 (root) / 1.23.4 (submodules) + Gin, `golang.org/x/image` (already a dependency: `bmp`, `webp` decoders, `draw` for resizing), stdlib `image/jpeg`, `image/png`, `image/gif` (all with both decode and encode). No new external dependencies. Plain JS modules on the frontend (no framework), matching `static/js/songs/`.

**Spec:** [docs/superpowers/specs/2026-09-12-upload-exibicao-imagens-design.md](../specs/2026-09-12-upload-exibicao-imagens-design.md)

## Global Constraints

- Accepted formats, detected by content (magic bytes), never by the uploaded filename's extension: JPEG (`FF D8 FF`), PNG (`89 50 4E 47 0D 0A 1A 0A`), GIF (`GIF87a`/`GIF89a`), BMP (`BM`), WEBP (`RIFF????WEBP`).
- Max upload size: 10MB (`10 << 20` bytes). Larger uploads are rejected before being fully buffered in memory when possible.
- Thumbnails are always written as `.png`, regardless of the original format — there is no WEBP encoder available without cgo.
- A name containing `/`, `\`, or `..` is invalid and rejected on every operation (save, read content, read thumbnail) — no path traversal.
- Uploading a name that already exists overwrites the existing file and its thumbnail (no rename-on-conflict, no rejection).
- A thumbnail-generation failure (corrupt file that still matched a valid signature) must not fail the upload — the original file is kept, the thumbnail is simply absent.
- Only the `main` provider gains image rendering in the panel this round. `aux` and `preview` are untouched.
- `POST /api/images` requires Basic Auth (`app.AuthMiddleware`), matching every other write endpoint. `GET /api/images`, `GET /api/images/content`, `GET /api/images/thumb` are public, matching `GET /api/songs/content` and `GET /api/content`.
- No image category/subfolder system (no `storage.Category` for images) — a flat list under `storage.BasePath()+"images/"`.

---

## File Structure

- **Create `images/go.mod`** — new submodule `presenter/images`, depending on `presenter/fsutil`, `presenter/storage`, and `golang.org/x/image` (all already used elsewhere in the repo).
- **Create `images/images.go`** — format sniffing, name validation, `Save` (validate + size-limit + write original), `List`, `ContentPath`. (Task 1)
- **Create `images/images_test.go`** — unit tests for the above. (Task 1)
- **Create `images/thumbnail.go`** — per-format decode + resize + PNG encode, wired into `Save`; `ThumbPath`. (Task 2)
- **Create `images/thumbnail_test.go`** — unit tests for the above. (Task 2)
- **Modify root `go.mod`** — add `presenter/images` require + replace. (Task 3)
- **Create `handlers_images.go`** — the 4 Gin handlers (`uploadImage` replaces the old one, `listImages`, `getImageContent`, `getImageThumb`). (Task 3)
- **Create `handlers_images_test.go`** — HTTP-level tests for the 4 endpoints. (Task 3)
- **Modify `handlers_api.go`** — remove the old, buggy `uploadImage`. (Task 3)
- **Modify `handlers.go`** — remove `createThumbnail`/`getThumbnailDimensions` and their now-unused imports. (Task 3)
- **Modify `routes.go`** — remove `POST /api/images` from `registerMediaRoutes`; add `registerImageRoutes` with all 4 image routes. (Task 3)
- **Modify `routes_test.go`** — update `TestRegisterMediaRoutes`'s expected route list; add `TestRegisterImageRoutes`. (Task 3)
- **Modify `main.go`** — call `registerImageRoutes`. (Task 3)
- **Modify `templates/panels/default.html`** — `processMainProviderData` learns to render `type === 'IMAGE'` as an `<img>`; small CSS rule for it. (Task 4)
- **Create `templates/controllers/images.html`** — upload form + thumbnail gallery. (Task 5)
- **Create `static/js/images/client.js`** — API calls (`uploadImage`, `fetchImageList`, `emitImageContent`). (Task 5)
- **Create `static/js/images/images.js`** — DOM wiring (render gallery, handle upload/click). (Task 5)

---

### Task 1: `presenter/images` — format validation, save, list, read

**Files:**
- Create: `images/go.mod`
- Create: `images/images.go`
- Test: `images/images_test.go`

**Interfaces:**
- Produces: `const MaxUploadSize = 10 << 20`; `var ErrUnsupportedFormat, ErrTooLarge, ErrInvalidName, ErrNotFound error`; `func Save(name string, content io.Reader) (string, error)` (no thumbnail generation yet — that's Task 2); `func List() ([]string, error)`; `func ContentPath(name string) (string, error)`; unexported `sniffFormat(data []byte) (string, error)` and `validateName(name string) error`, reused by Task 2.

- [ ] **Step 1: Create the submodule's go.mod**

Create `images/go.mod`:

```go
module presenter/images

go 1.23.4

require (
	golang.org/x/image v0.32.0
	presenter/fsutil v0.0.0-00010101000000-000000000000
	presenter/storage v0.0.0-00010101000000-000000000000
)

replace presenter/fsutil => ../fsutil

replace presenter/storage => ../storage
```

- [ ] **Step 2: Write the failing tests**

Create `images/images_test.go`:

```go
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run (from the repo root, with the asdf env vars set — see note at the end of this plan): `cd images && go test ./... -v`
Expected: fails to compile (`package images: no Go files` / `undefined: Save` etc.), since `images.go` doesn't exist yet.

- [ ] **Step 4: Implement**

Create `images/images.go`:

```go
// Package images handles validated upload, on-disk storage, listing and
// retrieval of presentation images (and, from Task 2 onward, their
// thumbnails), stored flat under storage.BasePath()+"images/".
package images

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	if _, err := sniffFormat(data); err != nil {
		return "", err
	}
	if _, err := fsutil.CreateFolder(imagesDir()); err != nil {
		return "", fmt.Errorf("creating images dir: %w", err)
	}
	if err := os.WriteFile(imagesDir()+name, data, 0644); err != nil {
		return "", fmt.Errorf("saving image: %w", err)
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
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd images && go test ./... -v`
Expected: PASS (all tests from Step 2)

- [ ] **Step 6: Commit**

```bash
git add images/go.mod images/images.go images/images_test.go
git commit -m "feat(images): add format validation, save, list, content path"
```

---

### Task 2: Thumbnail generation

**Files:**
- Create: `images/thumbnail.go`
- Modify: `images/images.go`
- Test: `images/thumbnail_test.go`

**Interfaces:**
- Consumes: `sniffFormat`, `validateName`, `imagesDir`/`thumbsDir`, `ErrInvalidName`, `ErrNotFound` from Task 1.
- Produces: unexported `saveThumbnail(name, format string, data []byte) error`, called from `Save` (non-fatal on failure); `func ThumbPath(name string) (string, error)`; unexported `thumbName(name string) string`.

- [ ] **Step 1: Get a real WEBP fixture for the decode test**

`golang.org/x/image/webp` only decodes — there's no way to encode a WEBP file from Go without cgo, so the test needs a real, valid WEBP file's bytes. The `golang.org/x/image` module itself ships small WEBP test images under its own `webp/testdata/` directory, which `go mod download` already fetched into your local module cache.

Run this to find one:

```bash
find "$(go env GOMODCACHE)/golang.org/x/image@v0.32.0/webp/testdata" -name '*.webp' | head -5
```

Pick the **smallest** file listed (`ls -la` them, or just try the first) and copy it in:

```bash
mkdir -p images/testdata
cp "<the path you found>" images/testdata/sample.webp
```

If that directory doesn't exist in this module version (module layouts change), fall back to generating one with any image tool available in this environment (e.g. `cwebp -q 80 <any small png> -o images/testdata/sample.webp`, or Python + Pillow: `python3 -c "from PIL import Image; Image.new('RGB',(4,4),'red').save('images/testdata/sample.webp')"`). If neither `golang.org/x/image`'s testdata nor any encoding tool is available in this environment, report **BLOCKED** with what you tried — don't fabricate WEBP bytes by hand, a hand-rolled file is unlikely to be valid and would make the test lie.

Once you have `images/testdata/sample.webp`, confirm it's a real file with content: `ls -la images/testdata/sample.webp` (should be a few hundred bytes to a few KB, not empty).

- [ ] **Step 2: Write the failing tests**

Create `images/thumbnail_test.go`:

```go
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd images && go test ./... -run 'TestSaveGeneratesThumbnail|TestThumbPath|TestSaveTolerates' -v`
Expected: fails to compile (`undefined: ThumbPath`) — `bmp` package isn't imported by the test build yet either, but it's already a declared dependency from Task 1's `go.mod`, so only the missing production symbols block compilation.

- [ ] **Step 4: Implement**

Create `images/thumbnail.go`:

```go
package images

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"

	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"

	"presenter/fsutil"
)

const thumbnailRatio = 8

// thumbName returns name with its extension replaced by .png.
func thumbName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[:i] + ".png"
		}
	}
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
```

Now wire thumbnail generation into `Save`. In `images/images.go`, change the `Save` function: capture the detected format (it's currently discarded), and call `saveThumbnail` non-fatally after the original file is written:

```go
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
```

Add `"log"` to `images/images.go`'s import block.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd images && go test ./... -v`
Expected: PASS (all tests from Task 1 and Task 2)

- [ ] **Step 6: Commit**

```bash
git add images/thumbnail.go images/images.go images/thumbnail_test.go images/testdata
git commit -m "feat(images): generate PNG thumbnails for every accepted format"
```

---

### Task 3: HTTP endpoints, route registration, and removal of the old code

**Files:**
- Modify: `go.mod` (root)
- Create: `handlers_images.go`
- Create: `handlers_images_test.go`
- Modify: `handlers_api.go`
- Modify: `handlers.go`
- Modify: `routes.go`
- Modify: `routes_test.go`
- Modify: `main.go`

**Interfaces:**
- Consumes: `images.Save`, `images.List`, `images.ContentPath`, `images.ThumbPath`, `images.ErrUnsupportedFormat`, `images.ErrTooLarge`, `images.ErrInvalidName`, `images.ErrNotFound` (Tasks 1-2); `app.AuthMiddleware` (existing, `app.go`); `withTempWorkDir(t)` (existing, `app_test.go`).
- Produces: `POST /api/images`, `GET /api/images`, `GET /api/images/content`, `GET /api/images/thumb` — wired via `registerImageRoutes(router *gin.Engine, app *App)`.

- [ ] **Step 1: Wire the new submodule into the root module**

In root `go.mod`, add to the `require (...)` block (alongside the other `presenter/*` entries):

```go
	presenter/images v0.0.0-00010101000000-000000000000
```

And add a `replace` line alongside the others:

```go
replace presenter/images => ./images
```

Run `go mod tidy` in the repo root afterward (Step 6 below) to reconcile versions/checksums — do this once `handlers_images.go` (Step 3) actually imports the package, since `go mod tidy` needs a real import to know the module is used.

- [ ] **Step 2: Write the failing HTTP tests**

Create `handlers_images_test.go`:

```go
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newImageTestRouter(t *testing.T) (*gin.Engine, *App) {
	t.Helper()
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	app := newTestApp()
	router := gin.New()
	registerImageRoutes(router, app)
	return router, app
}

func pngUploadBody(t *testing.T, filename string) (*bytes.Buffer, string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("encoding png fixture: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	if _, err := part.Write(pngBuf.Bytes()); err != nil {
		t.Fatalf("writing form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func TestUploadImageRequiresAuth(t *testing.T) {
	router, _ := newImageTestRouter(t)
	body, contentType := pngUploadBody(t, "foto.png")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/images", body)
	req.Header.Set("Content-Type", contentType)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUploadImageThenListAndFetch(t *testing.T) {
	router, app := newImageTestRouter(t)
	body, contentType := pngUploadBody(t, "foto.png")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/images", body)
	req.Header.Set("Content-Type", contentType)
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/images", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", w.Code, http.StatusOK)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("foto.png")) {
		t.Errorf("list body %q does not contain %q", w.Body.String(), "foto.png")
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/images/content?name=foto.png", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("content status = %d, want %d", w.Code, http.StatusOK)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/images/thumb?name=foto.png", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("thumb status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUploadImageRejectsUnsupportedFormat(t *testing.T) {
	router, app := newImageTestRouter(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files", "nota.txt")
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	part.Write([]byte("just some text, not an image"))
	writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetBasicAuth(app.Config.BasicAuthUser, app.Config.BasicAuthPass)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestGetImageContentNotFound(t *testing.T) {
	router, _ := newImageTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/images/content?name=nao-existe.png", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetImageThumbNotFound(t *testing.T) {
	router, _ := newImageTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/images/thumb?name=nao-existe.png", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./... -run 'TestUploadImage|TestGetImage' -v`
Expected: fails to compile (`undefined: registerImageRoutes`)

- [ ] **Step 4: Implement the handlers**

Create `handlers_images.go`:

```go
package main

import (
	"errors"
	"net/http"
	"presenter/images"

	"github.com/gin-gonic/gin"
)

// uploadImage saves one or more uploaded images (multipart field "files"),
// validating each one's format and size via the images package.
func uploadImage(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}
	saved := make([]string, 0, len(files))
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		savedName, err := images.Save(fileHeader.Filename, file)
		file.Close()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		saved = append(saved, savedName)
	}
	c.JSON(http.StatusOK, gin.H{"message": "OK", "names": saved})
}

// listImages returns every uploaded image's file name.
func listImages(c *gin.Context) {
	names, err := images.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"images": names})
}

// getImageContent serves the original file for ?name=.
func getImageContent(c *gin.Context) {
	path, err := images.ContentPath(c.Query("name"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, images.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.File(path)
}

// getImageThumb serves the thumbnail for ?name=.
func getImageThumb(c *gin.Context) {
	path, err := images.ThumbPath(c.Query("name"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, images.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.File(path)
}
```

Remove the old `uploadImage` function (and its `mediaList`-adjacent nothing — it doesn't share code with anything else) from `handlers_api.go`:

```go
func uploadImage(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["files"]
	for _, file := range files {
		log.Println(file.Filename)
		savedName := "./" + storage.BasePath() + "images/" + file.Filename
		c.SaveUploadedFile(file, savedName)
		createThumbnail(savedName)
	}
	c.JSON(http.StatusOK, gin.H{"message": "OK"})
}
```

Check `handlers_api.go`'s remaining functions still use `log`, `os`, `storage` (they do — `moveMedia` uses all three) — no import cleanup needed there.

Remove `createThumbnail` and `getThumbnailDimensions` from `handlers.go`:

```go
func createThumbnail(fileName string) {
	input, _ := os.Open(fileName)
	defer input.Close()
	fmt.Println("input: " + fileName)
	outName := strings.Replace(fileName, "images/", "images/thumbs/", 1)
	outName = strings.Replace(outName, ".JPG", ".png", 1)
	fmt.Println("output: " + outName)
	output, _ := os.Create(outName)
	defer output.Close()
	src, _ := jpeg.Decode(input)
	ratio := 8
	b, h := getThumbnailDimensions(src.Bounds(), ratio)
	dst := image.NewRGBA(image.Rect(0, 0, b, h))
	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	png.Encode(output, dst)
}

func getThumbnailDimensions(rect image.Rectangle, ratio int) (int, int) {
	return rect.Max.X / ratio, rect.Max.Y / ratio
}
```

After removing those two functions, `handlers.go`'s import block loses users for `"fmt"`, `"image"`, `"image/jpeg"`, `"image/png"`, `"strings"`, and `"golang.org/x/image/draw"` — remove all six. The remaining functions (`getAuthAsB64`, `insertAddressOnContent`, `insertAuthTokenOnContent`, `getHtmlPage`, `viewPanel`, `viewController`, `viewHome`) only need `"bytes"`, `"encoding/base64"`, `"io"`, `"log"`, `"net/http"`, `"os"`, and `"github.com/gin-gonic/gin"`. Confirm with `go build ./...` in Step 6 — a leftover unused import fails the build immediately, so this is self-checking.

- [ ] **Step 5: Wire the routes**

In `routes.go`, remove this line from `registerMediaRoutes`:

```go
	router.POST("/api/images", app.AuthMiddleware, uploadImage)
```

and update that function's doc comment (currently "wires up song/image/provider content management") to drop the "image" mention, since image routes move to their own group.

Add a new function:

```go
// registerImageRoutes wires up image upload, listing and retrieval — the
// endpoints the images controller and the live panel use to publish, list
// and read back uploaded images and their thumbnails.
func registerImageRoutes(router *gin.Engine, app *App) {
	router.POST("/api/images", app.AuthMiddleware, uploadImage)
	router.GET("/api/images", listImages)
	router.GET("/api/images/content", getImageContent)
	router.GET("/api/images/thumb", getImageThumb)
}
```

In `main.go`, add the call alongside the other `register*Routes` calls in `main()`:

```go
	registerMediaRoutes(router, app)
	registerImageRoutes(router, app)
	registerProviderRoutes(router, app)
	registerViewRoutes(router, app)
```

- [ ] **Step 6: Update `TestRegisterMediaRoutes` and add `TestRegisterImageRoutes`**

In `routes_test.go`, remove `{"POST", "/api/images"}` from `TestRegisterMediaRoutes`'s `want` slice (and its route-count assertion adjusts automatically since it's `len(want)`).

Add:

```go
func TestRegisterImageRoutes(t *testing.T) {
	withTempWorkDir(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerImageRoutes(router, NewApp(Config{}))

	want := []struct{ method, path string }{
		{"POST", "/api/images"},
		{"GET", "/api/images"},
		{"GET", "/api/images/content"},
		{"GET", "/api/images/thumb"},
	}
	for _, r := range want {
		assertRouteRegistered(t, router, r.method, r.path)
	}
	if got := len(router.Routes()); got != len(want) {
		t.Errorf("got %d routes registered, want %d", got, len(want))
	}
}
```

- [ ] **Step 7: Run `go mod tidy` and the full test suite**

```bash
go mod tidy
go build -v ./...
go test ./... -v
go vet ./...
cd images && go test ./... -v && go vet ./...
```

Expected: everything PASS, `go build`/`go vet` clean in both modules. Also run `git status --short` — no `media/` artifacts should be left in the repo root (root-module image tests all go through `withTempWorkDir`).

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum handlers_images.go handlers_images_test.go handlers_api.go handlers.go routes.go routes_test.go main.go
git commit -m "feat(images): wire upload/list/content/thumb HTTP endpoints, remove old broken code"
```

---

### Task 4: Panel renders `type === 'IMAGE'` as an `<img>`

**Files:**
- Modify: `templates/panels/default.html`

**Interfaces:**
- Consumes: `GET /api/content` response shape `{content, type, contentId}` per provider (existing); `GET /api/images/content?name=` (Task 3) to actually fetch the image bytes the `<img>` tag points at.
- Produces: `processMainProviderData` now renders an `<img>` when `data.type === 'IMAGE'`; a new `lastMainType` module-level variable tracks the last-rendered type so a same-content-different-type transition still re-renders.

There is no automated test harness for this file (plain JS, no test framework in the repo) — this task is verified with a manual smoke check at the end, matching how the rest of the panel is validated today.

- [ ] **Step 1: Add the `lastMainType` state variable**

In `templates/panels/default.html`, find:

```js
        var lastMainText = '';
        var lastAuxText = '';
```

Change to:

```js
        var lastMainText = '';
        var lastMainType = '';
        var lastAuxText = '';
```

- [ ] **Step 2: Add a CSS rule for the rendered image**

Find the `.stroke` rule:

```css
        .stroke {
            -webkit-text-stroke: 2px black;
        }
```

Add immediately after it:

```css
        .live-image {
            max-width: 100%;
            max-height: 100vh;
            object-fit: contain;
        }
```

- [ ] **Step 3: Branch on `data.type` in `processMainProviderData`**

Replace the whole function:

```js
        function processMainProviderData(data) {
            const liveBlock = document.getElementById('liveblock'); 
            const metaBlock = document.getElementById('metablock');
            if (isShowClockMode()) {
                if (data.content === '') {
                    liveBlock.style.backgroundColor = 'transparent';
                } else {
                    liveBlock.style.backgroundColor = 'black';
                }
            }
            if (lastMainText === data.content) {
                return;
            }
            const contentLines = data.content.split('\n');
            const categorized = {
                meta: contentLines.filter(e => e.startsWith('meta:')).map(e => e.replace('meta:', '')),
                live: contentLines.filter(e => !e.startsWith('meta:'))
            }
            const metaContent = categorized.meta.join('\n');
            const liveContent = categorized.live.join('\n');
            lastMainText = data.content;
            liveBlock.innerHTML = processarConteudo(liveContent);
            metaBlock.innerHTML = processarConteudo(metaContent);
            if (data.contentId && data.contentId.startsWith('bible ')) {
                processBibleData(data.contentId);
                toggleVideo({content: 'off'}); // Desliga o vídeo se for um capítulo da Bíblia
            } else {
                // Se não for um capítulo da Bíblia, remove o capítulo
                const bibleChapter = document.querySelector('.bible-chapter');
                bibleChapter.textContent = '';
                toggleVideo({content: 'on'}); // Liga o vídeo para outros conteúdos
            }
        }
```

with:

```js
        function processMainProviderData(data) {
            const liveBlock = document.getElementById('liveblock'); 
            const metaBlock = document.getElementById('metablock');
            if (isShowClockMode()) {
                if (data.content === '') {
                    liveBlock.style.backgroundColor = 'transparent';
                } else {
                    liveBlock.style.backgroundColor = 'black';
                }
            }
            if (lastMainText === data.content && lastMainType === data.type) {
                return;
            }
            lastMainText = data.content;
            lastMainType = data.type;

            if (data.type === 'IMAGE') {
                metaBlock.innerHTML = '';
                liveBlock.innerHTML = data.content
                    ? `<img class="live-image" src="/api/images/content?name=${encodeURIComponent(data.content)}">`
                    : '';
                const bibleChapter = document.querySelector('.bible-chapter');
                bibleChapter.textContent = '';
                toggleVideo({content: 'on'});
                return;
            }

            const contentLines = data.content.split('\n');
            const categorized = {
                meta: contentLines.filter(e => e.startsWith('meta:')).map(e => e.replace('meta:', '')),
                live: contentLines.filter(e => !e.startsWith('meta:'))
            }
            const metaContent = categorized.meta.join('\n');
            const liveContent = categorized.live.join('\n');
            liveBlock.innerHTML = processarConteudo(liveContent);
            metaBlock.innerHTML = processarConteudo(metaContent);
            if (data.contentId && data.contentId.startsWith('bible ')) {
                processBibleData(data.contentId);
                toggleVideo({content: 'off'}); // Desliga o vídeo se for um capítulo da Bíblia
            } else {
                // Se não for um capítulo da Bíblia, remove o capítulo
                const bibleChapter = document.querySelector('.bible-chapter');
                bibleChapter.textContent = '';
                toggleVideo({content: 'on'}); // Liga o vídeo para outros conteúdos
            }
        }
```

- [ ] **Step 4: Manual smoke check**

Run the server (see the full end-to-end smoke check at the end of Task 5 — steps 4 of this task and Task 5 are naturally done together once the controller page exists). For now, at minimum, confirm the panel still works for plain text: build and run the server, open `/live` in a browser, and from a `curl` terminal send `curl -u admin:admin -X POST http://localhost:8080/api/content/set/main -d '{"content":"ola mundo","type":"TEXT"}'` — the panel should still show the text exactly as before this change (regression check, since `processMainProviderData` was rewritten).

- [ ] **Step 5: Commit**

```bash
git add templates/panels/default.html
git commit -m "feat(panel): render type=IMAGE content as an img instead of text"
```

---

### Task 5: `images` controller page and JS module

**Files:**
- Create: `templates/controllers/images.html`
- Create: `static/js/images/client.js`
- Create: `static/js/images/images.js`

**Interfaces:**
- Consumes: `POST /api/images` (multipart, field `files`), `GET /api/images` (`{images: [name, ...]}`), `GET /api/images/thumb?name=`, `POST /api/content/set/main` `{content, type}` (all from Task 3 / existing).
- Produces: a working `/controller/images` page (reachable automatically — `viewController` in `handlers.go` already serves any `templates/controllers/<page>.html` by name, no Go change needed).

No automated test — this is a frontend-only task, verified with a manual smoke check (Step 4) exactly like Task 4's panel change, and the two should be checked together end-to-end.

- [ ] **Step 1: Create the API client module**

Create `static/js/images/client.js`:

```js
function getAuthHeader() {
    const token = document.getElementById('basic-auth-token').value;
    return `Basic ${token}`;
}

export async function uploadImage(file) {
    const formData = new FormData();
    formData.append('files', file);
    const res = await fetch('/api/images', {
        method: 'POST',
        headers: { Authorization: getAuthHeader() },
        body: formData
    });
    if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `upload failed with status ${res.status}`);
    }
    return res.json();
}

export async function fetchImageList() {
    const res = await fetch('/api/images');
    const body = await res.json();
    return body.images || [];
}

export function emitImageContent(name) {
    return fetch('/api/content/set/main', {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: name, type: 'IMAGE' })
    });
}

export function clearMainContent() {
    return fetch('/api/content/set/main', {
        method: 'POST',
        headers: {
            Authorization: getAuthHeader(),
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: '', type: 'TEXT' })
    });
}

export function thumbUrl(name) {
    return `/api/images/thumb?name=${encodeURIComponent(name)}`;
}
```

- [ ] **Step 2: Create the UI wiring module**

Create `static/js/images/images.js`:

```js
import { uploadImage, fetchImageList, emitImageContent, clearMainContent, thumbUrl } from './client.js';

async function renderGallery() {
    const gallery = document.getElementById('gallery');
    gallery.innerHTML = '';
    const names = await fetchImageList();
    for (const name of names) {
        const item = document.createElement('button');
        item.className = 'gallery-item';
        item.type = 'button';
        item.title = name;

        const img = document.createElement('img');
        img.src = thumbUrl(name);
        img.alt = name;
        img.onerror = () => { img.replaceWith(document.createTextNode(name)); };

        item.appendChild(img);
        item.addEventListener('click', () => emitImageContent(name));
        gallery.appendChild(item);
    }
}

function setupUploadForm() {
    const form = document.getElementById('upload-form');
    const input = document.getElementById('file-input');
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        if (!input.files.length) {
            return;
        }
        try {
            await uploadImage(input.files[0]);
            input.value = '';
            await renderGallery();
        } catch (err) {
            alert(err.message);
        }
    });
}

function setupClearButton() {
    document.getElementById('clear-main').addEventListener('click', () => {
        clearMainContent();
    });
}

document.addEventListener('DOMContentLoaded', () => {
    setupUploadForm();
    setupClearButton();
    renderGallery();
});
```

- [ ] **Step 3: Create the controller page**

Create `templates/controllers/images.html`:

```html
<!DOCTYPE html>
<html lang="pt-br">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Presenter - Imagens</title>
    <style>
        * {
            box-sizing: border-box;
        }
        body {
            font-family: sans-serif;
            margin: 0;
            padding: 20px;
            background: #1e1e1e;
            color: #eee;
        }
        h1 {
            font-size: 20px;
            margin-bottom: 16px;
        }
        #upload-form {
            display: flex;
            gap: 8px;
            margin-bottom: 20px;
        }
        button {
            cursor: pointer;
        }
        #clear-main {
            margin-bottom: 20px;
        }
        #gallery {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
            gap: 12px;
        }
        .gallery-item {
            background: #2a2a2a;
            border: 2px solid #444;
            border-radius: 4px;
            padding: 4px;
            cursor: pointer;
        }
        .gallery-item:hover {
            border-color: #888;
        }
        .gallery-item img {
            width: 100%;
            height: 100px;
            object-fit: cover;
            display: block;
        }
    </style>
</head>

<body>
    <input type="hidden" name="basic-auth-token" id="basic-auth-token" value="{{BASIC_AUTH_TOKEN}}">

    <h1>Imagens</h1>

    <form id="upload-form">
        <input type="file" id="file-input" accept="image/*">
        <button type="submit">Enviar</button>
    </form>

    <button id="clear-main">Limpar tela</button>

    <div id="gallery"></div>

    <script type="module" src="/static/js/images/images.js"></script>
</body>

</html>
```

- [ ] **Step 4: Manual smoke check (end-to-end, covers Task 4 too)**

```bash
export ASDF_DATA_DIR=/media/WORK/apps/.asdf
export ASDF_DIR=/media/WORK/apps/asdf
go build -v ./... && ./presenter -usuario admin -senha admin
```

In a browser:
1. Open `http://localhost:8080/controller/images` — the upload form and an empty gallery should appear.
2. Upload one image of each format (JPEG, PNG, WEBP, GIF, BMP) — after each upload the gallery should show a new thumbnail.
3. Upload a non-image file (e.g. a `.txt`) — it should be rejected with a visible error (the `alert()`), and the gallery should not gain an entry.
4. Open `http://localhost:8080/live` in a second tab (or window). Click one of the gallery thumbnails in the controller tab — the image should appear on the panel within the polling interval.
5. Go to `/controller/songs` and send some plain text to `main` — the panel should switch back to showing text (not still show the last image).
6. Back in `/controller/images`, click "Limpar tela" — the panel should go blank.

Stop the server (Ctrl+C). Confirm `git status --short` shows only the files this task is expected to touch (no leftover `media/images/*` files committed — `media/` is gitignored, but double check nothing unexpected got staged).

- [ ] **Step 5: Commit**

```bash
git add templates/controllers/images.html static/js/images/client.js static/js/images/images.js
git commit -m "feat(images): add controller page for uploading and displaying images"
```

---

## Post-implementation

- Update `CLAUDE.md`: the "O que é" section already lists "imagens/vídeos" as a target content type — add a short paragraph (matching the style of the "Providers" section) describing `presenter/images`, the 4 endpoints, the 5 accepted formats, the 10MB limit, and that thumbnails are always PNG.
- Update `TODO.md`: note that image upload/display is now implemented, if the file's "Feito" convention from the previous round is still in use for this kind of tracking.

## Note on this repo's Go toolchain

This machine's Go toolchain is managed by asdf under a non-default path. Every `go` command in every step above needs these exported first, in any shell that hasn't already set them:

```bash
export ASDF_DATA_DIR=/media/WORK/apps/.asdf
export ASDF_DIR=/media/WORK/apps/asdf
```

Without them, `go` fails with "unknown command: go. Perhaps you have to reshim?".
