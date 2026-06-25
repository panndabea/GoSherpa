package sherpa

type SymbolKind string

const (
	SymbolKindStruct    SymbolKind = "struct"
	SymbolKindInterface SymbolKind = "interface"
	SymbolKindFunction  SymbolKind = "function"
	SymbolKindMethod    SymbolKind = "method"
)

type Position struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type Symbol struct {
	Name     string     `json:"name"`
	Kind     SymbolKind `json:"kind"`
	Position Position   `json:"position"`
	Receiver string     `json:"receiver"`
}
