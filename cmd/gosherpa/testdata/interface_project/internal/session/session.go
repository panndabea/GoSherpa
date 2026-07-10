package session

import "example.com/interfaces/internal/auth"

func Run(authenticator auth.Authenticator) error {
	return authenticator.Authenticate()
}
