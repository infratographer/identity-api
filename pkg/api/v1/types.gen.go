// Package v1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package v1

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"go.infratographer.com/identity-api/internal/crdbx"
	"go.infratographer.com/x/gidx"
)

// AddGroupMembers defines model for AddGroupMembers.
type AddGroupMembers struct {
	// MemberIDs IDs of the members to add to the group
	MemberIDs []gidx.PrefixedID `json:"member_ids"`
}

// AddGroupMembersResponse defines model for AddGroupMembersResponse.
type AddGroupMembersResponse struct {
	// Success true if the members were added successfully
	Success bool `json:"success"`
}

// CreateGroup defines model for CreateGroup.
type CreateGroup struct {
	// Description a description for the group
	Description *string `json:"description,omitempty"`

	// Name a name for the group
	Name string `json:"name"`
}

// CreateIssuer defines model for CreateIssuer.
type CreateIssuer struct {
	// ClaimMappings CEL expressions mapping token claims to other claims
	ClaimMappings *map[string]string `json:"claim_mappings,omitempty"`

	// JWKSURI JWKS URI
	JWKSURI string `json:"jwks_uri"`

	// Name A human-readable name for the issuer
	Name string `json:"name"`

	// URI URI for the issuer. Must match the "iss" claim value in incoming JWTs
	URI string `json:"uri"`
}

// CreateOAuthClient defines model for CreateOAuthClient.
type CreateOAuthClient struct {
	// Audience Audiences that this client can request
	Audience *[]string `json:"audience,omitempty"`

	// Name A human-readable name for the client
	Name string `json:"name"`
}

// DeleteResponse defines model for DeleteResponse.
type DeleteResponse struct {
	// Success Always true.
	Success bool `json:"success"`
}

// Group defines model for Group.
type Group struct {
	// Description a description for the group
	Description *string `json:"description,omitempty"`

	// ID ID of the group
	ID gidx.PrefixedID `json:"id"`

	// Name a name for the group
	Name string `json:"name"`

	// OwnerID ID of the owner of the group
	OwnerID *gidx.PrefixedID `json:"owner_id,omitempty"`
}

// Issuer defines model for Issuer.
type Issuer struct {
	// ClaimMappings CEL expressions mapping token claims to other claims
	ClaimMappings map[string]string `json:"claim_mappings"`

	// ID ID of the issuer
	ID gidx.PrefixedID `json:"id"`

	// JWKSURI JWKS URI
	JWKSURI string `json:"jwks_uri"`

	// Name A human-readable name for the issuer
	Name string `json:"name"`

	// URI URI for the issuer. Must match the "iss" claim value in incoming JWTs
	URI string `json:"uri"`
}

// IssuerUpdate defines model for IssuerUpdate.
type IssuerUpdate struct {
	// ClaimMappings CEL expressions mapping token claims to other claims
	ClaimMappings *map[string]string `json:"claim_mappings,omitempty"`

	// JWKSURI JWKS URI
	JWKSURI *string `json:"jwks_uri,omitempty"`

	// Name A human-readable name for the issuer
	Name *string `json:"name,omitempty"`

	// URI URI for the issuer. Must match the "iss" claim value in incoming JWTs
	URI *string `json:"uri,omitempty"`
}

// OAuthClient defines model for OAuthClient.
type OAuthClient struct {
	// Audience Grantable audiences
	Audience []string `json:"audience"`

	// ID OAuth 2.0 Client ID
	ID gidx.PrefixedID `json:"id"`

	// Name Description of Client
	Name string `json:"name"`

	// Secret OAuth2.0 Client Secret
	Secret *string `json:"secret,omitempty"`
}

// Pagination collection response pagination
type Pagination struct {
	// Limit the limit used for the collection response
	Limit int `json:"limit"`

	// Next the cursor for the next page
	Next *crdbx.Cursor `json:"next,omitempty"`
}

// UpdateGroup defines model for UpdateGroup.
type UpdateGroup struct {
	// Description a description for the group
	Description *string `json:"description,omitempty"`

	// Name a name for the group
	Name *string `json:"name,omitempty"`
}

// User defines model for User.
type User struct {
	// Email Email of the user
	Email *string `json:"email,omitempty"`

	// ID OAuth 2.0 User ID
	ID gidx.PrefixedID `json:"id"`

	// Issuer OAuth 2.0 Issuer of the user
	Issuer string `json:"iss"`

	// Name Name of the user
	Name *string `json:"name,omitempty"`

	// Subject OAuth 2.0 Subject for the user
	Subject string `json:"sub"`
}

// GroupID defines model for groupID.
type GroupID = gidx.PrefixedID

// IssuerID defines model for issuerID.
type IssuerID = gidx.PrefixedID

// OwnerID defines model for ownerID.
type OwnerID = gidx.PrefixedID

// PageCursor defines model for pageCursor.
type PageCursor = crdbx.Cursor

// PageLimit defines model for pageLimit.
type PageLimit = int

// SubjectID defines model for subjectID.
type SubjectID = gidx.PrefixedID

