package validate

import (
	"strings"
)

// builtinTypeNames lists PostgreSQL 17 built-in type names, covering both the
// canonical spellings and the identifier aliases (int4, float8, ...) that reach
// the validator unqualified. Names are lower-case because Postgres folds
// unquoted identifiers.
var builtinTypeNames = []string{
	"bigint", "bigserial", "bit", "bool", "boolean", "box", "bpchar", "bytea",
	"char", "cid", "cidr", "circle", "date", "daterange", "datemultirange", "decimal",
	"float", "float4", "float8",
	"inet", "int", "int2", "int4", "int8", "integer", "interval", "int4range", "int8range", "int4multirange", "int8multirange",
	"json", "jsonb", "jsonpath",
	"line", "lseg",
	"macaddr", "macaddr8", "money",
	"name", "numeric", "numrange", "nummultirange",
	"oid",
	"path", "pg_lsn", "point", "polygon",
	"real", "record", "regclass", "regcollation", "regconfig", "regdictionary", "regnamespace", "regoper", "regoperator",
	"regproc", "regprocedure", "regrole", "regtype",
	"serial", "serial2", "serial4", "serial8", "smallint", "smallserial",
	"text", "tid", "time", "timestamp", "timestamptz", "timetz", "tsquery", "tsrange", "tsvector", "tstzrange", "txid_snapshot",
	"tsmultirange", "tstzmultirange",
	"uuid",
	"varbit", "varchar", "void",
	"xml", "xid", "xid8",
}

var builtinTypeSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(builtinTypeNames))
	for _, name := range builtinTypeNames {
		set[name] = struct{}{}
	}
	return set
}()

func isBuiltinType(name string) bool {
	_, ok := builtinTypeSet[strings.ToLower(name)]
	return ok
}
func suggestTypeName(name string) (string, bool) {
	lower := strings.ToLower(name)
	for _, candidate := range builtinTypeNames {
		if withinOneEdit(lower, candidate) {
			return candidate, true
		}
	}
	return "", false
}

// withinOneEdit reports whether a and b differ by a single substitution,
// a single adjacent transposition, or a single insertion/deletion.
func withinOneEdit(a, b string) bool {
	switch {
	case a == b:
		return false
	case len(a) == len(b):
		diff := 0
		for i := 0; i < len(a); i++ {
			if a[i] != b[i] {
				diff++
			}
		}
		if diff == 1 {
			return true
		}
		for i := 0; i+1 < len(a); i++ {
			if a[i] == b[i+1] && a[i+1] == b[i] && a[:i] == b[:i] && a[i+2:] == b[i+2:] {
				return true
			}
		}
		return false
	case len(a) == len(b)+1:
		return isSingleInsertion(a, b)
	case len(b) == len(a)+1:
		return isSingleInsertion(b, a)
	default:
		return false
	}
}

// isSingleInsertion reports whether long equals short with exactly one extra
// character inserted.
func isSingleInsertion(long, short string) bool {
	i, j := 0, 0
	skipped := false
	for i < len(long) && j < len(short) {
		if long[i] != short[j] {
			if skipped {
				return false
			}
			skipped = true
			i++
			continue
		}
		i++
		j++
	}
	return true
}
