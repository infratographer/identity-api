package types

const (
	// IdentityService represents the service portion of the prefix.
	IdentityService = "idnt"

	// IdentityUserResource represents the resource portion of the prefix.
	IdentityUserResource = "usr"

	// IdentityUserIDPrefix represents the full identity id prefix for a user resource.
	IdentityUserIDPrefix = IdentityService + IdentityUserResource

	// IdentityClientResource represents the client resource type in a ID.
	IdentityClientResource = "cli"

	// IdentityClientIDPrefix represents the full identity id prefix for a client resource.
	IdentityClientIDPrefix = IdentityService + IdentityClientResource

	// IdentityIssuerResource represents the issuer resource type in an ID.
	IdentityIssuerResource = "iss"

	// IdentityIssuerIDPrefix represents the full identity id prefix for an issuer resource.
	IdentityIssuerIDPrefix = IdentityService + IdentityIssuerResource

	// IdentityGroupResource represents the group resource type in an ID.
	IdentityGroupResource = "grp"

	// IdentityGroupIDPrefix represents the full identity id prefix for a group resource.
	IdentityGroupIDPrefix = IdentityService + IdentityGroupResource
)
