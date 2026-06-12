package sherpa

type SymbolKind string

const (
	SymbolKindStruct    SymbolKind = "struct"
	SymbolKindInterface SymbolKind = "interface"
	SymbolKindFunction  SymbolKind = "function"
	SymbolKindMethod    SymbolKind = "method"
)

type Position struct {
	File string
	Line int
}

type Symbol struct {
	Name     string
	Kind     SymbolKind
	Position Position
	Receiver string
}