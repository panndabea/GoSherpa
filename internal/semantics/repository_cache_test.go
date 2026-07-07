package semantics

import "testing"

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
