# Authentication

Hexabase KaaS uses OAuth2/OIDC for authentication with JWT tokens for API access.

## Overview

The authentication flow follows these steps:

1. User initiates login with an OAuth provider
2. User is redirected to the provider's login page
3. After successful authentication, user is redirected back with an authorization code
4. The code is exchanged for Hexabase KaaS JWT tokens
5. JWT tokens are used for all subsequent API requests

## Supported Providers

### Google

- **Provider ID**: `google`
- **Required Scopes**: `openid`, `email`, `profile`
- **Configuration**: OAuth 2.0 client credentials required

### GitHub

- **Provider ID**: `github`
- **Required Scopes**: `user:email`, `read:org`
- **Configuration**: OAuth App credentials required

### Microsoft Azure AD

- **Provider ID**: `azure`
- **Required Scopes**: `openid`, `email`, `profile`
- **Configuration**: App registration in Azure AD required

### Custom OIDC Provider

- **Provider ID**: Custom identifier
- **Required Claims**: `sub`, `email`, `name`
- **Configuration**: OIDC discovery endpoint required

## Authentication Flows

### Authorization Code Flow (Web Applications)

This is the recommended flow for web applications.

#### 1. Initiate Login

Redirect user to:
```
https://api.hexabase.ai/auth/login/google?redirect_uri=https://app.hexabase.ai/auth/callback
```

#### 2. Handle Callback

After authentication, the user is redirected to:
```
https://app.hexabase.ai/auth/callback?code=AUTHORIZATION_CODE&state=STATE
```

#### 3. Exchange Code for Tokens

```http
POST /auth/callback/google
Content-Type: application/json

{
  "code": "AUTHORIZATION_CODE",
  "redirect_uri": "https://app.hexabase.ai/auth/callback"
}
```

Response:
```json
{
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "user": {
      "id": "user-123",
      "email": "user@example.com",
      "name": "John Doe",
      "picture": "https://..."
    }
  }
}
```

### PKCE Flow (Single Page Applications)

For enhanced security in SPAs, use the PKCE (Proof Key for Code Exchange) flow.

#### 1. Generate Code Verifier and Challenge

```javascript
// Generate code verifier
const codeVerifier = generateRandomString(128);

// Generate code challenge
const codeChallenge = await sha256(codeVerifier);
const codeChallengeBase64 = base64UrlEncode(codeChallenge);
```

#### 2. Initiate Login with PKCE

```
https://api.hexabase.ai/auth/login/google?
  redirect_uri=https://app.hexabase.ai/auth/callback&
  code_challenge=CODE_CHALLENGE&
  code_challenge_method=S256
```

#### 3. Exchange Code with Verifier

```http
POST /auth/callback/google
Content-Type: application/json

{
  "code": "AUTHORIZATION_CODE",
  "redirect_uri": "https://app.hexabase.ai/auth/callback",
  "code_verifier": "CODE_VERIFIER"
}
```

### Direct Token Exchange (Mobile/Desktop Apps)

For native applications that can securely handle OAuth flows.

```http
POST /auth/login/google
Content-Type: application/json

{
  "id_token": "GOOGLE_ID_TOKEN"
}
```

## JWT Tokens

### Token Structure

Hexabase KaaS issues JWT tokens with the following structure:

#### Access Token Claims

```json
{
  "sub": "user-123",
  "email": "user@example.com",
  "name": "John Doe",
  "picture": "https://...",
  "iss": "https://api.hexabase.ai",
  "aud": "hexabase-ai",
  "exp": 1705753200,
  "iat": 1705749600,
  "jti": "token-unique-id",
  "organizations": [
    {
      "id": "org-123",
      "role": "admin"
    }
  ],
  "fingerprint": "device-fingerprint-hash"
}
```

#### Refresh Token Claims

```json
{
  "sub": "user-123",
  "type": "refresh",
  "iss": "https://api.hexabase.ai",
  "aud": "hexabase-ai",
  "exp": 1708341600,
  "iat": 1705749600,
  "jti": "refresh-token-id",
  "family": "token-family-id"
}
```

### Token Lifetimes

- **Access Token**: 1 hour (3600 seconds)
- **Refresh Token**: 30 days
- **Session Maximum**: 90 days

### Token Usage

Include the access token in the Authorization header:

```http
GET /api/v1/organizations
Authorization: Bearer eyJhbGciOiJSUzI1NiIs...
```

### Token Refresh

When the access token expires, use the refresh token to get a new one:

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

Response:
```json
{
  "data": {
    "access_token": "new-access-token",
    "refresh_token": "new-refresh-token",
    "expires_in": 3600
  }
}
```

## Security Features

### JWT Fingerprinting

To prevent token theft, Hexabase KaaS implements JWT fingerprinting:

1. A device fingerprint is generated during login
2. The fingerprint hash is embedded in the JWT
3. Each request validates the fingerprint matches
4. Tokens are bound to the originating device/browser

