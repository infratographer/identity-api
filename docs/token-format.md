# Token Format

Clients can expect to exchange OAuth 2.0 access tokens for OAuth 2.0 access tokens issued by identity-manager-sts. This document outlines the parameters for token requests, how they affect the response and the meaning of the claims contained in the STS token.

We anticipate that these tokens will be used to inform further AuthZ decisions later in the API Gateway stack. While we could instead exchange for ID Tokens issued by the STS, the access token gives us the ability to use standard claims to provide information necessary for AuthZ.

### Token Request

This is as defined in [RFC 8693]. The minimal requests includes:
- `grant_type`
- `subject_token`
- `subject_token_type`

STS tokens can be restricted further through the use of `audience` and `scope`. The former restricts the class of resources for which this token may be used. The latter restricts the type of actions that that token can perform on resources.

As part of that RFC the requester may specify the `audience` and/or `resource`. For the first iteration of STS, we are supporting just `audience` which will allow us to expose a set of services that a user may use to restrict the context in which the token is valid. As we start to use this more, logical names may become cumbersome and being able to namespace with the URI format given by the `resource` parameter can be implemented. The `resource` parameter affects the `aud` claim on the STS token.


### Token Response

The token returned will be an OAuth 2.0 access token JWT  [RFC 9068] with the following claims:
| claim              | description                                                                                         |
|--------------------|-----------------------------------------------------------------------------------------------------|
| iss                | FQDN for identity-gate-sts issuer                                                                   |
| iat                | denotes the time the token was issued                                                               |
| jti                | unique ID for the token issued
| exp | date of expiry                                                                  |
| sub                | subject of the token in the infratographer namespace<br>example "sub": `infratographer:user:{uuid}` |
| aud                | Resource classes on which that token may operate                                                    |
| scope              | Actions which the token is restricted to within the audiences                                       |
| client_id | id of the client requesting the token |
| act:sub            | original JWT subject                                             |
| infratographer.sub | this is a private claim which indicates the subject be used in policy enforcement                   |


The following are in the spec, but for now aren't supported until we run into/identify the use-cases for them:

- `may_act` claims in the subject_token are used together with   `actor_token` to determine if the actor is authorized to proceed as the authority for the original subject.


### Claim Mapping

As an organization, I may have my own set of way of assigning permissions to users, and I want that to map the same way when operating with an exchanged token. In order to do so, a user may provide custom claim mappings.

Users may provide mappings on `infratographer.sub` which build a subject based on claims in the subject token. This claim in the STS token is expected to be used in-place of the JWT "sub" claim for policy enforcement.


[RFC 8693]:  https://www.rfc-editor.org/rfc/rfc8693.html#name-token-exchange-request-and-
[RFC 9068]: https://www.rfc-editor.org/rfc/rfc9068.html#name-data-structure
