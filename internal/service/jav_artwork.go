package service

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

const verticalPosterAspectWidth = 2
const verticalPosterAspectHeight = 3

// writeJavArtwork keeps the downloaded full cover as fanart and writes a
// media-server-friendly vertical poster. Wide JAV sleeve images contain the
// front cover on the right, so the poster uses a right-aligned 2:3 crop.
func writeJavArtwork(coverPath, base string, managed bool) error {
	ext := strings.ToLower(filepath.Ext(coverPath))
	if ext == "" {
		ext = ".jpg"
	}

	fanartPath := base + "-fanart" + ext
	writeFanart, err := canWriteSidecarArtwork(fanartPath, managed)
	if err != nil {
		return fmt.Errorf("inspect fanart: %w", err)
	}
	if writeFanart {
		if err := copyFileAtomically(coverPath, fanartPath); err != nil {
			return fmt.Errorf("write fanart: %w", err)
		}
	}

	posterPath := base + "-poster.jpg"
	writePoster, err := canWriteSidecarArtwork(posterPath, managed)
	if err != nil {
		return fmt.Errorf("inspect poster: %w", err)
	}
	if !writePoster {
		return nil
	}
	if err := writeVerticalPoster(coverPath, posterPath); err == nil {
		return nil
	}

	// Some cover formats are preserved by JavBoss but are not decodable by the
	// standard image package. Keep the previous behavior for those files rather
	// than failing the entire Sidecar operation.
	fallbackPath := base + "-poster" + ext
	writeFallback, err := canWriteSidecarArtwork(fallbackPath, managed)
	if err != nil {
		return fmt.Errorf("inspect fallback poster: %w", err)
	}
	if !writeFallback {
		return nil
	}
	if err := copyFileAtomically(coverPath, fallbackPath); err != nil {
		return fmt.Errorf("write fallback poster: %w", err)
	}
	return nil
}

func canWriteSidecarArtwork(path string, managed bool) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case errorsIsNotExist(err):
		return true, nil
	case err != nil:
		return false, err
	case managed:
		return true, nil
	default:
		return false, nil
	}
}

func errorsIsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func writeVerticalPoster(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	decoded, _, err := image.Decode(input)
	if err != nil {
		return fmt.Errorf("decode cover: %w", err)
	}
	bounds := decoded.Bounds()
	if bounds.Empty() {
		return fmt.Errorf("decode cover: empty image")
	}

	crop := bounds
	if bounds.Dx() >= bounds.Dy() {
		width := bounds.Dy() * verticalPosterAspectWidth / verticalPosterAspectHeight
		if width < 1 {
			width = 1
		}
		if width < bounds.Dx() {
			crop.Min.X = bounds.Max.X - width
		}
	}

	poster := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(poster, poster.Bounds(), decoded, crop.Min, draw.Src)

	var output bytes.Buffer
	if err := jpeg.Encode(&output, poster, &jpeg.Options{Quality: 90}); err != nil {
		return fmt.Errorf("encode poster: %w", err)
	}
	if err := writeFileAtomically(target, &output, 0o644); err != nil {
		return fmt.Errorf("save poster: %w", err)
	}
	return nil
}
