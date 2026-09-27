package validate

import (
	"strings"
	"testing"
)

func TestIsBuiltinType(t *testing.T) {
	tests := []struct {
		label string
		input string
		want  bool
	}{
		{label: "alias int4", input: "int4", want: true},
		{label: "canonical varchar", input: "varchar", want: true},
		{label: "canonical boolean", input: "boolean", want: true},
		{label: "uuid", input: "uuid", want: true},
		{label: "timestamptz", input: "timestamptz", want: true},
		{label: "tsvector", input: "tsvector", want: true},
		{label: "underscore name pg_lsn", input: "pg_lsn", want: true},
		{label: "numeric multirange", input: "nummultirange", want: true},

		// Postgres folds unquoted identifiers, so lookup is case-insensitive.
		{label: "upper case INT4", input: "INT4", want: true},
		{label: "mixed case VarChar", input: "VarChar", want: true},
		{label: "upper case BOOLEAN", input: "BOOLEAN", want: true},

		// Misspellings are not built-ins; that is what the typo check catches.
		{label: "misspelled varchat", input: "varchat", want: false},
		{label: "misspelled boolen", input: "boolen", want: false},

		// User-defined and schema-qualified names are not built-ins.
		{label: "user-defined type", input: "changezap_status", want: false},
		{label: "qualified builtin", input: "pg_catalog.int4", want: false},
		{label: "empty", input: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := isBuiltinType(tt.input); got != tt.want {
				t.Errorf("isBuiltinType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSuggestTypeName(t *testing.T) {
	// suggestTypeName is only reached for names that are not built-ins (see
	// validateTypeName), so built-in inputs are deliberately not covered here.
	tests := []struct {
		label  string
		input  string
		want   string
		wantOK bool
	}{
		{label: "substitution", input: "varchat", want: "varchar", wantOK: true},
		{label: "upper case folds", input: "VARCHAT", want: "varchar", wantOK: true},
		{label: "insertion", input: "intger", want: "integer", wantOK: true},
		{label: "deletion", input: "varchr", want: "varchar", wantOK: true},
		{label: "transposition", input: "tsvetcor", want: "tsvector", wantOK: true},
		{label: "missing letter boolean", input: "boolen", want: "boolean", wantOK: true},

		{label: "two edits", input: "nubmer", want: "", wantOK: false},
		{label: "unrelated name", input: "changezap_status", want: "", wantOK: false},
		{label: "single letter", input: "x", want: "", wantOK: false},
		{label: "empty", input: "", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got, ok := suggestTypeName(tt.input)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("suggestTypeName(%q) = (%q, %v), want (%q, %v)",
					tt.input, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestWithinOneEdit(t *testing.T) {
	tests := []struct {
		label string
		a     string
		b     string
		want  bool
	}{
		{label: "substitution", a: "varchat", b: "varchar", want: true},
		{label: "substitution reversed", a: "varchar", b: "varchat", want: true},
		{label: "insertion", a: "intger", b: "integer", want: true},
		{label: "insertion at end", a: "textt", b: "text", want: true},
		{label: "deletion", a: "varchr", b: "varchar", want: true},
		{label: "adjacent transposition", a: "varchra", b: "varchar", want: true},
		{label: "adjacent transposition tsvector", a: "tsvetcor", b: "tsvector", want: true},
		{label: "insertion into empty", a: "", b: "a", want: true},

		{label: "identical", a: "varchar", b: "varchar", want: false},
		{label: "both empty", a: "", b: "", want: false},
		{label: "two substitutions", a: "nubmer", b: "numeric", want: false},
		{label: "two non-adjacent swaps", a: "abcd", b: "cbad", want: false},
		{label: "length differs by three", a: "ab", b: "abcde", want: false},
		{label: "unrelated names", a: "changezap_status", b: "varchar", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := withinOneEdit(tt.a, tt.b); got != tt.want {
				t.Errorf("withinOneEdit(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsSingleInsertion(t *testing.T) {
	// long must be exactly one character longer than short; withinOneEdit
	// enforces that precondition before calling this helper.
	tests := []struct {
		label string
		long  string
		short string
		want  bool
	}{
		{label: "extra letter in middle", long: "integer", short: "intger", want: true},
		{label: "extra letter near end", long: "varchar", short: "varchr", want: true},
		{label: "extra letter in middle of varchar", long: "varchar", short: "vachar", want: true},
		{label: "single character pair", long: "a", short: "", want: true},
		{label: "extra letter at start", long: "ba", short: "a", want: true},

		{label: "two letters removed", long: "integer", short: "intgr", want: false},
		{label: "different suffix", long: "varchar", short: "vachr", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := isSingleInsertion(tt.long, tt.short); got != tt.want {
				t.Errorf("isSingleInsertion(%q, %q) = %v, want %v", tt.long, tt.short, got, tt.want)
			}
		})
	}
}

func TestBuiltinTypeNamesMatchIndex(t *testing.T) {
	seen := make(map[string]struct{}, len(builtinTypeNames))
	for _, name := range builtinTypeNames {
		if name != strings.ToLower(name) {
			t.Errorf("builtinTypeNames contains %q; lookup folds identifiers to lower case", name)
		}
		if _, dup := seen[name]; dup {
			t.Errorf("builtinTypeNames contains duplicate entry %q", name)
		}
		seen[name] = struct{}{}

		if _, ok := builtinTypeSet[name]; !ok {
			t.Errorf("builtinTypeSet is missing %q", name)
		}
	}

	if len(builtinTypeSet) != len(seen) {
		t.Errorf("builtinTypeSet has %d entries, want %d", len(builtinTypeSet), len(seen))
	}
}
