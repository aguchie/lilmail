package utils

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/nfnt/resize"
)

// IsImage checks if the content type is a supported image format
func IsImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/jpeg") || 
		   strings.HasPrefix(contentType, "image/png")
}

// OptimizeImage resizes and compresses an image
func OptimizeImage(data []byte, maxWidth uint) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Calculate new dimensions
	bounds := img.Bounds()
	if uint(bounds.Dx()) <= maxWidth {
		return data, nil // No resize needed
	}

	// Resize using Lanczos3 for quality
	m := resize.Resize(maxWidth, 0, img, resize.Lanczos3)

	var buf bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, m, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(&buf, m)
	default:
		// Fallback for unsupported formats (e.g. gif), just return original or encode as jpeg?
		// Better to Encode as original format if supported, or JPEG if not.
		// Since we decoded it, the format string is usually "jpeg", "png", "gif".
		// Standard image package supports decoding gif but encoding needs imports.
		// For safety, let's stick to jpeg/png support for optimization.
		// If unknown format that was decoded, return original.
		return data, nil
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
