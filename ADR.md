# Go SDK Architecture Decision Record

This document records the technical decisions made for the Waffo Go SDK.

## Summary Table

| Decision Area | Choice | Rationale |
|---------------|--------|-----------|
| Runtime Version | Go 1.20+ | Generics maturity, improved crypto library |
| Dependency Strategy | Zero runtime dependencies | Security, minimal attack surface, Go philosophy |
| HTTP Client | net/http | Built-in, TLS 1.2+ support, connection pooling |
| JSON Framework | encoding/json | Built-in, struct tags support |
| Build Tool | go build / go mod | Native Go toolchain |
| Test Framework | testing | Built-in, table-driven tests |
| Type System | Static typing + interface | Native Go type system with generics |
| Code Style | gofmt + golangci-lint | Go community standard |
| Framework Adaptation | Gin, Echo, Fiber, Chi | Top 4 Go web frameworks |

## Detailed Decisions

### 1. Runtime Version

**Decision:** Go 1.20+

**Rationale:**
- Go 1.18 introduced generics, 1.20 has more mature implementation
- Improved `crypto` package for better RSA support
- Wide production adoption
- Go supports the two most recent major versions (good LTS strategy)

**Alternatives Considered:**
- Go 1.18+: Initial generics version, some features not fully polished
- Go 1.21+: Too restrictive, would exclude some production environments

**Version Compatibility Matrix:**

| Go Version | Support Status | Notes |
|------------|----------------|-------|
| 1.22.x | Fully Supported | Latest |
| 1.21.x | Fully Supported | Recommended |
| 1.20.x | Fully Supported | Minimum |
| < 1.20 | Not Supported | Generics not mature |

---

### 2. Dependency Strategy

**Decision:** Zero runtime dependencies

**Rationale:**
- **Security**: No supply chain attack surface from third-party packages
- **Maintenance**: No dependency updates to track
- **Bundle Size**: Minimal package size
- **Compatibility**: No version conflicts with user's dependencies
- **Trust**: Payment SDK should minimize external code
- **Go Philosophy**: Go standard library is very powerful, encourages zero deps

**Trade-offs:**
- Need to implement some functionality ourselves (but Go stdlib covers most needs)

**Dev Dependencies (test only):**
- None required (standard `testing` package is sufficient)
- Optional: `testify` for enhanced assertions

---

### 3. HTTP Client / Network Layer

**Decision:** net/http (standard library)

**Rationale:**
- Built into Go standard library, no external dependency
- Supports TLS 1.2+ (configurable minimum version)
- Connection pooling and Keep-Alive support
- Timeout control (ConnectTimeout, ReadTimeout)
- Custom Transport support

**Implementation Details:**
```go
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,
    },
    MaxIdleConns:        100,
    IdleConnTimeout:     90 * time.Second,
}
client := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,
}
```

**Alternatives Considered:**

| Library | Pros | Cons | Decision |
|---------|------|------|----------|
| net/http | Standard library, no deps | Sufficient features | Chosen |
| resty | Fluent API, convenient | External dependency | Rejected |
| fasthttp | High performance | Incompatible with net/http | Rejected |

---

### 4. JSON Framework

**Decision:** encoding/json (standard library)

**Rationale:**
- Built into Go standard library, no external dependency
- Struct tags for custom field names
- Supports omitempty, string, and other options
- Well-optimized performance
- Wide community usage

**Usage Example:**
```go
type CreateOrderParams struct {
    MerchantInfo MerchantInfo `json:"merchantInfo"`
    Amount       string       `json:"amount"`
    Currency     string       `json:"currency"`
}
```

**Alternatives Considered:**
- json-iterator: Higher performance, but adds dependency
- easyjson: Requires code generation
- sonic: Requires amd64 architecture

---

### 5. Build Tool

**Decision:** go build / go mod

**Rationale:**
- Native Go toolchain, no additional installation
- go mod for dependency management
- go build for compilation
- go test for testing
- go vet for static analysis

**Commands:**
```bash
# Initialize module
go mod init github.com/waffo-com/waffo-go

# Build
go build ./...

# Test
go test ./...

# Static analysis
go vet ./...
```

---

### 6. Test Framework

**Decision:** testing (standard library)

**Rationale:**
- `testing` package is Go's standard testing framework
- Supports table-driven tests (Go recommended pattern)
- Coverage reporting support
- No external dependencies required

**Usage Example:**
```go
func TestSign(t *testing.T) {
    tests := []struct {
        name       string
        data       string
        privateKey string
        expected   string
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Sign(tt.data, tt.privateKey)
            if err != nil {
                t.Errorf("Sign() error = %v", err)
            }
            if result != tt.expected {
                t.Errorf("Sign() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

**Alternatives Considered:**

| Framework | Pros | Cons | Decision |
|-----------|------|------|----------|
| testing | Standard library, zero deps | Simple assertion syntax | Chosen |
| testify | Rich assertions | External dependency | Optional |
| ginkgo | BDD style | Learning curve | Rejected |

---

### 7. Type System

**Decision:** Static typing + interface

**Rationale:**
- Go is a statically typed language, compile-time type checking
- Interface provides polymorphism and abstraction
- Generics support (Go 1.18+) for ApiResponse[T]

**Usage Example:**
```go
// Interface abstraction
type HttpTransport interface {
    Send(ctx context.Context, req *HttpRequest) (*HttpResponse, error)
}

// Generic response
type ApiResponse[T any] struct {
    Code    string `json:"code"`
    Message string `json:"message,omitempty"`
    Data    T      `json:"data,omitempty"`
}
```

---

### 8. Code Style

**Decision:** gofmt + golangci-lint

**Rationale:**
- gofmt: Go official formatting tool, no configuration debates
- golangci-lint: Meta-linter integrating multiple linters
- Unified code style, reduces review discussions

**Configuration (golangci.yml):**
```yaml
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - ineffassign
    - misspell
```

---

### 9. Framework Adaptation

**Decision:** Gin, Echo, Fiber, Chi

**Rationale:**
These are the top 4 Go web frameworks:

| Framework | GitHub Stars | Market Position |
|-----------|--------------|-----------------|
| Gin | ~75k | #1, most popular |
| Echo | ~28k | #2, clean API |
| Fiber | ~30k | #3, Express-like |
| Chi | ~16k | #4, lightweight |

**Integration Approach:**
- SDK is framework-agnostic (only depends on net/http)
- README provides integration examples for each framework
- Webhook handling supports http.Handler interface

**Integration Examples Provided:**

1. **Gin**
   - Middleware pattern
   - Raw body reading for webhook

2. **Echo**
   - Handler pattern
   - Request body binding

3. **Fiber**
   - Handler pattern
   - Fast body parsing

4. **Chi**
   - Standard http.Handler
   - Middleware chain

5. **net/http**
   - Standard library pattern
   - Basic http.HandlerFunc

**Frameworks NOT Prioritized:**
- Iris: Declining community
- Beego: Enterprise-focused, heavy
- Hertz: ByteDance internal, newer

---

## Decision Log

| Date | Decision | Change | Author |
|------|----------|--------|--------|
| 2026-02-03 | Initial ADR | Created with all decisions | SDK Team |

## Future Considerations

1. **Go 1.23+**: Evaluate new features (iterators, etc.)
2. **WebAssembly**: May support WASM runtime
3. **More Frameworks**: Add Hertz, Iris based on community demand
