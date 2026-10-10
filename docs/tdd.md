# TDD record

The user requests a local, single-command CLI with tests for main behavior,
boundaries, and failure paths. The agreed public seam is the CLI process:
arguments, stdout, stderr, exit status, and unchanged input files. Tests create
real fixture directories inside this project and execute the real binary.
No Docker daemon, external service, private implementation calls, or mocks.

Each behavior is added as one failing test, followed by the smallest implementation
that passes the suite. Actual red/green transcripts are saved in `docs/test-runs/`.
Go build caches and temporary directories are explicitly kept in this project.

The first red run failed because there was no production Go source yet.
Every subsequent red run failed an observable CLI assertion. No green transcript
is a cached test result. Cycles 2–21 use `go test -count=1 ./...`; cycle 1 used
`go test ./...`. Each run set PATH to the project-local Go toolchain and set
GOCACHE, GOPATH, GOTMPDIR, TMPDIR, and GOTOOLCHAIN locally.

Boundary audit: those early Go invocations did **not** explicitly isolate the Go
telemetry directory. The Go source shows that the default may write local
counters under the user's configuration directory. Whether such files were
written was not checked; no outside files were inspected or cleaned up. Final
Makefile verification isolates XDG_CONFIG_HOME and TEST_TELEMETRY_DIR, writes
`off` only to the project's telemetry mode file, disables GOENV, and disables
dependency downloads. No global configuration was changed deliberately.

## Actual red/green cycles

| Cycle | Public behavior | Red | Green |
| --- | --- | --- | --- |
| 01 | `TestExcludedDirectoryExplainsRuleLine` | [FAIL](test-runs/01-red.txt) | [PASS](test-runs/01-green.txt) |
| 02 | `TestNegationReincludesFileBelowExcludedParent` | [FAIL](test-runs/02-red.txt) | [PASS](test-runs/02-green.txt) |
| 03 | `TestBatchDoubleStarReinclusionKeepsExcludedSiblings` | [FAIL](test-runs/03-red.txt) | [PASS](test-runs/03-green.txt) |
| 04 | `TestRootGlobDoesNotExcludeNestedFiles` | [FAIL](test-runs/04-red.txt) | [PASS](test-runs/04-green.txt) |
| 05 | `TestMissingIgnoreFileIncludesWithoutCreatingFiles` | [FAIL](test-runs/05-red.txt) | [PASS](test-runs/05-green.txt) |
| 06 | `TestContextRootIsNeverExcluded` | [FAIL](test-runs/06-red.txt) | [PASS](test-runs/06-green.txt) |
| 07 | `TestOnlyFirstLineBOMIsRemovedAndOriginalLinesSurvive` | [FAIL](test-runs/07-red.txt) | [PASS](test-runs/07-green.txt) |
| 08 | `TestPathsAreNormalizedRelativeToContext` | [FAIL](test-runs/08-red.txt) | [PASS](test-runs/08-green.txt) |
| 09 | `TestInvalidPathsRejectWholeBatchWithoutPartialOutput` | [FAIL](test-runs/09-red.txt) | [PASS](test-runs/09-green.txt) |
| 10 | `TestMissingArgumentsShowUsageWithoutPanic` | [FAIL](test-runs/10-red.txt) | [PASS](test-runs/10-green.txt) |
| 11 | `TestHelpExplainsExitCodesAndRelativePaths` | [FAIL](test-runs/11-red.txt) | [PASS](test-runs/11-green.txt) |
| 12 | `TestDockerfileSpecificRulesReplaceRootRules` | [FAIL](test-runs/12-red.txt) | [PASS](test-runs/12-green.txt) |
| 13 | `TestDefaultDockerfileUsesItsOwnIgnoreFile` | [FAIL](test-runs/13-red.txt) | [PASS](test-runs/13-green.txt) |
| 14 | `TestContextMustBeAnExistingDirectory` | [FAIL](test-runs/14-red.txt) | [PASS](test-runs/14-green.txt) |
| 15 | `TestInvalidPatternNamesSourceAndLineWithoutOutput` | [FAIL](test-runs/15-red.txt) | [PASS](test-runs/15-green.txt) |
| 16 | `TestDockerfilePathMustStayInsideContext` | [FAIL](test-runs/16-red.txt) | [PASS](test-runs/16-green.txt) |
| 17 | `TestIgnoreSymlinkCannotReadOutsideContext` | [FAIL](test-runs/17-red.txt) | [PASS](test-runs/17-green.txt) |
| 18 | `TestOversizedIgnoreLineFailsWithSourceLocation` | [FAIL](test-runs/18-red.txt) | [PASS](test-runs/18-green.txt) |
| 19 | `TestSpecialIgnoreFileIsRejectedWithoutBlocking` | [FAIL](test-runs/19-red.txt) | [PASS](test-runs/19-green.txt) |
| 20 | `TestIgnoreFileNamesWithSpacesAreQuoted` | [FAIL](test-runs/20-red.txt) | [PASS](test-runs/20-green.txt) |
| 21 | `TestHelpTakesPrecedenceOverPositionalArguments` | [FAIL](test-runs/21-red.txt) | [PASS](test-runs/21-green.txt) |

## Final verification

Executed once after formatting and vendoring:

```sh
PATH="$PWD/.tools/go/bin:$PATH" make test vet build
```

This ran `go test -count=1 -v ./...`, `go vet ./...`, and
`go build -trimpath -o bin/ignoretrail .` with vendored dependencies and Go network
downloads disabled. Result: **21 top-level tests passed**, including three path
subtests; `ok ignoretrail 9.575s`. Vet and build exited 0.
[Full transcript](test-runs/final.txt).

Both documented example commands were then executed against the built binary.
Their complete stdout and exit codes (0 and 1) matched the README exactly.
[Example transcript](test-runs/examples.txt). No Docker daemon or external
service was used. No feature work or repeated acceptance run follows DONE.
