# 外部IdPとしてOIDCプロバイダーを使用できるかの調査結果

## 調査日時
2024年1月

## 調査概要
Hexabase AIプラットフォームにおいて、外部Identity Provider（IdP）としてOIDC（OpenID Connect）プロバイダーを使用できるか、`api/internal/`配下のコードを調査しました。

## 調査結果サマリー

**結論：現在の実装では部分的にOIDCプロバイダーをサポートできるが、完全なOIDC準拠の実装にはいくつかの改修が必要です。**

## 現在の実装状況

### 1. OAuth2実装の現状
- **実装済み**：OAuth2の認証フローが実装されており、GoogleとGitHubのプロバイダーがサポートされている
- **実装場所**：`api/internal/auth/oauth_client.go`および`api/internal/auth/repository/oauth.go`
- **認証フロー**：Authorization Code Flow with PKCE（RFC 7636）が実装済み

### 2. 現在サポートされているプロバイダー
```go
// api/internal/auth/oauth_client.go より
- Google OAuth2（google.Endpoint使用）
- GitHub OAuth2（github.Endpoint使用）
```

### 3. 設定構造
```go
// api/internal/shared/config/config.go より
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

### 4. ADR-002による設計決定
- HexabaseはOIDC Relying Party（RP/クライアント）としてのみ動作
- OIDC Provider（IdP）としては動作しない
- JWKS エンドポイント（`/.well-known/jwks.json`）は自身のJWT検証用に提供
- OIDC Discovery エンドポイント（`/.well-known/openid-configuration`）は意図的に削除済み

## なぜ現在の実装では完全なOIDC対応ができないのか

### 1. IDトークンの処理が未実装
現在の実装はOAuth2のアクセストークンのみを処理しており、OIDC特有のIDトークンの検証・処理が実装されていません。

```go
// 現在の実装（api/internal/auth/repository/oauth.go）
func (r *oauthRepository) ExchangeCode(ctx context.Context, provider, code string) (*domain.OAuthToken, error) {
    token, err := config.Exchange(ctx, code)
    // IDトークンの処理がない
    return &domain.OAuthToken{
        AccessToken:  token.AccessToken,
        RefreshToken: token.RefreshToken,
        // ...
    }, nil
}
```

### 2. UserInfo取得がプロバイダー固有実装
各プロバイダーごとにUserInfo取得ロジックがハードコードされており、OIDC標準のUserInfoエンドポイントを使用する汎用実装がありません。

```go
// 現在の実装
switch provider {
case "google":
    return r.getGoogleUserInfo(ctx, client)
case "github":
    return r.getGithubUserInfo(ctx, client)
default:
    return nil, fmt.Errorf("user info not implemented for provider %s", provider)
}
```

### 3. OIDC Discovery機能の欠如
OIDC Discovery（`/.well-known/openid-configuration`）を使用した動的なエンドポイント検出機能がありません。

## 外部OIDCプロバイダーを使用するための改修案

### 1. IDトークンの検証機能を追加

```go
// 新しい構造体を追加
type OIDCToken struct {
    IDToken      string // JWT形式のIDトークン
    AccessToken  string
    RefreshToken string
}

// IDトークンの検証メソッドを追加
func (s *service) ValidateIDToken(ctx context.Context, idToken string, provider string) (*IDTokenClaims, error) {
    // 1. プロバイダーのJWKSエンドポイントから公開鍵を取得
    // 2. IDトークンの署名を検証
    // 3. クレームを検証（iss, aud, exp等）
    // 4. ユーザー情報を抽出
}
```

### 2. 汎用的なOIDCプロバイダーサポートを追加

```go
// api/internal/auth/repository/oauth.go を修正
func (r *oauthRepository) GetUserInfo(ctx context.Context, provider string, token *domain.OAuthToken) (*domain.UserInfo, error) {
    config, ok := r.providers[provider]
    if !ok {
        return nil, fmt.Errorf("provider %s not configured", provider)
    }

    // プロバイダー固有の実装がある場合
    switch provider {
    case "google", "github":
        // 既存の実装を使用
    default:
        // 汎用的なOIDC UserInfoエンドポイントを使用
        if config.UserInfoURL != "" {
            return r.getOIDCUserInfo(ctx, client, config.UserInfoURL)
        }
    }
}

