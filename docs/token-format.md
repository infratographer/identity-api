# Token Format

Clients can expect to exchange JWTs from one security domain for OAuth 2.0 access tokens issued by identity-manager-sts. This document outlines the parameters for token requests, how they affect the response and the meaning of the claims contained in the issued access token.

We anticipate that these tokens will be used to inform further authorization decisions later in the API gateway stack. While RFC 8693 allows for issuing ID tokens, access tokens give us the ability to use standard claims to provide information necessary for making authorization decisions.

### Token Request

This is as defined in [RFC 8693][rfc-8693]. At minimum, a token request includes the following parameters:

- `grant_type` (always set to `urn:ietf:params:oauth:grant-type:token-exchange`)
- `subject_token`
- `subject_token_type`

Issued access tokens can be restricted further through the use of `audience`, `resource`, and `scope` parameters in the token request. `audience` and `resource` restrict the class of resources for which the issued token may be used. `scope` restricts the type of actions the token can perform on resources.

For the first iteration of identity-manager-sts, we are only supporting the use of `resource`, which will allow us to expose a set of services that a user may use to restrict the context in which the token is valid. The `resource` parameter maps to the `aud` claim on the issued token, as outlined in [RFC 9068][rfc-9068].


### Token Response

The token returned is an [RFC 9068][rfc-9068]-compliant access token JWT with the following claims:

| claim              | description                                                                                      |
|--------------------|--------------------------------------------------------------------------------------------------|
| iss                | FQDN for identity-manager-sts issuer                                                             |
| iat                | The time the token was issued                                                                    |
| jti                | Unique ID for the token issued                                                                   |
| exp                | Token expiration time                                                                            |
| sub                | URN of the user in the infratographer namespace, e.g. `urn:infratographer:user/{uuid}`           |
| aud                | Resources on which the token may operate                                                         |
| client_id          | ID of the client requesting the token, or `null` if no client was used when requesting the token |
| infratographer.sub | Private claim which indicates the subject be used in policy enforcement                          |

The following are defined in [RFC 8693][rfc-8693] and [RFC 9068][rfc-9068], but for now aren't supported until we run into/identify the use cases for them:

- `act`: Describes the acting party to whom access has been delegated. identity-manager-sts currently only supports impersonation semantics, not delegation.
- `may_act`: Describes the set of claims that identify an actor that can act on behalf of the subject identified by the subject token.
- `groups`: Describes groups that the subject is a member of in the context of the issued access token.
- `roles`: Describes roles assigned to the subject in the context of the issued access token.
- `entitlements`: Describes individual resources the subject can access in the context of the issued access token.

### Claim Mapping

In many scenarios, organizations may have their own set of ways of assigning permissions to users, and may want that to map the same way when operating with an exchanged token. In order to accomplish this, a user may provide custom claim mappings when defining an issuer in identity-manager-sts.

Users may provide a mapping for the `infratographer.sub` claim which builds a subject based on claims in the subject token. This claim in the issued access token is expected to be used in place of the JWT "sub" claim for policy enforcement.

[rfc-8693]: https://www.rfc-editor.org/rfc/rfc8693.html
[rfc-9068]: https://www.rfc-editor.org/rfc/rfc9068.html
