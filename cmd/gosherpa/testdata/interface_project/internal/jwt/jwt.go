package jwt

type JWTAuthenticator struct{}

func (JWTAuthenticator) Authenticate() error {
	return nil
}
