package diagnostic

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Results is a single validation finding. When offset is non-negative it
// marks the byte range in the source that the finding applies to; a negative
// offset means the finding has no source position (an internal tree invariant).
type Results struct {
	Message string
	Offset  int
	Length  int
}

// sourceLine describes the line of source containing a byte offset.
type sourceLine struct {
	number int // 1-based
	start  int // offset of the first byte of the line
	end    int // offset just past the last byte, excluding the newline
}

// lineAt returns the line of src containing offset.
func lineAt(src []byte, offset int) sourceLine {
	if offset > len(src) {
		offset = len(src)
	}

	line := sourceLine{number: 1}
	for i := 0; i < offset; i++ {
		if src[i] == '\n' {
			line.number++
			line.start = i + 1
		}
	}

	line.end = line.start
	for line.end < len(src) && src[line.end] != '\n' {
		line.end++
	}
	for line.end > line.start && src[line.end-1] == '\r' {
		line.end--
	}
	return line
}

// Write renders findings in a rustc-like format: the message, a
// file:line:column location, and the offending line with a caret underline.
func Write(w io.Writer, name string, src []byte, diags []Results) {
	for i, d := range diags {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "error: %s\n", d.Message)
		if d.Offset < 0 {
			continue
		}

		line := lineAt(src, d.Offset)
		if d.Offset > line.end {
			d.Offset = line.end
		}

		gutter := strings.Repeat(" ", len(strconv.Itoa(line.number)))
		column := d.Offset - line.start + 1
		fmt.Fprintf(w, "%s--> %s:%d:%d\n", gutter, name, line.number, column)
		fmt.Fprintf(w, "%s |\n", gutter)
		fmt.Fprintf(w, "%d | %s\n", line.number, expandTabs(src[line.start:line.end]))

		span := d.Length
		if span < 1 {
			span = 1
		}
		if span > line.end-d.Offset {
			span = line.end - d.Offset
		}
		// Tabs are rendered as a single space above, so one space per source
		// byte keeps the carets aligned.
		fmt.Fprintf(w, "%s | %s%s\n",
			gutter,
			strings.Repeat(" ", d.Offset-line.start),
			strings.Repeat("^", span),
		)
	}
}

// expandTabs replaces each tab with a single space. Substituting one character
// for one keeps caret alignment intact.
func expandTabs(b []byte) string {
	return strings.ReplaceAll(string(b), "\t", " ")
}

// isIdentByte reports whether b can appear in an unquoted identifier.
func isIdentByte(b byte) bool {
	return b == '_' || b == '$' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

// IdentifierSpan locates the identifier name in src at offset, returning the
// span of the full identifier run. The parser records byte offsets for tokens,
// but it only records them for the names it recognises, so the offset is
// verified against the name before being trusted.
func IdentifierSpan(src []byte, offset int, name string) (int, int) {
	if offset < 0 || offset >= len(src) || name == "" {
		return -1, 0
	}
	if offset+len(name) > len(src) || !strings.EqualFold(string(src[offset:offset+len(name)]), name) {
		return -1, 0
	}
	end := offset
	for end < len(src) && isIdentByte(src[end]) {
		end++
	}
	return offset, end - offset
}
