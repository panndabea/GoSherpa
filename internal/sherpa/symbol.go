package sherpa

type SymbolKind string

const (
	SymbolKindStruct    SymbolKind = "struct"
	SymbolKindInterface SymbolKind = "interface"
	SymbolKindAlias     SymbolKind = "alias"
	SymbolKindFunction  SymbolKind = "function"
	SymbolKindMethod    SymbolKind = "method"
)

type Position struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column,omitempty"`
}

type SourceRange struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type SymbolField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Tag      string `json:"tag,omitempty"`
	Embedded bool   `json:"embedded,omitempty"`
}

type SymbolMethod struct {
	Name      string `json:"name"`
	Signature string `json:"signature"`
	Embedded  bool   `json:"embedded,omitempty"`
}

type Symbol struct {
	Name          string         `json:"name"`
	Kind          SymbolKind     `json:"kind"`
	Package       string         `json:"package,omitempty"`
	PackageName   string         `json:"packageName,omitempty"`
	QualifiedName string         `json:"qualifiedName,omitempty"`
	Signature     string         `json:"signature,omitempty"`
	Documentation string         `json:"documentation,omitempty"`
	Position      Position       `json:"position"`
	Range         *SourceRange   `json:"range,omitempty"`
	Receiver      string         `json:"receiver"`
	ReceiverType  string         `json:"receiverType,omitempty"`
	Fields        []SymbolField  `json:"fields,omitempty"`
	Methods       []SymbolMethod `json:"methods,omitempty"`
}

func (symbol Symbol) DisplayName() string {
	if symbol.Receiver == "" {
		return symbol.Name
	}

	return symbol.Receiver + "." + symbol.Name
}
