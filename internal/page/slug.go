package page

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

const maxSlugLength = 128

// GenerateSlug returns a random 16-character hex slug.
func GenerateSlug() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate slug: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// ValidateSlug checks only length constraints for user-provided slugs.
func ValidateSlug(slug string) error {
	switch {
	case slug == "":
		return errors.New("slug must not be empty")
	case len(slug) > maxSlugLength:
		return fmt.Errorf("slug must be %d characters or fewer", maxSlugLength)
	default:
		return nil
	}
}
