package sherpa

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

type RepositoryLayout struct {
	Root                    string             `json:"root"`
	ModulePath              string             `json:"modulePath"`
	Manifest                string             `json:"manifest,omitempty"`
	GoModPath               string             `json:"goModPath,omitempty"`
	GoWork                  GoWorkLayout       `json:"goWork"`
	AnalysisBoundary        string             `json:"analysisBoundary"`
	GoFiles                 int                `json:"goFiles"`
	TestFiles               int                `json:"testFiles"`
	GeneratedFiles          int                `json:"generatedFiles"`
	NestedModules           []string           `json:"nestedModules"`
	SkippedNestedModules    []string           `json:"skippedNestedModules"`
	WorkspaceModules        []WorkspaceModule  `json:"workspaceModules"`
	SkippedWorkspaceModules []string           `json:"skippedWorkspaceModules"`
	LocalReplacements       []LocalReplacement `json:"localReplacements"`
}

type GoWorkLayout struct {
	Detected bool   `json:"detected"`
	Path     string `json:"path,omitempty"`
	Scope    string `json:"scope,omitempty"`
}

type WorkspaceModule struct {
	Path       string `json:"path"`
	ModulePath string `json:"modulePath,omitempty"`
	InsideRoot bool   `json:"insideRoot"`
	Included   bool   `json:"included"`
}

type LocalReplacement struct {
	Owner       string `json:"owner"`
	ModulePath  string `json:"modulePath"`
	Replacement string `json:"replacement"`
	Path        string `json:"path,omitempty"`
	InsideRoot  bool   `json:"insideRoot"`
}

type workspaceModuleRecord struct {
	Summary WorkspaceModule
	Dir     string
}

func AnalyzeRepositoryLayout(root string) (RepositoryLayout, []string) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return RepositoryLayout{}, []string{fmt.Sprintf("repository layout unavailable: %v", err)}
	}

	layout := RepositoryLayout{
		Root:             rootPath,
		ModulePath:       readModulePath(rootPath),
		GoWork:           detectRepositoryGoWork(rootPath),
		AnalysisBoundary: "directory",
	}
	var warnings []string

	if regularFileExists(filepath.Join(rootPath, "go.mod")) {
		layout.Manifest = "go.mod"
		layout.GoModPath = "go.mod"
		layout.AnalysisBoundary = "module"
	}
	if layout.GoWork.Detected && layout.GoWork.Scope == "root" {
		layout.Manifest = "go.work"
		layout.AnalysisBoundary = "workspace"
	}
	if layout.GoWork.Detected && layout.GoWork.Scope == "parent" && layout.AnalysisBoundary == "module" {
		layout.AnalysisBoundary = "module-with-parent-workspace"
	}

	goFiles, err := FindGoFiles(rootPath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("go file scan failed: %v", err))
	} else {
		layout.GoFiles = len(goFiles)
		layout.TestFiles = CountTestGoFiles(goFiles)
		layout.GeneratedFiles = CountGeneratedGoFiles(goFiles)
	}

	nestedModules, nestedWarnings := DiscoverNestedModules(rootPath)
	layout.NestedModules = nestedModules
	warnings = append(warnings, nestedWarnings...)

	if layout.GoWork.Detected {
		records, recordWarnings := workspaceModuleRecords(rootPath, layout.GoWork.absolutePath(rootPath))
		warnings = append(warnings, recordWarnings...)
		layout.WorkspaceModules = workspaceModuleSummaries(records)
		layout.SkippedWorkspaceModules = skippedWorkspaceModules(records)
	}
	layout.SkippedNestedModules = skippedNestedModules(layout.NestedModules, layout.WorkspaceModules)
	layout.LocalReplacements = localReplacements(rootPath, layout.WorkspaceModules)

	if len(layout.SkippedNestedModules) > 0 {
		warnings = append(warnings, fmt.Sprintf("nested modules skipped by the selected analysis boundary: %s; inspect them with --root <path> when needed", strings.Join(layout.SkippedNestedModules, ", ")))
	}
	if len(layout.SkippedWorkspaceModules) > 0 {
		warnings = append(warnings, fmt.Sprintf("workspace modules outside the selected root are not scanned as repository packages: %s", strings.Join(layout.SkippedWorkspaceModules, ", ")))
	}
	if external := outsideRootLocalReplacementPaths(layout.LocalReplacements); len(external) > 0 {
		warnings = append(warnings, fmt.Sprintf("local replacements outside the selected root may affect typechecking but are not scanned as repository packages: %s", strings.Join(external, ", ")))
	}

	return NormalizeRepositoryLayout(layout), uniqueSorted(warnings)
}

