package sherpa

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseFile(t *testing.T) {
	_, err := ParseFile("does-not-exist.go")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFileFindsStruct(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

type UserService struct {
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}

	if symbols[0].Name != "UserService" {
		t.Fatalf("expected UserService, got %s", symbols[0].Name)
	}

	if symbols[0].Kind != SymbolKindStruct {
		t.Fatalf("expected struct kind, got %s", symbols[0].Kind)
	}
}

func TestParseFileFindsFunction(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

func CreateUser() {
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}

	if symbols[0].Name != "CreateUser" {
		t.Fatalf("expected CreateUser, got %s", symbols[0].Name)
	}

	if symbols[0].Kind != SymbolKindFunction {
		t.Fatalf("expected function kind, got %s", symbols[0].Kind)
	}
}

func TestParseFileFindsMethod(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

type UserService struct{}

func (s *UserService) Create() {
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(symbols))
	}

	method := symbols[1]

	if method.Name != "Create" {
		t.Fatalf("expected Create, got %s", method.Name)
	}

	if method.Kind != SymbolKindMethod {
		t.Fatalf("expected method kind")
	}

	if method.Receiver != "UserService" {
		t.Fatalf("expected UserService receiver, got %s", method.Receiver)
	}
}

func TestParseFileFindsValueMethod(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

type UserService struct{}

func (s UserService) Create() {
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	method := symbols[1]

	if method.Receiver != "UserService" {
		t.Fatalf("expected UserService receiver, got %s", method.Receiver)
	}
}

func TestParseFileFindsInterface(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

type UserRepository interface {
	Save() error
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}

	if symbols[0].Name != "UserRepository" {
		t.Fatalf("expected UserRepository, got %s", symbols[0].Name)
	}

	if symbols[0].Kind != SymbolKindInterface {
		t.Fatalf("expected interface kind, got %s", symbols[0].Kind)
	}
}

func TestParseFileFindsFunctionDetails(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`package auth

// CreateUser creates a user.
func CreateUser(name string) (string, error) {
	return name, nil
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	symbol := findParseTestSymbol(t, symbols, "CreateUser")
	if symbol.PackageName != "auth" {
		t.Fatalf("expected package name auth, got %s", symbol.PackageName)
	}
	if symbol.Signature != "func CreateUser(name string) (string, error)" {
		t.Fatalf("expected function signature, got %s", symbol.Signature)
	}
	if symbol.Documentation != "CreateUser creates a user." {
		t.Fatalf("expected doc comment, got %q", symbol.Documentation)
	}
	if symbol.Position.Column != 1 {
		t.Fatalf("expected column 1, got %d", symbol.Position.Column)
	}
	if symbol.Range == nil || symbol.Range.End.Line != 6 {
		t.Fatalf("expected end line 6, got %#v", symbol.Range)
	}
}

func TestParseFileFindsMethodDetails(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cache.go")

	err := os.WriteFile(path, []byte(`package auth

type Cache[T any] struct{}

// Get returns a cached value.
func (c *Cache[T]) Get(key string) (T, bool) {
	var zero T
	return zero, false
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	symbol := findParseTestSymbol(t, symbols, "Get")
	if symbol.Receiver != "Cache" {
		t.Fatalf("expected Cache receiver, got %s", symbol.Receiver)
	}
	if symbol.ReceiverType != "*Cache[T]" {
		t.Fatalf("expected generic pointer receiver type, got %s", symbol.ReceiverType)
	}
	if symbol.Signature != "func (c *Cache[T]) Get(key string) (T, bool)" {
		t.Fatalf("expected method signature, got %s", symbol.Signature)
	}
	if symbol.Documentation != "Get returns a cached value." {
		t.Fatalf("expected doc comment, got %q", symbol.Documentation)
	}
}

func TestParseFileFindsStructFields(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte("package auth\n\n"+
		"type Embedded struct{}\n\n"+
		"// User stores auth data.\n"+
		"type User struct {\n"+
		"\tEmbedded\n"+
		"\tID string `json:\"id\"`\n"+
		"\tCount int\n"+
		"}\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	symbol := findParseTestSymbol(t, symbols, "User")
	want := []SymbolField{
		{Name: "Embedded", Type: "Embedded", Embedded: true},
		{Name: "ID", Type: "string", Tag: `json:"id"`},
		{Name: "Count", Type: "int"},
	}
	if !reflect.DeepEqual(symbol.Fields, want) {
		t.Fatalf("expected fields %#v, got %#v", want, symbol.Fields)
	}
	if symbol.Documentation != "User stores auth data." {
		t.Fatalf("expected type doc, got %q", symbol.Documentation)
	}
}

func TestParseFileFindsInterfaceMethods(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "repository.go")

	err := os.WriteFile(path, []byte(`package auth

type Closer interface {
	Close() error
}

// UserRepository stores users.
type UserRepository interface {
	Closer
	Save(ctx Context, id string) error
	Find(id string) (User, bool)
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	symbol := findParseTestSymbol(t, symbols, "UserRepository")
	want := []SymbolMethod{
		{Name: "Closer", Signature: "Closer", Embedded: true},
		{Name: "Save", Signature: "Save(ctx Context, id string) error"},
		{Name: "Find", Signature: "Find(id string) (User, bool)"},
	}
	if !reflect.DeepEqual(symbol.Methods, want) {
		t.Fatalf("expected methods %#v, got %#v", want, symbol.Methods)
	}
}

func TestParseFileFindsTypeAlias(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "auth.go")

	err := os.WriteFile(path, []byte(`package auth

import "example.com/app/internal/model"

// UserID is the auth-facing user identifier.
type UserID = model.UserID
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	symbol := findParseTestSymbol(t, symbols, "UserID")
	if symbol.Kind != SymbolKindAlias {
		t.Fatalf("expected alias kind, got %s", symbol.Kind)
	}
	if symbol.Signature != "type UserID = model.UserID" {
		t.Fatalf("expected alias signature, got %s", symbol.Signature)
	}
	if symbol.Documentation != "UserID is the auth-facing user identifier." {
		t.Fatalf("expected alias doc comment, got %q", symbol.Documentation)
	}
	if symbol.Range == nil || symbol.Range.End.Line != 6 {
		t.Fatalf("expected alias range ending at line 6, got %#v", symbol.Range)
	}
}

func TestParseRepositoryAddsPackageAndQualifiedName(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), `package auth

func Run() {}
`)

	symbols, err := ParseRepository(tmp)
	if err != nil {
		t.Fatal(err)
	}

	symbol := findParseTestSymbol(t, symbols, "Run")
	if symbol.Package != "./internal/auth" {
		t.Fatalf("expected package ./internal/auth, got %s", symbol.Package)
	}
	if symbol.QualifiedName != "./internal/auth.Run" {
		t.Fatalf("expected qualified name, got %s", symbol.QualifiedName)
	}
	if symbol.Position.File != "internal/auth/service.go" {
		t.Fatalf("expected root-relative file, got %s", symbol.Position.File)
	}
	if symbol.Range == nil || symbol.Range.Start.File != "internal/auth/service.go" {
		t.Fatalf("expected root-relative range, got %#v", symbol.Range)
	}
}

func findParseTestSymbol(t *testing.T, symbols []Symbol, name string) Symbol {
	t.Helper()

	for _, symbol := range symbols {
		if symbol.Name == name {
			return symbol
		}
	}

	t.Fatalf("expected symbol %s in %#v", name, symbols)
	return Symbol{}
}
