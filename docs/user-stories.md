# User stories

The following user stories comprise the bulk of functionality identity-api aims to provide, based on the described [personas][personas].

[personas]: ./personas.md

## As a customer, I want to use my existing OIDC provider's access tokens to access protected resources

Customers may access resources using any OIDC provider that identity-api supports and which issues appropriately formatted JWTs. Upstream services should support consuming these tokens directly and exchanging them in identity-api, as well as consuming and validating tokens issued by identity-api.

## As a service, I want to trust exactly one JWT issuer for authenticating requests

Requiring that downstream services trust multiple JWT issuers introduces a greater attack surface area and increases the complexity of configuring a service. For simplicity, we should prefer trusting exactly one issuer for authenticating all subjects.

## As an application, I want a persistent token I can use to authenticate as myself

Many use cases, particularly those focused on applications, assume the existence of some kind of persistent token for authentication. We need to be able to mint these, validate them, and pass a JWT derived from them to backend services.

## As a user, I want to be able to restrict tokens to individual owners

In general, it is desirable to be able to limit a token to be usable only in the context of a single owner, so that said tokens have a lower blast radius if leaked.

## As a user, I want to manage persistent tokens that belong to applications I own

Users should be able to revoke, rotate, and issue tokens for applications when they have sufficient permissions to do so.

## As an enterprise, I want to define the mappings from an OIDC provider to a known token format

OIDC providers, assuming they issue access tokens as JWTs, support a number of different claims and formats for those claim values. Enterprises that manage an identity-api instance should be able to define mappings from these claims to other values when constructing access tokens during token exchange.
