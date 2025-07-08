# OIDC Provider Support Analysis for Hexabase AI

## Executive Summary

**判断結果**: **可能 (Possible)** - 外部IdPとしてOIDCプロバイダーを使用することは可能ですが、現在の実装には改善が必要です。

## Current State Analysis

### 1. Existing OAuth2/OIDC Infrastructure

The platform already has a solid OAuth2/OIDC foundation:

#### 1.1 Configuration Support
- **Location**: `api/internal/shared/config/config.go`
- **Structure**: 
  ```go
  type OAuthProvider struct {
      ClientID     string   `mapstructure:"client_id"`
      ClientSecret string   `mapstructure:"client_secret"`
      RedirectURL  string   `mapstructure:"redirect_url"`
      Scopes       []string `mapstructure:"scopes"`
      AuthURL      string   `mapstructure:"auth_url"`
      TokenURL     string   `mapstructure:"token_url"`
      UserInfoURL  string   `mapstructure:"userinfo_url"`
  }
  ```

#### 1.2 Current Provider Support
- **Google OAuth2**: Full support via `golang.org/x/oauth2/google`
- **GitHub OAuth2**: Full support via `golang.org/x/oauth2/github`
- **Generic OIDC**: Partial support through configuration

#### 1.3 Provider Architecture
- **Repository Pattern**: `api/internal/auth/repository/oauth.go`
- **Service Layer**: `api/internal/auth/service/service.go`
- **Domain Models**: `api/internal/auth/domain/models.go`

### 2. OIDC-Specific Features

#### 2.1 Current OIDC Implementation
- **OIDC Config**: Already exists in `api/internal/shared/observability/manager.go`
- **JWT Support**: Complete with RSA signing and JWKS endpoint
- **Workspace Integration**: OIDC configuration per workspace in vCluster

#### 2.2 Security Features
- **PKCE Support**: RFC 7636 implemented
- **State Validation**: Secure state management with Redis
- **Token Fingerprinting**: Enhanced JWT security
- **Session Management**: Comprehensive session handling

## Gap Analysis

### 1. Missing OIDC-Specific Features

#### 1.1 Discovery Endpoint Support
**Current**: Manual configuration required
**Needed**: Automatic discovery via `.well-known/openid-configuration`

#### 1.2 Standard OIDC Claims
**Current**: Limited to `id`, `email`, `name`, `picture`
**Needed**: Full OpenID Connect claims support:
- `sub` (subject identifier)
- `iss` (issuer)
- `aud` (audience)
- `exp`, `iat`, `nbf` (time claims)
- `groups` (group membership)

#### 1.3 ID Token Validation
**Current**: Only access token validation
**Needed**: Proper ID token validation with:
- Signature verification
- Claims validation
- Issuer verification

### 2. Provider Interface Enhancement

#### 2.1 Generic OIDC Provider
**Current**: Hardcoded Google/GitHub providers
**Needed**: Generic OIDC provider implementation

#### 2.2 Provider Factory Pattern
**Current**: Manual provider instantiation
**Needed**: Factory pattern for dynamic provider creation

## Recommendations

### 1. Immediate Improvements (High Priority)

#### 1.1 Add Generic OIDC Provider
Create a new provider in `api/internal/auth/repository/oidc_provider.go`:

```go
type OIDCProvider struct {
    config   *OIDCProviderConfig
    client   *http.Client
    logger   *slog.Logger
}

type OIDCProviderConfig struct {
    ProviderConfig
    IssuerURL       string `mapstructure:"issuer_url"`
    JWKSUri         string `mapstructure:"jwks_uri"`
    UserInfoEndpoint string `mapstructure:"userinfo_endpoint"`
}
```

#### 1.2 Implement Discovery Support
Add auto-discovery capability:

```go
func (p *OIDCProvider) DiscoverConfiguration(ctx context.Context) (*OIDCConfiguration, error) {
    discoveryURL := p.config.IssuerURL + "/.well-known/openid-configuration"
    // Implement discovery logic
}
```

#### 1.3 Enhanced Configuration
Update `config.go` to support OIDC-specific parameters:

```go
type OIDCProvider struct {
    OAuthProvider
    IssuerURL       string `mapstructure:"issuer_url"`
    DiscoveryURL    string `mapstructure:"discovery_url"`
    JWKSUri         string `mapstructure:"jwks_uri"`
    EnableDiscovery bool   `mapstructure:"enable_discovery"`
}
```

### 2. Medium-term Enhancements

