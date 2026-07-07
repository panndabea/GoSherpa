package semantics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

type repositoryCache struct {
	mu      sync.Mutex
	max     int
	entries map[string]repositoryCacheEntry
	order   []string
}

type repositoryCacheEntry struct {
	Fingerprint string
	Repository  Repository
}

func newRepositoryCache(max int) *repositoryCache {
	return &repositoryCache{
		max:     max,
		entries: make(map[string]repositoryCacheEntry),
	}
}

func (cache *repositoryCache) Get(key string, fingerprint string) (Repository, bool) {
	if cache == nil {
		return Repository{}, false
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	entry, ok := cache.entries[key]
	if !ok || entry.Fingerprint != fingerprint {
		return Repository{}, false
	}

	return cloneRepository(entry.Repository), true
}

func (cache *repositoryCache) Put(key string, fingerprint string, repo Repository) {
	if cache == nil || cache.max <= 0 {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	if _, ok := cache.entries[key]; !ok {
		cache.order = append(cache.order, key)
	}
	cache.entries[key] = repositoryCacheEntry{
		Fingerprint: fingerprint,
		Repository:  cloneRepository(repo),
	}

	for len(cache.order) > cache.max {
		oldest := cache.order[0]
		cache.order = cache.order[1:]
		delete(cache.entries, oldest)
	}
}

func (cache *repositoryCache) Clear() {
	if cache == nil {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.entries = make(map[string]repositoryCacheEntry)
	cache.order = nil
}

func repositoryCacheKey(root string, options LoadOptions, patterns []string) string {
	parts := []string{
		filepath.Clean(root),
		fmt.Sprintf("tests=%t", options.IncludeTests),
		"tags=" + strings.Join(NormalizeBuildTags(options.BuildTags), ","),
		"flags=" + strings.Join(options.BuildFlags, "\x1f"),
		"patterns=" + strings.Join(patterns, "\x1f"),
	}

	return strings.Join(parts, "\x00")
}

func repositoryInputFingerprint(root string, options LoadOptions) (string, error) {
	hash := sha256.New()
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
		if !repositoryInputFile(entry.Name(), options.IncludeTests) {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		writeRepositoryHashLine(hash, filepath.ToSlash(relative))
		writeRepositoryHashLine(hash, fmt.Sprintf("%d", info.Size()))
		writeRepositoryHashLine(hash, fmt.Sprintf("%d", info.ModTime().UnixNano()))

		return nil
	})
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func repositoryInputFile(name string, includeTests bool) bool {
	if strings.HasSuffix(name, ".go") {
		if !includeTests && strings.HasSuffix(name, "_test.go") {
			return false
		}
		return true
	}

	switch name {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return false
	}
}

func writeRepositoryHashLine(hash interface{ Write([]byte) (int, error) }, value string) {
	_, _ = hash.Write([]byte(value))
	_, _ = hash.Write([]byte{0})
}

func cloneRepository(repo Repository) Repository {
	clone := Repository{
		Root:     repo.Root,
		Packages: append([]Package{}, repo.Packages...),
		Warnings: append([]string{}, repo.Warnings...),
	}
	for i := range clone.Packages {
		clone.Packages[i].GoFiles = append([]string{}, clone.Packages[i].GoFiles...)
		clone.Packages[i].CompiledGoFiles = append([]string{}, clone.Packages[i].CompiledGoFiles...)
		clone.Packages[i].Files = append([]*ast.File{}, clone.Packages[i].Files...)
	}

	return clone
}
