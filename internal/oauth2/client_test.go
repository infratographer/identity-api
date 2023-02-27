package oauth2

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
)

var _ fosite.Client = Client{}

func TestClient(t *testing.T) {
	id := uuid.New()
	c := Client{
		ID:                      id,
		Secret:                  "fakesecret",
		Audience:                []string{"aud1", "aud2"},
		Scope:                   "openid profile email",
		TokenEndpointAuthMethod: "none",
	}

	assert.True(t, c.IsPublic())
	assert.Equal(t, fosite.Arguments([]string{"openid", "profile", "email"}), c.GetScopes())
	assert.Equal(t, []byte("fakesecret"), c.GetHashedSecret())
	assert.Equal(t, id.String(), c.GetID())
	assert.Equal(t, fosite.Arguments([]string{"aud1", "aud2"}), c.GetAudience())
}
