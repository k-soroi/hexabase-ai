# Authentication Sequence Diagrams

## 1. OAuth Login/Registration Flow

This flow handles both new user registration and existing user login through OAuth providers.

```mermaid
sequenceDiagram
    participant U as User/Browser
    participant C as Client App
    participant API as Hexabase API
    participant AS as Auth Service
    participant DB as Database
    participant OP as OAuth Provider (Google/GitHub)

    U->>C: Click "Login with [Provider]"
    C->>API: POST /auth/login/:provider
    API->>AS: GetAuthURL(provider)
    AS->>AS: Generate state & PKCE verifier
    AS->>DB: Store auth state
    AS-->>API: Return auth URL
    API-->>C: 200 OK {authURL}
    C->>U: Redirect to OAuth provider
    
    U->>OP: Login & Authorize
    OP->>U: Redirect to callback URL with code
    U->>C: Callback with code & state
    C->>API: GET/POST /auth/callback/:provider?code=xxx&state=xxx
    
    API->>AS: HandleCallback(provider, code, state)
    AS->>DB: Validate state
    AS->>AS: Verify PKCE (if used)
    AS->>OP: Exchange code for tokens
    OP-->>AS: OAuth tokens
    
    AS->>OP: Get user info
    OP-->>AS: User profile (email, name, etc)
    
    AS->>DB: Check if user exists
    alt New User
        AS->>DB: Create new user record
        AS->>DB: Log security event (first login)
    else Existing User
        AS->>DB: Update user info
        AS->>DB: Log security event (login)
    end
    
    AS->>AS: Generate JWT access token
    AS->>AS: Generate refresh token (selector/validator)
    AS->>DB: Store session with refresh token
    AS-->>API: TokenPair (access + refresh)
    
    API-->>C: 200 OK {access_token, refresh_token}
    C->>C: Store tokens
    C->>API: GET /auth/me (with access token)
    API->>AS: ValidateToken(access_token)
    AS->>DB: Get user info
    AS-->>API: User data
    API-->>C: 200 OK {user}
    C->>U: Show authenticated UI
```

## 2. Token Refresh Flow

This flow is used when the access token expires and needs to be refreshed.

```mermaid
sequenceDiagram
    participant C as Client App
    participant API as Hexabase API
    participant AS as Auth Service
    participant DB as Database

    C->>API: POST /auth/refresh
    Note over C,API: Body: {refresh_token: "selector.validator"}
    
    API->>AS: RefreshAccessToken(refreshToken)
    AS->>AS: Parse selector from token
    AS->>DB: Find session by selector
    
    alt Session not found
        AS-->>API: Error: Invalid token
        API-->>C: 401 Unauthorized
    else Session found
        AS->>AS: Validate token (hash validator)
        
        alt Invalid token
            AS->>DB: Delete session (security)
            AS-->>API: Error: Invalid token
            API-->>C: 401 Unauthorized
        else Valid token
            AS->>AS: Check session expiry
            
            alt Session expired
                AS->>DB: Delete expired session
                AS-->>API: Error: Session expired
                API-->>C: 401 Unauthorized
            else Session valid
                AS->>AS: Generate new JWT access token
                AS->>AS: Generate new refresh token
                AS->>DB: Update session with new refresh token
                AS->>DB: Log security event (token refresh)
                AS-->>API: New TokenPair
                API-->>C: 200 OK {access_token, refresh_token}
                C->>C: Update stored tokens
            end
        end
    end
```

## 3. Logout Flow

```mermaid
sequenceDiagram
    participant U as User
    participant C as Client App
    participant API as Hexabase API
    participant AS as Auth Service
    participant DB as Database

    U->>C: Click "Logout"
    C->>API: POST /auth/logout
    Note over C,API: Headers: Authorization: Bearer <access_token>
    
    API->>AS: ValidateToken(access_token)
    AS->>AS: Extract user ID from token
    AS->>DB: Find current session
    AS->>DB: Delete session
    AS->>DB: Log security event (logout)
    AS-->>API: Success
    API-->>C: 200 OK
    C->>C: Clear stored tokens
    C->>U: Redirect to login page
```

## 4. API Request with Authentication

```mermaid
sequenceDiagram
    participant C as Client App
    participant API as Hexabase API
    participant MW as Auth Middleware
    participant AS as Auth Service
    participant H as API Handler

    C->>API: GET /api/protected-resource
    Note over C,API: Headers: Authorization: Bearer <access_token>
    
    API->>MW: Process request
    MW->>AS: ValidateToken(access_token)
    
    alt Invalid/Expired Token
        AS-->>MW: Error: Invalid token
        MW-->>API: Set 401 status
        API-->>C: 401 Unauthorized
    else Valid Token
        AS-->>MW: Valid claims
        MW->>MW: Set user context
        MW->>H: Forward to handler
        H->>H: Process request
        H-->>API: Response data
        API-->>C: 200 OK {data}
    end
```

## Key Security Features

1. **PKCE (Proof Key for Code Exchange)**: Prevents authorization code interception attacks
2. **State Parameter**: Prevents CSRF attacks in OAuth flow
3. **Selector/Validator Pattern**: Prevents timing attacks on refresh tokens
4. **Token Rotation**: New refresh token issued on each refresh
5. **Session Management**: Track all active sessions per user
6. **Security Event Logging**: Audit trail of all auth events

## Notes

- No traditional username/password signup - all users authenticate via OAuth providers
- New users are automatically created on first successful OAuth login
- User profile information is synced from OAuth provider
- Access tokens are short-lived JWTs (configurable, typically 15-30 minutes)
- Refresh tokens are long-lived (configurable, typically 7-30 days)
- All tokens are invalidated on logout