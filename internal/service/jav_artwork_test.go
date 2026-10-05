package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteVerticalPosterCropsRightSide(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "cover.jpg")
	target := filepath.Join(root, "poster.jpg")

	cover := image.NewRGBA(image.Rect(0, 0, 900, 600))
	fillImageRect(cover, image.Rect(0, 0, 500, 600), color.RGBA{R: 255, A: 255})
	fillImageRect(cover, image.Rect(500, 0, 900, 600), color.RGBA{B: 255, A: 255})
	writeTestJPEG(t, source, cover)

	if err := writeVerticalPoster(source, target); err != nil {
		t.Fatalf("write vertical poster: %v", err)
	}
	poster := readTestJPEG(t, target)
	if got := poster.Bounds().Size(); got.X != 400 || got.Y != 600 {
		t.Fatalf("poster size = %dx%d, want 400x600", got.X, got.Y)
	}
	r, _, b, _ := poster.At(10, 300).RGBA()
	if b <= r {
		t.Fatalf("poster did not use right side of sleeve: red=%d blue=%d", r, b)
	}
}

func TestWriteVerticalPosterKeepsPortraitComposition(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "cover.jpg")
	target := filepath.Join(root, "poster.jpg")
	writeTestJPEG(t, source, image.NewRGBA(image.Rect(0, 0, 400, 600)))

	if err := writeVerticalPoster(source, target); err != nil {
		t.Fatalf("write vertical poster: %v", err)
	}
	poster := readTestJPEG(t, target)
	if got := poster.Bounds().Size(); got.X != 400 || got.Y != 600 {
		t.Fatalf("poster size = %dx%d, want 400x600", got.X, got.Y)
	}
}

func TestWriteJavArtworkExportsFanartAndPoster(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "cover.jpg")
	base := filepath.Join(root, "movie")
	writeTestJPEG(t, source, image.NewRGBA(image.Rect(0, 0, 900, 600)))
	sourceData, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	if err := writeJavArtwork(source, base, false); err != nil {
		t.Fatalf("write artwork: %v", err)
	}
	fanartData, err := os.ReadFile(base + "-fanart.jpg")
	if err != nil {
		t.Fatalf("read fanart: %v", err)
	}
	if !bytes.Equal(fanartData, sourceData) {
		t.Fatal("fanart must preserve the original full cover")
	}
	if _, err := os.Stat(base + "-poster.jpg"); err != nil {
		t.Fatalf("generated poster missing: %v", err)
	}
}

func TestWriteJavArtworkPreservesExistingUserPoster(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "cover.jpg")
	base := filepath.Join(root, "movie")
	writeTestJPEG(t, source, image.NewRGBA(image.Rect(0, 0, 900, 600)))
	const existing = "user poster"
	if err := os.WriteFile(base+"-poster.jpg", []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing poster: %v", err)
	}

	if err := writeJavArtwork(source, base, false); err != nil {
		t.Fatalf("write artwork: %v", err)
	}
	data, err := os.ReadFile(base + "-poster.jpg")
	if err != nil {
		t.Fatalf("read existing poster: %v", err)
	}
	if string(data) != existing {
		t.Fatal("existing user poster was overwritten")
	}
}

func fillImageRect(img *image.RGBA, rect image.Rectangle, value color.Color) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.Set(x, y, value)
		}
	}
}

func writeTestJPEG(t *testing.T, path string, img image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create JPEG: %v", err)
	}
	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 95}); err != nil {
		file.Close()
		t.Fatalf("encode JPEG: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close JPEG: %v", err)
	}
}

func readTestJPEG(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open JPEG: %v", err)
	}
	defer file.Close()
	img, err := jpeg.Decode(file)
	if err != nil {
		t.Fatalf("decode JPEG: %v", err)
	}
	return img
}
