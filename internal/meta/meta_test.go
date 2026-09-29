package meta

import (
	"strings"
	"testing"
)

func spaces(n int) string { return strings.Repeat(" ", n) }

func TestMask(t *testing.T) {
	tests := []struct {
		label string
		src   string
		want  string
	}{
		{
			label: "no meta commands",
			src:   "select 1;\n",
			want:  "select 1;\n",
		},
		{
			label: "plain meta line is blanked",
			src:   "\\echo hi\nselect 1;\n",
			want:  spaces(8) + "\nselect 1;\n",
		},
		{
			label: "at-sign meta form is blanked",
			src:   "\\@echo hi\nselect 1;\n",
			want:  spaces(9) + "\nselect 1;\n",
		},
		{
			label: "leading whitespace before meta marker",
			src:   "   \\echo hi\nselect 1;\n",
			want:  spaces(11) + "\nselect 1;\n",
		},
		{
			label: "indented after a blank line",
			src:   "\n  \\echo x\n",
			want:  "\n" + spaces(9) + "\n",
		},
		{
			label: "meta line at end without trailing newline",
			src:   "select 1;\n\\echo done",
			want:  "select 1;\n" + spaces(10),
		},
		{
			label: "consecutive meta lines",
			src:   "\\echo a\n\\echo b\n",
			want:  spaces(7) + "\n" + spaces(7) + "\n",
		},
		{
			label: "meta line between statements",
			src:   "select 1;\n\\echo mid\nselect 2;\n",
			want:  "select 1;\n" + spaces(9) + "\nselect 2;\n",
		},
		{
			label: "CRLF line ending is blanked with the line",
			src:   "\\echo hi\r\nselect 1;\r\n",
			want:  spaces(9) + "\nselect 1;\r\n",
		},
		{
			label: "backslash inside single-quoted string is left alone",
			src:   "select '\n\\x\n';\n",
			want:  "select '\n\\x\n';\n",
		},
		{
			label: "escaped quote does not close the string",
			src:   "select 'a''b';\n\\echo x\n",
			want:  "select 'a''b';\n" + spaces(7) + "\n",
		},
		{
			label: "backslash inside double-quoted identifier is left alone",
			src:   "select \"\n\\x\n\";\n",
			want:  "select \"\n\\x\n\";\n",
		},
		{
			label: "backslash inside dollar-quoted body is left alone",
			src:   "select $$\n\\x\n$$;\n",
			want:  "select $$\n\\x\n$$;\n",
		},
		{
			label: "tagged dollar-quoted body is left alone",
			src:   "select $tag$\n\\i file.sql\n$tag$;\n",
			want:  "select $tag$\n\\i file.sql\n$tag$;\n",
		},
		{
			label: "parameter placeholder is not a dollar quote",
			src:   "select $1;\n\\echo z\n",
			want:  "select $1;\n" + spaces(7) + "\n",
		},
		{
			label: "backslash inside line comment is left alone",
			src:   "-- \\echo\nselect 1;\n",
			want:  "-- \\echo\nselect 1;\n",
		},
		{
			label: "backslash inside nested block comment is left alone",
			src:   "/* \\echo /* \\i */ */\nselect 1;\n",
			want:  "/* \\echo /* \\i */ */\nselect 1;\n",
		},
		{
			label: "meta command after SQL on the same line",
			src:   "select 1; \\echo done\n",
			want:  "select 1; " + spaces(10) + "\n",
		},
		{
			label: "meta command mid-statement blanked to end of line",
			src:   "select 1 \\echo\n",
			want:  "select 1 " + spaces(5) + "\n",
		},
		{
			label: "backslash-semicolon escape keeps the semicolon",
			src:   "select 1\\;\n",
			want:  "select 1 ;\n",
		},
		{
			label: "backslash-colon escape keeps the colon",
			src:   "arr[1\\:2]\n",
			want:  "arr[1 :2]\n",
		},
		{
			label: "escape then meta command on the same line",
			src:   "select 1\\; \\echo x\n",
			want:  "select 1 ;" + spaces(8) + "\n",
		},
		{
			label: "escape inside a string is left alone",
			src:   "select '\\;';\n",
			want:  "select '\\;';\n",
		},
		{
			label: "backslash inside a string on a SQL line is left alone",
			src:   "select 'a; \\echo';\n",
			want:  "select 'a; \\echo';\n",
		},
		{
			label: "empty input",
			src:   "",
			want:  "",
		},
		{
			label: "only whitespace",
			src:   "  \t \n",
			want:  "  \t \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got := Mask([]byte(tt.src))
			if string(got) != tt.want {
				t.Errorf("Mask(%q) = %q, want %q", tt.src, got, tt.want)
			}
			if len(got) != len(tt.src) {
				t.Errorf("Mask(%q) changed length: got %d, want %d", tt.src, len(got), len(tt.src))
			}
		})
	}
}

// TestMaskUnchangedInputIsReturnedAsIs documents that nothing is allocated when
// there is no meta command to blank.
func TestMaskUnchangedInputIsReturnedAsIs(t *testing.T) {
	src := []byte("select 1;\n")
	got := Mask(src)
	if &got[0] != &src[0] {
		t.Error("Mask returned a copy for input without meta commands")
	}
}

