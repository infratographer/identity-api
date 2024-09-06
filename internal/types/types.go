// Package types defines all non-http types used in the STS.
package types

import (
	"context"
	"encoding/json"

	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/encoding/prototext"

	"go.infratographer.com/identity-api/internal/celutils"
	"go.infratographer.com/identity-api/internal/crdbx"
	v1 "go.infratographer.com/identity-api/pkg/api/v1"
	"go.infratographer.com/x/gidx"
)

// Issuers represents a list of token issuers.
type Issuers []*Issuer

// ToV1Issuers converts an slice of issuers to a slice of API issuers.
func (i Issuers) ToV1Issuers() ([]v1.Issuer, error) {
	issuers := make([]v1.Issuer, len(i))

	for i, iss := range i {
		v1iss, err := iss.ToV1Issuer()
		if err != nil {
			return nil, err
		}

		issuers[i] = v1iss
	}

	return issuers, nil
}

// Issuer represents a token issuer.
type Issuer struct {
	// OwnerID represents the ID of the owner the issuer belongs to.
	OwnerID gidx.PrefixedID
	// ID represents the ID of the issuer in identity-api.
	ID gidx.PrefixedID
	// Name represents the human-readable name of the issuer.
	Name string
	// URI represents the issuer URI as found in the "iss" claim of a JWT.
	URI string
	// JWKSURI represents the URI where the issuer's JWKS lives. Must be accessible by identity-api.
	JWKSURI string
	// ClaimMappings represents a map of claims to a CEL expression that will be evaluated
	ClaimMappings ClaimsMapping
}

// ToV1Issuer converts an issuer to an API issuer.
func (i Issuer) ToV1Issuer() (v1.Issuer, error) {
	claimsMappingRepr, err := i.ClaimMappings.Repr()
	if err != nil {
		return v1.Issuer{}, err
	}

	out := v1.Issuer{
		ID:            i.ID,
		Name:          i.Name,
		URI:           i.URI,
		JWKSURI:       i.JWKSURI,
		ClaimMappings: claimsMappingRepr,
	}

	return out, nil
}

// IssuerUpdate represents an update operation on an issuer.
type IssuerUpdate struct {
	Name          *string
	URI           *string
	JWKSURI       *string
	ClaimMappings ClaimsMapping
}

// IssuerService represents a service for managing issuers.
type IssuerService interface {
	CreateIssuer(ctx context.Context, iss Issuer) (*Issuer, error)
	GetIssuerByID(ctx context.Context, id gidx.PrefixedID) (*Issuer, error)
	GetOwnerIssuers(ctx context.Context, id gidx.PrefixedID, pagination crdbx.Paginator) (Issuers, error)
	GetIssuerByURI(ctx context.Context, uri string) (*Issuer, error)
	UpdateIssuer(ctx context.Context, id gidx.PrefixedID, update IssuerUpdate) (*Issuer, error)
	DeleteIssuer(ctx context.Context, id gidx.PrefixedID) error
}

// ClaimsMapping represents a map of claims to a CEL expression that will be evaluated
type ClaimsMapping map[string]*cel.Ast

// NewClaimsMapping creates a ClaimsMapping from the given map of CEL expressions.
func NewClaimsMapping(exprs map[string]string) (ClaimsMapping, error) {
	out := make(ClaimsMapping, len(exprs))

	for k, v := range exprs {
		ast, err := celutils.ParseCEL(v)
		if err != nil {
			return nil, err
		}

		out[k] = ast
	}

	return out, nil
}

