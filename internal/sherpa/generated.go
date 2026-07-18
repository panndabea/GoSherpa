package sherpa

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const generatedHeaderScanLines = 20

var generatedCodeLinePattern = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

type GeneratedFileSummary struct {
	Path        string `json:"path"`
	Package     string `json:"package"`
	PackageName string `json:"packageName,omitempty"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type GeneratedPackageSummary struct {
	Package              string `json:"package"`
	PackageName          string `json:"packageName,omitempty"`
	Files                int    `json:"files"`
	SizeBytes            int64  `json:"sizeBytes"`
	LargestFile          string `json:"largestFile,omitempty"`
	LargestFileSizeBytes int64  `json:"largestFileSizeBytes,omitempty"`
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

	source := generatedHeaderScanSource(file, data)
	lines := strings.SplitN(source, "\n", generatedHeaderScanLines+1)
	for _, line := range lines {
		if generatedCodeLinePattern.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func generatedHeaderScanSource(file string, data []byte) string {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, file, data, parser.PackageClauseOnly)
	if err != nil || parsed == nil || !parsed.Package.IsValid() {
		return string(data)
	}

	position := fileSet.Position(parsed.Package)
	if position.Offset < 0 || position.Offset > len(data) {
		return string(data)
	}
	return string(data[:position.Offset])
}

func GeneratedGoFileSummaries(root string, files []string) []GeneratedFileSummary {
	root = filepath.Clean(root)
	summaries := make([]GeneratedFileSummary, 0, len(files))
	for _, file := range files {
		filePath := generatedFileAbsolutePath(root, file)
		if !IsGeneratedGoFile(filePath) {
			continue
		}

		summary := GeneratedFileSummary{
			Path:        generatedDisplayPath(root, filePath),
			Package:     generatedPackagePath(root, filePath),
			PackageName: generatedPackageName(filePath),
		}
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			summary.SizeBytes = info.Size()
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i int, j int) bool {
		return summaries[i].Path < summaries[j].Path
	})
	return summaries
}

func MajorGeneratedGoPackages(root string, files []string, limit int) []GeneratedPackageSummary {
	generatedFiles := GeneratedGoFileSummaries(root, files)
	if len(generatedFiles) == 0 {
		return nil
	}

	byPackage := make(map[string]GeneratedPackageSummary)
	for _, file := range generatedFiles {
		key := file.Package
		if strings.TrimSpace(key) == "" {
			key = "."
		}
		summary := byPackage[key]
		summary.Package = key
		if strings.TrimSpace(summary.PackageName) == "" {
			summary.PackageName = file.PackageName
		}
		summary.Files++
		summary.SizeBytes += file.SizeBytes
		if file.SizeBytes > summary.LargestFileSizeBytes ||
			(file.SizeBytes == summary.LargestFileSizeBytes && (summary.LargestFile == "" || file.Path < summary.LargestFile)) {
			summary.LargestFile = file.Path
			summary.LargestFileSizeBytes = file.SizeBytes
		}
		byPackage[key] = summary
	}

	packages := make([]GeneratedPackageSummary, 0, len(byPackage))
	for _, summary := range byPackage {
		packages = append(packages, summary)
	}
	sort.Slice(packages, func(i int, j int) bool {
		if packages[i].Files != packages[j].Files {
			return packages[i].Files > packages[j].Files
		}
		if packages[i].SizeBytes != packages[j].SizeBytes {
			return packages[i].SizeBytes > packages[j].SizeBytes
		}
		return packages[i].Package < packages[j].Package
	})

	if limit > 0 && len(packages) > limit {
		packages = packages[:limit]
	}
	return packages
}

func generatedFileAbsolutePath(root string, file string) string {
	filePath := filepath.FromSlash(strings.TrimSpace(file))
	if filepath.IsAbs(filePath) || strings.TrimSpace(root) == "" {
		return filepath.Clean(filePath)
	}
	return filepath.Clean(filepath.Join(root, filePath))
}

func generatedDisplayPath(root string, file string) string {
	if strings.TrimSpace(root) == "" {
		return filepath.ToSlash(filepath.Clean(file))
	}
	return displayPath(root, file)
}

func generatedPackagePath(root string, file string) string {
	if strings.TrimSpace(root) == "" {
		return "."
	}

	relativeDir, err := filepath.Rel(filepath.Clean(root), filepath.Dir(file))
	if err != nil || relativeDir == "." {
		return "."
	}
	if relativeDir == ".." || strings.HasPrefix(relativeDir, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(filepath.Clean(filepath.Dir(file)))
	}

	return filepath.ToSlash(filepath.Clean(relativeDir))
}

func generatedPackageName(file string) string {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.PackageClauseOnly)
	if err != nil || parsed == nil || parsed.Name == nil {
		return ""
	}
	return strings.TrimSpace(parsed.Name.Name)
}
