package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

const (
	FormatVersion = 1
	DirectoryName = ".gosherpa"
	FileName      = "snapshot.json"

	StatusMissing = "missing"
	StatusValid   = "valid"
	StatusStale   = "stale"
	StatusInvalid = "invalid"
)

type BuildOptions struct {
	BuildTags []string
}

type Snapshot struct {
	FormatVersion int                     `json:"formatVersion"`
	CreatedAt     string                  `json:"createdAt"`
	Root          string                  `json:"root"`
	ModulePath    string                  `json:"modulePath"`
	GoVersion     string                  `json:"goVersion"`
	GOOS          string                  `json:"goos"`
	GOARCH        string                  `json:"goarch"`
	BuildTags     []string                `json:"buildTags"`
	GitState      string                  `json:"gitState,omitempty"`
	Fingerprint   string                  `json:"fingerprint"`
	Files         []File                  `json:"files"`
	Packages      []sherpa.PackageSummary `json:"packages"`
	Symbols       []sherpa.Symbol         `json:"symbols"`
}

type File struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ModTime     string `json:"modTime"`
	ModTimeUnix int64  `json:"modTimeUnix"`
	SHA256      string `json:"sha256"`
}

type InspectResult struct {
	Supported          bool     `json:"supported"`
	Status             string   `json:"status"`
	Path               string   `json:"path"`
	Message            string   `json:"message"`
	FormatVersion      int      `json:"formatVersion,omitempty"`
	CreatedAt          string   `json:"createdAt,omitempty"`
	Fingerprint        string   `json:"fingerprint,omitempty"`
	CurrentFingerprint string   `json:"currentFingerprint,omitempty"`
	FileCount          int      `json:"fileCount,omitempty"`
	PackageCount       int      `json:"packageCount,omitempty"`
	SymbolCount        int      `json:"symbolCount,omitempty"`
	StaleReasons       []string `json:"staleReasons"`
}

func Build(root string, options BuildOptions) (Snapshot, error) {
	rootPath, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return Snapshot{}, fmt.Errorf("resolve repository root %s: %w", root, err)
	}
	rootPath = filepath.Clean(rootPath)

	modulePath, err := sherpa.ModulePath(rootPath)
	if err != nil {
		return Snapshot{}, err
	}

	files, err := collectFiles(rootPath)
	if err != nil {
		return Snapshot{}, err
	}

	packages, err := sherpa.FindPackageSummaries(rootPath, sherpa.PackageInventoryOptions{
		IncludeTests: true,
	})
	if err != nil {
		return Snapshot{}, err
	}

	symbols, err := sherpa.ParseRepository(rootPath)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot := Snapshot{
		FormatVersion: FormatVersion,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Root:          rootPath,
		ModulePath:    modulePath,
		GoVersion:     runtime.Version(),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		BuildTags:     semantics.NormalizeBuildTags(options.BuildTags),
		GitState:      readGitState(rootPath),
		Files:         files,
		Packages:      nonNilSlice(packages),
		Symbols:       nonNilSlice(symbols),
	}
	snapshot.Fingerprint = fingerprintSnapshotInputs(snapshot)

	return snapshot, nil
}

func Write(root string, snapshot Snapshot) (string, error) {
	path := Path(root)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create snapshot directory: %w", err)
	}

	data, err := json.MarshalIndent(normalizeSnapshot(snapshot), "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode snapshot: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write snapshot: %w", err)
	}

	return path, nil
}

func Load(root string) (Snapshot, error) {
	data, err := os.ReadFile(Path(root))
	if err != nil {
		return Snapshot{}, err
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}

	return normalizeSnapshot(snapshot), nil
}

func Inspect(root string, options BuildOptions) InspectResult {
	path := Path(root)
	displayPath := displayPath(root, path)
	stored, err := Load(root)
	if err != nil {
		if os.IsNotExist(err) {
			return normalizeInspectResult(InspectResult{
				Supported: true,
				Status:    StatusMissing,
				Path:      displayPath,
				Message:   "No snapshot found. Run gosherpa snapshot to create one.",
			})
		}

		return normalizeInspectResult(InspectResult{
			Supported:    true,
			Status:       StatusInvalid,
			Path:         displayPath,
			Message:      fmt.Sprintf("Snapshot could not be read: %v", err),
			StaleReasons: []string{"snapshot could not be read"},
		})
	}

	current, currentErr := Build(root, options)
	if currentErr != nil {
		return normalizeInspectResult(InspectResult{
			Supported:     true,
			Status:        StatusInvalid,
			Path:          displayPath,
			Message:       fmt.Sprintf("Snapshot could not be checked: %v", currentErr),
			FormatVersion: stored.FormatVersion,
			CreatedAt:     stored.CreatedAt,
			Fingerprint:   stored.Fingerprint,
			FileCount:     len(stored.Files),
			PackageCount:  len(stored.Packages),
			SymbolCount:   len(stored.Symbols),
			StaleReasons:  []string{"current repository fingerprint could not be computed"},
		})
	}

	reasons := staleReasons(stored, current)
	status := StatusValid
	message := "Snapshot is valid for the current repository inputs."
	if len(reasons) > 0 {
		status = StatusStale
		message = "Snapshot is stale. Run gosherpa snapshot to refresh it."
	}

	return normalizeInspectResult(InspectResult{
		Supported:          true,
		Status:             status,
		Path:               displayPath,
		Message:            message,
		FormatVersion:      stored.FormatVersion,
		CreatedAt:          stored.CreatedAt,
		Fingerprint:        stored.Fingerprint,
		CurrentFingerprint: current.Fingerprint,
		FileCount:          len(stored.Files),
		PackageCount:       len(stored.Packages),
		SymbolCount:        len(stored.Symbols),
		StaleReasons:       reasons,
	})
}

