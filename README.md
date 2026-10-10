# ignoretrail

Explain which `.dockerignore` lines exclude or re-include a path, with one local command.

No Docker daemon, login, network access, or source-file execution. The command
reads the applicable ignore file and leaves the project unchanged.

[中文说明与讲解](docs/guide.zh-CN.md) · [Problem and public sources](docs/problem.md)

## Install

Requires Go 1.24+ and Make. Tested on Linux amd64 with Go 1.27.2.
The Moby matching library is pinned and vendored, so building needs no network.

```sh
make build
./bin/ignoretrail --help
```

The binary is installed only in this project's `bin/` directory. Run it by its
absolute path from another project. In the delivered workspace, Go is also
available locally: `PATH="$PWD/.tools/go/bin:$PATH" make build`.

## Smallest example

The bundled `examples/negation/.dockerignore` excludes `sub/`, then re-includes
`sub/b.txt`:

```sh
./bin/ignoretrail examples/negation sub/b.txt
```

```text
INCLUDED "sub/b.txt"
  exclude .dockerignore:2 "sub/"
  include .dockerignore:3 "!sub/b.txt"
```

To check both files in one command:

```sh
./bin/ignoretrail examples/negation sub/a.txt sub/b.txt
```

This prints both explanations and exits **1**, because `sub/a.txt` is excluded.
An unrelated `!` rule never disables the other exclusions.

## Usage

```text
ignoretrail [-f Dockerfile] CONTEXT PATH...
```

- Both `PATH` and `-f` are relative to `CONTEXT`, not the current directory.
  Paths are literal; the tool does not expand path globs.
- `-f` defaults to `Dockerfile`. `<Dockerfile>.dockerignore` takes precedence
  over the root `.dockerignore`; the files are not merged. If neither exists,
  paths are included. An empty specific ignore file also takes precedence.
- For example: `ignoretrail -f docker/build.Dockerfile . src/app.py`.
  Flags must come before positional arguments.
- Output lists **changes in inclusion state**, in order. A redundant rule is
  omitted. Line numbers refer to the original ignore file, including comments
  and blank lines. Paths and rule text are quoted; special source names are too.
- Exit codes: **0** all paths included (or help); **1** at least one excluded;
  **2** invalid arguments, invalid rules, or file errors. Errors go to stderr.

## Limits

- This answers Docker ignore-rule questions. It does not repair Docker SDK or
  LangGraph bugs, modify rules, build images, or guarantee a successful build.
- It does not verify queried paths exist, inspect their contents, follow their
  symlinks, expand `COPY`/`ADD`, evaluate build stages, or enumerate archives.
  Docker's special handling of transferring Dockerfiles/ignore files is outside
  this rule-only answer. Named/remote contexts are not supported.
- The Dockerfile itself is not parsed or required to exist. Specify `-f` when
  using a name other than `Dockerfile`, including a lowercase `dockerfile`.
- Ignore files must be regular files. Links within the context are allowed;
  links escaping it are rejected. Do not change ignore files while querying.
- Rules use Moby patternmatcher **v0.6.1**, with its standard line-length limit
  (approximately 64 KiB). Errors identify the source and line.
- Intended for a few paths and modest ignore files: explaining successive rule
  prefixes takes O(paths × rules²) matching work, in addition to path depth.
- Linux is the verified platform. The tests use POSIX FIFOs; Windows support
  has not been validated.

## Tests

```sh
make test
make vet
```

`make` keeps caches and temporary files inside this directory and disables Go
network downloads. Tests build and invoke the real binary against local fixture
directories. The 21 TDD cycles cover negation, directory inheritance, `**`, root
globs, source selection, BOM/line numbers, path normalization, quoting, missing
files, invalid arguments/patterns, escaping symlinks, oversized lines, and FIFOs.

See [the red/green record](docs/tdd.md) and [actual transcripts](docs/test-runs/).

## License

MIT for original project code. Vendored Moby code retains its Apache-2.0 license.
