package agentcontext

import (
	"sort"

	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func prioritizeContextTests(tests []sherpa.RelatedTest) []sherpa.RelatedTest {
	if len(tests) < 2 {
		return tests
	}

	prioritized := append([]sherpa.RelatedTest{}, tests...)
	sort.SliceStable(prioritized, func(i int, j int) bool {
		left := prioritized[i]
		right := prioritized[j]
		if left.DirectReference != right.DirectReference {
			return left.DirectReference
		}
		if left.Package != right.Package {
			return left.Package < right.Package
		}
		if left.Position.File != right.Position.File {
			return left.Position.File < right.Position.File
		}
		if left.Position.Line != right.Position.Line {
			return left.Position.Line < right.Position.Line
		}
		return left.Name < right.Name
	})

	return prioritized
}
