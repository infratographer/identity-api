// Package v1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.12.5-0.20230118012357-f4cf8f9a5703 DO NOT EDIT.
package v1

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	openapi_types "github.com/deepmap/oapi-codegen/pkg/types"
	"github.com/getkin/kin-openapi/openapi3"
)

// CreateIssuer defines model for CreateIssuer.
type CreateIssuer struct {
	// ClaimMappings CEL expressions mapping token claims to other claims
	ClaimMappings *map[string]string `json:"claim_mappings,omitempty"`

	// JwksUri JWKS URI
	JWKSURI string `json:"jwks_uri"`

	// Name A human-readable name for the issuer
	Name string `json:"name"`

	// Uri URI for the issuer. Must match the "iss" claim value in incoming JWTs
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

// Error defines model for Error.
type Error struct {
	// Errors List of error messages
	Errors []string `json:"errors"`
}

// Issuer defines model for Issuer.
type Issuer struct {
	// ClaimMappings CEL expressions mapping token claims to other claims
	ClaimMappings map[string]string `json:"claim_mappings"`

	// Id ID of the issuer
	ID openapi_types.UUID `json:"id"`

	// JwksUri JWKS URI
	JWKSURI string `json:"jwks_uri"`

	// Name A human-readable name for the issuer
	Name string `json:"name"`

	// Uri URI for the issuer. Must match the "iss" claim value in incoming JWTs
	URI string `json:"uri"`
}

// IssuerUpdate defines model for IssuerUpdate.
type IssuerUpdate struct {
	// ClaimMappings CEL expressions mapping token claims to other claims
	ClaimMappings *map[string]string `json:"claim_mappings,omitempty"`

	// JwksUri JWKS URI
	JWKSURI *string `json:"jwks_uri,omitempty"`

	// Name A human-readable name for the issuer
	Name *string `json:"name,omitempty"`

	// Uri URI for the issuer. Must match the "iss" claim value in incoming JWTs
	URI *string `json:"uri,omitempty"`
}

// OAuthClient defines model for OAuthClient.
type OAuthClient struct {
	// Audience Grantable audiences
	Audience []string `json:"audience"`

	// Id OAuth 2.0 Client ID
	ID openapi_types.UUID `json:"id"`

	// Name Description of Client
	Name string `json:"name"`

	// Secret OAuth2.0 Client Secret
	Secret *string `json:"secret,omitempty"`
}

// NotFound defines model for NotFound.
type NotFound = Error

// UpdateIssuerJSONRequestBody defines body for UpdateIssuer for application/json ContentType.
type UpdateIssuerJSONRequestBody = IssuerUpdate

// CreateOAuthClientJSONRequestBody defines body for CreateOAuthClient for application/json ContentType.
type CreateOAuthClientJSONRequestBody = CreateOAuthClient

// CreateIssuerJSONRequestBody defines body for CreateIssuer for application/json ContentType.
type CreateIssuerJSONRequestBody = CreateIssuer

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/+xY32/bNhD+VwhuDxug2O6PJ7+lcReoa7ciTtCHNCgY6mwzlUiNRzo1Av3vw5GyLUVK",
	"XBvd0Gx9oyjq7vjd992dfcelKUqjQTvk4ztuAUujEcLDH8b9ZrzOaC2NdqAdLUVZ5koKp4we3qDRtIdy",
	"AYWg1c8WZnzMfxpuDQ/jWxy+ttZYXlVVwjNAaVVJRviYnwEabyWwW4FMG8dmwS8drL8l0ycWhIMU0YOl",
	"59KaEqxTMVqZC1V8KkRZKj0POyLLFDkQ+fvWSbcqgY85Oqv0nHeCOXn9lsGX0gKiMhpZbZI58xk0C26Q",
	"OcOMW4Ctn3mytmqub0A6snpz+xk/eavIZdvDmw+/T9nFWbr9qo4l4V+O5uZIiwLqY3SqSnjcuW/nmC18",
	"IfSRBZGJ6xwYHWMzY5lbAFMRqKR7396gLs7Se58O2DuPjhXCyUXY/sgV4kce78yWIvfAlGZKS1MQQm8+",
	"nOOOO4X7VAm38JdXFjI+voyXi1E1ULuqkjrjfx57tzjJVc2/dtqFzxRo2YdO/QaZWwjH3EIhk8EKk0Iz",
	"igDQ8YQrB0U/MeoNYa1YHZqG6LKbhj4Q6M4TyMHBWS3E7oXRSwmIPWHkt2KFzFkPg627a2NyELrjb22G",
	"XEZddjwBbfc4eqvQMTNj4T0rAFHMAfcA8l4otZ+rHg09GbGrrItTOiGUWlKcGVsIx8fce5XtkEo6+VFF",
	"9qgiAc/+UpLc58vVhloXZSYc/OgmT5kHVcIPbBGnVmgX7ro+g3v1gz7Vh1DY88GIxXhYOjlI+P1Jmmyf",
	"qLicPNBcEo4gLbgHwmtEN43ndrWnpro2eF4F7JWema6fKUhvlVux80D0KdilksB+mZ5Pf2XvhBZzKMj/",
	"8fuUKWRChxVRpqCXRIDp+ZRJo2dq7m0YNzH0NeVyeNhB2zRP+BIsxpBGg9HgGWFjStCiVHzMXwxGgxc8",
	"4aVwi5DyoSjVcPlsGLs2Du/iIp1U8YrUm2lF/AoxpVnICu03SUgmrSjAAXXQy36SyAZBFG1TGGucx3zt",
	"mjdTQe09aczbjxOrqq6S9lD/fDT6ZvP8vVGlZ7Cfxilj5nPWOJZw9EUh7GoDXSBABGVLaUGF97Kp7Vi5",
	"55HX7RScgvvf4d+88NeCn/CXo5cPGd5EOtz89mtn6xQcMhI8XZpqkLg23m2Tty0sg4czWCUbmcXWgMM7",
	"lX2FwNJ1D3o0t3Hsipaps9Y2e1McUvYfEJfdKa4aj1vlYvedqyVolk6aeYr4Pi6yeObVKshirzzMQ5t5",
	"WkmoGXcQ+EErW+SvV4+gXdJU1MU7TqeH0d7HyfYfQzz8en5lstU3BrueyKv2BEIhVt9pomPEjVz3Z7lR",
	"9hxoEaaLuEgn1XrgCNOrwR7tdf8L2UGI82CbqFBas1Q0BAXlt3qf11mgVg9J1rF9j1TpgvEv8+WQ1tsi",
	"TbxBY+6Re7TNHv7UnXQXf/apJW5DIBm+XRcXpZ8qYZoqfxq1pUGTR2tLVf0dAAD//3k5DefLFwAA",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
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
	var res = make(map[string]func() ([]byte, error))
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
	var resolvePath = PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		var pathToFile = url.String()
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
