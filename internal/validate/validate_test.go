package validate

import (
	"strings"
	"testing"

	"github.com/serhii-chechun/pg-query-validate/internal/diagnostic"
	"github.com/serhii-chechun/pg-query-validate/internal/meta"

	pgq "github.com/pganalyze/pg_query_go/v6"
)

func TestValidateValidSQL(t *testing.T) {
	src := "create table t (id uuid primary key); select a, count(*) from t group by a;"
	tree, err := pgq.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if diags := validate(tree, []byte(src)); len(diags) != 0 {
		t.Fatalf("validate returned findings for valid SQL: %v", diags)
	}
}

func TestValidateMissingStatement(t *testing.T) {
	tree := &pgq.ParseResult{Stmts: []*pgq.RawStmt{nil, {}}}
	diags := validate(tree, nil)
	if len(diags) != 2 {
		t.Fatalf("got %d findings, want 2: %v", len(diags), diags)
	}
	for i, want := range []string{"statement 1: missing statement", "statement 2: missing statement node"} {
		if !strings.Contains(diags[i].Message, want) {
			t.Errorf("finding %d = %q, want it to contain %q", i, diags[i].Message, want)
		}
	}
}

func TestValidateEmptyNodeTopLevel(t *testing.T) {
	tree := &pgq.ParseResult{Stmts: []*pgq.RawStmt{{Stmt: &pgq.Node{}}}}
	diags := validate(tree, nil)
	if len(diags) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "statement 1.stmt: empty node") {
		t.Errorf("unexpected finding: %q", diags[0].Message)
	}
}

// TestValidateEmptyNodeNested proves the walk descends into list fields.
func TestValidateEmptyNodeNested(t *testing.T) {
	tree := &pgq.ParseResult{Stmts: []*pgq.RawStmt{{
		Stmt: &pgq.Node{Node: &pgq.Node_SelectStmt{SelectStmt: &pgq.SelectStmt{
			TargetList: []*pgq.Node{{}},
		}}},
	}}}
	diags := validate(tree, nil)
	if len(diags) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "target_list[0]: empty node") {
		t.Errorf("unexpected finding: %q", diags[0].Message)
	}
}

func TestValidateNilTree(t *testing.T) {
	if diags := validate(nil, nil); len(diags) != 1 {
		t.Fatalf("got %d findings, want 1 for nil parse result", len(diags))
	}
}

func TestValidateTypeNameTypo(t *testing.T) {
	src := "create table t (name VARCHAT(255) NOT NULL);"
	tree, err := pgq.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diags := validate(tree, []byte(src))
	if len(diags) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(diags), diags)
	}
	// Unquoted identifiers are folded to lower case by the parser, so the
	// reported name is lower-case even though the input was VARCHAT.
	for _, want := range []string{`unknown type "varchat"`, `did you mean "varchar"`} {
		if !strings.Contains(diags[0].Message, want) {
			t.Errorf("finding %q does not contain %q", diags[0].Message, want)
		}
	}
	// The finding must point at the VARCHAT token so it can be underlined.
	if diags[0].Offset != 21 || diags[0].Length != 7 {
		t.Errorf("span = (%d, %d), want (21, 7)", diags[0].Offset, diags[0].Length)
	}
}

func TestValidateTypeNameTypoInCast(t *testing.T) {
	src := "select '1'::INTGER;"
	tree, err := pgq.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if diags := validate(tree, []byte(src)); len(diags) == 0 {
		t.Fatal("expected a finding for a misspelled type name in a cast")
	}
}

func TestValidateTypeNameAcceptsValid(t *testing.T) {
	queries := []string{
		"create table t (a int4, b float8, c timestamptz, d bool);",
		"create table t (a pg_catalog.int4);",
		"create table t (a varchar(255), b numeric(14, 2), c uuid);",
		"create table t (a changezap_status);",
		"create table t (a myschema.weird_type);",
	}
	for _, q := range queries {
		tree, err := pgq.Parse(q)
		if err != nil {
			t.Fatalf("parse %q: %v", q, err)
		}
		if diags := validate(tree, []byte(q)); len(diags) != 0 {
			t.Errorf("validate %q returned findings: %v", q, diags)
		}
	}
}

