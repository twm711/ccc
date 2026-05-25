package redact

import "regexp"

var patterns = []*regexp.Regexp{
	// Credit card numbers (13-19 digits, optionally separated by spaces/dashes)
	regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
	// Chinese ID card (18 digits, last may be X)
	regexp.MustCompile(`\b\d{17}[\dXx]\b`),
	// Phone numbers: Chinese mobile (11 digits starting with 1)
	regexp.MustCompile(`\b1[3-9]\d{9}\b`),
	// Email addresses
	regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),
}

// Text replaces sensitive patterns (credit card, ID card, phone, email) with [REDACTED].
func Text(s string) string {
	for _, p := range patterns {
		s = p.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}

var (
	phoneRe = regexp.MustCompile(`\b(1[3-9]\d)\d{4}(\d{4})\b`)
	idRe    = regexp.MustCompile(`\b(\d{6})\d{8}(\d{2}[\dXx]\d?)\b`)
)

// Mask partially masks PII for display (e.g. 138****1234, 110101********1234).
func Mask(s string) string {
	s = phoneRe.ReplaceAllString(s, "${1}****${2}")
	s = idRe.ReplaceAllString(s, "${1}********${2}")
	return s
}
