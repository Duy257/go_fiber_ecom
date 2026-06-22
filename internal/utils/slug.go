package utils

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]`)
var multiHyphenRegex = regexp.MustCompile(`-{2,}`)

// GenerateSlug normalizes text to URL-safe slug.
// Handles Unicode (Vietnamese, etc.) via NFD decomposition + diacritics removal.
func GenerateSlug(name string) string {
	// NFD decompose, then remove combining diacritical marks (Mn category)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)))
	result, _, _ := transform.String(t, name)

	slug := strings.ToLower(strings.TrimSpace(result))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonAlphanumericRegex.ReplaceAllString(slug, "")
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "untitled"
	}
	return slug
}
