# pg-query-validate

`pgqv` is a standalone query validator and linter for PostgreSQL. Point it at a
`.sql` file and it reports problems with rustc-style diagnostics: the offending
line, a `file:line:column` location, and a caret underline.

It works entirely offline - no PostgreSQL server, no database connection.

```console
$ pgqv schema.sql
error: unknown type "varchat" (did you mean "varchar"?)
 --> schema.sql:3:22
  |
3 |     name             VARCHAT(255) NOT NULL,
  |                      ^^^^^^^
```

## What it checks

- **Syntax.** The file is parsed with the PostgreSQL grammar. Anything that does
  not parse is reported with its location.
- **Misspelled type names.** PostgreSQL accepts *any* identifier as a type name
  and only resolves it when the statement runs, so `VARCHAT(255)` is valid
  syntax but fails on a real server. `pgqv` flags unqualified type names that
  are one edit away from a built-in type (`varchat` -> `varchar`), which catches
  typos without false positives on your own user-defined types.

### Meta commands

psql treats an unquoted backslash as the start of a meta-command (`\echo`,
`\@echo`, `\i`, ...) and consumes input through the end of that line, so a
meta-command may follow SQL on the same line (`select 1; \echo done`).

These are not SQL and would otherwise be reported as syntax errors, so `pgqv`
blanks the command text before parsing and validates only the surrounding SQL.
A backslash inside a string, quoted identifier, dollar-quoted body, or comment
does not start a meta-command and is left alone. Meta commands are recognised,
not executed, and the source is only blanked in memory, so the line and column
numbers in diagnostics still refer to the original file.

psql's `\;` and `\:` escapes are honoured: the backslash is dropped and the
`;` or `:` it protects is kept. Multi-line meta-command continuation (a line
ending in a backslash) is not handled.

## Usage

```console
pgqv <filename.sql>
```

Exactly one file is expected. Running `pgqv` with no arguments prints the usage
banner and exits non-zero. There are currently no flags, and input is not read
from stdin.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | No problems found |
| `1` | Problems found, or the file could not be read or parsed |

This makes `pgqv` straightforward to wire into a pre-commit hook or CI step.

## Installation

### Homebrew (macOS and Linux)

The formula is hosted in this repository, so there is no separate
`homebrew-tap` repository to add:

```console
$ brew tap serhii-chechun/pg-query-validate https://github.com/serhii-chechun/pg-query-validate
$ brew install pgqv
```

Tapping the main repository is what lets Homebrew find `Formula/pgqv.rb`. If the
short name is ambiguous, use the fully qualified form:
`brew install serhii-chechun/pg-query-validate/pgqv`. Remove the tap with
`brew untap serhii-chechun/pg-query-validate`.

The formula installs the prebuilt release binary for your platform, so no
compiler is needed.

### Prebuilt binaries

Prebuilt archives are attached to each
[release](https://github.com/serhii-chechun/pg-query-validate/releases/latest).
They need no compiler and no other dependencies.

| Platform | Archive |
| --- | --- |
| macOS, Apple Silicon | `pgqv_1.0.1_darwin_arm64.tar.gz` |
| macOS, Intel | `pgqv_1.0.1_darwin_amd64.tar.gz` |
| Linux, x86-64 | `pgqv_1.0.1_linux_amd64.tar.gz` |
| Linux, arm64 | `pgqv_1.0.1_linux_arm64.tar.gz` |
| Windows, x86-64 | `pgqv_1.0.1_windows_amd64.zip` |

Checksums for every archive are published alongside them in `SHA256SUMS`.

**macOS and Linux**

```console
$ tar -xzf pgqv_1.0.1_linux_amd64.tar.gz
$ sudo install -m 755 pgqv /usr/local/bin/pgqv
```

**Windows** - extract the `.zip` and put `pgqv.exe` on your `PATH`.

Confirm it runs:

```console
$ pgqv
PostgreSQL Query Validator v1.0.1 (c) 2026, Serhii Chechun
Usage: pgqv <filename.sql>
```

On macOS you may need to allow the unsigned binary under
*System Settings -> Privacy & Security* the first time you run it. Building from
source (below) avoids that.

### With `go install`

```console
go install github.com/serhii-chechun/pg-query-validate/cmd/pgqv@v1.0.1
```

The binary is placed in `$(go env GOPATH)/bin` (usually `~/go/bin`), which must
be on your `PATH`.

This requires **Go 1.27.1 or newer** and **a C compiler**, because the parser is
built with cgo. See [Build notes](#build-notes).

### From source

```console
git clone https://github.com/serhii-chechun/pg-query-validate.git
cd pg-query-validate
go build -o pgqv ./cmd/pgqv
```

## Build notes

`pgqv` embeds the PostgreSQL query parser (libpg_query) as C code, so building
has two consequences:

- **A C compiler is required.** cgo cannot be disabled: `CGO_ENABLED=0` fails,
  because the parser has no pure-Go implementation. On macOS install the Xcode
  Command Line Tools; on Debian/Ubuntu `build-essential`; on Alpine `build-base`.
- **The first build is slow.** It compiles the bundled PostgreSQL C sources and
  can take a few minutes. Later builds are cached. (use build -x to see the progress)

Cross-compiling needs a C cross-toolchain for the target platform - a plain
`GOOS=linux go build` from macOS will not work.

The resulting binary is self-contained: it links only against the system C
library, and needs no PostgreSQL installation or `libpg_query` at runtime.
Prebuilt release binaries therefore also run on machines without a compiler.

The Linux binaries are dynamically linked against glibc, so they run on any
reasonably recent glibc distribution but not on musl-based systems such as
Alpine. Build from source there instead.

## Limitations

- **PostgreSQL 17 grammar.** Parsing uses libpg_query 17 (via
  `pg_query_go` v6.2.2). Syntax introduced in PostgreSQL 18 is not recognised
  yet and will be reported as a syntax error.
- **No database connection.** Because nothing is resolved against a live
  catalog, `pgqv` cannot check table or column names, function signatures, or
  whether a type you define elsewhere actually exists. The type check is
  deliberately limited to near-misses of built-in type names.
- **One file per run.** Directories, globs, and multiple arguments are not
  supported.

## Development

```console
go test ./...
gofmt -l ./cmd ./internal
go vet ./...
```

To build the release archives locally, run:

```console
./scripts/build-release.sh
```

The macOS archives are built natively and the Linux and Windows ones inside a
container, so docker or podman must be installed. Archives and `SHA256SUMS` are
written to `dist/`. Tagged pushes build and publish the same archives through
[`.github/workflows/release.yml`](.github/workflows/release.yml).

## License

Apache License 2.0 - see [LICENSE](LICENSE).
