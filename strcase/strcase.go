// SPDX-License-Identifier: BSD-3-Clause

package strcase

import (
	"strings"
	"unicode"
)

type Mode int

const (
	_ Mode = iota
	FlatCase
	CamelCase
	SnakeCase
	KebabCase

	// noSep is a special internal separator value for specific cases (e.g. flatcase)
	noSep = rune(0)
	// maxWordLength limits iterator over a string when looking for any separator
	maxWordLength = 100
)

type Converter struct {
	s    string
	mode Mode
}

func Convert(s string) Converter {
	return Converter{
		mode: Detect(s),
		s:    s,
	}
}

func (c Converter) To(target Mode) string {
	return c.mode.ConvertTo(target, c.s)
}

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

func Detect(s string) Mode {
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

func toCamel(s string, sep rune) string {
	if s == "" {
		return ""
	}

	rs := []rune(s)
	sb := &strings.Builder{}
	sb.Grow(len(rs))

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
	return sb.String()
}

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

func replaceSep(s string, fromSep, toSep rune) string {
	if fromSep == noSep {
		// flat case or other non-delimited string
		return s
	}

	sb := &strings.Builder{}
	sb.Grow(len([]rune(s)))

	for _, r := range s {
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

	return sb.String()
}
