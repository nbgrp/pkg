// SPDX-License-Identifier: BSD-3-Clause

// Package strcase provides detection of the case of a string and conversion
// between the supported case styles.
package strcase

import (
	"strings"
	"unicode"
)

// Mode is a case style of a string.
type Mode int

const (
	_ Mode = iota
	// FlatCase is a lowercase string without any separator, e.g. "helloworld".
	// It is also the fallback mode, used when the case of a string cannot be
	// detected.
	FlatCase
	// CamelCase is a string with capitalized word boundaries and no separator,
	// e.g. "helloWorld".
	CamelCase
	// SnakeCase is a lowercase string separated by underscores, e.g. "hello_world".
	SnakeCase
	// KebabCase is a lowercase string separated by dashes, e.g. "hello-world".
	KebabCase
)

const (
	// noSep is a special internal separator value for specific cases (e.g. flatcase)
	noSep = rune(0)
	// maxWordLength limits iterator over a string when looking for any separator
	maxWordLength = 100
)

// Converter converts a string from the case it was detected in to another one.
type Converter struct {
	s    string
	mode Mode
}

// Convert detects the case of the given string and returns a converter for it.
//
// Use [Converter.To] to convert the string to another case.
func Convert(s string) Converter {
	return Converter{
		mode: Detect(s),
		s:    s,
	}
}

// To converts the string to the given target case.
func (c Converter) To(target Mode) string {
	return c.mode.ConvertTo(target, c.s)
}

// ConvertTo converts the string s from the c case to the target case.
//
// Converting a string to the case it already has returns it unchanged.
// [FlatCase] means that the case of the string is unknown, so converting from
// it is a no-op as well.
//
// The leading and the trailing separators of the string are standalone symbols,
// not the word boundaries, so they are carried over as they are: they are
// neither replaced nor removed.
func (c Mode) ConvertTo(target Mode, s string) string {
	switch {
	case c == target, c == FlatCase:
		return s
	case c == CamelCase:
		return fromCamel(s, target.sep())
	case target == CamelCase:
		return toCamel(s, c.sep())
	default:
		return replaceSep(s, c.sep(), target.sep())
	}
}

// sep returns the separator rune of the mode.
func (c Mode) sep() rune {
	switch c {
	case SnakeCase:
		return '_'
	case KebabCase:
		return '-'
	default:
		return noSep
	}
}

// Detect returns the case of the given string.
//
// The detection is based on the separators and on the presence of both lower
// and upper case letters: a string containing an underscore is a snake case
// one, a string containing a dash is a kebab case one, and a string with both
// lower and upper case letters is a camel case one.
//
// The leading and the trailing separators are standalone symbols, not the word
// boundaries, so they are not taken into account: the case is detected by the
// rest of the string.
//
// Only the first 100 bytes of the string are inspected, so that the detection
// stays cheap on long strings.
//
// An empty string, as well as a string without any case marker, is detected as
// [FlatCase].
func Detect(s string) Mode {
	if s == "" {
		return FlatCase // FlatCase used as a fallback value
	}

	_, core, _ := splitEdges(s)
	return detectCore(core)
}

// detectCore detects the case of a string without the leading and the trailing
// separators.
func detectCore(s string) Mode {
	if s == "" {
		return FlatCase // FlatCase used as a fallback value
	}

	lowerFound := false
	upperFound := false

	for i, r := range s {
		if i > maxWordLength {
			// the word is too long
			break
		}

		switch {
		case r == '_':
			return SnakeCase
		case r == '-':
			return KebabCase
		case !lowerFound || !upperFound:
			isLower := unicode.IsLower(r)
			isUpper := unicode.IsUpper(r)

			if isUpper && !isLower {
				upperFound = true
			} else if !isUpper && isLower {
				lowerFound = true
			}
		}
	}

	if lowerFound && upperFound {
		// hump found
		return CamelCase
	}
	return FlatCase
}

// splitEdges splits the leading and the trailing separators of the string from
// the rest of it.
//
// The separators at the edges are standalone symbols: they separate nothing, so
// they are kept as they are by the conversions.
func splitEdges(s string) (prefix, core, suffix string) {
	start := 0
	for start < len(s) && isSep(s[start]) {
		start++
	}

	end := len(s)
	for end > start && isSep(s[end-1]) {
		end--
	}

	return s[:start], s[start:end], s[end:]
}

// isSep reports whether the byte is a separator character.
//
// Only the ASCII separators are checked, so the byte-wise scanning of a UTF-8
// string is safe.
func isSep(b byte) bool {
	return b == '_' || b == '-'
}

// toCamel removes the separator from the string and capitalizes the letter
// that follows it.
//
// The leading and the trailing separators are standalone symbols, not the word
// boundaries, so they are kept as they are.
func toCamel(s string, sep rune) string {
	if s == "" {
		return ""
	}

	prefix, core, suffix := splitEdges(s)

	rs := []rune(core)
	sb := &strings.Builder{}
	sb.Grow(len(rs) + len(prefix) + len(suffix))

	sb.WriteString(prefix)

	upperNext := false
	for _, r := range rs {
		if r == sep {
			upperNext = true
			continue
		}

		if upperNext {
			sb.WriteRune(unicode.ToUpper(r))
			upperNext = false
		} else {
			sb.WriteRune(r)
		}
	}

	sb.WriteString(suffix)

	return sb.String()
}

// fromCamel splits the camel case string into words and joins them with the
// given separator, lowercasing the result.
//
// The abbreviations are handled: "ReadJSONData" becomes "read_json_data".
func fromCamel(s string, sep rune) string {
	if s == "" {
		return ""
	}
	if sep == noSep {
		// FlatCase is assumed
		return strings.ToLower(s)
	}

	rs := []rune(s)
	sb := &strings.Builder{}
	sb.Grow(int(float32(len(rs)) * 1.2)) //nolint:mnd // 1.2x capacity should be enough for everyone

	var (
		prev      rune
		prevUpper bool
		separated bool
	)

	for i, r := range rs {
		curUpper := unicode.IsUpper(r)
		switch {
		case i == 0: // first letter doesn't mean anything
		case i > 1 && !curUpper && prevUpper:
			if !separated {
				sb.WriteRune(sep)
			}
			sb.WriteRune(prev)
			separated = false
		case curUpper && !prevUpper:
			sb.WriteRune(prev)
			if !separated {
				sb.WriteRune(sep)
				separated = true
			}
		default:
			sb.WriteRune(prev)
			separated = false
		}

		prev = unicode.ToLower(r)
		prevUpper = curUpper
	}
	sb.WriteRune(prev)

	return sb.String()
}

// replaceSep replaces the fromSep separator with the toSep one.
//
// The leading and the trailing separators are standalone symbols, not the word
// boundaries, so they are kept as they are.
//
// A noSep source separator means that the string has no separators at all and
// is returned as is; a noSep target separator removes the separators.
func replaceSep(s string, fromSep, toSep rune) string {
	if fromSep == noSep {
		// flat case or other non-delimited string
		return s
	}

	prefix, core, suffix := splitEdges(s)

	sb := &strings.Builder{}
	sb.Grow(len(s))

	sb.WriteString(prefix)

	for _, r := range core {
		if r == fromSep {
			if toSep == noSep {
				// flat case
				continue
			}
			sb.WriteRune(toSep)
		} else {
			sb.WriteRune(r)
		}
	}

	sb.WriteString(suffix)

	return sb.String()
}
