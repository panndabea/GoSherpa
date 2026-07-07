package semantics

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

func TestRepositoryCacheKeySeparatesOptionsAndPatterns(t *testing.T) {
	root := t.TempDir()
	baseOptions := LoadOptions{}
	basePatterns := []string{"./..."}
	base := repositoryCacheKey(root, baseOptions, basePatterns)

	tests := []struct {
		name     string
		options  LoadOptions
		patterns []string
	}{
		{
			name:     "include tests",
			options:  LoadOptions{IncludeTests: true},
			patterns: basePatterns,
		},
		{
			name:     "build tags",
			options:  LoadOptions{BuildTags: []string{"enterprise"}},
			patterns: basePatterns,
		},
		{
			name:     "build flags",
			options:  LoadOptions{BuildFlags: []string{"-race"}},
			patterns: basePatterns,
		},
		{
			name:     "patterns",
			options:  baseOptions,
			patterns: []string{"./cmd/..."},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := repositoryCacheKey(root, test.options, test.patterns)
			if got == base {
				t.Fatalf("expected cache key to differ from base")
			}
		})
	}
}

func TestRepositoryCacheKeyNormalizesBuildTags(t *testing.T) {
	root := t.TempDir()
	patterns := []string{"./..."}

	first := repositoryCacheKey(root, LoadOptions{BuildTags: []string{"integration, enterprise"}}, patterns)
	second := repositoryCacheKey(root, LoadOptions{BuildTags: []string{"enterprise", "integration"}}, patterns)
	if first != second {
		t.Fatalf("expected equivalent build tags to share cache key")
	}
}

func TestRepositoryCacheEvictsOldestEntry(t *testing.T) {
	cache := newRepositoryCache(2)

	cache.Put("a", "fp", Repository{Root: "a"})
	cache.Put("b", "fp", Repository{Root: "b"})
	cache.Put("c", "fp", Repository{Root: "c"})

	if _, ok := cache.Get("a", "fp"); ok {
		t.Fatal("expected oldest cache entry to be evicted")
	}
	assertRepositoryCacheHit(t, cache, "b", "fp", "b")
	assertRepositoryCacheHit(t, cache, "c", "fp", "c")
}

func TestRepositoryBuildTagsIncludesBuildFlagTags(t *testing.T) {
	got := repositoryBuildTags(LoadOptions{
		BuildTags:  []string{"enterprise"},
		BuildFlags: []string{"-race", "-tags=integration,debug", "-tags", "canary nightly"},
	})
	want := []string{"canary", "debug", "enterprise", "integration", "nightly"}
	if !slices.Equal(got, want) {
		t.Fatalf("repositoryBuildTags() = %#v, want %#v", got, want)
	}
}

func assertRepositoryCacheHit(t *testing.T, cache *repositoryCache, key string, fingerprint string, root string) {
	t.Helper()

	repo, ok := cache.Get(key, fingerprint)
	if !ok {
		t.Fatalf("expected cache hit for %q", key)
	}
	if repo.Root != root {
		t.Fatalf("cached repository root = %q, want %q", repo.Root, root)
	}
}

func BenchmarkRepositoryInputFingerprint(b *testing.B) {
	root := repositoryFingerprintBenchmarkRoot(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := repositoryInputFingerprint(root, LoadOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepositoryInputContentFingerprint(b *testing.B) {
	root := repositoryFingerprintBenchmarkRoot(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := repositoryInputContentFingerprintForBenchmark(root); err != nil {
			b.Fatal(err)
		}
	}
}

func repositoryFingerprintBenchmarkRoot(b *testing.B) string {
	b.Helper()

	root := b.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n"), 0644); err != nil {
		b.Fatal(err)
	}

	const fileCount = 250
	for i := 0; i < fileCount; i++ {
		dir := filepath.Join(root, "pkg", "service")
		path := filepath.Join(dir, "service_"+strconv.Itoa(i)+".go")
		if err := os.MkdirAll(dir, 0755); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(repositoryFingerprintBenchmarkSource(i)), 0644); err != nil {
			b.Fatal(err)
		}
	}

	return root
}

func repositoryFingerprintBenchmarkSource(index int) string {
	value := strconv.Itoa(index)
	return `package service

type Value` + value + ` struct {
	Field string
	Count int
}

func NewValue` + value + `(field string, count int) Value` + value + ` {
	return Value` + value + `{Field: field, Count: count}
}
`
}

func repositoryInputContentFingerprintForBenchmark(root string) (string, error) {
	hash := sha256.New()
	options := LoadOptions{IncludeTests: true}
	buildContext := repositoryBuildContext(options)
	buildContextKey := repositoryBuildContextKey(buildContext)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".gosherpa":
				if filepath.Clean(path) != filepath.Clean(root) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !repositoryInputCandidate(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		input, err := repositoryInputFile(path, entry.Name(), info, options, buildContext, buildContextKey)
		if err != nil {
			return err
		}
		if !input {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		writeRepositoryHashLine(hash, filepath.ToSlash(relative))
		_, _ = hash.Write(contents)
		_, _ = hash.Write([]byte{0})

		return nil
	})
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