func NormalizeRepositoryLayout(layout RepositoryLayout) RepositoryLayout {
	layout.Root = filepath.Clean(layout.Root)
	layout.NestedModules = nonNilStrings(uniqueSorted(layout.NestedModules))
	layout.SkippedNestedModules = nonNilStrings(uniqueSorted(layout.SkippedNestedModules))
	layout.WorkspaceModules = normalizeWorkspaceModules(layout.WorkspaceModules)
	layout.SkippedWorkspaceModules = nonNilStrings(uniqueSorted(layout.SkippedWorkspaceModules))
	layout.LocalReplacements = normalizeLocalReplacements(layout.LocalReplacements)
	if strings.TrimSpace(layout.AnalysisBoundary) == "" {
		layout.AnalysisBoundary = "directory"
	}
	return layout
}

func CountTestGoFiles(files []string) int {
	count := 0
	for _, file := range files {
		if strings.HasSuffix(filepath.ToSlash(file), "_test.go") {
			count++
		}
	}
	return count
}

func CountGeneratedGoFiles(files []string) int {
	count := 0
	for _, file := range files {
		if IsGeneratedGoFile(file) {
			count++
		}
	}
	return count
}

func IsGeneratedGoFile(file string) bool {
	data, err := os.ReadFile(file)
	if err != nil {
		return false
	}

	lines := strings.SplitN(string(data), "\n", 12)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "// Code generated ") && strings.HasSuffix(trimmed, " DO NOT EDIT.") {
			return true
		}
	}
	return false
}

func DiscoverNestedModules(root string) ([]string, []string) {
	root = filepath.Clean(root)
	var modules []string
	var warnings []string

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, fmt.Sprintf("nested module scan skipped %s: %v", displayPath(root, path), walkErr))
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}
		if shouldSkipLayoutDir(entry.Name()) {
			return filepath.SkipDir
		}
		if regularFileExists(filepath.Join(path, "go.mod")) {
			modules = append(modules, displayPath(root, path))
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("nested module scan failed: %v", err))
	}

	return uniqueSorted(modules), uniqueSorted(warnings)
}

