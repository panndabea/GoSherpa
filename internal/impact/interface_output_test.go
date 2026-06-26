package impact

import (
	"fmt"
	"testing"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

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
