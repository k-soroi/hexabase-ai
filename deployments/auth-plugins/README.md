# Kubernetes OIDC Authentication Plugin

> Complete OIDC authentication system for Kubernetes exec credential plugin

## 🔐 Overview

This project implements a complete Kubernetes authentication plugin system with OIDC support, designed for secure and flexible authentication with Kubernetes clusters.

### Features

- ✅ **Full OIDC Authentication**: Complete OpenID Connect flow with JWT validation
- ✅ **Mock OIDC Server**: Development and testing OIDC server
- ✅ **Token Management**: Automatic token refresh and secure caching
- ✅ **ExecCredential API**: Full compliance with `client.authentication.k8s.io/v1`
- ✅ **RS256 JWT**: Secure JWT signature verification
- ✅ **JWKS Support**: Public key distribution via JSON Web Key Set

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   kubectl       │    │ Auth Plugin     │    │ OIDC Server     │
│                 │────│                 │────│                 │
│ kubectl get pods│    │ Token Request   │    │ JWT Generation  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         │              ┌─────────────────┐              │
         │              │ Token Cache     │              │
         │              │ (JSON file)     │              │
         │              └─────────────────┘              │
         │                                               │
         │              ┌─────────────────┐              │
         │              │ JWKS Endpoint   │              │
         └──────────────│ Public Key      │──────────────┘
                        └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- kubectl
- Access to a Kubernetes cluster

### 1. Build Components

```bash
# Build all components
make build-auth-plugins

# Or build individually
go build -o auth-plugin cmd/auth-plugin/main.go
go build -o mock-server cmd/mock-server/main.go
go build -o test-auth-plugin cmd/test-auth-plugin/main.go
```

### 2. Start Mock OIDC Server

```bash
# Start mock server
./mock-server

# Server will start on port 8080
# Endpoints available:
# - /.well-known/openid-configuration
# - /.well-known/jwks.json
# - /oauth/token
```

### 3. Configure kubectl

```bash
# Use example kubeconfig
export KUBECONFIG=configs/examples/k3s-test-kubeconfig.yaml

# Test authentication
kubectl get pods --all-namespaces
```

## 📂 Components

### 🔌 Auth Plugin (`cmd/auth-plugin/`)
Full-featured OIDC authentication plugin with:
- JWT token validation
- Automatic token refresh
- JWKS public key verification
- Secure token caching


### 🖥️ Mock OIDC Server (`cmd/mock-server/`)
Development OIDC server providing:
- OpenID Connect Discovery
- JWKS public key distribution
- OAuth2 token endpoint
- RS256 JWT signing

### 🧪 Test Auth Plugin (`cmd/test-auth-plugin/`)
Testing utility for:
- Basic authentication flow verification
- Token generation testing
- ExecCredential format validation

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBE_AUTH_ISSUER_URL` | OIDC issuer URL | `http://localhost:8080` |
| `KUBE_AUTH_TOKEN_URL` | Token endpoint URL | `http://localhost:8080/oauth/token` |
| `KUBE_AUTH_AUDIENCE` | JWT audience | `kubernetes-api` |
| `KUBE_AUTH_CACHE_PATH` | Token cache file path | `~/.kube/cache/auth-plugin/tokens.json` |

### Example kubeconfig

```yaml
users:
- name: oidc-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: "/path/to/auth-plugin"
      env:
      - name: "KUBE_AUTH_ISSUER_URL"
        value: "http://localhost:8080"
      - name: "KUBE_AUTH_TOKEN_URL"
        value: "http://localhost:8080/oauth/token"
      - name: "KUBE_AUTH_AUDIENCE"
        value: "kubernetes-api"
      interactiveMode: Never
```

## 🔧 Development

### Testing

```bash
# Run unit tests
go test ./...

# Test mock server endpoints
curl http://localhost:8080/.well-known/openid-configuration
curl http://localhost:8080/.well-known/jwks.json
curl -X POST http://localhost:8080/oauth/token -d "grant_type=client_credentials"

# Test auth plugin
./test-auth-plugin
```

### Building

```bash
# Build all binaries
make build-auth-plugins

# Clean build artifacts
make clean-auth-plugins
```

## 📋 API Reference

### ExecCredential Format

```json
{
  "apiVersion": "client.authentication.k8s.io/v1",
  "kind": "ExecCredential",
  "status": {
    "expirationTimestamp": "2024-01-01T12:00:00Z",
    "token": "eyJhbGciOiJSUzI1NiI..."
  }
}
```

### JWT Claims

```json
{
  "iss": "http://localhost:8080",
  "sub": "mock-user",
  "aud": ["kubernetes-api"],
  "exp": 1640995200,
  "iat": 1640991600,
  "kid": "key-id"
}
```

## 🔍 Troubleshooting

### Common Issues

1. **"Unauthorized" Error**
   - Ensure Kubernetes API server is configured for OIDC
   - Verify issuer URL matches server configuration
   - Check JWT token validity

2. **Token Refresh Failed**
   - Verify token endpoint accessibility
   - Check refresh token validity
   - Ensure proper grant_type in request

3. **JWKS Verification Failed**
   - Confirm JWKS endpoint is accessible
   - Verify key ID (kid) matches JWT header
   - Check public key format

### Debug Mode

```bash
# Enable verbose logging
export KUBE_AUTH_DEBUG=true

# Check plugin execution
kubectl get pods -v=8
```

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.

## 🔗 Related Documentation

- [Kubernetes Authentication](https://kubernetes.io/docs/reference/access-authn-authz/authentication/)
- [ExecCredential API](https://kubernetes.io/docs/reference/config-api/client-authentication.v1/)
- [OpenID Connect](https://openid.net/connect/)
- [JWT Specification](https://tools.ietf.org/html/rfc7519)