func detectRepositoryGoWork(root string) GoWorkLayout {
	root = filepath.Clean(root)
	for dir := root; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, "go.work")
		if regularFileExists(path) {
			scope := "parent"
			if dir == root {
				scope = "root"
			}
			return GoWorkLayout{
				Detected: true,
				Path:     displayPath(root, path),
				Scope:    scope,
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return GoWorkLayout{}
}

func (layout GoWorkLayout) absolutePath(root string) string {
	if !layout.Detected || strings.TrimSpace(layout.Path) == "" {
		return ""
	}
	if filepath.IsAbs(layout.Path) {
		return filepath.Clean(layout.Path)
	}
	return filepath.Clean(filepath.Join(root, filepath.FromSlash(layout.Path)))
}

func workspaceModuleRecords(root string, workPath string) ([]workspaceModuleRecord, []string) {
	if strings.TrimSpace(workPath) == "" {
		return nil, nil
	}

	contents, err := os.ReadFile(workPath)
	if err != nil {
		return nil, []string{fmt.Sprintf("workspace module scan failed: %v", err)}
	}

	workFile, err := modfile.ParseWork(workPath, contents, nil)
	if err != nil {
		return nil, []string{fmt.Sprintf("workspace module scan failed: %v", err)}
	}

	workDir := filepath.Dir(filepath.Clean(workPath))
	seen := make(map[string]workspaceModuleRecord)
	for _, use := range workFile.Use {
		if use == nil {
			continue
		}
		value := strings.TrimSpace(use.Path)
		if value == "" {
			continue
		}

		moduleDir := value
		if !filepath.IsAbs(moduleDir) {
			moduleDir = filepath.Join(workDir, filepath.FromSlash(moduleDir))
		}
		moduleDir = filepath.Clean(moduleDir)

		insideRoot := workspacePathInsideRoot(root, moduleDir)
		summary := WorkspaceModule{
			Path:       displayPath(root, moduleDir),
			ModulePath: readModulePath(moduleDir),
			InsideRoot: insideRoot,
			Included:   insideRoot,
		}
		seen[moduleDir] = workspaceModuleRecord{Summary: summary, Dir: moduleDir}
	}

	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	records := make([]workspaceModuleRecord, 0, len(dirs))
	for _, dir := range dirs {
		records = append(records, seen[dir])
	}
	return records, nil
}

func workspaceModuleSummaries(records []workspaceModuleRecord) []WorkspaceModule {
	summaries := make([]WorkspaceModule, 0, len(records))
	for _, record := range records {
		summaries = append(summaries, record.Summary)
	}
	return summaries
}

func skippedWorkspaceModules(records []workspaceModuleRecord) []string {
	var skipped []string
	for _, record := range records {
		if !record.Summary.InsideRoot {
			skipped = append(skipped, record.Summary.Path)
		}
	}
	return uniqueSorted(skipped)
}

func skippedNestedModules(nested []string, workspaceModules []WorkspaceModule) []string {
	included := make(map[string]struct{}, len(workspaceModules))
	for _, module := range workspaceModules {
		if module.InsideRoot {
			included[cleanSlashPath(module.Path)] = struct{}{}
		}
	}

	var skipped []string
	for _, module := range nested {
		cleaned := cleanSlashPath(module)
		if _, ok := included[cleaned]; ok {
			continue
		}
		skipped = append(skipped, cleaned)
	}
	return uniqueSorted(skipped)
}

func localReplacements(root string, workspaceModules []WorkspaceModule) []LocalReplacement {
	moduleDirs := map[string]string{}
	if regularFileExists(filepath.Join(root, "go.mod")) {
		moduleDirs[root] = "."
	}
	for _, module := range workspaceModules {
		if !module.InsideRoot {
			continue
		}
		dir := filepath.Join(root, filepath.FromSlash(module.Path))
		moduleDirs[filepath.Clean(dir)] = cleanSlashPath(module.Path)
	}

	dirs := make([]string, 0, len(moduleDirs))
	for dir := range moduleDirs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	var replacements []LocalReplacement
	for _, dir := range dirs {
		replacements = append(replacements, localReplacementsFromGoMod(root, dir, moduleDirs[dir])...)
	}
	return normalizeLocalReplacements(replacements)
}

func localReplacementsFromGoMod(root string, moduleDir string, owner string) []LocalReplacement {
	goModPath := filepath.Join(moduleDir, "go.mod")
	contents, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}
	modFile, err := modfile.Parse(goModPath, contents, nil)
	if err != nil {
		return nil
	}

	var replacements []LocalReplacement
	for _, replace := range modFile.Replace {
		if replace == nil || strings.TrimSpace(replace.New.Version) != "" {
			continue
		}
		replacement := strings.TrimSpace(replace.New.Path)
		if replacement == "" || (!filepath.IsAbs(replacement) && !strings.HasPrefix(replacement, ".")) {
			continue
		}

		replacementDir := replacement
		if !filepath.IsAbs(replacementDir) {
			replacementDir = filepath.Join(moduleDir, filepath.FromSlash(replacementDir))
		}
		replacementDir = filepath.Clean(replacementDir)
		replacements = append(replacements, LocalReplacement{
			Owner:       owner,
			ModulePath:  replace.Old.Path,
			Replacement: filepath.ToSlash(replacement),
			Path:        displayPath(root, replacementDir),
			InsideRoot:  workspacePathInsideRoot(root, replacementDir),
		})
	}
	return replacements
}

func outsideRootLocalReplacementPaths(replacements []LocalReplacement) []string {
	var paths []string
	for _, replacement := range replacements {
		if !replacement.InsideRoot {
			paths = append(paths, replacement.Path)
		}
	}
	return uniqueSorted(paths)
}

func normalizeWorkspaceModules(modules []WorkspaceModule) []WorkspaceModule {
	modules = append([]WorkspaceModule{}, modules...)
	sort.Slice(modules, func(i int, j int) bool {
		if modules[i].Path != modules[j].Path {
			return modules[i].Path < modules[j].Path
		}
		return modules[i].ModulePath < modules[j].ModulePath
	})
	if len(modules) == 0 {
		return []WorkspaceModule{}
	}
	return modules
}

func normalizeLocalReplacements(replacements []LocalReplacement) []LocalReplacement {
	replacements = append([]LocalReplacement{}, replacements...)
	sort.Slice(replacements, func(i int, j int) bool {
		if replacements[i].Owner != replacements[j].Owner {
			return replacements[i].Owner < replacements[j].Owner
		}
		if replacements[i].ModulePath != replacements[j].ModulePath {
			return replacements[i].ModulePath < replacements[j].ModulePath
		}
		return replacements[i].Path < replacements[j].Path
	})
	if len(replacements) == 0 {
		return []LocalReplacement{}
	}
	return replacements
}

func cleanSlashPath(value string) string {
	value = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(value))))
	if value == "." || value == "" {
		return "."
	}
	return value
}

func shouldSkipLayoutDir(name string) bool {
	switch name {
	case ".git", ".gosherpa", "vendor":
		return true
	default:
		return false
	}
}

func displayPath(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err == nil && relative != "." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != ".." {
		return filepath.ToSlash(relative)
	}
	if err == nil && relative == "." {
		return "."
	}
	return filepath.Clean(path)
}
