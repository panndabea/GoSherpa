package sherpa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

const TestInventoryLimitationDynamicSubtests = "Literal t.Run subtest names are recorded when statically visible; dynamic table-driven names remain limitations."

type TestInventory struct {
	Packages    []TestPackageRecord  `json:"packages"`
	Files       []TestFileRecord     `json:"files"`
	Functions   []TestFunctionRecord `json:"functions"`
	Limitations []string             `json:"limitations"`
}

type TestPackageRecord struct {
	Package          string   `json:"package"`
	PackageName      string   `json:"packageName"`
	ExternalPackage  bool     `json:"externalPackage"`
	Files            []string `json:"files"`
	TestFunctions    int      `json:"testFunctions"`
	Subtests         int      `json:"subtests"`
	SuiteLikeMethods int      `json:"suiteLikeMethods"`
}

type TestFileRecord struct {
	File             string               `json:"file"`
	Package          string               `json:"package"`
	PackageName      string               `json:"packageName"`
	ExternalPackage  bool                 `json:"externalPackage"`
	Functions        []TestFunctionRecord `json:"functions"`
	SuiteLikeMethods []TestFunctionRecord `json:"suiteLikeMethods"`

	info testFileInfo
}

type TestFunctionRecord struct {
	Name            string              `json:"name"`
	Package         string              `json:"package"`
	PackageName     string              `json:"packageName"`
	Position        Position            `json:"position"`
	Range           *SourceRange        `json:"range,omitempty"`
	Subtests        []TestSubtestRecord `json:"subtests"`
	SuiteLike       bool                `json:"suiteLike,omitempty"`
	ExternalPackage bool                `json:"externalPackage"`
}

type TestSubtestRecord struct {
	Name     string       `json:"name"`
	Position Position     `json:"position"`
	Range    *SourceRange `json:"range,omitempty"`
}

func BuildTestInventory(root string) (TestInventory, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return TestInventory{}, err
	}

	files, err := FindGoFiles(rootPath)
	if err != nil {
		return TestInventory{}, err
	}

	sort.Strings(files)

	records := make([]TestFileRecord, 0)
	for _, path := range files {
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}

		record, err := buildTestFileRecord(rootPath, path)
		if err != nil {
			return TestInventory{}, err
		}
		records = append(records, record)
	}

	return normalizeTestInventory(TestInventory{
		Files:       records,
		Limitations: []string{TestInventoryLimitationDynamicSubtests},
	}), nil
}

func buildTestFileRecord(root string, path string) (TestFileRecord, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return TestFileRecord{}, fmt.Errorf("parse %s: %w", path, err)
	}

	packagePath, err := packagePathForFile(root, path)
	if err != nil {
		return TestFileRecord{}, err
	}

	record := TestFileRecord{
		File:             displayPath(root, path),
		Package:          packagePath,
		PackageName:      file.Name.Name,
		ExternalPackage:  isExternalTestPackage(file.Name.Name),
		Functions:        []TestFunctionRecord{},
		SuiteLikeMethods: []TestFunctionRecord{},
		info: testFileInfo{
			Package:     packagePath,
			PackageName: file.Name.Name,
			FileSet:     fileSet,
			File:        file,
		},
	}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(funcDecl.Name.Name, "Test") {
			continue
		}

		testRecord := testFunctionRecord(root, fileSet, packagePath, file.Name.Name, funcDecl)
		testRecord.ExternalPackage = record.ExternalPackage
		if isGoTestFunction(funcDecl) {
			record.Functions = append(record.Functions, testRecord)
			continue
		}
		if funcDecl.Recv != nil {
			testRecord.SuiteLike = true
			record.SuiteLikeMethods = append(record.SuiteLikeMethods, testRecord)
		}
	}

	sortTestFunctionRecords(record.Functions)
	sortTestFunctionRecords(record.SuiteLikeMethods)

	return record, nil
}

func testFunctionRecord(root string, fileSet *token.FileSet, packagePath string, packageName string, funcDecl *ast.FuncDecl) TestFunctionRecord {
	position := fileSet.Position(funcDecl.Pos())
	return TestFunctionRecord{
		Name:        funcDecl.Name.Name,
		Package:     packagePath,
		PackageName: packageName,
		Position: positionRelativeToRoot(root, Position{
			File:   position.Filename,
			Line:   position.Line,
			Column: position.Column,
		}),
		Range:    sourceRangeRelativeToRoot(root, fileSet, funcDecl.Pos(), funcDecl.End()),
		Subtests: testSubtestRecords(root, fileSet, literalSubtests(funcDecl)),
	}
}

