package types

import (
	"github.com/ory/fosite"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/gidx"
)

// OAuthClient is an OAuth 2.0 Client
type OAuthClient struct {
	ID       gidx.PrefixedID
	OwnerID  gidx.PrefixedID
	Name     string
	Secret   string
	Audience []string
}

// GetAudience implements fosite.Client
func (c OAuthClient) GetAudience() fosite.Arguments {
	return fosite.Arguments(c.Audience)
}

// GetGrantTypes implements fosite.Client
func (OAuthClient) GetGrantTypes() fosite.Arguments {
	panic("unimplemented")
}

// GetHashedSecret implements fosite.Client
func (c OAuthClient) GetHashedSecret() []byte {
	return []byte(c.Secret)
}

// GetID implements fosite.Client
func (c OAuthClient) GetID() string {
	return c.ID.String()
}

// GetRedirectURIs implements fosite.Client
func (OAuthClient) GetRedirectURIs() []string {
	panic("unimplemented")
}

// GetResponseTypes implements fosite.Client
func (OAuthClient) GetResponseTypes() fosite.Arguments {
	panic("unimplemented")
}

// GetScopes implements fosite.Client
func (c OAuthClient) GetScopes() fosite.Arguments {
	panic("unimplemented")
}

// IsPublic implements fosite.Client
func (OAuthClient) IsPublic() bool {
	return false
}

// ToV1OAuthClient converts to the OAS OAuth Client type.
func (c OAuthClient) ToV1OAuthClient() v1.OAuthClient {
	var client v1.OAuthClient

	client.ID = c.ID
	client.Name = c.Name
	client.Audience = c.Audience

	return client
}
