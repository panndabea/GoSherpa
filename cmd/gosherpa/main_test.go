package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintUsageIncludesCallees(t *testing.T) {
	output := captureMainTestStdout(t, func() {
		printUsage()
	})

	if !strings.Contains(output, "callees <function-or-method>") {
		t.Fatalf("expected usage to contain callees command, got:\n%s", output)
	}
}

func TestPrintUsageIncludesCallers(t *testing.T) {
	output := captureMainTestStdout(t, func() {
		printUsage()
	})

	if !strings.Contains(output, "callers <function-or-method>") {
		t.Fatalf("expected usage to contain callers command, got:\n%s", output)
	}
}

func TestMainPrintsCallersUsageWhenArgumentIsMissing(t *testing.T) {
	setMainTestArgs(t, []string{"gosherpa", "callers"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	want := "usage: gosherpa callers <function-or-method>\n"
	if output != want {
		t.Fatalf("expected %q, got %q", want, output)
	}
}

func TestMainRunsCallersCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	oldWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := os.Chdir(oldWorkingDirectory)
		if err != nil {
			t.Fatal(err)
		}
	})

	setMainTestArgs(t, []string{"gosherpa", "callers", "Step"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"CALLERS", "Step", "Run", "Found 1 callers"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}
}

func TestMainPrintsCalleesUsageWhenArgumentIsMissing(t *testing.T) {
	setMainTestArgs(t, []string{"gosherpa", "callees"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	want := "usage: gosherpa callees <function-or-method>\n"
	if output != want {
		t.Fatalf("expected %q, got %q", want, output)
	}
}

func TestMainRunsCalleesCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	oldWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := os.Chdir(oldWorkingDirectory)
		if err != nil {
			t.Fatal(err)
		}
	})

	setMainTestArgs(t, []string{"gosherpa", "callees", "Run"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"CALLEES", "Run", "Step", "Found 1 callees"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}
}

func captureMainTestStdout(t *testing.T, run func()) string {
	t.Helper()

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	os.Stdout = writePipe

	run()

	err = writePipe.Close()
	if err != nil {
		t.Fatal(err)
	}

	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatal(err)
	}

	err = readPipe.Close()
	if err != nil {
		t.Fatal(err)
	}

	return string(output)
}

func setMainTestArgs(t *testing.T, args []string) {
	t.Helper()

	oldArgs := os.Args
	os.Args = args
	t.Cleanup(func() {
		os.Args = oldArgs
	})
}

func writeMainTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(path, []byte(contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
}
