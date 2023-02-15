# Users

A user as defined in identity-manager-sts is any unique subject as identified by an issuer. Users are bound to the issuer that provides identifying JWTs for them (i.e., `foo@example.com` from one OIDC provider is not the same user as `foo@example.com` from another OIDC provider).

## Tenant scoping

Currently, identity-manager-sts does not support any particular mechanism for binding an issuer to a given tenant. This work is being tracked as part of Issue [#23][issue-23].

At minimum, it is expected that identity-manager-sts should ultimately support constraining tokens to tenants using the OAuth 2.0 `audience` parameter.

[issue-23]: https://github.com/infratographer/identity-manager-sts/issues/23

## Provisioning

identity-manager-sts assumes that each issuer provides an OIDC [UserInfo][userinfo] endpoint. During token exchange, the user is looked up in the service's storage backend using the subject token's `iss` and `sub` claim values. If no user is found, identity-manager-sts creates and stores a new user with profile information populated from the issuer's UserInfo endpoint.

### Example

Suppose a person has a user account in an OIDC provider registered as an issuer in identity-manager-sts using the email address `foo@example.com` as their user ID. Assuming their OIDC identity is issued by `https://iam.example.com/`, during token exchange the user will present identity-manager-sts with a token with the following claims:

```
{
  "iss": "https://iam.example.com/",
  "sub": "foo@example.com"
}
```

If this is the first time the user has performed token exchange, no user record will exist for the (`iss`, `sub`) pair (`https://iam.example.com/`, `foo@example.com`). identity-manager-sts will thus look up the user using the provided subject token at `https://iam.example.com/userinfo` and get the following result:

```
{
  "sub": "foo@example.com",
  "name": "Foo Bar",
  "given_name": "Foo",
  "family_name": "Bar",
  "email": "foo@example.com"
}
```

A new user will be created with a globally unique ID using the profile information from the issuer's UserInfo endpoint, and the resulting access token will have the following claims:

```
{
  "iss": "https://iam.infratographer.com/",
  "sub": "urn:infratographer:user/898d5a69-2163-412b-90ea-5795f17f099e",
  "aud": ["https://iam.infratographer.com/userinfo"],
  "exp": 1676319675,
  "iat": 1676319575,
  "jti": "fcdee0ab-e1a7-4d27-94d2-ee4510c8a0cc"
}
```

Subsequent token exchanges with the same issuer and subject will use the stored user information and result in an access token with the same subject `urn:infratographer:user/898d5a69-2163-412b-90ea-5795f17f099e` in the `sub` claim.

[userinfo]: https://openid.net/specs/openid-connect-core-1_0.html#UserInfo

## Similarities with SCIM

As defined, the user model in identity-manager-sts bears some resemblance to the SCIM User resource defined in [RFC 7643][rfc-7643], particularly with respect to the storage of profile information as well as internal and external identifiers. [SCIM protocol][rfc-7644] support is not currently planned as part of identity-manager-sts, but could be offered in future iterations of the service as it provides a mechanism for issuers to send user lifecycle updates back to identity-manager-sts, including profile changes for and deactivation of existing users.

[rfc-7643]: https://www.rfc-editor.org/rfc/rfc7643#section-4.1
[rfc-7644]: https://www.rfc-editor.org/rfc/rfc7644
