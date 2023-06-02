# Token Format

Clients can expect to exchange JWTs from one security domain for OAuth 2.0 access tokens issued by identity-api. This document outlines the parameters for token requests, how they affect the response and the meaning of the claims contained in the issued access token.

These tokens are used to inform further authorization decisions later in the API gateway stack. While RFC 8693 allows for issuing ID tokens, Infratographer in general does not enforce particular conventions around identity management and thus ID tokens are not in the scope of what identity-api provides.

### Token Request

This is as defined in [RFC 8693][rfc-8693]. At minimum, a token request includes the following parameters:

- `grant_type` (always set to `urn:ietf:params:oauth:grant-type:token-exchange`)
- `subject_token`
- `subject_token_type` (currently only `urn:iet:params:oauth:token-type:jwt`)

### Token Response

The token returned is an [RFC 9068][rfc-9068]-compliant access token JWT with the following claims:

| claim     | description                                                                                      |
|-----------|--------------------------------------------------------------------------------------------------|
| iss       | FQDN for identity-manager-sts issuer                                                             |
| iat       | The time the token was issued                                                                    |
| jti       | Unique ID for the token issued                                                                   |
| exp       | Token expiration time                                                                            |
| sub       | ID of the user as defined in [Subject Identifier Generation](#subject-identifier-generation)     |
| aud       | Resources on which the token may operate                                                         |
| client_id | ID of the client requesting the token, or `null` if no client was used when requesting the token |

The following are defined in [RFC 8693][rfc-8693] and [RFC 9068][rfc-9068], but for now aren't supported until we run into/identify the use cases for them:

- `act`: Describes the acting party to whom access has been delegated. identity-manager-sts currently only supports impersonation semantics, not delegation.
- `may_act`: Describes the set of claims that identify an actor that can act on behalf of the subject identified by the subject token.
- `groups`: Describes groups that the subject is a member of in the context of the issued access token.
- `roles`: Describes roles assigned to the subject in the context of the issued access token.
- `entitlements`: Describes individual resources the subject can access in the context of the issued access token.

### Subject Identifier Generation

[RFC 7519][rfc-7519] requires that the `sub` value of a JWT be either globally unique or unique in the context of the issuer. Thus, exchanged tokens must have a `sub` value that is at minimum unique to identity-api itself. However, in many scenarios it can be useful to know what the value of the `sub` claim of an exchanged token will be before the exchange occurs. For example, automation accounts may be configured to access resources before those accounts are created. For this reason this document includes a simple deterministic algorithm for subject ID generation.

The `sub` value of an access token is the concatenation of:

* A selected 7-character prefix `prefix`
* A literal `-`
* The SHA256 digest of the concatenation of the `iss` and `sub` claims in the subject token, base64-encoded with URL safe alphabet as defined in [RFC 4648][rfc-4648]

In pseudocode, this looks something like:

```
sub = prefix + "-" + base64encode(sha256(iss + sub))
```

For example, consider a token exchange for a subject token with an `iss` claim of `https://example.com` and a `sub` claim of `foo@example.com`. Assuming a prefix of `idntusr`, the algorithm for subject ID generation looks like so:

```
sub = "idntusr" + "-" + base64encode(sha256("https://example.comfoo@example.com"))
```

The resulting `sub` value will be `idntusr-G9KRgCBGlE6lYkoLKCdK`.

[rfc-4648]: https://www.rfc-editor.org/rfc/rfc4648.html#section-5
[rfc-7519]: https://www.rfc-editor.org/rfc/rfc7519#section-4.1.2
[rfc-8693]: https://www.rfc-editor.org/rfc/rfc8693.html
[rfc-9068]: https://www.rfc-editor.org/rfc/rfc9068.html
