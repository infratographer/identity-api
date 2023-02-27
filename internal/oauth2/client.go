package oauth2

import (
	"strings"

	"github.com/google/uuid"
	"github.com/ory/fosite"
)

type Client struct {
	ID         uuid.UUID
	Secret     string
	Audience   []string
	Scope      string
	GrantTypes []string

	// Requested Client Authentication method for the Token Endpoint. The options are:
	//
	// - `client_secret_post`: (default) Send `client_id` and `client_secret` as `application/x-www-form-urlencoded` in the HTTP body.
	// - `client_secret_basic`: Send `client_id` and `client_secret` as `application/x-www-form-urlencoded` encoded in the HTTP Authorization header.
	// - `private_key_jwt`: Use JSON Web Tokens to authenticate the client.
	// - `none`: Used for public clients (native apps, mobile apps) which can not have secrets.
	TokenEndpointAuthMethod string
}

// GetAudience implements fosite.Client
func (c Client) GetAudience() fosite.Arguments {
	return fosite.Arguments(c.Audience)
}

// GetGrantTypes implements fosite.Client
func (c Client) GetGrantTypes() fosite.Arguments {
	if len(c.GrantTypes) == 0 {
		authCode := string(fosite.GrantTypeAuthorizationCode)
		return fosite.Arguments([]string{authCode})
	}

	return fosite.Arguments(c.GrantTypes)
}

// GetHashedSecret implements fosite.Client
func (c Client) GetHashedSecret() []byte {
	return []byte(c.Secret)
}

// GetID implements fosite.Client
func (c Client) GetID() string {
	return c.ID.String()
}

// GetRedirectURIs implements fosite.Client
func (Client) GetRedirectURIs() []string {
	panic("unimplemented")
}

// GetResponseTypes implements fosite.Client
func (Client) GetResponseTypes() fosite.Arguments {
	panic("unimplemented")
}

// GetScopes implements fosite.Client
func (c Client) GetScopes() fosite.Arguments {
	return fosite.Arguments(strings.Fields(c.Scope))
}

// IsPublic implements fosite.Client
func (c Client) IsPublic() bool {
	return c.TokenEndpointAuthMethod == "none"
}
