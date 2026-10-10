package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

type result struct {
	code int
	out  string
	err  string
}

func invoke(t *testing.T, args ...string) result {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "ignoretrail")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return result{code, stdout.String(), stderr.String()}
}

func fixture(t *testing.T, rules string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".dockerignore"), []byte(rules), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func expect(t *testing.T, got result, code int, stdout string) {
	t.Helper()
	if got.code != code || got.out != stdout || got.err != "" {
		t.Fatalf("got code=%d stdout=%q stderr=%q; want code=%d stdout=%q", got.code, got.out, got.err, code, stdout)
	}
}

func TestExcludedDirectoryExplainsRuleLine(t *testing.T) {
	root := fixture(t, "# generated assets\nbuild/\n")
	expect(t, invoke(t, root, "build/app.js"), 1,
		"EXCLUDED \"build/app.js\"\n  exclude .dockerignore:2 \"build/\"\n")
}

func TestNegationReincludesFileBelowExcludedParent(t *testing.T) {
	root := fixture(t, "vendor/\n!vendor/keep/\n!vendor/keep/**\n")
	expect(t, invoke(t, root, "vendor/keep/important.py"), 0,
		"INCLUDED \"vendor/keep/important.py\"\n"+
			"  exclude .dockerignore:1 \"vendor/\"\n"+
			"  include .dockerignore:2 \"!vendor/keep/\"\n")
}

func TestBatchDoubleStarReinclusionKeepsExcludedSiblings(t *testing.T) {
	root := fixture(t, "**\n!**/a.txt\n")
	expect(t, invoke(t, root, "foo/bar/a.txt", "foo/b.txt", "a.txt"), 1,
		"INCLUDED \"foo/bar/a.txt\"\n"+
			"  exclude .dockerignore:1 \"**\"\n"+
			"  include .dockerignore:2 \"!**/a.txt\"\n"+
			"EXCLUDED \"foo/b.txt\"\n"+
			"  exclude .dockerignore:1 \"**\"\n"+
			"INCLUDED \"a.txt\"\n"+
			"  exclude .dockerignore:1 \"**\"\n"+
			"  include .dockerignore:2 \"!**/a.txt\"\n")
}

func TestRootGlobDoesNotExcludeNestedFiles(t *testing.T) {
	root := fixture(t, "*.log\n")
	expect(t, invoke(t, root, "nested/app.log", "app.log"), 1,
		"INCLUDED \"nested/app.log\"\n  no matching exclusion\n"+
			"EXCLUDED \"app.log\"\n  exclude .dockerignore:1 \"*.log\"\n")
}

func TestMissingIgnoreFileIncludesWithoutCreatingFiles(t *testing.T) {
	root := t.TempDir()
	expect(t, invoke(t, root, "new file.txt"), 0,
		"INCLUDED \"new file.txt\"\n  no matching exclusion\n")
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("input directory changed: entries=%v err=%v", entries, err)
	}
}

func TestContextRootIsNeverExcluded(t *testing.T) {
	root := fixture(t, ".*\n**\n")
	expect(t, invoke(t, root, "."), 0,
		"INCLUDED \".\"\n  context root is never excluded\n")
}

func TestOnlyFirstLineBOMIsRemovedAndOriginalLinesSurvive(t *testing.T) {
	root := fixture(t, "\ufeff# comment\r\n\r\n\ufeffcache/\r\n  #literal\r\n")
	expect(t, invoke(t, root, "cache/item", "#literal"), 1,
		"INCLUDED \"cache/item\"\n  no matching exclusion\n"+
			"EXCLUDED \"#literal\"\n  exclude .dockerignore:4 \"  #literal\"\n")
}

func TestPathsAreNormalizedRelativeToContext(t *testing.T) {
	root := fixture(t, "/src/\n")
	expect(t, invoke(t, root, "./other/../src/app.py"), 1,
		"EXCLUDED \"src/app.py\"\n  exclude .dockerignore:1 \"/src/\"\n")
}

func TestInvalidPathsRejectWholeBatchWithoutPartialOutput(t *testing.T) {
	root := fixture(t, "cache/\n")
	for _, path := range []string{"../outside", "/outside", ""} {
		t.Run(path, func(t *testing.T) {
			got := invoke(t, root, "cache/item", path)
			if got.code != 2 || got.out != "" || !strings.Contains(got.err, "path must stay relative to context") {
				t.Fatalf("unexpected invalid-path result: %+v", got)
			}
		})
	}
}

func TestMissingArgumentsShowUsageWithoutPanic(t *testing.T) {
	for _, args := range [][]string{nil, {t.TempDir()}} {
		got := invoke(t, args...)
		if got.code != 2 || got.out != "" || !strings.Contains(got.err, "usage: ignoretrail") || strings.Contains(got.err, "panic") {
			t.Fatalf("unexpected usage result: %+v", got)
		}
	}
}