func TestSkipSingleQuoted(t *testing.T) {
	tests := []struct {
		label string
		src   string
		i     int
		want  int
	}{
		{label: "simple literal", src: "'abc'", i: 0, want: 5},
		{label: "empty literal", src: "''", i: 0, want: 2},
		{label: "only the opening quote", src: "'", i: 0, want: 1},
		{label: "doubled quote is escaped", src: "'a''b'", i: 0, want: 6},
		{label: "only escaped quotes then close", src: "''''", i: 0, want: 4},
		{label: "newline inside literal", src: "'a\nb'", i: 0, want: 5},
		{label: "start offset mid-input", src: "x 'ab' y", i: 2, want: 6},
		{label: "trailing bytes after literal are ignored", src: "'ab' rest", i: 0, want: 4},
		{label: "unterminated runs to end", src: "'abc", i: 0, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := skipSingleQuoted([]byte(tt.src), tt.i); got != tt.want {
				t.Errorf("skipSingleQuoted(%q, %d) = %d, want %d", tt.src, tt.i, got, tt.want)
			}
		})
	}
}

func TestSkipDoubleQuoted(t *testing.T) {
	tests := []struct {
		label string
		src   string
		i     int
		want  int
	}{
		{label: "simple identifier", src: "\"abc\"", i: 0, want: 5},
		{label: "empty identifier", src: "\"\"", i: 0, want: 2},
		{label: "only the opening quote", src: "\"", i: 0, want: 1},
		{label: "doubled quote is escaped", src: "\"a\"\"b\"", i: 0, want: 6},
		{label: "newline inside identifier", src: "\"a\nb\"", i: 0, want: 5},
		{label: "start offset mid-input", src: "x \"ab\" y", i: 2, want: 6},
		{label: "unterminated runs to end", src: "\"abc", i: 0, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := skipDoubleQuoted([]byte(tt.src), tt.i); got != tt.want {
				t.Errorf("skipDoubleQuoted(%q, %d) = %d, want %d", tt.src, tt.i, got, tt.want)
			}
		})
	}
}

func TestSkipLineComment(t *testing.T) {
	tests := []struct {
		label string
		src   string
		i     int
		want  int
	}{
		{label: "returns the newline index", src: "-- hi\nselect", i: 0, want: 5},
		{label: "empty comment", src: "--\n", i: 0, want: 2},
		{label: "crlf returns the newline index", src: "-- hi\r\nx", i: 0, want: 6},
		{label: "unterminated returns len", src: "-- hi", i: 0, want: 5},
		{label: "only dashes to end", src: "--", i: 0, want: 2},
		{label: "start offset mid-input", src: "x -- y\nz", i: 2, want: 6},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := skipLineComment([]byte(tt.src), tt.i); got != tt.want {
				t.Errorf("skipLineComment(%q, %d) = %d, want %d", tt.src, tt.i, got, tt.want)
			}
		})
	}
}

func TestSkipBlockComment(t *testing.T) {
	tests := []struct {
		label string
		src   string
		i     int
		want  int
	}{
		{label: "simple comment", src: "/* x */ y", i: 0, want: 7},
		{label: "empty comment", src: "/**/", i: 0, want: 4},
		{label: "star before close", src: "/* **/ x", i: 0, want: 6},
		{label: "newline inside comment", src: "/* a\nb */", i: 0, want: 9},
		{label: "nested comment", src: "/* a /* b */ c */ d", i: 0, want: 17},
		{label: "start offset mid-input", src: "x /* y */ z", i: 2, want: 9},
		{label: "unterminated runs to end", src: "/* abc", i: 0, want: 6},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := skipBlockComment([]byte(tt.src), tt.i); got != tt.want {
				t.Errorf("skipBlockComment(%q, %d) = %d, want %d", tt.src, tt.i, got, tt.want)
			}
		})
	}
}

func TestSkipDollarQuoted(t *testing.T) {
	tests := []struct {
		label  string
		src    string
		i      int
		want   int
		wantOK bool
	}{
		{label: "empty tag", src: "$$abc$$", i: 0, want: 7, wantOK: true},
		{label: "empty tag empty body", src: "$$$$", i: 0, want: 4, wantOK: true},
		{label: "tagged body", src: "$tag$abc$tag$", i: 0, want: 13, wantOK: true},
		{label: "tag with digits after letter", src: "$a1$x$a1$", i: 0, want: 9, wantOK: true},
		{label: "unterminated empty tag runs to end", src: "$$abc", i: 0, want: 5, wantOK: true},
		{label: "unterminated tagged runs to end", src: "$t$abc", i: 0, want: 6, wantOK: true},
		{label: "start offset mid-input", src: "x$$y$$z", i: 1, want: 6, wantOK: true},

		{label: "digit-only tag is a parameter", src: "$1", i: 0, want: 0, wantOK: false},
		{label: "digit tag with closing dollar is a parameter", src: "$1$", i: 0, want: 0, wantOK: false},
		{label: "tag without closing dollar", src: "$x", i: 0, want: 0, wantOK: false},
		{label: "dollar then space", src: "$ abc", i: 0, want: 0, wantOK: false},
		{label: "lone dollar at end", src: "$", i: 0, want: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got, ok := skipDollarQuoted([]byte(tt.src), tt.i)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("skipDollarQuoted(%q, %d) = (%d, %v), want (%d, %v)",
					tt.src, tt.i, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestIsDollarTagChar(t *testing.T) {
	tests := []struct {
		label string
		input byte
		want  bool
	}{
		{label: "underscore", input: '_', want: true},
		{label: "lower a", input: 'a', want: true},
		{label: "lower z", input: 'z', want: true},
		{label: "upper A", input: 'A', want: true},
		{label: "upper Z", input: 'Z', want: true},
		{label: "digit 0", input: '0', want: true},
		{label: "digit 9", input: '9', want: true},

		{label: "dollar sign", input: '$', want: false},
		{label: "space", input: ' ', want: false},
		{label: "dash", input: '-', want: false},
		{label: "dot", input: '.', want: false},
		{label: "quote", input: '\'', want: false},
		{label: "non-ASCII byte", input: 0xC3, want: false},
		{label: "nul", input: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := isDollarTagChar(tt.input); got != tt.want {
				t.Errorf("isDollarTagChar(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
