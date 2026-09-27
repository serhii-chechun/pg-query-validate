package diagnostic

import (
	"strings"
	"testing"
)

func TestExpandTabs(t *testing.T) {
	tests := []struct {
		label string
		input string
		want  string
	}{
		{label: "no tabs", input: "no tabs here", want: "no tabs here"},
		{label: "empty", input: "", want: ""},
		{label: "single tab", input: "\t", want: " "},
		{label: "tab between words", input: "a\tb", want: "a b"},
		{label: "consecutive tabs", input: "a\t\tb", want: "a  b"},
		{label: "leading tab", input: "\tindented", want: " indented"},
		{label: "trailing tab", input: "trailing\t", want: "trailing "},
		{label: "preserves multibyte runes", input: "héllo\twörld", want: "héllo wörld"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := expandTabs([]byte(tt.input)); got != tt.want {
				t.Errorf("expandTabs(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsIdentByte(t *testing.T) {
	tests := []struct {
		label string
		input byte
		want  bool
	}{
		{label: "underscore", input: '_', want: true},
		{label: "dollar sign", input: '$', want: true},
		{label: "lower a", input: 'a', want: true},
		{label: "lower z", input: 'z', want: true},
		{label: "upper A", input: 'A', want: true},
		{label: "upper Z", input: 'Z', want: true},
		{label: "digit 0", input: '0', want: true},
		{label: "digit 9", input: '9', want: true},

		{label: "space", input: ' ', want: false},
		{label: "open paren", input: '(', want: false},
		{label: "dot", input: '.', want: false},
		{label: "dash", input: '-', want: false},
		{label: "newline", input: '\n', want: false},
		{label: "at sign", input: '@', want: false},
		{label: "quoted string delimiter", input: '"', want: false},
		{label: "non-ASCII byte", input: 0xC3, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := isIdentByte(tt.input); got != tt.want {
				t.Errorf("isIdentByte(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLineAt(t *testing.T) {
	const multi = "line one\nline two\nline three" // len 28
	const crlf = "a\r\nb"                          // len 4

	tests := []struct {
		label  string
		src    string
		offset int
		want   sourceLine
	}{
		{label: "first line start", src: multi, offset: 0, want: sourceLine{1, 0, 8}},
		{label: "first line middle", src: multi, offset: 5, want: sourceLine{1, 0, 8}},
		{label: "first line last char", src: multi, offset: 7, want: sourceLine{1, 0, 8}},
		{label: "second line start", src: multi, offset: 9, want: sourceLine{2, 9, 17}},
		{label: "second line middle", src: multi, offset: 12, want: sourceLine{2, 9, 17}},
		{label: "second line last char", src: multi, offset: 16, want: sourceLine{2, 9, 17}},
		{label: "third line start", src: multi, offset: 18, want: sourceLine{3, 18, 28}},
		{label: "third line last char", src: multi, offset: 27, want: sourceLine{3, 18, 28}},
		{label: "exactly at end of source", src: multi, offset: 28, want: sourceLine{3, 18, 28}},
		{label: "past end of source clamps", src: multi, offset: 1000, want: sourceLine{3, 18, 28}},

		{label: "single line", src: "select 1", offset: 3, want: sourceLine{1, 0, 8}},
		{label: "empty source", src: "", offset: 0, want: sourceLine{1, 0, 0}},

		// CRLF: the trailing carriage return is excluded from the line.
		{label: "crlf first line strips carriage return", src: crlf, offset: 0, want: sourceLine{1, 0, 1}},
		{label: "crlf second line", src: crlf, offset: 3, want: sourceLine{2, 3, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := lineAt([]byte(tt.src), tt.offset); got != tt.want {
				t.Errorf("lineAt(%q, %d) = %+v, want %+v", tt.src, tt.offset, got, tt.want)
			}
		})
	}
}

func TestIdentifierSpan(t *testing.T) {
	const stmt = "create table t (name VARCHAT(255));" // VARCHAT at offset 21

	tests := []struct {
		label      string
		src        string
		offset     int
		name       string
		wantOffset int
		wantLength int
	}{
		{label: "exact match", src: stmt, offset: 21, name: "varchat", wantOffset: 21, wantLength: 7},
		{label: "case-insensitive match", src: stmt, offset: 21, name: "VARCHAT", wantOffset: 21, wantLength: 7},
		{label: "stops at non-identifier byte", src: stmt, offset: 21, name: "varch", wantOffset: 21, wantLength: 7},
		{label: "extends to full identifier run", src: "type x1$y", offset: 5, name: "x1", wantOffset: 5, wantLength: 4},
		{label: "stops at space", src: "\tvarchar ", offset: 1, name: "varchar", wantOffset: 1, wantLength: 7},
		{label: "underscore included in run", src: "a my_type_b", offset: 2, name: "my", wantOffset: 2, wantLength: 9},
		{label: "single byte source", src: "a", offset: 0, name: "a", wantOffset: 0, wantLength: 1},

		{label: "negative offset", src: stmt, offset: -1, name: "varchat", wantOffset: -1, wantLength: 0},
		{label: "offset at end of source", src: "abc", offset: 3, name: "a", wantOffset: -1, wantLength: 0},
		{label: "offset past end of source", src: "abc", offset: 10, name: "a", wantOffset: -1, wantLength: 0},
		{label: "empty source", src: "", offset: 0, name: "a", wantOffset: -1, wantLength: 0},
		{label: "empty name", src: stmt, offset: 21, name: "", wantOffset: -1, wantLength: 0},
		{label: "name longer than remaining source", src: "abc", offset: 1, name: "bcdef", wantOffset: -1, wantLength: 0},
		{label: "offset points at other text", src: stmt, offset: 0, name: "varchar", wantOffset: -1, wantLength: 0},
		{label: "name crosses identifier boundary", src: stmt, offset: 21, name: "varcharx", wantOffset: -1, wantLength: 0},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			gotOffset, gotLength := IdentifierSpan([]byte(tt.src), tt.offset, tt.name)
			if gotOffset != tt.wantOffset || gotLength != tt.wantLength {
				t.Errorf("IdentifierSpan(%q, %d, %q) = (%d, %d), want (%d, %d)",
					tt.src, tt.offset, tt.name, gotOffset, gotLength, tt.wantOffset, tt.wantLength)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	const typoDiag = `unknown type "varchat" (did you mean "varchar"?)`

	tests := []struct {
		label string
		name  string
		src   string
		diags []Results
		want  string
	}{
		{
			label: "underlines the offending token",
			name:  "test.sql",
			src:   "create table t (name VARCHAT(255));",
			diags: []Results{{Message: typoDiag, Offset: 21, Length: 7}},
			want: "error: " + typoDiag + "\n" +
				" --> test.sql:1:22\n" +
				"  |\n" +
				"1 | create table t (name VARCHAT(255));\n" +
				"  | " + strings.Repeat(" ", 21) + "^^^^^^^\n",
		},
		{
			label: "reports a two-digit line with a wider gutter",
			name:  "m.sql",
			src:   strings.Repeat("select 1;\n", 9) + "create table t (name VARCHAT(255));\n",
			diags: []Results{{Message: typoDiag, Offset: 9*len("select 1;\n") + 21, Length: 7}},
			want: "error: " + typoDiag + "\n" +
				"  --> m.sql:10:22\n" +
				"   |\n" +
				"10 | create table t (name VARCHAT(255));\n" +
				"   | " + strings.Repeat(" ", 21) + "^^^^^^^\n",
		},
		{
			label: "tabs are rendered as one space so carets stay aligned",
			name:  "t.sql",
			src:   "a\tVARCHAT",
			diags: []Results{{Message: "bad type", Offset: 2, Length: 7}},
			want: "error: bad type\n" +
				" --> t.sql:1:3\n" +
				"  |\n" +
				"1 | a VARCHAT\n" +
				"  |   ^^^^^^^\n",
		},
		{
			label: "caret span is clamped to the end of the line",
			name:  "c.sql",
			src:   "abc",
			diags: []Results{{Message: "bad type", Offset: 1, Length: 100}},
			want: "error: bad type\n" +
				" --> c.sql:1:2\n" +
				"  |\n" +
				"1 | abc\n" +
				"  |  ^^\n",
		},
		{
			label: "zero-length span still draws one caret",
			name:  "z.sql",
			src:   "abc",
			diags: []Results{{Message: "bad type", Offset: 1, Length: 0}},
			want: "error: bad type\n" +
				" --> z.sql:1:2\n" +
				"  |\n" +
				"1 | abc\n" +
				"  |  ^\n",
		},
		{
			// Degenerate input: the offset is past the end of the source, so
			// the clamped span is zero and no caret is drawn.
			label: "offset past end of source draws no caret",
			name:  "e.sql",
			src:   "abc",
			diags: []Results{{Message: "m", Offset: 1000, Length: 4}},
			want: "error: m\n" +
				" --> e.sql:1:4\n" +
				"  |\n" +
				"1 | abc\n" +
				"  |    \n",
		},
		{
			label: "findings without a source span print the message only",
			name:  "n.sql",
			src:   "abc",
			diags: []Results{{Message: "statement 1: missing statement", Offset: -1}},
			want:  "error: statement 1: missing statement\n",
		},
		{
			label: "findings are separated by a blank line",
			name:  "d.sql",
			src:   "line one\nline two\n",
			diags: []Results{
				{Message: "first", Offset: 0, Length: 4},
				{Message: "second", Offset: 9, Length: 3},
			},
			want: "error: first\n" +
				" --> d.sql:1:1\n" +
				"  |\n" +
				"1 | line one\n" +
				"  | ^^^^\n" +
				"\n" +
				"error: second\n" +
				" --> d.sql:2:1\n" +
				"  |\n" +
				"2 | line two\n" +
				"  | ^^^\n",
		},
		{
			label: "no findings write nothing",
			name:  "none.sql",
			src:   "select 1;",
			diags: nil,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			var out strings.Builder
			Write(&out, tt.name, []byte(tt.src), tt.diags)
			if got := out.String(); got != tt.want {
				t.Errorf("Write() mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestWriteDoesNotMutateFindings(t *testing.T) {
	diags := []Results{{Message: "m", Offset: 1000, Length: 4}}
	Write(&strings.Builder{}, "e.sql", []byte("abc"), diags)
	if diags[0].Offset != 1000 {
		t.Errorf("Write mutated the caller's finding: offset = %d, want 1000", diags[0].Offset)
	}
}
