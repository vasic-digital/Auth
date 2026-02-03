# Changelog

All notable changes to the `digital.vasic.auth` module are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2025-01-01

### Added

- **pkg/token**: `Token` interface with `AccessToken`, `RefreshToken`, `ExpiresAt`, `IsExpired`, and `NeedsRefresh` methods.
- **pkg/token**: `Claims` map type with typed accessors for standard JWT claims (`Subject`, `Issuer`, `Audience`, `ExpiresAt`, `IssuedAt`) and generic `Get`/`GetString`.
- **pkg/token**: `Store` interface with `Get`, `Set`, `Delete`, and `Revoke` operations.
- **pkg/token**: `InMemoryStore` thread-safe implementation with TTL support, `Cleanup`, and `Len` methods.
- **pkg/token**: `SimpleToken` basic `Token` implementation with constructor `NewSimpleToken`.
- **pkg/jwt**: `Config` struct with `SigningMethod`, `Secret`, `Expiration`, and `Issuer` fields.
- **pkg/jwt**: `DefaultConfig` factory function (HS256, 1-hour expiration).
- **pkg/jwt**: `Token` struct with `Claims`, `ExpiresAt`, `IssuedAt`, and `Raw` fields.
- **pkg/jwt**: `Manager` with `Create`, `Validate`, and `Refresh` methods.
- **pkg/jwt**: Signing method enforcement in `Validate` to prevent algorithm substitution attacks.
- **pkg/apikey**: `APIKey` struct with `ID`, `Key`, `Name`, `Scopes`, `ExpiresAt`, `CreatedAt` fields and `IsExpired`, `HasScope`, `HasAllScopes` methods.
- **pkg/apikey**: `GeneratorConfig` and `DefaultGeneratorConfig` (prefix `"ak-"`, 32-byte length).
- **pkg/apikey**: `Generator` with `Generate` method using `crypto/rand`.
- **pkg/apikey**: `KeyStore` interface with `Store`, `Get`, `GetByID`, `Delete`, `List` operations.
- **pkg/apikey**: `InMemoryStore` with dual-map (by key string and by UUID) O(1) lookup.
- **pkg/apikey**: `Validate` package-level function combining store lookup and expiration check.
- **pkg/apikey**: `MaskKey` utility for safe display of API keys.
- **pkg/oauth**: `Credentials` struct with `AccessToken`, `RefreshToken`, `ExpiresAt`, `Scopes`, `Metadata` and `IsExpired`/`NeedsRefresh` methods.
- **pkg/oauth**: `CredentialReader` interface and `FileCredentialReader` JSON file implementation.
- **pkg/oauth**: `TokenRefresher` interface and `HTTPTokenRefresher` implementation with configurable HTTP client, client ID, and extra parameters.
- **pkg/oauth**: `RefreshResponse` struct for standard OAuth2 token responses.
- **pkg/oauth**: `Config` and `DefaultConfig` for `AutoRefresher` (10-min threshold, 5-min cache, 30-sec rate limit).
- **pkg/oauth**: `AutoRefresher` with `GetCredentials`, `ClearCache`, and `ClearCacheFor` methods.
- **pkg/oauth**: `NeedsRefresh` and `IsExpired` package-level utility functions.
- **pkg/middleware**: `Middleware` type alias `func(http.Handler) http.Handler`.
- **pkg/middleware**: `TokenValidator` and `APIKeyValidator` interfaces.
- **pkg/middleware**: `BearerToken` middleware for `Authorization: Bearer` header validation.
- **pkg/middleware**: `APIKeyHeader` middleware for custom header API key validation.
- **pkg/middleware**: `RequireScopes` middleware for scope-based authorization.
- **pkg/middleware**: `Chain` function for composing multiple middleware.
- **pkg/middleware**: `ClaimsFromContext`, `ScopesFromContext`, `APIKeyFromContext` accessor functions.
- **pkg/middleware**: Context keys `ClaimsKey`, `ScopesKey`, `APIKeyKey` with unexported key type for collision prevention.
- Unit tests for all five packages with race detection support.