// GroupCollection defines model for GroupCollection.
type GroupCollection struct {
	Groups []Group `json:"groups"`

	// Pagination collection response pagination
	Pagination Pagination `json:"pagination"`
}

// GroupIDCollection defines model for GroupIDCollection.
type GroupIDCollection struct {
	GroupIDs []gidx.PrefixedID `json:"group_ids"`

	// Pagination collection response pagination
	Pagination Pagination `json:"pagination"`
}

// GroupMemberCollection defines model for GroupMemberCollection.
type GroupMemberCollection struct {
	GroupID   gidx.PrefixedID   `json:"group_id"`
	MemberIDs []gidx.PrefixedID `json:"member_ids"`

	// Pagination collection response pagination
	Pagination Pagination `json:"pagination"`
}

// IssuerCollection defines model for IssuerCollection.
type IssuerCollection struct {
	Issuers []Issuer `json:"issuers"`

	// Pagination collection response pagination
	Pagination Pagination `json:"pagination"`
}

// OAuthClientCollection defines model for OAuthClientCollection.
type OAuthClientCollection struct {
	Clients []OAuthClient `json:"clients"`

	// Pagination collection response pagination
	Pagination Pagination `json:"pagination"`
}

// UserCollection defines model for UserCollection.
type UserCollection struct {
	// Pagination collection response pagination
	Pagination Pagination `json:"pagination"`
	Users      []User     `json:"users"`
}

// ListGroupMembersParams defines parameters for ListGroupMembers.
type ListGroupMembersParams struct {
	// Cursor the cursor to the results to return
	Cursor *PageCursor `form:"cursor,omitempty" json:"cursor,omitempty" query:"cursor"`

	// Limit limits the response collections
	Limit *PageLimit `form:"limit,omitempty" json:"limit,omitempty" query:"limit"`
}

// GetIssuerUsersParams defines parameters for GetIssuerUsers.
type GetIssuerUsersParams struct {
	// Cursor the cursor to the results to return
	Cursor *PageCursor `form:"cursor,omitempty" json:"cursor,omitempty" query:"cursor"`

	// Limit limits the response collections
	Limit *PageLimit `form:"limit,omitempty" json:"limit,omitempty" query:"limit"`
}

// GetOwnerOAuthClientsParams defines parameters for GetOwnerOAuthClients.
type GetOwnerOAuthClientsParams struct {
	// Cursor the cursor to the results to return
	Cursor *PageCursor `form:"cursor,omitempty" json:"cursor,omitempty" query:"cursor"`

	// Limit limits the response collections
	Limit *PageLimit `form:"limit,omitempty" json:"limit,omitempty" query:"limit"`
}

// ListGroupsParams defines parameters for ListGroups.
type ListGroupsParams struct {
	// Cursor the cursor to the results to return
	Cursor *PageCursor `form:"cursor,omitempty" json:"cursor,omitempty" query:"cursor"`

	// Limit limits the response collections
	Limit *PageLimit `form:"limit,omitempty" json:"limit,omitempty" query:"limit"`
}

// ListOwnerIssuersParams defines parameters for ListOwnerIssuers.
type ListOwnerIssuersParams struct {
	// Cursor the cursor to the results to return
	Cursor *PageCursor `form:"cursor,omitempty" json:"cursor,omitempty" query:"cursor"`

	// Limit limits the response collections
	Limit *PageLimit `form:"limit,omitempty" json:"limit,omitempty" query:"limit"`
}

// ListUserGroupsParams defines parameters for ListUserGroups.
type ListUserGroupsParams struct {
	// Cursor the cursor to the results to return
	Cursor *PageCursor `form:"cursor,omitempty" json:"cursor,omitempty" query:"cursor"`

	// Limit limits the response collections
	Limit *PageLimit `form:"limit,omitempty" json:"limit,omitempty" query:"limit"`
}

// UpdateGroupJSONRequestBody defines body for UpdateGroup for application/json ContentType.
type UpdateGroupJSONRequestBody = UpdateGroup

// AddGroupMembersJSONRequestBody defines body for AddGroupMembers for application/json ContentType.
type AddGroupMembersJSONRequestBody = AddGroupMembers

// ReplaceGroupMembersJSONRequestBody defines body for ReplaceGroupMembers for application/json ContentType.
type ReplaceGroupMembersJSONRequestBody = AddGroupMembers

// UpdateIssuerJSONRequestBody defines body for UpdateIssuer for application/json ContentType.
type UpdateIssuerJSONRequestBody = IssuerUpdate

// CreateOAuthClientJSONRequestBody defines body for CreateOAuthClient for application/json ContentType.
type CreateOAuthClientJSONRequestBody = CreateOAuthClient

// CreateGroupJSONRequestBody defines body for CreateGroup for application/json ContentType.
type CreateGroupJSONRequestBody = CreateGroup

// CreateIssuerJSONRequestBody defines body for CreateIssuer for application/json ContentType.
type CreateIssuerJSONRequestBody = CreateIssuer

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