// Repr produces a representation of the claim map using human-readable CEL expressions.
func (c ClaimsMapping) Repr() (map[string]string, error) {
	out := make(map[string]string, len(c))

	for k, v := range c {
		var err error

		out[k], err = cel.AstToString(v)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

// MarshalJSON implements the json.Marshaler interface.
func (c ClaimsMapping) MarshalJSON() ([]byte, error) {
	out := make(map[string][]byte, len(c))

	for k, v := range c {
		expr, err := cel.AstToCheckedExpr(v)
		if err != nil {
			return nil, err
		}

		b, err := prototext.Marshal(expr)
		if err != nil {
			return nil, err
		}

		out[k] = b
	}

	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *ClaimsMapping) UnmarshalJSON(data []byte) error {
	in := make(map[string][]byte)
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}

	out := make(ClaimsMapping, len(in))

	for k, v := range in {
		var expr exprpb.CheckedExpr

		err := prototext.Unmarshal(v, &expr)
		if err != nil {
			return err
		}

		out[k] = cel.CheckedExprToAst(&expr)
	}

	*c = out

	return nil
}

// BuildClaimsMappingFromMap builds a ClaimsMapping from a map of strings.
func BuildClaimsMappingFromMap(in map[string]*exprpb.CheckedExpr) ClaimsMapping {
	out := make(ClaimsMapping, len(in))

	for k, v := range in {
		out[k] = cel.CheckedExprToAst(v)
	}

	return out
}

// UserInfos represents a list of token issuers.
type UserInfos []UserInfo

// ToV1Users converts an slice of issuers to a slice of API issuers.
func (i UserInfos) ToV1Users() ([]v1.User, error) {
	users := make([]v1.User, len(i))

	for i, user := range i {
		v1user, err := user.ToV1User()
		if err != nil {
			return nil, err
		}

		users[i] = v1user
	}

	return users, nil
}

// UserInfo contains information about the user from the source OIDC provider.
// As defined in https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
type UserInfo struct {
	ID      gidx.PrefixedID `json:"-"`
	Name    string          `json:"name,omitempty"`
	Email   string          `json:"email,omitempty"`
	Issuer  string          `json:"iss"`
	Subject string          `json:"sub"`
}

// ToV1User converts an user info to an API user info.
func (u UserInfo) ToV1User() (v1.User, error) {
	var (
		name  *string
		email *string
	)

	if u.Name != "" {
		name = &u.Name
	}

	if u.Email != "" {
		email = &u.Email
	}

	out := v1.User{
		ID:      u.ID,
		Name:    name,
		Email:   email,
		Issuer:  u.Issuer,
		Subject: u.Subject,
	}

	return out, nil
}

// UserInfoService defines the storage class for storing User
// information related to the subject tokens.
type UserInfoService interface {
	// LookupUserInfoByClaims returns the User information object for a issuer, subject pair.
	LookupUserInfoByClaims(ctx context.Context, iss, sub string) (UserInfo, error)

	// LookupUserInfoByID returns the user info for a STS user ID
	LookupUserInfoByID(ctx context.Context, id gidx.PrefixedID) (UserInfo, error)

	// LookupUserOwnerID finds the Owner ID of the Issuer for the given User ID.
	LookupUserOwnerID(ctx context.Context, id gidx.PrefixedID) (gidx.PrefixedID, error)

	// LookupUserInfosByIssuerID returns the user infos for an STS issuer ID
	LookupUserInfosByIssuerID(ctx context.Context, id gidx.PrefixedID, paginator crdbx.Paginator) (UserInfos, error)

	// StoreUserInfo stores the userInfo into the storage backend.
	StoreUserInfo(ctx context.Context, userInfo UserInfo) (UserInfo, error)

	// ParseUserInfoFromClaims parses OIDC ID token claims from the given claim map.
	ParseUserInfoFromClaims(claims map[string]any) (UserInfo, error)
}

// OAuthClientManager defines the storage interface for OAuth clients.
type OAuthClientManager interface {
	CreateOAuthClient(ctx context.Context, client OAuthClient) (OAuthClient, error)
	LookupOAuthClientByID(ctx context.Context, clientID gidx.PrefixedID) (OAuthClient, error)
	DeleteOAuthClient(ctx context.Context, clientID gidx.PrefixedID) error
	GetOwnerOAuthClients(ctx context.Context, ownerID gidx.PrefixedID, pagination crdbx.Paginator) (OAuthClients, error)
}

// Group represents a set of subjects
type Group struct {
	// ID is the group's ID
	ID gidx.PrefixedID
	// OwnerID is the ID of the OU that owns the group
	OwnerID gidx.PrefixedID
	// Name is the group's name
	Name string
	// Description is the group's description
	Description string
}

// ToV1Group converts a group to an API group.
func (g *Group) ToV1Group() (v1.Group, error) {
	group := v1.Group{
		ID:    g.ID,
		Owner: g.OwnerID,
		Name:  g.Name,
	}

	if g.Description != "" {
		group.Description = &g.Description
	}

	return group, nil
}

// GroupUpdate represents an update operation on a group.
type GroupUpdate struct {
	Name        *string
	Description *string
}

// GroupService represents a service for managing groups.
type GroupService interface {
	CreateGroup(ctx context.Context, group Group) (*Group, error)
	GetGroupByID(ctx context.Context, id gidx.PrefixedID) (*Group, error)
	// UpdateGroup(ctx context.Context, id gidx.PrefixedID, update GroupUpdate) (*Group, error)
	// DeleteGroup(ctx context.Context, id gidx.PrefixedID) error
}