func testSubtestRecords(root string, fileSet *token.FileSet, subtests []literalSubtest) []TestSubtestRecord {
	records := make([]TestSubtestRecord, 0, len(subtests))
	for _, subtest := range subtests {
		position := fileSet.Position(subtest.Pos)
		records = append(records, TestSubtestRecord{
			Name: subtest.Name,
			Position: positionRelativeToRoot(root, Position{
				File:   position.Filename,
				Line:   position.Line,
				Column: position.Column,
			}),
			Range: sourceRangeRelativeToRoot(root, fileSet, subtest.Pos, subtest.End),
		})
	}

	sort.Slice(records, func(i int, j int) bool {
		if records[i].Position.File != records[j].Position.File {
			return records[i].Position.File < records[j].Position.File
		}
		if records[i].Position.Line != records[j].Position.Line {
			return records[i].Position.Line < records[j].Position.Line
		}
		if records[i].Position.Column != records[j].Position.Column {
			return records[i].Position.Column < records[j].Position.Column
		}
		return records[i].Name < records[j].Name
	})

	return records
}

func normalizeTestInventory(inventory TestInventory) TestInventory {
	inventory.Files = nonNilTestFileRecords(inventory.Files)
	sort.Slice(inventory.Files, func(i int, j int) bool {
		return inventory.Files[i].File < inventory.Files[j].File
	})

	inventory.Packages = testPackageRecords(inventory.Files)
	inventory.Functions = testFunctionRecords(inventory.Files)
	inventory.Limitations = uniqueSorted(inventory.Limitations)

	return inventory
}

func testPackageRecords(files []TestFileRecord) []TestPackageRecord {
	byKey := make(map[string]*TestPackageRecord)
	for _, file := range files {
		key := file.Package + "\x00" + file.PackageName
		record := byKey[key]
		if record == nil {
			record = &TestPackageRecord{
				Package:         file.Package,
				PackageName:     file.PackageName,
				ExternalPackage: isExternalTestPackage(file.PackageName),
				Files:           []string{},
			}
			byKey[key] = record
		}

		record.Files = append(record.Files, file.File)
		record.TestFunctions += len(file.Functions)
		record.SuiteLikeMethods += len(file.SuiteLikeMethods)
		for _, function := range file.Functions {
			record.Subtests += len(function.Subtests)
		}
	}

	records := make([]TestPackageRecord, 0, len(byKey))
	for _, record := range byKey {
		record.Files = uniqueSorted(record.Files)
		records = append(records, *record)
	}

	sort.Slice(records, func(i int, j int) bool {
		if records[i].Package != records[j].Package {
			return records[i].Package < records[j].Package
		}
		return records[i].PackageName < records[j].PackageName
	})

	return records
}

func testFunctionRecords(files []TestFileRecord) []TestFunctionRecord {
	var records []TestFunctionRecord
	for _, file := range files {
		records = append(records, file.Functions...)
	}
	sortTestFunctionRecords(records)
	return nonNilTestFunctionRecords(records)
}

func testFilesFromInventory(inventory TestInventory) []testFileInfo {
	files := make([]testFileInfo, 0, len(inventory.Files))
	for _, file := range inventory.Files {
		if file.info.File == nil || file.info.FileSet == nil {
			continue
		}
		files = append(files, file.info)
	}
	return files
}

func sortTestFunctionRecords(records []TestFunctionRecord) {
	sort.Slice(records, func(i int, j int) bool {
		if records[i].Package != records[j].Package {
			return records[i].Package < records[j].Package
		}
		if records[i].Position.File != records[j].Position.File {
			return records[i].Position.File < records[j].Position.File
		}
		if records[i].Position.Line != records[j].Position.Line {
			return records[i].Position.Line < records[j].Position.Line
		}
		if records[i].Position.Column != records[j].Position.Column {
			return records[i].Position.Column < records[j].Position.Column
		}
		return records[i].Name < records[j].Name
	})
}

func nonNilTestFileRecords(values []TestFileRecord) []TestFileRecord {
	if values == nil {
		return []TestFileRecord{}
	}
	return values
}

func nonNilTestFunctionRecords(values []TestFunctionRecord) []TestFunctionRecord {
	if values == nil {
		return []TestFunctionRecord{}
	}
	return values
}
