package impact

import (
	"sort"
	"strings"

	"github.com/panndabea/GoSherpa/internal/sherpa"
)

const (
	InterfaceRelationshipKindImplementation = "interface-implementation"
	InterfaceRelationshipKindSatisfied      = "satisfied-interface"
)

type InterfaceRelationship struct {
	Kind           string
	Package        string
	File           string
	Interface      sherpa.RelationshipSymbolIdentity
	Implementation sherpa.RelationshipSymbolIdentity
	AnalysisMode   string
	Position       sherpa.Position
	Limitations    []string
}

func BuildInterfaceRelationshipsWithOptions(root string, options InterfaceOptions) ([]InterfaceRelationship, string, []string, error) {
	graph, err := buildInterfaceGraph(root, options)
	if err != nil {
		return nil, "", nil, err
	}

	interfacesByQualified := interfacesByQualifiedName(graph.Interfaces)
	typesByQualified := typesByQualifiedName(graph.Types)

	var records []InterfaceRelationship
	for interfaceName, implementations := range graph.ImplementationsByIface {
		iface, ok := interfacesByQualified[interfaceName]
		if !ok {
			continue
		}

		for _, implementationName := range implementations {
			typ, ok := typesByQualified[implementationName]
			if !ok {
				continue
			}

			records = append(records, InterfaceRelationship{
				Kind:           InterfaceRelationshipKindImplementation,
				Package:        typ.Package,
				File:           typ.Position.File,
				Interface:      relationshipIdentityFromInterfaceInfo(iface),
				Implementation: relationshipIdentityFromTypeInfo(typ),
				AnalysisMode:   graph.AnalysisMode,
				Position:       typ.Position,
			})
			records = append(records, InterfaceRelationship{
				Kind:           InterfaceRelationshipKindSatisfied,
				Package:        typ.Package,
				File:           iface.Position.File,
				Interface:      relationshipIdentityFromInterfaceInfo(iface),
				Implementation: relationshipIdentityFromTypeInfo(typ),
				AnalysisMode:   graph.AnalysisMode,
				Position:       iface.Position,
			})
		}
	}

	sort.Slice(records, func(i int, j int) bool {
		return interfaceRelationshipKey(records[i]) < interfaceRelationshipKey(records[j])
	})

	return records, graph.AnalysisMode, graph.Warnings, nil
}

func relationshipIdentityFromInterfaceInfo(iface interfaceInfo) sherpa.RelationshipSymbolIdentity {
	return sherpa.RelationshipSymbolIdentity{
		Package:       iface.Package,
		Name:          iface.Name,
		QualifiedName: iface.Qualified,
		Kind:          sherpa.SymbolKindInterface,
		Position:      iface.Position,
	}
}

func relationshipIdentityFromTypeInfo(typ typeInfo) sherpa.RelationshipSymbolIdentity {
	return sherpa.RelationshipSymbolIdentity{
		Package:       typ.Package,
		Name:          typ.Name,
		QualifiedName: typ.Qualified,
		Position:      typ.Position,
	}
}

func interfaceRelationshipKey(record InterfaceRelationship) string {
	return strings.Join([]string{
		record.Kind,
		record.Package,
		record.File,
		record.Interface.QualifiedName,
		record.Implementation.QualifiedName,
		record.AnalysisMode,
		record.Position.File,
	}, "\x00")
}