func TestHelpExplainsExitCodesAndRelativePaths(t *testing.T) {
	got := invoke(t, "--help")
	if got.code != 0 || got.err != "" || !strings.Contains(got.out, "PATH is relative to CONTEXT") || !strings.Contains(got.out, "0 included, 1 excluded, 2 error") {
		t.Fatalf("unexpected help result: %+v", got)
	}
}

func TestDockerfileSpecificRulesReplaceRootRules(t *testing.T) {
	root := fixture(t, "*\n")
	if err := os.Mkdir(filepath.Join(root, "docker"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docker", "build.Dockerfile.dockerignore"), []byte("temp/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	expect(t, invoke(t, "-f", "docker/build.Dockerfile", root, "app.py", "temp/a"), 1,
		"INCLUDED \"app.py\"\n  no matching exclusion\n"+
			"EXCLUDED \"temp/a\"\n  exclude docker/build.Dockerfile.dockerignore:1 \"temp/\"\n")
}

func TestDefaultDockerfileUsesItsOwnIgnoreFile(t *testing.T) {
	root := fixture(t, "*\n")
	if err := os.WriteFile(filepath.Join(root, "Dockerfile.dockerignore"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	expect(t, invoke(t, root, "app.py"), 0,
		"INCLUDED \"app.py\"\n  no matching exclusion\n")
}

func TestContextMustBeAnExistingDirectory(t *testing.T) {
	root := fixture(t, "")
	for _, context := range []string{filepath.Join(root, "missing"), filepath.Join(root, ".dockerignore"), ""} {
		got := invoke(t, context, "app.py")
		if got.code != 2 || got.out != "" || !strings.Contains(got.err, "context must be a directory") {
			t.Fatalf("unexpected context error: %+v", got)
		}
	}
}

func TestInvalidPatternNamesSourceAndLineWithoutOutput(t *testing.T) {
	for _, pattern := range []string{"[broken", "!"} {
		root := fixture(t, "# header\ncache/\n"+pattern+"\n")
		got := invoke(t, root, "cache/item", "other")
		if got.code != 2 || got.out != "" || !strings.Contains(got.err, ".dockerignore:3:") {
			t.Fatalf("unexpected pattern error: %+v", got)
		}
	}
}

func TestDockerfilePathMustStayInsideContext(t *testing.T) {
	root := fixture(t, "")
	for _, dockerfile := range []string{"../Dockerfile", "/Dockerfile", ""} {
		got := invoke(t, "-f", dockerfile, root, "app.py")
		if got.code != 2 || got.out != "" || !strings.Contains(got.err, "Dockerfile must stay relative to context") {
			t.Fatalf("unexpected Dockerfile error: %+v", got)
		}
	}
}

func TestIgnoreSymlinkCannotReadOutsideContext(t *testing.T) {
	root := t.TempDir()
	outside := fixture(t, "*\n")
	if err := os.Symlink(filepath.Join(outside, ".dockerignore"), filepath.Join(root, ".dockerignore")); err != nil {
		t.Fatal(err)
	}
	got := invoke(t, root, "app.py")
	if got.code != 2 || got.out != "" || !strings.Contains(got.err, "escapes") {
		t.Fatalf("unexpected outside-context result: %+v", got)
	}
}

func TestOversizedIgnoreLineFailsWithSourceLocation(t *testing.T) {
	root := fixture(t, strings.Repeat("x", 70*1024)+"\n")
	got := invoke(t, root, "app.py")
	if got.code != 2 || got.out != "" || !strings.Contains(got.err, ".dockerignore:1:") || !strings.Contains(got.err, "token too long") {
		t.Fatalf("unexpected oversized-line result: %+v", got)
	}
}

func TestSpecialIgnoreFileIsRejectedWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, ".dockerignore"), 0600); err != nil {
		t.Fatal(err)
	}
	got := invoke(t, root, "app.py")
	if got.code != 2 || got.out != "" || !strings.Contains(got.err, "ignore file must be regular") {
		t.Fatalf("unexpected special-file result: %+v", got)
	}
}

func TestIgnoreFileNamesWithSpacesAreQuoted(t *testing.T) {
	root := fixture(t, "")
	if err := os.WriteFile(filepath.Join(root, "构建 file.Dockerfile.dockerignore"), []byte("[ab]?.txt\n"), 0600); err != nil {
		t.Fatal(err)
	}
	expect(t, invoke(t, "-f", "构建 file.Dockerfile", root, "a1.txt"), 1,
		"EXCLUDED \"a1.txt\"\n  exclude \"构建 file.Dockerfile.dockerignore\":1 \"[ab]?.txt\"\n")
}

func TestHelpTakesPrecedenceOverPositionalArguments(t *testing.T) {
	got := invoke(t, "-h", t.TempDir(), "app.py")
	if got.code != 0 || got.err != "" || !strings.Contains(got.out, "usage: ignoretrail") {
		t.Fatalf("unexpected help result: %+v", got)
	}
}
