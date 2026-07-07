package semantics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/build"
	"io/fs"
	"path/filepath"
	"strconv"
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

type repositoryInputFileMatchCache struct {
	mu      sync.Mutex
	max     int
	entries map[string]bool
	order   []string
}

var repositoryInputMatches = newRepositoryInputFileMatchCache(4096)

func newRepositoryCache(max int) *repositoryCache {
	return &repositoryCache{
		max:     max,
		entries: make(map[string]repositoryCacheEntry),
	}
}

func newRepositoryInputFileMatchCache(max int) *repositoryInputFileMatchCache {
	return &repositoryInputFileMatchCache{
		max:     max,
		entries: make(map[string]bool),
	}
}

func (cache *repositoryInputFileMatchCache) MatchFile(path string, name string, info fs.FileInfo, buildContext build.Context, buildContextKey string) (bool, error) {
	if cache == nil || cache.max <= 0 {
		return buildContext.MatchFile(filepath.Dir(path), name)
	}

	key := repositoryInputFileMatchCacheKey(path, info, buildContextKey)
	cache.mu.Lock()
	match, ok := cache.entries[key]
	cache.mu.Unlock()
	if ok {
		return match, nil
	}

	match, err := buildContext.MatchFile(filepath.Dir(path), name)
	if err != nil {
		return false, err
	}

	cache.mu.Lock()
	if _, ok := cache.entries[key]; !ok {
		cache.order = append(cache.order, key)
	}
	cache.entries[key] = match
	for len(cache.order) > cache.max {
		oldest := cache.order[0]
		cache.order = cache.order[1:]
		delete(cache.entries, oldest)
	}
	cache.mu.Unlock()

	return match, nil
}

func repositoryInputFileMatchCacheKey(path string, info fs.FileInfo, buildContextKey string) string {
	cleaned := filepath.Clean(path)
	var builder strings.Builder
	builder.Grow(len(cleaned) + len(buildContextKey) + 64)
	builder.WriteString(cleaned)
	builder.WriteByte(0)
	writeRepositoryBuilderInt64(&builder, info.Size())
	builder.WriteByte(0)
	writeRepositoryBuilderInt64(&builder, info.ModTime().UnixNano())
	builder.WriteByte(0)
	builder.WriteString(buildContextKey)
	return builder.String()
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
	rootPrefix := filepath.Clean(root) + string(filepath.Separator)
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

		relative, err := repositoryInputRelativePath(root, rootPrefix, path)
		if err != nil {
			return err
		}
		writeRepositoryHashLine(hash, relative)
		writeRepositoryHashInt64(hash, info.Size())
		writeRepositoryHashInt64(hash, info.ModTime().UnixNano())

		return nil
	})
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func repositoryInputRelativePath(root string, rootPrefix string, path string) (string, error) {
	if relative, ok := strings.CutPrefix(path, rootPrefix); ok {
		return repositorySlashPath(relative), nil
	}

	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}

	return repositorySlashPath(relative), nil
}

func repositorySlashPath(path string) string {
	if filepath.Separator == '/' {
		return path
	}

	return filepath.ToSlash(path)
}

func repositoryBuildContext(options LoadOptions) build.Context {
	context := build.Default
	context.BuildTags = repositoryBuildTags(options)
	if compiler := repositoryBuildCompiler(options.BuildFlags); compiler != "" {
		context.Compiler = compiler
	}
	return context
}

func repositoryBuildContextKey(buildContext build.Context) string {
	return strings.Join([]string{
		buildContext.GOOS,
		buildContext.GOARCH,
		buildContext.Compiler,
		fmt.Sprintf("cgo=%t", buildContext.CgoEnabled),
		"tags=" + strings.Join(buildContext.BuildTags, ","),
		"tool=" + strings.Join(buildContext.ToolTags, ","),
		"release=" + strings.Join(buildContext.ReleaseTags, ","),
	}, "\x00")
}

func repositoryBuildTags(options LoadOptions) []string {
	values := append([]string{}, options.BuildTags...)
	values = append(values, repositoryBuildFlagTags(options.BuildFlags)...)
	return NormalizeBuildTags(values)
}

func repositoryBuildFlagTags(flags []string) []string {
	var values []string
	for i := 0; i < len(flags); i++ {
		flag := strings.TrimSpace(flags[i])
		switch {
		case repositoryBoolBuildFlagEnabled(flag, "-race"):
			values = append(values, "race")
		case repositoryBoolBuildFlagEnabled(flag, "-msan"):
			values = append(values, "msan")
		case repositoryBoolBuildFlagEnabled(flag, "-asan"):
			values = append(values, "asan")
		case flag == "-tags" && i+1 < len(flags):
			i++
			values = append(values, flags[i])
		case strings.HasPrefix(flag, "-tags="):
			values = append(values, strings.TrimPrefix(flag, "-tags="))
		}
	}

	return values
}

func repositoryBoolBuildFlagEnabled(flag string, name string) bool {
	if flag == name {
		return true
	}
	value, ok := strings.CutPrefix(flag, name+"=")
	if !ok {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true":
		return true
	default:
		return false
	}
}

func repositoryBuildCompiler(flags []string) string {
	for i := 0; i < len(flags); i++ {
		flag := strings.TrimSpace(flags[i])
		if flag == "-compiler" && i+1 < len(flags) {
			i++
			if compiler := strings.TrimSpace(flags[i]); compiler != "" {
				return compiler
			}
			continue
		}

		if compiler, ok := strings.CutPrefix(flag, "-compiler="); ok {
			if compiler = strings.TrimSpace(compiler); compiler != "" {
				return compiler
			}
		}
	}

	return ""
}

func repositoryInputCandidate(name string) bool {
	if strings.HasSuffix(name, ".go") {
		return true
	}

	switch name {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return false
	}
}

func repositoryInputFile(path string, name string, info fs.FileInfo, options LoadOptions, buildContext build.Context, buildContextKey string) (bool, error) {
	if strings.HasSuffix(name, ".go") {
		if !options.IncludeTests && strings.HasSuffix(name, "_test.go") {
			return false, nil
		}
		return repositoryInputMatches.MatchFile(path, name, info, buildContext, buildContextKey)
	}

	switch name {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true, nil
	default:
		return false, nil
	}
}

func writeRepositoryHashLine(hash interface{ Write([]byte) (int, error) }, value string) {
	_, _ = hash.Write([]byte(value))
	_, _ = hash.Write([]byte{0})
}

func writeRepositoryHashInt64(hash interface{ Write([]byte) (int, error) }, value int64) {
	var buffer [20]byte
	_, _ = hash.Write(strconv.AppendInt(buffer[:0], value, 10))
	_, _ = hash.Write([]byte{0})
}

func writeRepositoryBuilderInt64(builder *strings.Builder, value int64) {
	var buffer [20]byte
	_, _ = builder.Write(strconv.AppendInt(buffer[:0], value, 10))
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
