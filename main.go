package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
)

func main() {
	code, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "ignoretrail:", err)
		code = 2
	}
	os.Exit(code)
}

func run(args []string) (int, error) {
	flags := flag.NewFlagSet("ignoretrail", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dockerfile := flags.String("f", "Dockerfile", "Dockerfile path relative to context")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Println("usage: ignoretrail [-f Dockerfile] CONTEXT PATH...\nPATH is relative to CONTEXT. Rules only; path existence is not checked.\nExit: 0 included, 1 excluded, 2 error")
			return 0, nil
		}
		return 0, err
	}
	args = flags.Args()
	if len(args) < 2 {
		return 0, fmt.Errorf("usage: ignoretrail [-f Dockerfile] CONTEXT PATH...")
	}
	if !filepath.IsLocal(*dockerfile) {
		return 0, fmt.Errorf("Dockerfile must stay relative to context: %q", *dockerfile)
	}
	*dockerfile = filepath.Clean(*dockerfile)
	info, err := os.Stat(args[0])
	if err != nil || !info.IsDir() {
		return 0, fmt.Errorf("context must be a directory: %q", args[0])
	}
	context, err := os.OpenRoot(args[0])
	if err != nil {
		return 0, err
	}
	defer context.Close()
	for _, path := range args[1:] {
		if !filepath.IsLocal(path) {
			return 0, fmt.Errorf("path must stay relative to context: %q", path)
		}
	}
	ignoreName := ".dockerignore"
	if *dockerfile != "" {
		candidate := *dockerfile + ".dockerignore"
		if _, err := context.Stat(candidate); err == nil {
			ignoreName = candidate
		} else if !os.IsNotExist(err) {
			return 0, err
		}
	}
	code := 0
	for _, path := range args[1:] {
		path = filepath.ToSlash(filepath.Clean(path))
		status, err := explain(context, ignoreName, path)
		if err != nil {
			return 0, err
		}
		code = max(code, status)
	}
	return code, nil
}

func explain(context *os.Root, ignoreName, path string) (int, error) {
	source := filepath.ToSlash(ignoreName)
	if strings.IndexFunc(source, func(r rune) bool {
		return unicode.IsSpace(r) || !unicode.IsPrint(r) || r == '"' || r == '\\'
	}) >= 0 {
		source = fmt.Sprintf("%q", source)
	}
	info, err := context.Stat(ignoreName)
	if os.IsNotExist(err) {
		if path == "." {
			fmt.Printf("INCLUDED %q\n  context root is never excluded\n", path)
		} else {
			fmt.Printf("INCLUDED %q\n  no matching exclusion\n", path)
		}
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("ignore file must be regular: %q", ignoreName)
	}
	file, err := context.Open(ignoreName)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	var trail strings.Builder
	excluded := false
	var rules []string
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		raw := scanner.Text()
		input := raw
		if line > 1 {
			// A BOM is special only at the start of the complete file.
			input = "#\n" + input
		}
		patterns, err := ignorefile.ReadAll(strings.NewReader(input))
		if err != nil {
			return 0, fmt.Errorf("%s:%d: %w", source, line, err)
		}
		rules = append(rules, patterns...)
		matcher, err := patternmatcher.New(rules)
		if err != nil {
			return 0, fmt.Errorf("%s:%d: %w", source, line, err)
		}
		matched, err := matcher.MatchesOrParentMatches(path)
		if err != nil {
			return 0, fmt.Errorf("%s:%d: %w", source, line, err)
		}
		if path == "." {
			matched = false
		}
		if matched != excluded {
			action := "include"
			if matched {
				action = "exclude"
			}
			excluded = matched
			fmt.Fprintf(&trail, "  %s %s:%d %q\n", action, source, line, raw)
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("%s:%d: %w", source, line+1, err)
	}
	if excluded {
		fmt.Printf("EXCLUDED %q\n%s", path, trail.String())
		return 1, nil
	}
	if path == "." {
		trail.WriteString("  context root is never excluded\n")
	} else if trail.Len() == 0 {
		trail.WriteString("  no matching exclusion\n")
	}
	fmt.Printf("INCLUDED %q\n%s", path, trail.String())
	return 0, nil
}
