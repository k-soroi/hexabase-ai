# ADR-002: OAuth2/OIDC Authentication and Security Architecture

**Date**: 2025-06-30  
**Status**: Implemented  
**Authors**: Security Team

## 1. Background

Hexabase AI required a robust authentication and authorization system that could:
- Support multiple identity providers (GitHub, Google, Microsoft, etc.)
- Enable Single Sign-On (SSO) across all platform services
- Provide secure API access for programmatic clients
- Support multi-factor authentication
- Scale to thousands of concurrent users
- Maintain audit trails for compliance

The platform handles sensitive customer workloads and needed enterprise-grade security.

## 2. Status

**Implemented** - OAuth2/OIDC authentication with PKCE, JWT fingerprinting, and comprehensive audit logging is fully deployed.

## 3. Other Options Considered

### Option A: Basic JWT with Local User Database
- Local user management with bcrypt passwords
- Simple JWT tokens for session management
- Custom RBAC implementation

### Option B: SAML 2.0 Integration
- Enterprise SAML SSO
- XML-based assertions
- Session management via SAML

### Option C: OAuth2/OIDC with Enhanced Security
- OAuth2 with PKCE flow
- JWT with fingerprinting
- Integration with external IdPs
- Redis-based session management

## 4. What Was Decided

We chose **Option C: OAuth2/OIDC with Enhanced Security** featuring:
- OAuth2 authorization code flow with PKCE
- JWT tokens with browser fingerprinting
- Support for multiple OIDC providers as a Relying Party (RP)
- Hexabase acts as OIDC client only, not as an OIDC provider (IdP)
- Redis session storage with 24-hour expiry
- Comprehensive audit logging
- Rate limiting and DDoS protection
- JWKS endpoint for JWT verification (no OIDC discovery endpoint)

## 5. Why Did You Choose It?

- **Industry Standard**: OAuth2/OIDC is widely supported and understood
- **Security**: PKCE prevents authorization code interception attacks
- **Flexibility**: Easy to add new identity providers
- **User Experience**: Seamless SSO with existing accounts
- **Auditability**: Comprehensive logging for compliance requirements

## 6. Why Didn't You Choose the Other Options?

### Why not Basic JWT?
- No SSO capability
- Password management overhead
- Less secure than delegated authentication
- No built-in MFA support

### Why not SAML 2.0?
- Complex XML processing
- Poor mobile/SPA support
- Heavier protocol overhead
- Less developer-friendly

## 7. What Has Not Been Decided

- Support for WebAuthn/FIDO2 passwordless authentication
- Integration with enterprise Active Directory
- Advanced threat detection mechanisms
- Biometric authentication support

## 8. Considerations

### Security Considerations
- Regular rotation of signing keys
- Monitoring for abnormal authentication patterns
- Protection against token replay attacks
- Secure storage of OAuth client secrets

### Performance Considerations
- Redis clustering for session storage scale
- JWT validation caching strategies
- Efficient JWT key management via JWKS endpoint

### Architecture Clarifications (2025-06-30)
- **OIDC Relying Party Role**: Hexabase acts solely as an OIDC Relying Party (client), integrating with external IdPs
- **No IdP Functionality**: The platform does not act as an OIDC Provider, thus:
  - No OIDC discovery endpoint (`/.well-known/openid-configuration`) is exposed
  - No authorization_endpoint, token_endpoint, or userinfo_endpoint implementations exist
  - This reduces complexity and attack surface
- **JWKS Endpoint Retained**: The JWKS endpoint remains available for verifying JWTs issued by Hexabase

### Compliance Considerations
- GDPR compliance for user data
- SOC2 audit trail requirements
- Right to deletion implementation
- Data residency requirements

### Implementation Details

```go
// JWT with fingerprinting implementation
type EnhancedClaims struct {
    jwt.StandardClaims
    Fingerprint string `json:"fingerprint"`
    UserID      string `json:"user_id"`
    OrgID       string `json:"org_id"`
}

// PKCE verification
func VerifyPKCE(codeVerifier, codeChallenge string) bool {
    hash := sha256.Sum256([]byte(codeVerifier))
    computed := base64.RawURLEncoding.EncodeToString(hash[:])
    return computed == codeChallenge
}
```

### Future Enhancements
- Zero-trust network architecture integration
- Risk-based authentication
- Continuous authentication mechanisms
- Decentralized identity support
