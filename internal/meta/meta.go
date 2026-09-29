// Package meta masks psql-style meta-commands out of SQL source before it
// reaches the parser.
package meta

import (
	"bytes"
)

// Mask returns a copy of src in which every meta-command is blanked to spaces.
// src is returned unchanged when there is nothing to mask.
func Mask(src []byte) []byte {
	out := src
	copied := false

	for i := 0; i < len(src); {
		c := src[i]

		if c == '\\' {
			if !copied {
				out = make([]byte, len(src))
				copy(out, src)
				copied = true
			}
			// psql's \; and \: escapes inject a literal ';' or ':' into the
			// query. Blank the backslash and keep the character, so the masked
			// source carries the same SQL psql would send.
			if i+1 < len(src) && (src[i+1] == ';' || src[i+1] == ':') {
				out[i] = ' '
				i++
				continue
			}
			// Otherwise a meta-command runs to the end of the line.
			for i < len(src) && src[i] != '\n' {
				out[i] = ' '
				i++
			}
			continue
		}

		switch {
		case c == '\'':
			i = skipSingleQuoted(src, i)
		case c == '"':
			i = skipDoubleQuoted(src, i)
		case c == '-' && i+1 < len(src) && src[i+1] == '-':
			i = skipLineComment(src, i)
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			i = skipBlockComment(src, i)
		case c == '$':
			if end, ok := skipDollarQuoted(src, i); ok {
				i = end
			} else {
				i++
			}
		default:
			i++
		}
	}

	return out
}

// skipSingleQuoted returns the index just past the closing quote of the string
// literal starting at src[i] (a single quote). A doubled quote (two adjacent
// single quotes) is an escaped quote and does not close the literal. An
// unterminated literal runs to the end of the input.
func skipSingleQuoted(src []byte, i int) int {
	for j := i + 1; j < len(src); j++ {
		if src[j] != '\'' {
			continue
		}
		if j+1 < len(src) && src[j+1] == '\'' {
			j++ // skip the escaped quote
			continue
		}
		return j + 1
	}
	return len(src)
}

// skipDoubleQuoted returns the index just past the closing quote of the quoted
// identifier starting at src[i] (a double quote). A doubled quote ("") is an
// escaped quote and does not close the identifier.
func skipDoubleQuoted(src []byte, i int) int {
	for j := i + 1; j < len(src); j++ {
		if src[j] != '"' {
			continue
		}
		if j+1 < len(src) && src[j+1] == '"' {
			j++
			continue
		}
		return j + 1
	}
	return len(src)
}

// skipLineComment returns the index of the newline that ends the -- comment
// starting at src[i], or len(src) if it is unterminated.
func skipLineComment(src []byte, i int) int {
	for j := i + 2; j < len(src); j++ {
		if src[j] == '\n' {
			return j
		}
	}
	return len(src)
}

// skipBlockComment returns the index just past the closing */ of the block
// comment starting at src[i]. PostgreSQL block comments nest, so the depth is
// tracked. An unterminated comment runs to the end of the input.
func skipBlockComment(src []byte, i int) int {
	depth := 1
	for j := i + 2; j < len(src); {
		switch {
		case src[j] == '/' && j+1 < len(src) && src[j+1] == '*':
			depth++
			j += 2
		case src[j] == '*' && j+1 < len(src) && src[j+1] == '/':
			depth--
			j += 2
			if depth == 0 {
				return j
			}
		default:
			j++
		}
	}
	return len(src)
}

// skipDollarQuoted handles the dollar-quoted string starting at src[i] (a
// dollar sign). It returns the index just past the closing delimiter and true
// when src[i:] opens a valid $tag$...$tag$ literal, or (0, false) when it does
// not (for example a $1 parameter placeholder).
func skipDollarQuoted(src []byte, i int) (int, bool) {
	k := i + 1
	for k < len(src) && isDollarTagChar(src[k]) {
		k++
	}
	// A tag must not start with a digit, so $1 is a parameter, not a quote.
	if k > i+1 && src[i+1] >= '0' && src[i+1] <= '9' {
		return 0, false
	}
	if k >= len(src) || src[k] != '$' {
		return 0, false
	}

	delim := src[i : k+1]
	if idx := bytes.Index(src[k+1:], delim); idx >= 0 {
		return k + 1 + idx + len(delim), true
	}
	return len(src), true
}

// isDollarTagChar reports whether b can appear in a dollar-quote tag.
func isDollarTagChar(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}