// 新しいメソッドを追加
func (r *oauthRepository) getOIDCUserInfo(ctx context.Context, client *http.Client, userInfoURL string) (*domain.UserInfo, error) {
    // OIDC標準のUserInfoエンドポイントからユーザー情報を取得
}
```

### 3. 設定ファイルでの新規OIDCプロバイダー追加サポート

```yaml
# config.yaml の例
auth:
  external_providers:
    # 既存のプロバイダー
    google:
      client_id: "your-google-client-id"
      client_secret: "your-google-client-secret"
      redirect_url: "https://api.hexabase.com/auth/callback/google"
      scopes: ["openid", "email", "profile"]
    
    # 新規OIDCプロバイダーの例（Keycloak）
    keycloak:
      client_id: "hexabase-client"
      client_secret: "your-client-secret"
      redirect_url: "https://api.hexabase.com/auth/callback/keycloak"
      scopes: ["openid", "email", "profile"]
      auth_url: "https://keycloak.example.com/auth/realms/master/protocol/openid-connect/auth"
      token_url: "https://keycloak.example.com/auth/realms/master/protocol/openid-connect/token"
      userinfo_url: "https://keycloak.example.com/auth/realms/master/protocol/openid-connect/userinfo"
      jwks_url: "https://keycloak.example.com/auth/realms/master/protocol/openid-connect/certs"
```

### 4. OIDC Discovery対応（オプション）

```go
// 新しいサービスメソッドを追加
func (s *service) DiscoverOIDCProvider(ctx context.Context, issuerURL string) (*OIDCProviderMetadata, error) {
    // /.well-known/openid-configuration から設定を自動取得
    discoveryURL := fmt.Sprintf("%s/.well-known/openid-configuration", strings.TrimSuffix(issuerURL, "/"))
    // ...
}
```

## 推奨される実装手順

1. **フェーズ1：最小限の実装**
   - IDトークンの検証機能を追加
   - 汎用的なUserInfoエンドポイント対応を追加
   - 設定ファイルで新規OIDCプロバイダーを追加できるようにする

2. **フェーズ2：完全なOIDC準拠**
   - OIDC Discovery対応
   - 動的クライアント登録のサポート（オプション）
   - セッション管理の強化

3. **フェーズ3：エンタープライズ機能**
   - 複数のOIDCプロバイダーの同時サポート
   - カスタムクレームマッピング
   - 条件付きアクセスポリシー

## セキュリティ考慮事項

1. **IDトークンの検証は必須**
   - 署名検証
   - 発行者（iss）の検証
   - 対象者（aud）の検証
   - 有効期限（exp）の検証

2. **HTTPS通信の強制**
   - すべてのOIDC関連通信はHTTPS経由で行う

3. **状態管理の強化**
   - CSRF攻撃を防ぐためのstate パラメータの適切な管理
   - PKCEの使用（既に実装済み）

## まとめ

現在の実装でも、設定を追加することで基本的なOAuth2プロバイダーとしての外部IdPは使用可能ですが、完全なOIDC準拠のためには上記の改修が必要です。特に、IDトークンの検証と汎用的なUserInfoエンドポイントのサポートは、OIDC対応において重要な要素です。

推奨される次のステップ：
1. まず最小限の実装（フェーズ1）を行い、基本的なOIDCプロバイダー（Keycloak、Auth0、Oktaなど）との連携を可能にする
2. その後、必要に応じて完全なOIDC準拠機能を追加する
3. エンタープライズ要件に応じて、高度な機能を実装する