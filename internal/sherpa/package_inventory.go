package sherpa

import (
	"path/filepath"
	"sort"
	"strings"
)

type PackageInventoryOptions struct {
	IncludeTests bool
}

type PackageSummary struct {
	Package         string `json:"package"`
	PackageName     string `json:"packageName"`
	GoFiles         int    `json:"goFiles"`
	TestFiles       int    `json:"testFiles"`
	Symbols         int    `json:"symbols"`
	Imports         int    `json:"imports"`
	LocalImports    int    `json:"localImports"`
	ExternalImports int    `json:"externalImports"`
	ImportedBy      int    `json:"importedBy"`
	HasTests        bool   `json:"hasTests"`
}

type packageSummaryBuilder struct {
	summary            PackageSummary
	hasProductionName  bool
	localImportTargets map[string]struct{}
	externalImports    map[string]struct{}
	importedBy         map[string]struct{}
}

func FindPackageSummaries(root string, options PackageInventoryOptions) ([]PackageSummary, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	modulePath, err := modulePath(rootPath)
	if err != nil {
		return nil, err
	}

	files, err := FindGoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	builders := map[string]*packageSummaryBuilder{}
	for _, file := range files {
		packagePath, err := packagePathForFile(rootPath, file)
		if err != nil {
			return nil, err
		}

		metadata, err := parsePackageFileMetadata(file)
		if err != nil {
			return nil, err
		}

		builder := ensurePackageSummaryBuilder(builders, packagePath)
		isTestFile := strings.HasSuffix(filepath.ToSlash(file), "_test.go")
		applyPackageFileMetadata(builder, metadata, isTestFile)
		if isTestFile && !options.IncludeTests {
			continue
		}

		addPackageImports(builder, metadata.Imports, modulePath)

		symbols, err := ParseFile(file)
		if err != nil {
			return nil, err
		}
		builder.summary.Symbols += len(symbols)
	}

	addPackageImportedByCounts(builders)

	return packageSummariesFromBuilders(builders), nil
}

func ensurePackageSummaryBuilder(builders map[string]*packageSummaryBuilder, packagePath string) *packageSummaryBuilder {
	builder, ok := builders[packagePath]
	if ok {
		return builder
	}

	builder = &packageSummaryBuilder{
		summary: PackageSummary{
			Package: packagePath,
		},
		localImportTargets: map[string]struct{}{},
		externalImports:    map[string]struct{}{},
		importedBy:         map[string]struct{}{},
	}
	builders[packagePath] = builder

	return builder
}

func applyPackageFileMetadata(builder *packageSummaryBuilder, metadata packageFileMetadata, isTestFile bool) {
	if isTestFile {
		builder.summary.TestFiles++
		builder.summary.HasTests = true
		if !builder.hasProductionName && builder.summary.PackageName == "" {
			builder.summary.PackageName = metadata.PackageName
		}
		return
	}

	builder.summary.GoFiles++
	if metadata.PackageName != "" {
		builder.summary.PackageName = metadata.PackageName
		builder.hasProductionName = true
	}
}

func addPackageImports(builder *packageSummaryBuilder, imports []string, modulePath string) {
	for _, importPath := range imports {
		localPath, ok := localPackagePath(importPath, modulePath)
		if ok {
			builder.localImportTargets[localPath] = struct{}{}
			continue
		}

		builder.externalImports[importPath] = struct{}{}
	}
}

func addPackageImportedByCounts(builders map[string]*packageSummaryBuilder) {
	for packagePath, builder := range builders {
		for importedPackage := range builder.localImportTargets {
			importedBuilder, ok := builders[importedPackage]
			if !ok || importedPackage == packagePath {
				continue
			}

			importedBuilder.importedBy[packagePath] = struct{}{}
		}
	}
}

func packageSummariesFromBuilders(builders map[string]*packageSummaryBuilder) []PackageSummary {
	summaries := make([]PackageSummary, 0, len(builders))
	for _, builder := range builders {
		builder.summary.LocalImports = len(builder.localImportTargets)
		builder.summary.ExternalImports = len(builder.externalImports)
		builder.summary.Imports = builder.summary.LocalImports + builder.summary.ExternalImports
		builder.summary.ImportedBy = len(builder.importedBy)
		summaries = append(summaries, builder.summary)
	}

	sort.SliceStable(summaries, func(i int, j int) bool {
		return summaries[i].Package < summaries[j].Package
	})

	return summaries
}