#### 2.1 Provider Factory Implementation
Create a provider factory following existing patterns:

```go
type OIDCProviderFactory interface {
    CreateProvider(providerType string, config *OIDCProviderConfig) (Provider, error)
    ValidateProviderConfig(providerType string, config map[string]interface{}) error
}
```

#### 2.2 ID Token Validation
Implement proper ID token validation:

```go
func (p *OIDCProvider) ValidateIDToken(ctx context.Context, idToken string) (*Claims, error) {
    // Implement ID token validation
    // 1. Verify signature using JWKS
    // 2. Validate claims (iss, aud, exp, etc.)
    // 3. Extract user information
}
```

### 3. Long-term Architecture

#### 3.1 Multi-Provider Chain
Support multiple OIDC providers simultaneously:

```go
type ProviderChain struct {
    providers []Provider
    fallback  Provider
}
```

#### 3.2 Claims Mapping
Implement flexible claims mapping:

```go
type ClaimsMapper interface {
    MapClaims(providerClaims map[string]interface{}) (*domain.Claims, error)
}
```

## Implementation Steps

### Phase 1: Foundation (1-2 weeks)

1. **Update Configuration**
   - Add OIDC-specific fields to `OAuthProvider`
   - Add discovery endpoint support
   - Update validation logic

2. **Create Generic OIDC Provider**
   - Implement `OIDCProvider` struct
   - Add discovery capability
   - Implement ID token validation

3. **Update Provider Factory**
   - Modify `NewOAuthRepository` to support OIDC
   - Add provider type detection
   - Implement configuration validation

### Phase 2: Integration (1 week)

1. **Service Layer Updates**
   - Update `GetUserInfo` to handle OIDC claims
   - Add ID token processing
   - Update error handling

2. **Testing**
   - Add unit tests for OIDC provider
   - Add integration tests
   - Test with real OIDC providers

### Phase 3: Documentation (1 week)

1. **Configuration Guide**
   - Document OIDC provider setup
   - Provide examples for common providers
   - Update ADR-002 with OIDC specifics

2. **Migration Guide**
   - Document existing OAuth2 to OIDC migration
   - Provide configuration examples
   - Update deployment guides

## Example Configuration

### Generic OIDC Provider
```yaml
auth:
  external_providers:
    keycloak:
      client_id: "hexabase-ai"
      client_secret: "your-secret"
      issuer_url: "https://auth.company.com/realms/hexabase"
      scopes: ["openid", "profile", "email", "groups"]
      enable_discovery: true
      redirect_url: "https://api.hexabase.ai/auth/callback"
      
    okta:
      client_id: "your-okta-client-id"
      client_secret: "your-okta-secret"
      issuer_url: "https://company.okta.com/oauth2/default"
      scopes: ["openid", "profile", "email", "groups"]
      enable_discovery: true
      redirect_url: "https://api.hexabase.ai/auth/callback"
```

### Azure AD Example
```yaml
auth:
  external_providers:
    azure:
      client_id: "your-azure-client-id"
      client_secret: "your-azure-secret"
      issuer_url: "https://login.microsoftonline.com/{tenant-id}/v2.0"
      scopes: ["openid", "profile", "email", "https://graph.microsoft.com/User.Read"]
      enable_discovery: true
      redirect_url: "https://api.hexabase.ai/auth/callback"
```

## Security Considerations

### 1. Token Validation
- Implement proper ID token signature verification
- Validate token expiration and issuer claims
- Use JWKS for public key rotation

### 2. Claims Validation
- Validate audience (`aud`) claim
- Verify issuer (`iss`) claim
- Check token expiration (`exp`)

### 3. Session Management
- Maintain existing session security
- Implement proper token refresh
- Handle provider-specific logout

## Conclusion

**判断**: 外部IdPとしてOIDCプロバイダーを使用することは**可能**です。

**理由**:
1. 現在のOAuth2アーキテクチャがOIDCの基盤として機能
2. 設定とプロバイダーの抽象化が既に実装済み
3. JWT処理とセキュリティ機能が充実している
4. 既存のプロバイダーパターンを拡張可能

**実装方法**:
1. 汎用OIDCプロバイダーの実装
2. Discovery endpointのサポート追加
3. ID tokenの検証ロジック実装
4. 設定の拡張とプロバイダーファクトリーの更新

**推定工数**: 2-3週間で基本的なOIDC対応が可能、完全な実装には4-6週間が必要

現在のアーキテクチャの品質が高く、適切な抽象化がされているため、OIDC対応は比較的容易に実装できます。