package auth

type Config struct {
	JWTSecret    []byte
	JWTIssuer    string
	JWTAudience  string
	WebOrigin    string
	CookieSecure bool
}