func TestWriteDiagnosticsUnderlinesToken(t *testing.T) {
	src := "create table t (name VARCHAT(255));"
	tree, err := pgq.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var out strings.Builder
	diagnostic.Write(&out, "test.sql", []byte(src), validate(tree, []byte(src)))

	want := "error: unknown type \"varchat\" (did you mean \"varchar\"?)\n" +
		" --> test.sql:1:22\n" +
		"  |\n" +
		"1 | create table t (name VARCHAT(255));\n" +
		"  | " + strings.Repeat(" ", 21) + "^^^^^^^\n"
	if out.String() != want {
		t.Errorf("rendered diagnostics mismatch\n got:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestWriteDiagnosticsReportsCorrectLine(t *testing.T) {
	src := "create table t (\n    a int,\n    name VARCHAT(255)\n);\n"
	tree, err := pgq.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var out strings.Builder
	diagnostic.Write(&out, "m.sql", []byte(src), validate(tree, []byte(src)))

	for _, want := range []string{
		" --> m.sql:3:10\n",
		"3 |     name VARCHAT(255)\n",
		"  |          ^^^^^^^\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("rendered diagnostics missing %q\n got:\n%s", want, out.String())
		}
	}
}

func TestWriteDiagnosticsWithoutSourceSpan(t *testing.T) {
	var out strings.Builder
	diagnostic.Write(&out, "m.sql", nil, []diagnostic.Results{{Message: "statement 1: missing statement", Offset: -1}})
	if want := "error: statement 1: missing statement\n"; out.String() != want {
		t.Errorf("rendered = %q, want %q", out.String(), want)
	}
}

// TestValidateWithMetaCommands parses masked source and checks that a typo in
// the surrounding SQL is still reported, with an offset that refers to the
// original (unmasked) source.
func TestValidateWithMetaCommands(t *testing.T) {
	src := "\\echo building schema\ncreate table t (name VARCHAT(255));\n"
	masked := meta.Mask([]byte(src))
	tree, err := pgq.Parse(string(masked))
	if err != nil {
		t.Fatalf("parse masked source: %v", err)
	}

	diags := validate(tree, []byte(src))
	if len(diags) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(diags), diags)
	}

	wantOffset := strings.Index(src, "VARCHAT")
	if diags[0].Offset != wantOffset || diags[0].Length != 7 {
		t.Errorf("span = (%d, %d), want (%d, 7)", diags[0].Offset, diags[0].Length, wantOffset)
	}

	var out strings.Builder
	diagnostic.Write(&out, "m.sql", []byte(src), diags)
	if want := " --> m.sql:2:22\n"; !strings.Contains(out.String(), want) {
		t.Errorf("rendered diagnostics missing %q\n got:\n%s", want, out.String())
	}
}

// TestValidateMetaOnlyFile confirms a file of nothing but meta commands parses
// cleanly and produces no findings.
func TestValidateMetaOnlyFile(t *testing.T) {
	src := "\\echo one\n\\@echo two\n"
	tree, err := pgq.Parse(string(meta.Mask([]byte(src))))
	if err != nil {
		t.Fatalf("parse meta-only source: %v", err)
	}
	if diags := validate(tree, []byte(src)); len(diags) != 0 {
		t.Errorf("validate returned findings for meta-only source: %v", diags)
	}
}

// TestValidateMetaCommandSharesLineWithSQL covers a meta-command following SQL
// on the same line, e.g. "select 1; \echo done".
func TestValidateMetaCommandSharesLineWithSQL(t *testing.T) {
	src := "select 1; \\echo done\ncreate table t (name VARCHAT(255));\n"
	tree, err := pgq.Parse(string(meta.Mask([]byte(src))))
	if err != nil {
		t.Fatalf("parse masked source: %v", err)
	}

	diags := validate(tree, []byte(src))
	if len(diags) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(diags), diags)
	}

	wantOffset := strings.Index(src, "VARCHAT")
	if diags[0].Offset != wantOffset {
		t.Errorf("offset = %d, want %d", diags[0].Offset, wantOffset)
	}

	var out strings.Builder
	diagnostic.Write(&out, "m.sql", []byte(src), diags)
	if want := " --> m.sql:2:22\n"; !strings.Contains(out.String(), want) {
		t.Errorf("rendered diagnostics missing %q\n got:\n%s", want, out.String())
	}
}