func Path(root string) string {
	return filepath.Join(root, DirectoryName, FileName)
}

func displayPath(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(relative)
	}

	return filepath.Clean(path)
}

func collectFiles(root string) ([]File, error) {
	paths, err := sherpa.FindGoFiles(root)
	if err != nil {
		return nil, err
	}

	for _, name := range []string{"go.mod", "go.sum", "go.work"} {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			paths = append(paths, path)
		}
	}

	paths = uniqueCleanPaths(paths)
	files := make([]File, 0, len(paths))
	for _, path := range paths {
		file, err := fileMetadata(root, path)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	sort.SliceStable(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func fileMetadata(root string, path string) (File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return File{}, err
	}
	if info.IsDir() {
		return File{}, fmt.Errorf("snapshot input is a directory: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}

	hash := sha256.Sum256(data)
	modTime := info.ModTime().UTC()
	return File{
		Path:        displayPath(root, path),
		Size:        info.Size(),
		ModTime:     modTime.Format(time.RFC3339Nano),
		ModTimeUnix: modTime.UnixNano(),
		SHA256:      hex.EncodeToString(hash[:]),
	}, nil
}

func uniqueCleanPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var result []string
	for _, path := range paths {
		cleaned := filepath.Clean(path)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	sort.Strings(result)

	return result
}

func staleReasons(stored Snapshot, current Snapshot) []string {
	var reasons []string
	if stored.FormatVersion != current.FormatVersion {
		reasons = append(reasons, "snapshot format version changed")
	}
	if filepath.Clean(stored.Root) != filepath.Clean(current.Root) {
		reasons = append(reasons, "repository root changed")
	}
	if stored.ModulePath != current.ModulePath {
		reasons = append(reasons, "module path changed")
	}
	if !equalStrings(stored.BuildTags, current.BuildTags) {
		reasons = append(reasons, "build tags changed")
	}
	if stored.GitState != current.GitState {
		reasons = append(reasons, "git state changed")
	}
	if stored.Fingerprint != current.Fingerprint {
		reasons = append(reasons, "repository files changed")
	}

	return reasons
}

func fingerprintSnapshotInputs(snapshot Snapshot) string {
	hash := sha256.New()
	for _, file := range snapshot.Files {
		writeHashLine(hash, file.Path)
		writeHashLine(hash, fmt.Sprintf("%d", file.Size))
		writeHashLine(hash, fmt.Sprintf("%d", file.ModTimeUnix))
		writeHashLine(hash, file.SHA256)
	}

	return hex.EncodeToString(hash.Sum(nil))
}

func writeHashLine(hash interface{ Write([]byte) (int, error) }, value string) {
	_, _ = hash.Write([]byte(value))
	_, _ = hash.Write([]byte{0})
}

func readGitState(root string) string {
	gitDir, ok := resolveGitDir(root)
	if !ok {
		return ""
	}

	headData, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}
	head := strings.TrimSpace(string(headData))
	if !strings.HasPrefix(head, "ref: ") {
		return head
	}

	ref := strings.TrimSpace(strings.TrimPrefix(head, "ref: "))
	refValue := readGitRef(gitDir, ref)
	if refValue == "" {
		return "ref:" + ref
	}

	return "ref:" + ref + "@" + refValue
}

func resolveGitDir(root string) (string, bool) {
	path := filepath.Join(root, ".git")
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		return path, true
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(string(data))
	if !strings.HasPrefix(value, "gitdir:") {
		return "", false
	}
	gitDir := strings.TrimSpace(strings.TrimPrefix(value, "gitdir:"))
	if gitDir == "" {
		return "", false
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}

	return filepath.Clean(gitDir), true
}

func readGitRef(gitDir string, ref string) string {
	data, err := os.ReadFile(filepath.Join(gitDir, filepath.FromSlash(ref)))
	if err == nil {
		return strings.TrimSpace(string(data))
	}

	data, err = os.ReadFile(filepath.Join(gitDir, "packed-refs"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "^") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == ref {
			return fields[0]
		}
	}

	return ""
}

func normalizeSnapshot(snapshot Snapshot) Snapshot {
	snapshot.BuildTags = nonNilSlice(semantics.NormalizeBuildTags(snapshot.BuildTags))
	snapshot.Files = nonNilSlice(snapshot.Files)
	snapshot.Packages = nonNilSlice(snapshot.Packages)
	snapshot.Symbols = nonNilSlice(snapshot.Symbols)
	if snapshot.FormatVersion == 0 {
		snapshot.FormatVersion = FormatVersion
	}

	return snapshot
}

func normalizeInspectResult(result InspectResult) InspectResult {
	if strings.TrimSpace(result.Status) == "" {
		result.Status = StatusInvalid
	}
	result.StaleReasons = nonNilSlice(result.StaleReasons)

	return result
}

func equalStrings(left []string, right []string) bool {
	left = semantics.NormalizeBuildTags(left)
	right = semantics.NormalizeBuildTags(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}