### Refresh Token Rotation

For enhanced security, refresh tokens are rotated on each use:

1. Each refresh token can only be used once
2. Using a refresh token invalidates it and issues a new one
3. Reuse of an old refresh token invalidates the entire token family
4. This prevents refresh token replay attacks

### Token Revocation

Tokens can be revoked in several ways:

#### Logout

```http
POST /auth/logout
Authorization: Bearer <access-token>
```

This revokes the current session and associated tokens.

#### Revoke All Sessions

```http
POST /auth/sessions/revoke-all
Authorization: Bearer <access-token>
```

This revokes all active sessions for the user.

#### Revoke Specific Session

```http
DELETE /auth/sessions/:sessionId
Authorization: Bearer <access-token>
```

### Rate Limiting

Authentication endpoints have specific rate limits:

- **Login attempts**: 5 per minute per IP
- **Token refresh**: 10 per minute per user
- **Failed attempts**: Exponential backoff after 3 failures

## Session Management

### List Active Sessions

```http
GET /auth/sessions
Authorization: Bearer <access-token>
```

Response:
```json
{
  "data": [
    {
      "id": "session-123",
      "device": "Chrome on Mac OS",
      "ip_address": "192.168.1.1",
      "location": "San Francisco, CA",
      "created_at": "2024-01-20T10:00:00Z",
      "last_active": "2024-01-20T15:30:00Z",
      "is_current": true
    }
  ]
}
```

### Security Events

View security-related events for your account:

```http
GET /auth/security-logs
Authorization: Bearer <access-token>
```

Response:
```json
{
  "data": [
    {
      "id": "evt-123",
      "type": "login_success",
      "timestamp": "2024-01-20T10:00:00Z",
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "location": "San Francisco, CA",
      "provider": "google"
    },
    {
      "id": "evt-124",
      "type": "login_failed",
      "timestamp": "2024-01-20T09:55:00Z",
      "ip_address": "192.168.1.1",
      "reason": "invalid_credentials"
    }
  ]
}
```

## Token Verification

### JWKS Endpoint

```http
GET /.well-known/jwks.json
```

Response:
```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "key-1",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

## Multi-Factor Authentication (MFA)

### Enable MFA

```http
POST /auth/mfa/enable
Authorization: Bearer <access-token>
```

Response includes QR code for authenticator app setup.

### Verify MFA

```http
POST /auth/mfa/verify
Content-Type: application/json
Authorization: Bearer <access-token>

{
  "code": "123456"
}
```

### Login with MFA

After initial authentication, if MFA is enabled:

```http
POST /auth/mfa/challenge
Content-Type: application/json

{
  "session_token": "mfa-session-token",
  "code": "123456"
}
```

## API Keys (Service Accounts)

For automated systems and CI/CD pipelines:

### Create API Key

```http
POST /api/v1/organizations/:orgId/api-keys
Content-Type: application/json
Authorization: Bearer <access-token>

{
  "name": "CI/CD Pipeline",
  "scopes": ["workspaces:read", "workspaces:write"],
  "expires_at": "2025-01-01T00:00:00Z"
}
```

Response:
```json
{
  "data": {
    "id": "key-123",
    "name": "CI/CD Pipeline",
    "key": "hxb_live_1234567890abcdef",
    "created_at": "2024-01-20T10:00:00Z",
    "expires_at": "2025-01-01T00:00:00Z"
  }
}
```

### Use API Key

```http
GET /api/v1/workspaces
Authorization: Bearer hxb_live_1234567890abcdef
```

## Best Practices

1. **Token Storage**
   - Store tokens securely (httpOnly cookies or secure storage)
   - Never store tokens in localStorage for production
   - Clear tokens on logout

2. **Token Refresh**
   - Implement automatic token refresh before expiration
   - Handle refresh failures gracefully
   - Don't expose refresh tokens to client-side JavaScript

3. **PKCE Usage**
   - Always use PKCE for public clients (SPAs, mobile apps)
   - Generate cryptographically secure code verifiers
   - Never reuse code verifiers

4. **Session Security**
   - Regularly review active sessions
   - Revoke sessions from unknown devices
   - Enable MFA for sensitive accounts

5. **API Key Management**
   - Rotate API keys regularly
   - Use minimal required scopes
   - Monitor API key usage

## Error Codes

| Code | Description |
|------|-------------|
| `invalid_request` | Request is missing required parameters |
| `invalid_client` | Client authentication failed |
| `invalid_grant` | Authorization code or refresh token is invalid |
| `unauthorized_client` | Client is not authorized for this grant type |
| `unsupported_grant_type` | Grant type is not supported |
| `invalid_scope` | Requested scope is invalid or exceeds granted scope |
| `token_expired` | Access token has expired |
| `token_revoked` | Token has been revoked |
| `mfa_required` | Multi-factor authentication is required |
| `rate_limit_exceeded` | Too many authentication attempts |