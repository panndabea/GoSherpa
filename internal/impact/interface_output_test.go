package impact

import (
	"fmt"
	"testing"

	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func TestFormatInterface(t *testing.T) {
	result := InterfaceResult{
		Target: "./internal/auth.Authenticator",
		Position: sherpa.Position{
			File: "internal/auth/auth.go",
			Line: 3,
		},
		AnalysisMode:            InterfaceAnalysisModeTypechecked,
		ReferenceAnalysisMode:   sherpa.ReferenceAnalysisModeTypechecked,
		MethodUsageAnalysisMode: InterfaceAnalysisModeTypechecked,
		Methods: []InterfaceMethod{
			{
				Name:      "Authenticate",
				Signature: "func() error",
				Usages: []InterfaceMethodUsage{
					{
						Kind: sherpa.ReferenceKindCall,
						Position: sherpa.Position{
							File: "internal/session/session.go",
							Line: 6,
						},
					},
				},
			},
		},
		Implementers: []Implementer{
			{
				Name: "./internal/jwt.JWTAuthenticator",
				Position: sherpa.Position{
					File: "internal/jwt/jwt.go",
					Line: 3,
				},
			},
		},
		References: []sherpa.Reference{
			{
				Kind: sherpa.ReferenceKindDefinition,
				Position: sherpa.Position{
					File: "internal/auth/auth.go",
					Line: 3,
				},
			},
			{
				Kind: sherpa.ReferenceKindTypeUsage,
				Position: sherpa.Position{
					File: "internal/session/session.go",
					Line: 5,
				},
			},
		},
		Limitations: []string{"Interface method usage reports statically visible selector usages only."},
	}

	got := FormatInterface(result)
	want := fmt.Sprintf(`INTERFACE

Target: ./internal/auth.Authenticator
Definition: internal/auth/auth.go:3
Analysis: typechecked
References: typechecked
Method usage: typechecked

METHODS
  Authenticate func() error
    call       internal/session/session.go:6

IMPLEMENTERS
  %-36s internal/jwt/jwt.go:3

REFERENCES
  definition internal/auth/auth.go:3
  type_usage internal/session/session.go:5

LIMITATIONS
  Interface method usage reports statically visible selector usages only.

Found 1 methods, 1 implementers, 2 references
`, "./internal/jwt.JWTAuthenticator")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatImplementers(t *testing.T) {
	result := ImplementersResult{
		Target: "./internal/auth.Authenticator",
		Implementers: []Implementer{
			{
				Name: "./internal/jwt.JWTAuthenticator",
				Position: sherpa.Position{
					File: "internal/jwt/jwt.go",
					Line: 3,
				},
			},
		},
	}

	got := FormatImplementers(result)
	want := fmt.Sprintf(`IMPLEMENTERS

./internal/auth.Authenticator

  %-36s internal/jwt/jwt.go:3

Found 1 implementers
`, "./internal/jwt.JWTAuthenticator")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatInterfaces(t *testing.T) {
	result := InterfacesResult{
		Target: "./internal/jwt.JWTAuthenticator",
		Interfaces: []SatisfiedInterface{
			{
				Name: "./internal/auth.Authenticator",
				Position: sherpa.Position{
					File: "internal/auth/auth.go",
					Line: 3,
				},
			},
		},
	}

	got := FormatInterfaces(result)
	want := fmt.Sprintf(`INTERFACES

./internal/jwt.JWTAuthenticator

  %-36s internal/auth/auth.go:3

Found 1 interfaces
`, "./internal/auth.Authenticator")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatImplementersWithEmptyList(t *testing.T) {
	got := FormatImplementers(ImplementersResult{Target: "Missing"})
	want := "no implementers found: Missing\n"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatInterfacesWithEmptyList(t *testing.T) {
	got := FormatInterfaces(InterfacesResult{Target: "PlainType"})
	want := "no interfaces found: PlainType\n"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
