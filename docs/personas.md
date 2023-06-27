# Personas

The following personas make up the bulk of the target audience and use cases for identity-api.

## User

A user is a generic entity that is accessing protected resources behind an API gateway.

## Customer

A customer is a (likely human) user of that interacts with protected resources through a web browser and tooling like Terraform and CLIs.

## Service

A service is a program that accepts requests from users, authorizes them, and performs some action. It sits behind an API gateway.

## Application

An application is a program that is able to perform actions on resources within an owner. It may or may not also function as a service.

## Enterprise

An enterprise is an entity that is self-hosting identity-api along with other Infratographer services.
